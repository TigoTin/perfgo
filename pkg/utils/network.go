package utils

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
)

// NetResult 网络适配器信息结构
type NetResult struct {
	Name        string // 网络接口名称
	RouteNum    int    // 路由表编号
	InAddressV4 string // IPv4地址
	Mac         string // MAC地址
	NATType     string // NAT类型
	PublicIP    string // 公网IP地址
}

// removeDuplicates 移除重复的网络适配器信息
func removeDuplicates(list []NetResult) []NetResult {
	seen := make(map[string]bool)
	var result []NetResult

	for _, item := range list {
		key := item.Name + item.InAddressV4
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

// GetAllNetworkAdapterWithRoutes 获取所有网络适配器信息（包含路由策略表）
// 保持原有基于命令行的实现方式，优先获取路由策略表，回退到全局网卡
func GetAllNetworkAdapterWithRoutes() ([]NetResult, error) {
	// 实现原逻辑，使用命令行方式
	TableNumString, err := RunCmd(fmt.Sprintf("ip ru s | grep fwmark | awk '{print $7}' | sort"))

	if err != nil {
		fmt.Printf("err run get fwmark command\n")
		return nil, errors.New("no fwmark")
	}

	var NicList []NetResult
	var Nic, AddressInV4 string
	var TableNum int

	// 遍历所有路由策略表编号
	for _, NumString := range strings.Split(TableNumString, "\n") {
		if NumString != "" {
			TableNum, _ = strconv.Atoi(NumString)

			// 获取路由表中的信息
			mark, _ := RunCmd(fmt.Sprintf("ip route show table %d | awk '{print $2}'", TableNum))

			if mark == "" {
				continue
			} else {
				mark = strings.Split(mark, "\n")[0]
			}

			// 根据路由表信息判断如何获取网卡名称和IP地址
			if mark == "dev" {
				// 格式如: table 1000 proto static dev eth0
				Nic, _ = RunCmd(fmt.Sprintf("ip route show table %d | awk '{print $3}'", TableNum))
				Nic = strings.Split(Nic, "\n")[0]
				AddressInV4, _ = RunCmd(fmt.Sprintf("ip addr show dev %s | grep inet | grep -v inet6 | awk '{print $2}'", Nic))
			} else {
				// 格式如: table 1000 proto static scope link nhid 1041 dev eth1
				Nic, _ = RunCmd(fmt.Sprintf("ip route show table %d | awk '{print $5}'", TableNum))
				Nic = strings.Split(Nic, "\n")[0]
				AddressInV4, _ = RunCmd(fmt.Sprintf("ip addr show dev %s | grep inet | grep -v inet6 | awk '{print $2}' | awk -F/ '{print $1}'", Nic))
			}

			AddressInV4 = strings.Split(AddressInV4, "\n")[0]

			// 获取MAC地址
			MacAddress, _ := RunCmd(fmt.Sprintf("ip link show dev %s | grep \"link/ether\" | awk '{print $2}'", Nic))
			MacAddress = strings.Split(MacAddress, "\n")[0]

			// 添加到结果列表
			natType, publicIP, _ := DetectNATType(AddressInV4)
			NicList = append(NicList, NetResult{
				Name:        Nic,
				RouteNum:    TableNum,
				InAddressV4: AddressInV4,
				Mac:         MacAddress,
				NATType:     natType,
				PublicIP:    publicIP,
			})
		}
	}

	// 如果没有识别到任何的线路，则为软路由或者专线模式，回退到获取全局主网卡
	if len(NicList) <= 0 {
		NicAddr, _ := RunCmd(fmt.Sprintf("ip a  | grep \"scope global\" | grep -v docker0 | grep -v inet6 | grep -v 192.168.100.100 | awk '{print $2,$NF}'"))
		NicAddr = strings.Split(NicAddr, "\n")[0]
		NicAddrList := strings.Split(NicAddr, " ")
		NicAddr = strings.Split(NicAddrList[0], "/")[0]
		Nic = NicAddrList[1]

		MacAddress, err := RunCmd(fmt.Sprintf("ip link show dev %s | grep link | awk '{print $2}'", Nic))
		if err != nil {
			MacAddress = ""
		} else {
			MacAddress = strings.Split(MacAddress, "\n")[0]
		}

		natType, publicIP, _ := DetectNATType(NicAddr)
		NicList = append(NicList, NetResult{
			Name:        Nic,
			InAddressV4: NicAddr,
			Mac:         MacAddress,
			NATType:     natType,
			PublicIP:    publicIP,
		})
	}

	NicList = removeDuplicates(NicList)
	return NicList, nil
}

// RunCmd 模拟运行命令的函数（实际实现可能在其他地方）
func RunCmd(cmd string) (string, error) {
	// 这里应该是实际的命令执行逻辑
	// 为了演示，我们返回空字符串
	return "", nil
}

// GetAllNetworkAdapter 获取所有网络适配器信息
// 使用 netlink 库替代命令行方式，提高效率和可靠性
func GetAllNetworkAdapter() ([]NetResult, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	var nicList []NetResult

	for _, link := range links {
		attrs := link.Attrs()
		// 只处理处于 UP 状态的网络接口
		if attrs.Flags&net.FlagUp != 0 && attrs.EncapType != "loopback" {
			// 获取接口的 IP 地址
			addrList, err := netlink.AddrList(link, syscall.AF_INET)
			if err != nil {
				continue
			}

			for _, addr := range addrList {
				if !addr.IP.IsLoopback() && !addr.IP.IsLinkLocalUnicast() {
					natType, publicIP, _ := DetectNATType(addr.IP.String())
					nic := NetResult{
						Name:        attrs.Name,
						InAddressV4: addr.IP.String(),
						Mac:         attrs.HardwareAddr.String(),
						NATType:     natType,
						PublicIP:    publicIP,
					}
					nicList = append(nicList, nic)
				}
			}
		}
	}

	nicList = removeDuplicates(nicList)
	return nicList, nil
}
