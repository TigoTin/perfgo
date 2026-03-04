package utils

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/ccding/go-stun/stun"
)

// NetworkInterfaceInfo 网络接口信息结构
type NetworkInterfaceInfo struct {
	Name     string // 网卡名称
	IP       string // IP地址
	NATType  string // NAT类型
	IsOnline bool   // 是否联网
}

// GetAllNetworkInterfaceDetails 获取所有网络接口详细信息
func GetAllNetworkInterfaceDetails() ([]NetworkInterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var results []NetworkInterfaceInfo

	for _, iface := range interfaces {
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		info := NetworkInterfaceInfo{
			Name:     iface.Name,
			IP:       "",
			NATType:  "Unknown",
			IsOnline: false,
		}

		// 检查接口状态
		if iface.Flags&net.FlagUp != 0 {
			info.IsOnline = true
		} else {
			// 接口未激活，直接添加到结果中但标记为离线
			results = append(results, info)
			continue
		}

		// 获取接口的IP地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 查找有效的IPv4地址
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					// 跳过回环地址和链路本地地址
					if !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() && !ip4.IsLinkLocalMulticast() {
						info.IP = ip4.String()
						break
					}
				}
			}
		}

		// 如果有有效IP且接口在线，检测NAT类型
		if info.IP != "" && info.IsOnline {
			natType, publicIP, natErr := detectNATTypeForInterface(info.Name, info.IP)
			if natErr != nil {
				// NAT类型检测失败，记录错误但不影响整体流程
				info.NATType = "Unknown"
			} else {
				info.NATType = natType
				// 可以选择性地保存公网IP信息
				_ = publicIP
			}
		}

		results = append(results, info)
	}

	return results, nil
}

func DetectNATTypeForInterface(localIP string) (natType, publicIP string, err error) {
	return detectNATTypeForInterface("", localIP)
}

// detectNATTypeForInterface 检测特定接口的NAT类型
func detectNATTypeForInterface(interfaceName, localIP string) (natType, publicIP string, err error) {
	// 定义多个STUN服务器，按优先级排列
	stunServers := []string{
		"stun.qcloud.com:3478",
		"stun.miwifi.com:3478",
		"stun.ekiga.net:3478",
		"stun.ideasip.com:3478",
		"stun.voiparound.com:3478",
	}

	for _, serverAddr := range stunServers {
		client := stun.NewClient()
		// 设置 STUN 服务器
		client.SetServerAddr(serverAddr)

		// 设置本地IP
		if localIP != "" {
			client.SetLocalIP(localIP)
		}

		nat, pubIP, discoverErr := client.Discover()
		if discoverErr != nil {
			// 尝试下一个STUN服务器
			continue
		}

		// 检查是否为无效的NAT类型
		if nat == stun.NATError {
			// 尝试下一个STUN服务器
			continue
		}

		// 处理可能为nil的pubIP
		var publicIPStr string
		if pubIP != nil {
			publicIPStr = pubIP.String()
		} else {
			publicIPStr = ""
		}

		switch nat {
		case stun.NATFull:
			natType = "NAT1"
		case stun.NATRestricted:
			natType = "NAT2"
		case stun.NATPortRestricted:
			natType = "NAT3"
		case stun.NATSymetric:
			natType = "NAT4"
		default:
			natType = "Unknown"
		}

		// 成功检测到NAT类型，返回结果
		return natType, strings.Split(publicIPStr, ":")[0], nil
	}

	// 所有STUN服务器都失败
	return "Unknown", "", fmt.Errorf("无法通过任何STUN服务器检测NAT类型")
}

// FilterOnlineInterfaces 过滤出已联网的接口
func FilterOnlineInterfaces(interfaces []NetworkInterfaceInfo) []NetworkInterfaceInfo {
	var onlineInterfaces []NetworkInterfaceInfo
	for _, iface := range interfaces {
		if iface.IsOnline && iface.IP != "" {
			onlineInterfaces = append(onlineInterfaces, iface)
		}
	}
	return onlineInterfaces
}

// PrintNetworkInterfaceInfo 打印网络接口信息
func PrintNetworkInterfaceInfo(interfaces []NetworkInterfaceInfo) {
	fmt.Printf("%-20s %-15s %-15s %-10s\n", "网卡名称", "IP地址", "NAT类型", "联网状态")
	fmt.Printf("%-20s %-15s %-15s %-10s\n",
		"--------------------", "---------------", "---------------", "----------")

	for _, iface := range interfaces {
		status := "离线"
		if iface.IsOnline {
			status = "在线"
		}

		fmt.Printf("%-20s %-15s %-15s %-10s\n",
			iface.Name,
			iface.IP,
			iface.NATType,
			status)
	}
}

// GetOnlineNetworkInterfaces 获取已联网的网络接口信息
func GetOnlineNetworkInterfaces() ([]NetworkInterfaceInfo, error) {
	allInterfaces, err := GetAllNetworkInterfaceDetails()
	if err != nil {
		return nil, err
	}

	onlineInterfaces := FilterOnlineInterfaces(allInterfaces)

	// 按网卡名称排序
	sort.Slice(onlineInterfaces, func(i, j int) bool {
		return strings.ToLower(onlineInterfaces[i].Name) < strings.ToLower(onlineInterfaces[j].Name)
	})

	return onlineInterfaces, nil
}
