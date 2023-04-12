package network

import (
	"net"
	"testing"
)


func TestAllocate(t *testing.T) {
	_, subnet,_ := net.ParseCIDR("192.168.0.0/24")
	ip,_ := ipAllocator.Allocate(subnet)
	t.Logf("alloc ip: %v",ip)
}

func TestRelease(t *testing.T) {
	ip, subnet,_ := net.ParseCIDR("192.168.0.1/24")
	ipAllocator.Release(subnet,&ip)
	t.Logf("release ip: %v",ip)
}
