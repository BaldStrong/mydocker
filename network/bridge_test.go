package network

import (
	"example/mydocker/container"
	"fmt"
	"log"
	"testing"

	"github.com/vishvananda/netlink"
)

func TestBridgeInit(t *testing.T) {
	d := BridgeNetworkDriver{}
	_, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("err: %v", err)
}

func TestBridgeConnect(t *testing.T) {
	ep := Endpoint{
		ID: "testcontainer",
	}

	n := Network{
		Name: "testbridge",
	}

	d := BridgeNetworkDriver{}
	err := d.Connect(&n, &ep)
	t.Logf("err: %v", err)
}

func TestNetworkConnect(t *testing.T) {

	cInfo := &container.ContainerInfo{
		Id: "testcontainer",
		Pid: "15438",
	}

	d := BridgeNetworkDriver{}
	n, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("err: %v", n)

	Init()

	networks[n.Name] = n
	err = Connect(n.Name, cInfo)
	t.Logf("err: %v", err)
}

func TestLoad(t *testing.T) {
	n := Network{
		Name: "testbridge",
	}

	n.load("/var/run/mydocker/network/network/testbridge")

	t.Logf("network: %v", n)
}


func TestNet003(t *testing.T) {
    bridgeName := "testbridge"
    // 根据设备名找到设备testbridge
    br, err := netlink.LinkByName(bridgeName)
    if err != nil {
        log.Printf("LinkByName err:%v\n", err)
        return
    }

    la := netlink.NewLinkAttrs()
    la.Name = "12345"

    log.Printf("br.attrs().index:%d\n", br.Attrs().Index)
    // 等于 ip link set dev 12345 master testbridge
    la.MasterIndex = br.Attrs().Index

    myVeth := netlink.Veth{
        LinkAttrs: la,
        PeerName:  "cif-" + la.Name,
    }
    // 等于 ip link add 12345 type veth peer name cif-12345
    if err = netlink.LinkAdd(&myVeth); err != nil {
        fmt.Errorf("Error Add Endpoint Device: %v", err)
        return
    }

    // 等于 ip link set 12345 up
    if err = netlink.LinkSetUp(&myVeth); err != nil {
        fmt.Errorf("Error Add Endpoint Device: %v", err)
        return
    }
}