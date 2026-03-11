package utils

import (
	"fmt"
	"net"
	"sort"
	"strings"
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

		if info.IP != "" && info.IsOnline {
			natType, _, natErr := DetectNATType(info.IP)
			if natErr != nil {
				info.NATType = "Unknown"
			} else {
				info.NATType = natType
			}
		}

		results = append(results, info)
	}

	return results, nil
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
