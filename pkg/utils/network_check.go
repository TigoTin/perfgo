package utils

import (
	"fmt"
	"net"
	"time"

	"github.com/ccding/go-stun/stun"
)

// NetworkCheckResult 网络检测结果结构
type NetworkCheckResult struct {
	InterfaceName string  // 网卡名称
	IsConnected   bool    // 是否联网
	Latency       float64 // 延迟时间(ms)
	NATType       string  // NAT类型
	Bandwidth     string  // 压测带宽
	Account       string  // 宽带账号
	PublicIP      string  // 公网IP
}

// CheckNetworkInterface 检测网络接口状态
func CheckNetworkInterface(iface net.Interface) NetworkCheckResult {
	result := NetworkCheckResult{
		InterfaceName: iface.Name,
		IsConnected:   false,
		Latency:       0,
		NATType:       "Unknown",
		Bandwidth:     "N/A",
		Account:       "N/A",
		PublicIP:      "",
	}

	// 检查接口状态
	if iface.Flags&net.FlagUp == 0 {
		return result
	}

	// 检查是否有IPv4地址
	addrs, err := iface.Addrs()
	if err != nil {
		return result
	}

	hasIPv4 := false
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				hasIPv4 = true
				break
			}
		}
	}

	if !hasIPv4 {
		return result
	}

	result.IsConnected = true

	// 检测延迟 (ping 一个公共DNS服务器)
	result.Latency = measureLatency("8.8.8.8")

	// 检测NAT类型
	result.NATType, result.PublicIP, _ = detectNATTypeByInterface(iface.Name)

	return result
}

// measureLatency 测量网络延迟
func measureLatency(target string) float64 {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target+":80", 3*time.Second)
	if err != nil {
		return -1 // 表示无法连接
	}
	defer conn.Close()
	latency := time.Since(start).Seconds() * 1000 // 转换为毫秒
	return latency
}

// detectNATTypeByInterface 检测特定接口的NAT类型
func detectNATTypeByInterface(interfaceName string) (natType, publicIP string, err error) {
	// 获取指定接口的IP地址
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "Unknown", "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "Unknown", "", err
	}

	var localIP string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				localIP = ip4.String()
				break
			}
		}
	}

	if localIP == "" {
		return "Unknown", "", fmt.Errorf("no IPv4 address found for interface %s", interfaceName)
	}

	client := stun.NewClient()
	client.SetServerAddr("stun.l.google.com:19302")
	client.SetLocalIP(localIP)

	nat, pubIP, discoverErr := client.Discover()
	if discoverErr != nil {
		return "Unknown", "", discoverErr
	}

	return nat.String(), pubIP.String(), nil
}

// PrintNetworkCheckResults 打印网络检测结果表格
func PrintNetworkCheckResults(results []NetworkCheckResult) {
	fmt.Printf("%-15s %-10s %-12s %-15s %-15s %-15s\n", "网卡", "是否联网", "延迟时间(ms)", "NAT类型", "压测带宽", "宽带账号")
	fmt.Printf("%-15s %-10s %-12s %-15s %-15s %-15s\n",
		"---------------", "--------", "------------", "---------------", "---------------", "---------------")

	for _, result := range results {
		status := "否"
		if result.IsConnected {
			status = "是"
		}

		latencyStr := "N/A"
		if result.Latency >= 0 {
			latencyStr = fmt.Sprintf("%.2f", result.Latency)
		}

		fmt.Printf("%-15s %-10s %-12s %-15s %-15s %-15s\n",
			result.InterfaceName,
			status,
			latencyStr,
			result.NATType,
			result.Bandwidth,
			result.Account)
	}
}

// CheckAllNetworkInterfaces 检查所有网络接口
func CheckAllNetworkInterfaces() ([]NetworkCheckResult, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var results []NetworkCheckResult
	for _, iface := range interfaces {
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 跳过没有UP标志的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		result := CheckNetworkInterface(iface)
		results = append(results, result)
	}

	return results, nil
}
