package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetworkDriver) Create(subnet, name string) (*Network, error) {
	ip, IPRange, _ := net.ParseCIDR(subnet)
	IPRange.IP = ip
	// 创建一个网络
	n := &Network{
		Name:    name,
		IPRange: IPRange,
		Driver:  d.Name(),
	}
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("")
	}
	return n, err
}

func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("Getting link with name %s failed: %v", bridgeName, err)
	}
	if err := netlink.LinkDel(br); err != nil {
		return fmt.Errorf("Failed to remove bridge interface %s delete: %v", bridgeName, err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = endpoint.ID[:5]
	linkAttrs.MasterIndex = br.Attrs().Index

	endpoint.Device = netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  "cif-" + linkAttrs.Name,
	}
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("Error setup Endpoint Device: %v", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

func (d *BridgeNetworkDriver) initBridge(network *Network) error {
	bridgeName := network.Name
	// 1. 创建bridge虚拟设备
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error add bridge: %s, Error: %v", bridgeName, err)
	}
	gatewayIP := *network.IPRange
	gatewayIP.IP = network.IPRange.IP

	// 2. 设置bridge设备的地址和路由
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("Error assigning address: %s on bridge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}

	// 打开设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up: %s, Error: %v", bridgeName, err)
	}

	// 设置 iptables的SNAT规则
	if err := setupIPTables(bridgeName, network.IPRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}

func createBridgeInterface(bridgeName string) error {
	// 检查释放已存在同名设备
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}
	// 初始化一个netlink的Link基础对象，Link的名字即Bridge虚拟设备的名字
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = bridgeName
	
	//创建网桥，LinkAttrs代表大多数类型都共有的数据 
	br := &netlink.Bridge{LinkAttrs: linkAttrs}
	// 创建虚拟网络设备，相当于ip link add xxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

func setInterfaceIP(name, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("Abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error:", err)
	}
	// 由于netlink.ParseIPNet 是对 net.ParseCIDR 的一个封装，可以对ParseCIDR返回的IP和net进行整合
	// 返回值中的 ipNet 既包含了网段的信息（掩码），192.168.0.0/24，也包含了原始的IP：192.168.0.1
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	// netlink.AddrAdd相当于ip addr add xxx
	// 同时还会配置路由信息，相当于 route add -net 172.18.0.0/24 dev iface
	// 将ipnet网段的流量路由到iface上
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ]: %v", iface.Attrs().Name, err)
	}

	// 相当于 ip link set iface up
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	// 
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}
