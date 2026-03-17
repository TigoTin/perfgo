package client

import (
	"fmt"

	"github.com/TigoTin/perfgo/pkg/utils"
)

func RunTest(config utils.TestConfig) (*utils.TestResultSummary, error) {
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
		onlineInterfaces, err := utils.GetOnlineNetworkInterfaces()
		if err != nil {
			return nil, fmt.Errorf("获取在线网卡失败：%v", err)
		}
		for _, iface := range onlineInterfaces {
			if iface.IP != "" {
				config.LocalIPs = append(config.LocalIPs, iface.IP)
			}
		}
		if len(config.LocalIPs) == 0 {
			return nil, fmt.Errorf("未找到任何在线网卡")
		}
	}

	interfaceInfoMap := buildInterfaceInfoMap()
	result := &utils.TestResultSummary{
		Results: make([]utils.InterfaceResult, 0, len(config.LocalIPs)),
	}

	for _, localIP := range config.LocalIPs {
		interfaceResult := executeTestForInterface(localIP, config, interfaceInfoMap)
		result.Results = append(result.Results, interfaceResult)
	}

	return result, nil
}

func buildInterfaceInfoMap() map[string]utils.NetworkInterfaceInfo {
	interfaceInfoMap := make(map[string]utils.NetworkInterfaceInfo)
	interfaces, err := utils.GetAllNetworkInterfaceDetails()
	if err == nil {
		for _, iface := range interfaces {
			if iface.IP != "" {
				interfaceInfoMap[iface.IP] = iface
			}
		}
	}
	return interfaceInfoMap
}

func executeTestForInterface(localIP string, config utils.TestConfig, interfaceInfoMap map[string]utils.NetworkInterfaceInfo) utils.InterfaceResult {
	interfaceResult := utils.InterfaceResult{
		LocalIP: localIP,
	}

	if ifaceInfo, ok := interfaceInfoMap[localIP]; ok {
		interfaceResult.InterfaceName = ifaceInfo.Name
	}

	natType, publicIP, _ := utils.DetectNATType(localIP)
	interfaceResult.NATType = natType
	interfaceResult.PublicIP = publicIP

	switch config.TestType {
	case utils.TestTypeTCP:
		executeTCPTest(&interfaceResult, localIP, config)
	case utils.TestTypeUDP:
		executeUDPTest(&interfaceResult, localIP, config)
	default:
		interfaceResult.Success = false
		interfaceResult.Error = fmt.Sprintf("不支持的测试类型：%s", config.TestType)
	}

	return interfaceResult
}

func executeTCPTest(result *utils.InterfaceResult, localIP string, config utils.TestConfig) error {
	tcpResult, err := runTCPTest(localIP, config.ServerAddr, config.Concurrency, config.Duration)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return err
	}

	result.Success = true
	result.Throughput = tcpResult.Throughput
	result.ThroughputMbps = tcpResult.ThroughputMbps
	result.AvgRTT = tcpResult.AvgRTT
	result.AvgJitter = tcpResult.AvgJitter
	result.TotalBytes = tcpResult.TotalBytes
	result.Duration = tcpResult.Duration
	result.PacketLoss = tcpResult.PacketLoss
	return nil
}

func executeUDPTest(result *utils.InterfaceResult, localIP string, config utils.TestConfig) error {
	udpResult, err := runUDPTest(localIP, config.ServerAddr, config.Concurrency, config.Duration, config.Bandwidth)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return err
	}

	result.Success = true
	result.Throughput = udpResult.Throughput
	result.ThroughputMbps = udpResult.ThroughputMbps
	result.AvgRTT = udpResult.AvgRTT
	result.AvgJitter = udpResult.AvgJitter
	result.TotalBytes = udpResult.TotalBytes
	result.Duration = udpResult.Duration
	result.PacketLoss = udpResult.PacketLoss
	return nil
}

func runTCPTest(localIP, serverAddr string, connections, duration int) (*TCPTestResult, error) {
	tester := NewTCPTester()
	return tester.RunTCPTestWithResult(serverAddr, connections, duration, localIP)
}

func runUDPTest(localIP, serverAddr string, connections, duration int, bandwidth string) (*UDPTestResult, error) {
	tester := NewUDPTester()
	return tester.RunUDPTestWithResult(serverAddr, connections, duration, bandwidth, localIP)
}
