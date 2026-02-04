package client

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"perfgo/pkg/protocol"
	"perfgo/pkg/utils"
)

// TCPTester TCP测试客户端
type TCPTester struct {
	Timeout         time.Duration
	bandwidthResult *utils.TestResult
	latencyResult   *utils.TestResult
}

// NewTCPTester 创建TCP测试客户端
func NewTCPTester() *TCPTester {
	return &TCPTester{
		Timeout: 30 * time.Second,
	}
}

// RunTCPTest 执行TCP测试（带宽和延迟） - 支持多连接并发
func (ct *TCPTester) RunTCPTest(serverAddr string, connections int, duration int, localIP string, interfaceName string) error {
	fmt.Printf("TCP测试参数 - 目标: %s, 连接数: %d, 时长: %d秒\n", serverAddr, connections, duration)
	if localIP != "" {
		fmt.Printf("本地IP: %s\n", localIP)
	}
	if interfaceName != "" {
		fmt.Printf("网络接口: %s\n", interfaceName)
	}

	// 如果interfaceName为"all"，则对所有在线网络接口进行测试并聚合结果
	if interfaceName == "all" {
		return ct.runTCPTestOnAllInterfacesAggregated(serverAddr, connections, duration)
	}

	// 用于收集所有连接的测试结果
	type connResult struct {
		bandwidth *utils.TestResult
		latency   *utils.TestResult
		err       error
	}

	resultChan := make(chan connResult, connections)
	var wg sync.WaitGroup

	// 启动多个独立的TCP连接进行并发测试
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			// 每个连接独立建立
			var conn net.Conn
			var err error

			if localIP != "" {
				// 使用本地IP绑定连接
				localAddr, err := net.ResolveTCPAddr("tcp", localIP+":0")
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 解析本地TCP地址失败: %v", connID, err)}
					return
				}
				remoteAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 解析远程TCP地址失败: %v", connID, err)}
					return
				}
				dialer := net.Dialer{
					LocalAddr: localAddr,
				}
				conn, err = dialer.Dial("tcp", remoteAddr.String())
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 连接到服务器失败: %v", connID, err)}
					return
				}
			} else {
				conn, err = net.Dial("tcp", serverAddr)
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 连接到服务器失败: %v", connID, err)}
					return
				}
			}
			defer conn.Close()

			// 发送测试请求
			msg := &protocol.Message{
				Type:     protocol.TypeTestRequest,
				TestType: "tcp",
				Payload: map[string]interface{}{
					"test_type": "tcp",
					"threads":   1, // 每个连接内部单线程
					"duration":  duration,
				},
			}

			err = msg.Send(conn)
			if err != nil {
				resultChan <- connResult{err: fmt.Errorf("连接%d: 发送TCP测试请求失败: %v", connID, err)}
				return
			}

			// 创建通道用于接收测试结果
			bwResultChan := make(chan *utils.TestResult, 1)
			latResultChan := make(chan *utils.TestResult, 1)
			errChan := make(chan error, 2)

			// 提取目标地址用于ping测试
			targetHost := serverAddr
			colonIndex := strings.LastIndex(serverAddr, ":")
			if colonIndex != -1 {
				targetHost = serverAddr[:colonIndex]
			}

			// 并行执行带宽测试和延迟测试
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

			// 等待测试结果
			var bwResult *utils.TestResult
			var latResult *utils.TestResult

			// 等待两个测试完成
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

			resultChan <- connResult{
				bandwidth: bwResult,
				latency:   latResult,
			}
		}(i)
	}

	// 等待所有连接完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集并聚合结果
	var totalBytes int64
	var totalDuration float64
	var latencies []float64
	var jitters []float64
	successCount := 0

	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("错误: %v\n", result.err)
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
		return fmt.Errorf("所有连接均失败")
	}

	// 计算聚合带宽（所有连接的总吞吐量）
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

	// 输出聚合结果
	combinedResult := utils.TestResult{
		Protocol:   "TCP",
		TestType:   "combined",
		Direction:  "uplink",
		Throughput: aggregateThroughput,
		TotalBytes: totalBytes,
		AvgRTT:     avgRTT,
		AvgJitter:  avgJitter,
		Duration:   avgDuration,
	}

	fmt.Printf("\n========== 聚合测试结果 (%d个连接) ==========\n", successCount)
	utils.PrintStructuredResult(combinedResult)

	return nil
}

// runSingleConnectionBandwidthTest 执行TCP带宽测试（单连接单线程）
func (ct *TCPTester) runSingleConnectionBandwidthTest(conn net.Conn, duration int) (*utils.TestResult, error) {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)

	// 设置总的写入超时时间
	conn.SetWriteDeadline(endTime)

	var totalBytes int64

	// 准备数据块
	data := make([]byte, 32*1024) // 32KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	// 单线程发送测试
	for {
		n, err := conn.Write(data)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err != nil {
			// 写入超时或连接关闭，退出
			break
		}
	}

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second // 防止除零错误
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

// runSingleConnectionLatencyTest 执行基于ping的延迟测试（单连接）
func (ct *TCPTester) runSingleConnectionLatencyTest(target string) (*utils.TestResult, error) {
	// 使用ping进行延迟测试
	pingResult, err := utils.PingTarget(target, 10) // 发送10个ping包
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
		AvgRTT:      pingResult.Latency, // 毫秒值
		AvgJitter:   pingResult.Jitter,  // 毫秒值
		SuccessRate: (100 - pingResult.PacketLoss) / 100,
		Duration:    pingResult.Latency * 10, // 估算值
	}, nil
}

// runTCPBandwidthTest 执行TCP带宽测试（旧方法，保留以兼容）
func (ct *TCPTester) runTCPBandwidthTest(conn net.Conn, threads int, duration int, serverAddr string) error {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)

	// 设置总的写入超时时间，避免在循环中重复检查时间或设置超时，提高压测准确性
	conn.SetWriteDeadline(endTime)

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 准备数据块
	data := make([]byte, 32*1024) // 32KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	// 多线程发送测试（测量上行带宽）
	// 注意：在单个 TCP 连接上使用多个 goroutine 并发写入通常不会增加物理带宽，
	// 甚至可能因为锁竞争导致性能下降。为了获得更好的压测效果，建议增加连接数。
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			threadBytes := int64(0)

			for {
				n, err := conn.Write(data)
				if n > 0 {
					threadBytes += int64(n)
				}
				if err != nil {
					// 写入超时或连接关闭，退出
					break
				}
			}

			mu.Lock()
			totalBytes += threadBytes
			mu.Unlock()
		}()
	}

	// 等待所有协程完成
	wg.Wait()

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second // 防止除零错误
	}
	throughput := float64(totalBytes) / elapsed.Seconds()

	// 创建测试结果并保存
	result := utils.TestResult{
		Protocol:   "TCP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}
	ct.bandwidthResult = &result

	// 执行延迟测试
	err := ct.runTCPLatencyTest(serverAddr)
	if err != nil {
		fmt.Printf("TCP延迟测试失败: %v\n", err)
	}

	return nil
}

// runTCPLatencyTest 执行基于ping的延迟测试
func (ct *TCPTester) runTCPLatencyTest(serverAddr string) error {
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
		// 创建临时变量来捕获单个接口的测试结果
		tempResults := make([]struct {
			bandwidth *utils.TestResult
			latency   *utils.TestResult
			err       error
		}, connections)

		// 实际执行测试
		tempResults = ct.runSingleInterfaceTestDetailed(iface.IP, serverAddr, connections, duration)

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

// runSingleInterfaceTestDetailed 辅助方法，用于测试单个接口
func (ct *TCPTester) runSingleInterfaceTestDetailed(localIP string, serverAddr string, connections int, duration int) []struct {
	bandwidth *utils.TestResult
	latency   *utils.TestResult
	err       error
} {
	// 用于收集所有连接的测试结果
	type connResult struct {
		bandwidth *utils.TestResult
		latency   *utils.TestResult
		err       error
	}

	resultChan := make(chan connResult, connections)
	var wg sync.WaitGroup

	// 启动多个独立的TCP连接进行并发测试
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			// 每个连接独立建立
			var conn net.Conn
			var err error

			if localIP != "" {
				// 使用本地IP绑定连接
				localAddr, err := net.ResolveTCPAddr("tcp", localIP+":0")
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 解析本地TCP地址失败: %v", connID, err)}
					return
				}
				remoteAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 解析远程TCP地址失败: %v", connID, err)}
					return
				}
				dialer := net.Dialer{
					LocalAddr: localAddr,
				}
				conn, err = dialer.Dial("tcp", remoteAddr.String())
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 连接到服务器失败: %v", connID, err)}
					return
				}
			} else {
				conn, err = net.Dial("tcp", serverAddr)
				if err != nil {
					resultChan <- connResult{err: fmt.Errorf("连接%d: 连接到服务器失败: %v", connID, err)}
					return
				}
			}
			defer conn.Close()

			// 发送测试请求
			msg := &protocol.Message{
				Type:     protocol.TypeTestRequest,
				TestType: "tcp",
				Payload: map[string]interface{}{
					"test_type": "tcp",
					"threads":   1, // 每个连接内部单线程
					"duration":  duration,
				},
			}

			err = msg.Send(conn)
			if err != nil {
				resultChan <- connResult{err: fmt.Errorf("连接%d: 发送TCP测试请求失败: %v", connID, err)}
				return
			}

			// 创建通道用于接收测试结果
			bwResultChan := make(chan *utils.TestResult, 1)
			latResultChan := make(chan *utils.TestResult, 1)
			errChan := make(chan error, 2)

			// 提取目标地址用于ping测试
			targetHost := serverAddr
			colonIndex := strings.LastIndex(serverAddr, ":")
			if colonIndex != -1 {
				targetHost = serverAddr[:colonIndex]
			}

			// 并行执行带宽测试和延迟测试
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

			// 等待测试结果
			var bwResult *utils.TestResult
			var latResult *utils.TestResult

			// 等待两个测试完成
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

			resultChan <- connResult{
				bandwidth: bwResult,
				latency:   latResult,
			}
		}(i)
	}

	// 等待所有连接完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var results []struct {
		bandwidth *utils.TestResult
		latency   *utils.TestResult
		err       error
	}

	for result := range resultChan {
		results = append(results, struct {
			bandwidth *utils.TestResult
			latency   *utils.TestResult
			err       error
		}{result.bandwidth, result.latency, result.err})
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
		Timeout: 30 * time.Second,
	}
}

// RunUDPTest 执行UDP测试（带宽和延迟）
func (ut *UDPTester) RunUDPTest(serverAddr string, threads int, duration int, targetBandwidth string, localIP string, interfaceName string) error {
	fmt.Printf("UDP测试参数 - 目标: %s, 线程数: %d, 时长: %d秒", serverAddr, threads, duration)
	if targetBandwidth != "" {
		fmt.Printf(", 目标带宽: %s", targetBandwidth)
	}
	if localIP != "" {
		fmt.Printf(", 本地IP: %s", localIP)
	}
	if interfaceName != "" {
		fmt.Printf(", 网络接口: %s", interfaceName)
	}
	fmt.Printf("\n")

	// 如果interfaceName为"all"，则对所有在线网络接口进行测试并聚合结果
	if interfaceName == "all" {
		return ut.runUDPTestOnAllInterfacesAggregated(serverAddr, threads, duration, targetBandwidth)
	}

	// 解析UDP地址
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("解析UDP地址失败: %v", err)
	}

	// 创建UDP连接
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("创建UDP连接失败: %v", err)
	}
	defer conn.Close()

	// 执行UDP带宽测试
	err = ut.runUDPBandwidthTest(conn, threads, duration, targetBandwidth)
	if err != nil {
		fmt.Printf("UDP带宽测试失败: %v\n", err)
	}

	// 提取目标地址用于ping测试
	addr := conn.RemoteAddr().String()
	targetHost := addr
	colonIndex := strings.LastIndex(addr, ":")
	if colonIndex != -1 {
		targetHost = addr[:colonIndex]
	}

	// 执行UDP延迟测试
	err = ut.runUDPLatencyTest(targetHost)
	if err != nil {
		fmt.Printf("UDP延迟测试失败: %v\n", err)
	}

	// 统一输出所有结果
	// 如果两个结果都有，则合并输出
	if ut.bandwidthResult != nil && ut.latencyResult != nil {
		combinedResult := utils.TestResult{
			Protocol:   ut.bandwidthResult.Protocol,
			TestType:   "combined",
			Direction:  ut.bandwidthResult.Direction,
			Throughput: ut.bandwidthResult.Throughput,
			AvgRTT:     ut.latencyResult.AvgRTT,
			AvgJitter:  ut.latencyResult.AvgJitter,
			Duration:   ut.bandwidthResult.Duration,
		}
		utils.PrintStructuredResult(combinedResult)
	} else {
		// 分别输出可用的结果
		if ut.bandwidthResult != nil {
			utils.PrintStructuredResult(*ut.bandwidthResult)
		}
		if ut.latencyResult != nil {
			utils.PrintStructuredResult(*ut.latencyResult)
		}
	}

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
