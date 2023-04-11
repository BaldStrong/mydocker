package network

import (
	"encoding/json"
	"example/mydocker/container"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

type Network struct {
	Name    string
	IPRange *net.IPNet
	Driver  string
}

type NetworkDriver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(network *Network) error
	Connect(network *Network, endpoint *Endpoint) error
	Disconnect(network *Network, endpoint *Endpoint) error
}

// 网络信息写到文件中
func (nw *Network) dump(dumpDir string) error {
	if _, err := os.Stat(dumpDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpDir, 0644)
		} else {
			return err
		}
	}

	nwPath := path.Join(dumpDir, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("OpenFile:%s error:%v", nwPath, err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)
	if err != nil {
		log.Errorf("Marshal error:%v", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		log.Errorf("Write error:%v", err)
		return err
	}
	return nil
}

func (nw *Network) load(dumpPath string) error {

	nwFile, err := os.Open(dumpPath)
	if err != nil {
		log.Errorf("Open:%s error:%v", dumpPath, err)
		return err
	}
	defer nwFile.Close()

	nwJson := make([]byte, 2000)
	n, err := nwFile.Read(nwJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		log.Errorf("Marshal error:%v", err)
		return err
	}
	return nil
}

func CreateNetwork(driver, subnet, name string) error {
	// ParseCIDR 是 Galang net 包的函数，功能是将网段的字符串转换成 net.IPNet 的对象
	_, cidr, _ := net.ParseCIDR(subnet)
	//  通过 IPAM 分配网关 IP，获取到网段中第一个 IP 作为网关的IP
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip

	/* 调用指定的网络驱动创建网络，这里的 drivers 字典是各个网络驱动的实例字典 ，
	通过调用网络驱动的 Create 方法创建网络，后面会以 Bridge 驱动为例介绍它的实现*/
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	return nw.dump(defaultNetworkPath)
}

// ------------------创建容器并连接网络

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such network: %s", networkName)
	}

	ip, err := ipAllocator.Allocate(nw.IPRange)
	if err != nil {
		return err
	}

	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress:   ip,
		Network:     nw,
		PortMapping: cinfo.PortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = drivers[nw.Driver].Connect(nw, ep); err != nil {
		return err
	}
	// 到容器的namespace配置容器网络设备IP地址
	if err = configEndpointAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	return configPortMapping(ep, cinfo)
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
	return nil
}

// ---------------------------展示网络列表

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IPRange.String(),
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

func Init() error {
	var bridgeDriver = BridgeNetworkDrvier{}
	drivers[bridgeDriver.Name()] = &bridgeDriver
	// 创建网络默认配置目录
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		// 如果是目录则跳过
		// if info.IsDir() {
		// 	return nil
		// }
		if strings.HasSuffix(nwPath, "/") {
			return nil
		}
		_, nwName := path.Split(nwPath)
		nw := &Network{Name: nwName}

		if err := nw.load(nwPath); err != nil {
			log.Errorf("error load network: %s", err)
		}

		networks[nwName] = nw
		return nil
	})

	log.Infof("networks: %v", networks)
	return nil
}

// ---------------------------删除网络

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such network: %s", networkName)
	}

	if err := ipAllocator.Release(nw.IPRange, &nw.IPRange.IP); err != nil {
		return fmt.Errorf("Error Remove Network gateway ip: %s", err)
	}

	// 调用网络驱动挂载和配置网络端点
	if err := drivers[nw.Driver].Delete(nw); err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %s", err)
	}
	return nw.remove(defaultNetworkPath)
}


func (nw *Network) remove(dumpDir string) error {
	nwPath := path.Join(dumpDir, nw.Name)
	if _, err := os.Stat(nwPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(nwPath)
	}
}