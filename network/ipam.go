package network

import (
	"encoding/json"
	"net"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.json"

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string // 网段和位图算法的数组map, key是网段，value是分配的位图数组
}

// 初始化一个 IPAM 的对象，默认使用/var/run/mydocker/network/ipam/subnet.json作为分配信息存储位置
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网段地址分配信息
func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("error dump allocation info, %v", err)
		return err
	}
	return nil
}

func (ipam *IPAM) dump() error {
	// path.Split 函数能够分隔目录和文件
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}

	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("OpenFile:%s error:%v", ipam.SubnetAllocatorPath, err)
		return err
	}
	defer subnetConfigFile.Close()

	subnetConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		log.Errorf("subnetConfigJson Marshal error:%v", err)
		return err
	}

	_, err = subnetConfigFile.Write(subnetConfigJson)
	if err != nil {
		log.Errorf("subnetConfigFile Write error:%v", err)
		return err
	}
	return nil
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	err = ipam.load()
	if err != nil {
		log.Errorf("error dump allocation info:%v", err)
	}
	subnetStr := subnet.String()
	// one是前导1的位数，size是掩码总位数
	// 比如"127.0.0.0/8" 网段的子网掩码是255.0.0.0
	// 那么one=8(11111111.0.0.0)，size=24
	one, size := subnet.Mask.Size()

	// 如果之前没有分配过这个网段，则初始化网段，全置为0
	if _, exist := (*ipam.Subnets)[subnetStr]; !exist {
		// 将0重复2^(size-one)次，赋值
		(*ipam.Subnets)[subnetStr] = strings.Repeat("0", 1<<uint8(size-one))
	}

	// 遍历网段的位图数组，最多遍历2^(size-one)次
	for c := range (*ipam.Subnets)[subnetStr] {
		// 找到未分配的ip
		if (*ipam.Subnets)[subnetStr][c] == '0' {
			// Go的字符串，创建之后就不能修改，所以通过转换成byte数组，修改后再转换成字符串赋值
			ipalloc := []byte((*ipam.Subnets)[subnetStr])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnetStr] = string(ipalloc)
			// 这里的IP为初始IP，比如对于网段 192.168.0.0/16 ，这里就是192.168.0.0
			ip = subnet.IP
			/* 上面的subnet.IP是基准地址，下面根据偏移c，获取最终分配的ip地址:
			通过网段的 IP 与上面的偏移相加计算出分配的 IP 地址，由于IP地址是uint的一个数组，需要通过数组中的每一项加所需要的值，
			比如网段是172.16.0.0/12，数组序号c是65555，那么在[172,16,0,0]上依次加[uint8(65555>>24),uint8(65555>>16),uint8(65555>>8),uint8(65555>>0)]
			由于是uint8，ip地址一共有32位，由四个点分十进制数字组成，每个数字占8位，所以只保留最后8位，即[0,1,0,19]，那么最终得到的ip为172.17.0.19
			*/
			for t := uint(4); t > 0; t-- {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			// 由于此处IP是从1开始分配的，所以最后再加1，最终得到分配的IP是172.17.0.20
			// 子网的第一个地址不可用，所以每次拿到的地址最后一个分组要加1
			ip[3]++
			break
		}
	}
	ipam.dump()
	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	err := ipam.load()
	if err != nil {
		log.Errorf("error dump allocation info:%v", err)
	}

	c := 0
	// 转换成4个字节的表示方式
	releaseIP := ipaddr.To4()
	// 由于之前分配时+1，现在需要重新计算偏移，所以先-1
	releaseIP[3]--
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	// 将位图数组中对应偏移位置置0，变成未分配状态
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)
	// 修改配置文件
	ipam.dump()
	return nil
}
