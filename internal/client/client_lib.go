package client

import (
	"fmt"

	"github.com/TigoTin/perfgo/pkg/utils"
)

type TestType string

const (
	TestTypeTCP TestType = "tcp"
	TestTypeUDP TestType = "udp"
)

type TestConfig struct {
	ServerAddr  string   `json:"server_addr"` // 服务端地址 (IP:端口)
	LocalIPs    []string `json:"local_ips"`   // 本地网卡IP列表 (支持多个)
	Duration    int      `json:"duration"`    // 测试持续时间 (秒)
	Concurrency int      `json:"concurrency"` // 并发连接数
	TestType    TestType `json:"test_type"`   // 测试类型: tcp 或 udp
	Bandwidth   string   `json:"bandwidth"`   // 仅UDP有效，目标带宽如 "10M"
}

type InterfaceResult struct {
	LocalIP        string  `json:"local_ip"`        // 本地网卡IP
	InterfaceName  string  `json:"interface_name"`  // 网卡名称
	NATType        string  `json:"nat_type"`        // NAT类型
	PublicIP       string  `json:"public_ip"`       // 公网IP
	Success        bool    `json:"success"`         // 测试是否成功
	Error          string  `json:"error,omitempty"` // 错误信息
	Throughput     float64 `json:"throughput"`      // 吞吐量 (bytes/s)
	ThroughputMbps float64 `json:"throughput_mbps"` // 吞吐量 (Mbps)
	AvgRTT         float64 `json:"avg_rtt_ms"`      // 平均延迟 (ms)
	AvgJitter      float64 `json:"avg_jitter_ms"`   // 平均抖动 (ms)
	TotalBytes     int64   `json:"total_bytes"`     // 总传输字节数
	Duration       float64 `json:"duration_sec"`    // 测试持续时间 (秒)
}

type TestResult struct {
	Results []InterfaceResult `json:"results"` // 每个网卡的测试结果
}

func RunTest(config TestConfig) (*TestResult, error) {
	if config.Duration <= 0 {
		config.Duration = 10
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}
	if config.ServerAddr == "" {
		return nil, fmt.Errorf("服务端地址不能为空")
	}
	if len(config.LocalIPs) == 0 {
		return nil, fmt.Errorf("本地网卡IP列表不能为空")
	}

	interfaceInfoMap := make(map[string]utils.NetworkInterfaceInfo)
	interfaces, err := utils.GetAllNetworkInterfaceDetails()
	if err == nil {
		for _, iface := range interfaces {
			if iface.IP != "" {
				interfaceInfoMap[iface.IP] = iface
			}
		}
	}

	publicIPMap := make(map[string]string)
	natTypeMap := make(map[string]string)
	for _, localIP := range config.LocalIPs {
		natType, publicIP, err := utils.DetectNATType(localIP)
		if err == nil {
			natTypeMap[localIP] = natType
			publicIPMap[localIP] = publicIP
		}
	}

	result := &TestResult{
		Results: make([]InterfaceResult, 0, len(config.LocalIPs)),
	}

	for _, localIP := range config.LocalIPs {
		interfaceResult := InterfaceResult{
			LocalIP: localIP,
		}

		if ifaceInfo, ok := interfaceInfoMap[localIP]; ok {
			interfaceResult.InterfaceName = ifaceInfo.Name
		}

		if natType, ok := natTypeMap[localIP]; ok {
			interfaceResult.NATType = natType
		}
		if publicIP, ok := publicIPMap[localIP]; ok {
			interfaceResult.PublicIP = publicIP
		}

		switch config.TestType {
		case TestTypeTCP:
			tcpResult, err := runTCPTest(localIP, config.ServerAddr, config.Concurrency, config.Duration)
			if err != nil {
				interfaceResult.Success = false
				interfaceResult.Error = err.Error()
			} else {
				interfaceResult.Success = true
				interfaceResult.Throughput = tcpResult.Throughput
				interfaceResult.ThroughputMbps = tcpResult.ThroughputMbps
				interfaceResult.AvgRTT = tcpResult.AvgRTT
				interfaceResult.AvgJitter = tcpResult.AvgJitter
				interfaceResult.TotalBytes = tcpResult.TotalBytes
				interfaceResult.Duration = tcpResult.Duration
			}
		case TestTypeUDP:
			udpResult, err := runUDPTest(localIP, config.ServerAddr, config.Concurrency, config.Duration, config.Bandwidth)
			if err != nil {
				interfaceResult.Success = false
				interfaceResult.Error = err.Error()
			} else {
				interfaceResult.Success = true
				interfaceResult.Throughput = udpResult.Throughput
				interfaceResult.ThroughputMbps = udpResult.ThroughputMbps
				interfaceResult.AvgRTT = udpResult.AvgRTT
				interfaceResult.AvgJitter = udpResult.AvgJitter
				interfaceResult.TotalBytes = udpResult.TotalBytes
				interfaceResult.Duration = udpResult.Duration
			}
		default:
			interfaceResult.Success = false
			interfaceResult.Error = fmt.Sprintf("不支持的测试类型: %s", config.TestType)
		}

		result.Results = append(result.Results, interfaceResult)
	}

	return result, nil
}

func runTCPTest(localIP, serverAddr string, connections, duration int) (*TCPTestResult, error) {
	tester := NewTCPTester()
	return tester.RunTCPTestWithResult(serverAddr, connections, duration, localIP)
}

func runUDPTest(localIP, serverAddr string, connections, duration int, bandwidth string) (*UDPTestResult, error) {
	tester := NewUDPTester()
	return tester.RunUDPTestWithResult(serverAddr, connections, duration, bandwidth, localIP)
}
