package client

import (
	"fmt"
	"sync"

	"github.com/TigoTin/perfgo/pkg/utils"
)

func RunTest(config utils.TestConfig) (*utils.TestResultSummary, error) {
	if config.Duration <= 0 {
		config.Duration = 10
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}
	if len(config.Servers) == 0 {
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
		executeMultiServerTCPTest(&interfaceResult, localIP, config)
	case utils.TestTypeUDP:
		executeMultiServerUDPTest(&interfaceResult, localIP, config)
	default:
		interfaceResult.Success = false
		interfaceResult.Error = fmt.Sprintf("不支持的测试类型：%s", config.TestType)
	}

	return interfaceResult
}

func executeMultiServerTCPTest(result *utils.InterfaceResult, localIP string, config utils.TestConfig) {
	servers := config.Servers
	if len(servers) == 0 {
		result.Success = false
		result.Error = "服务端列表为空"
		return
	}

	type tcpRes struct {
		result *TCPTestResult
		err    error
	}

	resultChan := make(chan tcpRes, len(servers))
	var wg sync.WaitGroup

	for _, server := range servers {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			tester := NewTCPTester()
			tcpResult, err := tester.RunTCPTestWithResult(addr, config.Concurrency, config.Duration, localIP)
			resultChan <- tcpRes{result: tcpResult, err: err}
		}(server.Addr)
	}

	wg.Wait()
	close(resultChan)

	var totalThroughput float64
	var totalBytes int64
	var totalDuration float64
	var successCount int
	var rttSum, jitterSum, packetLossSum float64
	var rttCount, jitterCount, packetLossCount int

	for r := range resultChan {
		if r.err != nil {
			continue
		}
		if r.result != nil {
			totalThroughput += r.result.Throughput
			totalBytes += r.result.TotalBytes
			totalDuration += r.result.Duration
			successCount++

			if r.result.AvgRTT > 0 {
				rttSum += r.result.AvgRTT
				rttCount++
			}
			if r.result.AvgJitter > 0 {
				jitterSum += r.result.AvgJitter
				jitterCount++
			}
			if r.result.PacketLoss >= 0 {
				packetLossSum += r.result.PacketLoss
				packetLossCount++
			}
		}
	}

	if successCount == 0 {
		result.Success = false
		result.Error = "所有服务端测试均失败"
		return
	}

	result.Success = true
	result.Throughput = totalThroughput
	result.ThroughputMbps = totalThroughput * 8 / 1000 / 1000
	result.TotalBytes = totalBytes
	if successCount > 0 {
		result.Duration = totalDuration / float64(successCount)
	}
	if rttCount > 0 {
		result.AvgRTT = rttSum / float64(rttCount)
	}
	if jitterCount > 0 {
		result.AvgJitter = jitterSum / float64(jitterCount)
	}
	if packetLossCount > 0 {
		result.PacketLoss = packetLossSum / float64(packetLossCount)
	}
}

func executeMultiServerUDPTest(result *utils.InterfaceResult, localIP string, config utils.TestConfig) {
	servers := config.Servers
	if len(servers) == 0 {
		result.Success = false
		result.Error = "服务端列表为空"
		return
	}

	type udpRes struct {
		result *UDPTestResult
		err    error
	}

	resultChan := make(chan udpRes, len(servers))
	var wg sync.WaitGroup

	for _, server := range servers {
		wg.Add(1)
		go func(addr, bandwidth string) {
			defer wg.Done()
			tester := NewUDPTester()
			udpResult, err := tester.RunUDPTestWithResult(addr, config.Concurrency, config.Duration, bandwidth, localIP)
			resultChan <- udpRes{result: udpResult, err: err}
		}(server.Addr, server.Bandwidth)
	}

	wg.Wait()
	close(resultChan)

	var totalThroughput float64
	var totalBytes int64
	var totalDuration float64
	var successCount int
	var rttSum, jitterSum, packetLossSum float64
	var rttCount, jitterCount, packetLossCount int

	for r := range resultChan {
		if r.err != nil {
			continue
		}
		if r.result != nil {
			totalThroughput += r.result.Throughput
			totalBytes += r.result.TotalBytes
			totalDuration += r.result.Duration
			successCount++

			if r.result.AvgRTT > 0 {
				rttSum += r.result.AvgRTT
				rttCount++
			}
			if r.result.AvgJitter > 0 {
				jitterSum += r.result.AvgJitter
				jitterCount++
			}
			if r.result.PacketLoss >= 0 {
				packetLossSum += r.result.PacketLoss
				packetLossCount++
			}
		}
	}

	if successCount == 0 {
		result.Success = false
		result.Error = "所有服务端测试均失败"
		return
	}

	result.Success = true
	result.Throughput = totalThroughput
	result.ThroughputMbps = totalThroughput * 8 / 1000 / 1000
	result.TotalBytes = totalBytes
	if successCount > 0 {
		result.Duration = totalDuration / float64(successCount)
	}
	if rttCount > 0 {
		result.AvgRTT = rttSum / float64(rttCount)
	}
	if jitterCount > 0 {
		result.AvgJitter = jitterSum / float64(jitterCount)
	}
	if packetLossCount > 0 {
		result.PacketLoss = packetLossSum / float64(packetLossCount)
	}
}
