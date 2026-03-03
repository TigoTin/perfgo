package client

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/TigoTin/perfgo/pkg/protocol"
	"github.com/TigoTin/perfgo/pkg/utils"
)

const (
	DefaultTimeout    = 30 * time.Second
	DefaultBufferSize = 32 * 1024 // 32KB
	DefaultPingCount  = 10
	channelBufferSize = 1
)

type connResult struct {
	bandwidth *utils.TestResult
	latency   *utils.TestResult
	err       error
}

func dialTCP(serverAddr, localIP string) (net.Conn, error) {
	if localIP != "" {
		localAddr, err := net.ResolveTCPAddr("tcp", localIP+":0")
		if err != nil {
			return nil, fmt.Errorf("解析本地TCP地址失败: %v", err)
		}
		remoteAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
		if err != nil {
			return nil, fmt.Errorf("解析远程TCP地址失败: %v", err)
		}
		dialer := net.Dialer{LocalAddr: localAddr}
		return dialer.Dial("tcp", remoteAddr.String())
	}
	return net.Dial("tcp", serverAddr)
}

func extractHost(serverAddr string) string {
	if idx := strings.LastIndex(serverAddr, ":"); idx != -1 {
		return serverAddr[:idx]
	}
	return serverAddr
}

type TCPTester struct {
	Timeout         time.Duration
	bandwidthResult *utils.TestResult
	latencyResult   *utils.TestResult
}

func NewTCPTester() *TCPTester {
	return &TCPTester{
		Timeout: DefaultTimeout,
	}
}

func (ct *TCPTester) RunTCPTest(serverAddr string, connections int, duration int, localIP string, interfaceName string) error {
	if interfaceName == "all" {
		return ct.runTCPTestOnAllInterfacesAggregated(serverAddr, connections, duration)
	}

	result, err := ct.RunTCPTestWithResult(serverAddr, connections, duration, localIP)
	if err != nil {
		return err
	}

	fmt.Printf("\n========== 聚合测试结果 (%d个连接) ==========\n", connections)
	fmt.Printf("吞吐量: %.2f MB/s (%.2f Mbps)\n", result.Throughput/1024/1024, result.ThroughputMbps)
	fmt.Printf("平均延迟: %.2f ms\n", result.AvgRTT)
	fmt.Printf("平均抖动: %.2f ms\n", result.AvgJitter)
	fmt.Printf("总传输字节: %d bytes\n", result.TotalBytes)
	fmt.Printf("测试时长: %.2f 秒\n", result.Duration)

	return nil
}

func (ct *TCPTester) RunTCPTestWithResult(serverAddr string, connections int, duration int, localIP string) (*TCPTestResult, error) {
	type connResult struct {
		bandwidth *utils.TestResult
		latency   *utils.TestResult
		err       error
	}

	resultChan := make(chan connResult, connections)
	var wg sync.WaitGroup

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			conn, err := dialTCP(serverAddr, localIP)
			if err != nil {
				resultChan <- connResult{err: err}
				return
			}
			defer conn.Close()

			msg := &protocol.Message{
				Type:     protocol.TypeTestRequest,
				TestType: "tcp",
				Payload: map[string]interface{}{
					"test_type": "tcp",
					"threads":   1,
					"duration":  duration,
				},
			}

			if err := msg.Send(conn); err != nil {
				resultChan <- connResult{err: err}
				return
			}

			bwResultChan := make(chan *utils.TestResult, channelBufferSize)
			latResultChan := make(chan *utils.TestResult, channelBufferSize)
			errChan := make(chan error, 2)

			targetHost := extractHost(serverAddr)

			go func() {
				bwResult, err := ct.runSingleConnectionBandwidthTest(conn, duration)
				if err != nil {
					errChan <- err
					bwResultChan <- nil
				} else {
					bwResultChan <- bwResult
				}
			}()

			go func() {
				latResult, err := ct.runSingleConnectionLatencyTest(targetHost)
				if err != nil {
					errChan <- err
					latResultChan <- nil
				} else {
					latResultChan <- latResult
				}
			}()

			var bwResult *utils.TestResult
			var latResult *utils.TestResult

			for j := 0; j < 2; j++ {
				select {
				case bwRes := <-bwResultChan:
					bwResult = bwRes
				case latRes := <-latResultChan:
					latResult = latRes
				case err := <-errChan:
					resultChan <- connResult{err: err}
					return
				}
			}

			resultChan <- connResult{bandwidth: bwResult, latency: latResult}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var totalBytes int64
	var totalDuration float64
	var latencies []float64
	var jitters []float64
	successCount := 0

	for result := range resultChan {
		if result.err != nil {
			continue
		}
		if result.bandwidth != nil {
			totalBytes += result.bandwidth.TotalBytes
			totalDuration += result.bandwidth.Duration
			successCount++
		}
		if result.latency != nil {
			latencies = append(latencies, result.latency.AvgRTT)
			jitters = append(jitters, result.latency.AvgJitter)
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("所有连接均失败")
	}

	avgDuration := totalDuration / float64(successCount)
	throughput := float64(totalBytes) / avgDuration

	var avgRTT, avgJitter float64
	if len(latencies) > 0 {
		for _, rtt := range latencies {
			avgRTT += rtt
		}
		avgRTT /= float64(len(latencies))

		for _, jitter := range jitters {
			avgJitter += jitter
		}
		avgJitter /= float64(len(jitters))
	}

	return &TCPTestResult{
		Throughput:     throughput,
		ThroughputMbps: throughput * 8 / 1000 / 1000,
		AvgRTT:         avgRTT,
		AvgJitter:      avgJitter,
		TotalBytes:     totalBytes,
		Duration:       avgDuration,
	}, nil
}

type TCPTestResult struct {
	Throughput     float64
	ThroughputMbps float64
	AvgRTT         float64
	AvgJitter      float64
	TotalBytes     int64
	Duration       float64
}

func (ct *TCPTester) runSingleConnectionBandwidthTest(conn net.Conn, duration int) (*utils.TestResult, error) {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)

	conn.SetWriteDeadline(endTime)

	var totalBytes int64

	data := make([]byte, DefaultBufferSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	for {
		n, err := conn.Write(data)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err != nil {
			break
		}
	}

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second
	}
	throughput := float64(totalBytes) / elapsed.Seconds()

	return &utils.TestResult{
		Protocol:   "TCP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}, nil
}

func (ct *TCPTester) runSingleConnectionLatencyTest(target string) (*utils.TestResult, error) {
	pingResult, err := utils.PingTarget(target, DefaultPingCount)
	if err != nil {
		return nil, fmt.Errorf("ping测试失败: %v", err)
	}

	if !pingResult.Success {
		return nil, fmt.Errorf("ping测试无响应")
	}

	return &utils.TestResult{
		Protocol:    "PING",
		TestType:    "latency",
		Direction:   "round-trip",
		AvgRTT:      pingResult.Latency,
		AvgJitter:   pingResult.Jitter,
		SuccessRate: (100 - pingResult.PacketLoss) / 100,
		Duration:    pingResult.Latency * DefaultPingCount,
	}, nil
}

func (ct *TCPTester) runTCPLatencyTest(serverAddr string) error {
	targetHost := extractHost(serverAddr)

	pingResult, err := utils.PingTarget(targetHost, DefaultPingCount)
	if err != nil {
		return fmt.Errorf("ping测试失败: %v", err)
	}

	if !pingResult.Success {
		return fmt.Errorf("ping测试无响应")
	}

	result := utils.TestResult{
		Protocol:    "PING",
		TestType:    "latency",
		Direction:   "round-trip",
		AvgRTT:      pingResult.Latency,
		AvgJitter:   pingResult.Jitter,
		SuccessRate: (100 - pingResult.PacketLoss) / 100,
		Duration:    pingResult.Latency * DefaultPingCount,
	}
	ct.latencyResult = &result

	return nil
}

// runTCPTestOnAllInterfacesAggregated 在所有在线网络接口上执行TCP测试并聚合结果
func (ct *TCPTester) runTCPTestOnAllInterfacesAggregated(serverAddr string, connections int, duration int) error {
	fmt.Println("正在获取所有在线网络接口...")
	interfaces, err := utils.GetOnlineNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("获取在线网络接口失败: %v", err)
	}

	if len(interfaces) == 0 {
		return fmt.Errorf("未找到任何在线网络接口")
	}

	fmt.Printf("发现 %d 个在线网络接口，开始逐个测试并聚合结果:\n\n", len(interfaces))

	// 存储每个接口的测试结果
	var results []utils.InterfaceTestResult

	for i, iface := range interfaces {
		fmt.Printf("=== 测试第 %d/%d 个网络接口: %s (IP: %s, NAT类型: %s) ===\n",
			i+1, len(interfaces), iface.Name, iface.IP, iface.NATType)

		// 使用接口的IP地址进行测试
		result := utils.TestResult{}
		tempResults := ct.runSingleInterfaceTestDetailed(iface.IP, serverAddr, connections, duration)

		// 计算单个接口的聚合结果
		var totalBytes int64
		var totalDuration float64
		var latencies []float64
		var jitters []float64
		successCount := 0

		for _, res := range tempResults {
			if res.err != nil {
				continue
			}
			if res.bandwidth != nil {
				totalBytes += res.bandwidth.TotalBytes
				totalDuration += res.bandwidth.Duration
				successCount++
			}
			if res.latency != nil {
				latencies = append(latencies, res.latency.AvgRTT)
				jitters = append(jitters, res.latency.AvgJitter)
			}
		}

		if successCount > 0 {
			avgDuration := totalDuration / float64(successCount)
			aggregateThroughput := float64(totalBytes) / avgDuration

			// 计算平均延迟和抖动
			var avgRTT, avgJitter float64
			if len(latencies) > 0 {
				for _, rtt := range latencies {
					avgRTT += rtt
				}
				avgRTT /= float64(len(latencies))

				for _, jitter := range jitters {
					avgJitter += jitter
				}
				avgJitter /= float64(len(jitters))
			}

			result = utils.TestResult{
				Protocol:   "TCP",
				TestType:   "combined",
				Direction:  "uplink",
				Throughput: aggregateThroughput,
				TotalBytes: totalBytes,
				AvgRTT:     avgRTT,
				AvgJitter:  avgJitter,
				Duration:   avgDuration,
			}

			fmt.Printf("接口 %s 测试完成\n", iface.Name)
		} else {
			fmt.Printf("接口 %s 测试失败: 所有连接均失败\n", iface.Name)
		}

		interfaceResult := utils.InterfaceTestResult{
			TestResult:    result,
			InterfaceName: iface.Name,
			NATType:       iface.NATType,
			Error:         nil,
		}
		if successCount == 0 {
			interfaceResult.Error = fmt.Errorf("所有连接均失败")
		}
		results = append(results, interfaceResult)
		fmt.Println()
	}

	// 输出每个接口的详细结果
	fmt.Println("========== 各网络接口详细测试结果 ==========")
	// fmt.Printf("%-15s %-20s %-15s %-20s %-15s %-15s\n", "网卡名称", "NAT类型", "协议", "吞吐量(B/s)", "平均RTT(ms)", "平均抖动(ms)")
	// fmt.Printf("%-15s %-20s %-15s %-20s %-15s %-15s\n",
	// 	"---------------", "--------------------", "---------------", "------------------", "---------------", "---------------")

	// for _, res := range results {
	// 	if res.Error == nil {
	// 		fmt.Printf("%-15s %-20s %-15s %-20.2f %-15.2f %-15.2f\n",
	// 			res.InterfaceName,
	// 			res.NATType,
	// 			res.Protocol,
	// 			res.Throughput,
	// 			res.AvgRTT,
	// 			res.AvgJitter)
	// 	} else {
	// 		fmt.Printf("%-15s %-20s %-15s %-20s %-15s %-15s\n",
	// 			res.InterfaceName,
	// 			res.NATType,
	// 			"ERROR",
	// 			"N/A",
	// 			"N/A",
	// 			"N/A")
	// 	}
	// }

	utils.PrintStructuredInterfaceResult(results)
	return nil
}

func (ct *TCPTester) runSingleInterfaceTestDetailed(localIP string, serverAddr string, connections int, duration int) []connResult {
	resultChan := make(chan connResult, connections)
	var wg sync.WaitGroup

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			conn, err := dialTCP(serverAddr, localIP)
			if err != nil {
				resultChan <- connResult{err: fmt.Errorf("连接%d: %v", connID, err)}
				return
			}
			defer conn.Close()

			msg := &protocol.Message{
				Type:     protocol.TypeTestRequest,
				TestType: "tcp",
				Payload: map[string]interface{}{
					"test_type": "tcp",
					"threads":   1,
					"duration":  duration,
				},
			}

			if err := msg.Send(conn); err != nil {
				resultChan <- connResult{err: fmt.Errorf("连接%d: 发送TCP测试请求失败: %v", connID, err)}
				return
			}

			bwResultChan := make(chan *utils.TestResult, channelBufferSize)
			latResultChan := make(chan *utils.TestResult, channelBufferSize)
			errChan := make(chan error, 2)

			targetHost := extractHost(serverAddr)

			go func() {
				bwResult, err := ct.runSingleConnectionBandwidthTest(conn, duration)
				if err != nil {
					errChan <- fmt.Errorf("连接%d: TCP带宽测试失败: %v", connID, err)
					bwResultChan <- nil
				} else {
					bwResultChan <- bwResult
				}
			}()

			go func() {
				latResult, err := ct.runSingleConnectionLatencyTest(targetHost)
				if err != nil {
					errChan <- fmt.Errorf("连接%d: TCP延迟测试失败: %v", connID, err)
					latResultChan <- nil
				} else {
					latResultChan <- latResult
				}
			}()

			var bwResult *utils.TestResult
			var latResult *utils.TestResult

			for i := 0; i < 2; i++ {
				select {
				case bwRes := <-bwResultChan:
					bwResult = bwRes
				case latRes := <-latResultChan:
					latResult = latRes
				case err := <-errChan:
					fmt.Printf("%v\n", err)
				}
			}

			resultChan <- connResult{bandwidth: bwResult, latency: latResult}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []connResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// UDPTester UDP测试客户端
type UDPTester struct {
	Timeout         time.Duration
	bandwidthResult *utils.TestResult
	latencyResult   *utils.TestResult
}

// NewUDPTester 创建UDP测试客户端
func NewUDPTester() *UDPTester {
	return &UDPTester{
		Timeout: DefaultTimeout,
	}
}

type UDPTestResult struct {
	Throughput     float64
	ThroughputMbps float64
	AvgRTT         float64
	AvgJitter      float64
	TotalBytes     int64
	Duration       float64
}

func (ut *UDPTester) RunUDPTestWithResult(serverAddr string, threads int, duration int, targetBandwidth string, localIP string) (*UDPTestResult, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("解析UDP地址失败: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("创建UDP连接失败: %v", err)
	}
	defer conn.Close()

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)
	conn.SetWriteDeadline(endTime)

	var totalBytes int64
	data := make([]byte, 32*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	for {
		n, err := conn.Write(data)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err != nil {
			break
		}
	}

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second
	}

	throughput := float64(totalBytes) / elapsed.Seconds()

	targetHost := extractHost(serverAddr)
	pingResult, _ := utils.PingTarget(targetHost, 10)

	result := &UDPTestResult{
		Throughput:     throughput,
		ThroughputMbps: throughput * 8 / 1000 / 1000,
		TotalBytes:     totalBytes,
		Duration:       elapsed.Seconds(),
	}

	if pingResult != nil && pingResult.Success {
		result.AvgRTT = pingResult.Latency
		result.AvgJitter = pingResult.Jitter
	}

	return result, nil
}

// RunUDPTest 执行UDP测试（带宽和延迟）
func (ut *UDPTester) RunUDPTest(serverAddr string, threads int, duration int, targetBandwidth string, localIP string, interfaceName string) error {
	if interfaceName == "all" {
		return ut.runUDPTestOnAllInterfacesAggregated(serverAddr, threads, duration, targetBandwidth)
	}

	result, err := ut.RunUDPTestWithResult(serverAddr, threads, duration, targetBandwidth, localIP)
	if err != nil {
		return err
	}

	fmt.Printf("\n========== UDP测试结果 ==========\n")
	fmt.Printf("吞吐量: %.2f MB/s (%.2f Mbps)\n", result.Throughput/1024/1024, result.ThroughputMbps)
	fmt.Printf("平均延迟: %.2f ms\n", result.AvgRTT)
	fmt.Printf("平均抖动: %.2f ms\n", result.AvgJitter)
	fmt.Printf("总传输字节: %d bytes\n", result.TotalBytes)
	fmt.Printf("测试时长: %.2f 秒\n", result.Duration)

	return nil
}

// runUDPBandwidthTest 执行UDP带宽测试
func (ut *UDPTester) runUDPBandwidthTest(conn *net.UDPConn, threads int, duration int, targetBandwidth string) error {

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 准备数据包
	packetSize := 1024 // 1KB UDP数据包
	data := make([]byte, packetSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// 多线程UDP带宽测试（上行带宽）
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			threadBytes := int64(0)

			for {
				if time.Now().After(endTime) {
					break
				}

				_, err := conn.Write(data)
				if err != nil {
					break
				}

				threadBytes += int64(len(data))

				// 只有在指定了目标带宽时才进行延迟，否则全速发送以进行压测
				if targetBandwidth != "" {
					time.Sleep(time.Millisecond * 10)
				}
			}

			mu.Lock()
			totalBytes += threadBytes
			mu.Unlock()
		}()
	}

	// 等待测试完成
	time.Sleep(time.Duration(duration) * time.Second)

	// 等待所有协程完成
	wg.Wait()

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second // 防止除零错误
	}
	throughput := float64(totalBytes) / elapsed.Seconds()

	// 创建测试结果并保存
	result := utils.TestResult{
		Protocol:   "UDP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}
	ut.bandwidthResult = &result

	return nil
}

// runUDPLatencyTest 执行基于ping的延迟测试
func (ut *UDPTester) runUDPLatencyTest(serverAddr string) error {
	// 提取目标主机用于ping测试
	targetHost := serverAddr
	colonIndex := strings.LastIndex(serverAddr, ":")
	if colonIndex != -1 {
		targetHost = serverAddr[:colonIndex]
	}

	// 使用ping进行延迟测试
	pingResult, err := utils.PingTarget(targetHost, 10) // 发送10个ping包
	if err != nil {
		return fmt.Errorf("ping测试失败: %v", err)
	}

	if !pingResult.Success {
		return fmt.Errorf("ping测试无响应")
	}

	// 创建测试结果并保存
	result := utils.TestResult{
		Protocol:    "PING",
		TestType:    "latency",
		Direction:   "round-trip",
		AvgRTT:      pingResult.Latency, // 毫秒值
		AvgJitter:   pingResult.Jitter,  // 毫秒值
		SuccessRate: (100 - pingResult.PacketLoss) / 100,
		Duration:    pingResult.Latency * 10, // 估算值
	}
	ut.latencyResult = &result

	return nil
}

// runUDPTestOnAllInterfacesAggregated 在所有在线网络接口上执行UDP测试并聚合结果
func (ut *UDPTester) runUDPTestOnAllInterfacesAggregated(serverAddr string, threads int, duration int, targetBandwidth string) error {
	fmt.Println("正在获取所有在线网络接口...")
	interfaces, err := utils.GetOnlineNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("获取在线网络接口失败: %v", err)
	}

	if len(interfaces) == 0 {
		return fmt.Errorf("未找到任何在线网络接口")
	}

	fmt.Printf("发现 %d 个在线网络接口，开始逐个测试并聚合结果:\n\n", len(interfaces))

	// 存储每个接口的测试结果
	var results []utils.InterfaceTestResult

	for i, iface := range interfaces {
		fmt.Printf("=== 测试第 %d/%d 个网络接口: %s (IP: %s, NAT类型: %s) ===\n",
			i+1, len(interfaces), iface.Name, iface.IP, iface.NATType)

		// 为每个接口创建单独的测试实例
		tester := NewUDPTester()
		// 使用接口的IP地址进行测试
		err := tester.RunUDPTest(serverAddr, threads, duration, targetBandwidth, iface.IP, "")
		if err != nil {
			fmt.Printf("接口 %s 测试失败: %v\n", iface.Name, err)
			interfaceResult := utils.InterfaceTestResult{
				TestResult:    utils.TestResult{},
				InterfaceName: iface.Name,
				NATType:       iface.NATType,
				Error:         err,
			}
			results = append(results, interfaceResult)
			fmt.Println()
			continue
		}

		// 构建包含接口信息的结果
		var result utils.TestResult
		if tester.bandwidthResult != nil && tester.latencyResult != nil {
			result = utils.TestResult{
				Protocol:   tester.bandwidthResult.Protocol,
				TestType:   "combined",
				Direction:  tester.bandwidthResult.Direction,
				Throughput: tester.bandwidthResult.Throughput,
				AvgRTT:     tester.latencyResult.AvgRTT,
				AvgJitter:  tester.latencyResult.AvgJitter,
				Duration:   tester.bandwidthResult.Duration,
			}
		} else if tester.bandwidthResult != nil {
			result = *tester.bandwidthResult
		} else if tester.latencyResult != nil {
			result = *tester.latencyResult
		}

		interfaceResult := utils.InterfaceTestResult{
			TestResult:    result,
			InterfaceName: iface.Name,
			NATType:       iface.NATType,
			Error:         nil,
		}
		results = append(results, interfaceResult)
		fmt.Printf("接口 %s 测试完成\n", iface.Name)
		fmt.Println()
	}

	// 输出每个接口的详细结果
	fmt.Println("========== 各网络接口详细测试结果 ==========")
	fmt.Printf("%-15s %-20s %-15s %-20s %-15s %-15s\n", "网卡名称", "NAT类型", "协议", "吞吐量(B/s)", "平均RTT(ms)", "平均抖动(ms)")
	fmt.Printf("%-15s %-20s %-15s %-20s %-15s %-15s\n",
		"---------------", "--------------------", "---------------", "------------------", "---------------", "---------------")

	for _, res := range results {
		if res.Error == nil {
			fmt.Printf("%-15s %-20s %-15s %-20.2f %-15.2f %-15.2f\n",
				res.InterfaceName,
				res.NATType,
				res.Protocol,
				res.Throughput,
				res.AvgRTT,
				res.AvgJitter)
		} else {
			fmt.Printf("%-15s %-20s %-15s %-20s %-15s %-15s\n",
				res.InterfaceName,
				res.NATType,
				"ERROR",
				"N/A",
				"N/A",
				"N/A")
		}
	}

	return nil
}
