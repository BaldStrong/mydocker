package network

import (
	"encoding/json"
	"example/mydocker/container"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`  // 网络端点的ip
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network					  // 网络端点所属的网络
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

func CreateNetwork(driver, cidr, name string) error {
	// ParseCIDR 是 Galang net 包的函数，功能是将网段的字符串转换成 net.IPNet 的对象
	// For example, ParseCIDR("192.0.2.1/24") returns the IP address 192.0.2.1 and the network 192.0.2.0/24.
	// CIDR记法："192.0.2.1/24"
	_, subnet, _ := net.ParseCIDR(cidr)
	// fmt.Println("CIDR return",ip,subnet)
	// 通过 IPAM 分配网关 IP，获取到网段中第一个 IP 作为网关的IP
	// 这里没有考虑 192.168.10.4/24表示192.168.10.4为第一个可分配的ip，
	// 而是直接从整个子网的第一个可分配地址开始，所以一个分配到的是192.168.10.1
	ip, err := ipAllocator.Allocate(subnet)
	// fmt.Println("ipAllocator return",ip,subnet.IP)
	if err != nil {
		return err
	}
	subnet.IP = ip

	/* 调用指定的网络驱动创建网络，这里的 drivers 字典是各个网络驱动的实例字典 ，
	通过调用网络驱动的 Create 方法创建网络，后面会以 Bridge 驱动为例介绍它的实现*/
	nw, err := drivers[driver].Create(subnet.String(), name)
	if err != nil {
		return err
	}
	return nw.dump(defaultNetworkPath)
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
	return nil
}

// ---------------------------展示网络列表

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIPRange\tDriver\n")
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
	var bridgeDriver = BridgeNetworkDriver{}
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
		return fmt.Errorf("error Remove Network DriverError: %s", err)
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




// ------------------创建容器并连接网络

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such network: %s", networkName)
	}
	// 给容器分配一个ip
	ip, err := ipAllocator.Allocate(nw.IPRange)
	if err != nil {
		return err
	}
	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress:   ip,
		Network:     nw,
		PortMapping: cinfo.PortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = drivers[nw.Driver].Connect(nw, ep); err != nil {
		return fmt.Errorf("drivers[nw.Driver].Connect(nw, ep) failed:%v",err)
	}
	// 到容器的namespace中配置容器网络设备IP地址
	if err = configEndpointAddressAndRoute(ep, cinfo); err != nil {
		return fmt.Errorf("configEndpointAddressAndRoute failed:%v",err)
	}
	// 配置容器和宿主机的端口映射
	return configPortMapping(ep, cinfo)
}

func configEndpointAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 这里切记是PeerName，将peer这端绑定在容器内，这样在容器内看到的是cif-81233，在宿主机看到的类似：81233@if51
	// LinkByName是最大前缀匹配寻找的，所以只指定@前面的就可以
	peerlink,err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	defer enterContainerNetns(&peerlink,cinfo)()
	/*
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net",cinfo.Pid),os.O_RDONLY,0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFd := f.Fd()
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go 语言的 goroutine是轻量级线程，可能会被调度到别的系统级线程上去
	// 就不能保证一直在所需要的net namespace中了
	// 所以调用 runtime.LockOSThread 时要先锁定当前程序执行的线程
	runtime.LockOSThread()

	// 将veth peer另一端移到容器的ns中
	if err= netlink.LinkSetNsFd(peerlink,int(nsFd)) ;err != nil {
		log.Errorf("set link netns failed: %v", err)
	}
	// 获取当前进程的net namespace
	origNS,err := netns.Get()
	if err != nil {
		log.Errorf("get origNS failed: %v", err)
	}
	// 设置当前进程到新的net namespace，
	//  nsFd是uintptr，netns.NsHandle()将其转换为int
	// netns.NsHandle(nsFd)根据nsFd获取net namespace句柄
	if err = netns.Set(netns.NsHandle(nsFd)); err != nil {
		log.Errorf("set netns failed: %v",err)
	}
	defer func() {
		netns.Set(origNS)
		origNS.Close()
		runtime.UnlockOSThread()
		f.Close()
	}()
	*/

	interfaceIP := *ep.Network.IPRange
	interfaceIP.IP = ep.IPAddress
	if err= setInterfaceIP(ep.Device.PeerName,interfaceIP.String()) ;err != nil {
		return fmt.Errorf("ep.Network: %v, setInterfaceIP failed: %s", ep.Network, err)
	}
	if err= setInterfaceUP(ep.Device.PeerName) ;err != nil {
		return fmt.Errorf("ep.Network: %v, setInterfaceUP failed: %s", ep.Network, err)
	}
	// 设置loop back，使自己能连通自己
	if err= setInterfaceUP("lo") ;err != nil {
		return err
	}
	_, cidr,_ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerlink.Attrs().Index,
		Gw: ep.Network.IPRange.IP,
		Dst: cidr,
	}

	if err= netlink.RouteAdd(defaultRoute) ;err != nil {
		return fmt.Errorf("RouteAdd failed: %s", err)
	}
	return nil
}

func enterContainerNetns(enterLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net",cinfo.Pid),os.O_RDONLY,0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFd := f.Fd()
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go 语言的 goroutine是轻量级线程，可能会被调度到别的系统级线程上去
	// 就不能保证一直在所需要的net namespace中了
	// 所以调用 runtime.LockOSThread 时要先锁定当前程序执行的线程
	runtime.LockOSThread()

	// 将veth peer另一端移到容器的ns中
	if err= netlink.LinkSetNsFd(*enterLink,int(nsFd)) ;err != nil {
		log.Errorf("set link netns failed: %v", err)
	}
	// 获取当前进程的net namespace
	origNS,err := netns.Get()
	if err != nil {
		log.Errorf("get origNS failed: %v", err)
	}
	// 设置当前进程到新的net namespace，
	//  nsFd是uintptr，netns.NsHandle()将其转换为int
	// netns.NsHandle(nsFd)根据nsFd获取net namespace句柄
	if err = netns.Set(netns.NsHandle(nsFd)); err != nil {
		log.Errorf("set netns failed: %v",err)
	}

	// 在容器的网络空间中，执行完容器配置之后调用此函数就可以将程序恢复到原来的Net Namespace
	return func() {
		netns.Set(origNS)
		origNS.Close()
		runtime.UnlockOSThread()
		f.Close()
	}	
}

// 通过iptables的DNAT规则来实现宿主机上的请求转发到容器上
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _,pm := range ep.PortMapping {
		portMapping := strings.Split(pm,":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error: %v",pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
								portMapping[0],ep.IPAddress.String(),portMapping[1])
		cmd := exec.Command("iptables",strings.Split(iptablesCmd," ")...)
		output,err := cmd.Output()
		if err != nil {
			log.Errorf("iptables output: %v",output)
			continue
		}
	}
	return nil
}