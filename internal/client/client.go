package client

import (
	"fmt"
	"net"
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
func (ct *TCPTester) RunTCPTest(serverAddr string, connections int, duration int, localIP string) error {
	fmt.Printf("TCP测试参数 - 目标: %s, 连接数: %d, 时长: %d秒\n", serverAddr, connections, duration)
	if localIP != "" {
		fmt.Printf("本地IP: %s\n", localIP)
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

			// 执行带宽测试（单连接单线程）
			bwResult, err := ct.runSingleConnectionBandwidthTest(conn, duration)
			if err != nil {
				fmt.Printf("连接%d: TCP带宽测试失败: %v\n", connID, err)
			}

			// 执行延迟测试
			latResult, err := ct.runSingleConnectionLatencyTest(conn)
			if err != nil {
				fmt.Printf("连接%d: TCP延迟测试失败: %v\n", connID, err)
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

// printCombinedResults 统一输出带宽和延迟测试结果
func (ct *TCPTester) printCombinedResults() {
	// 如果两个结果都有，则合并输出
	if ct.bandwidthResult != nil && ct.latencyResult != nil {
		combinedResult := utils.TestResult{
			Protocol:   ct.bandwidthResult.Protocol,
			TestType:   "combined",
			Direction:  ct.bandwidthResult.Direction,
			Throughput: ct.bandwidthResult.Throughput,
			AvgRTT:     ct.latencyResult.AvgRTT,
			AvgJitter:  ct.latencyResult.AvgJitter,
			Duration:   ct.bandwidthResult.Duration,
		}
		utils.PrintStructuredResult(combinedResult)
	} else {
		// 分别输出可用的结果
		if ct.bandwidthResult != nil {
			utils.PrintStructuredResult(*ct.bandwidthResult)
		}
		if ct.latencyResult != nil {
			utils.PrintStructuredResult(*ct.latencyResult)
		}
	}
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

// runSingleConnectionLatencyTest 执行TCP延迟测试（单连接）
func (ct *TCPTester) runSingleConnectionLatencyTest(conn net.Conn) (*utils.TestResult, error) {
	packetSize := 64 // 64字节的数据包
	numPackets := 10 // 发送10个包来测试延迟

	data := make([]byte, packetSize)
	for i := 0; i < packetSize; i++ {
		data[i] = byte(i % 256)
	}

	totalRTT := time.Duration(0)
	successfulPackets := 0
	jitter := time.Duration(0)

	var lastRTT time.Duration

	startTime := time.Now()
	for i := 0; i < numPackets; i++ {
		sendTime := time.Now()
		_, err := conn.Write(data)
		if err != nil {
			continue
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(ct.Timeout))

		// 尝试读取响应
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			continue
		}

		receiveTime := time.Now()
		rtt := receiveTime.Sub(sendTime)
		totalRTT += rtt
		successfulPackets++

		// 计算抖动（RTT变化）
		if lastRTT != 0 {
			diff := rtt - lastRTT
			if diff < 0 {
				diff = -diff
			}
			jitter += diff
		}
		lastRTT = rtt
	}

	if successfulPackets > 0 {
		avgRTT := totalRTT / time.Duration(successfulPackets)
		var avgJitter time.Duration
		if successfulPackets > 1 {
			avgJitter = jitter / time.Duration(successfulPackets-1)
		}

		return &utils.TestResult{
			Protocol:    "TCP",
			TestType:    "latency",
			Direction:   "round-trip",
			AvgRTT:      float64(avgRTT),
			AvgJitter:   float64(avgJitter),
			SuccessRate: float64(successfulPackets) / float64(numPackets),
			Duration:    time.Since(startTime).Seconds(),
		}, nil
	}

	return nil, fmt.Errorf("no packets received")
}

// runTCPBandwidthTest 执行TCP带宽测试（旧方法，保留以兼容）
func (ct *TCPTester) runTCPBandwidthTest(conn net.Conn, threads int, duration int) error {
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

	return nil
}

// runTCPLatencyTest 执行TCP延迟测试
func (ct *TCPTester) runTCPLatencyTest(conn net.Conn) error {

	packetSize := 64 // 64字节的数据包
	numPackets := 10 // 发送10个包来测试延迟

	data := make([]byte, packetSize)
	for i := 0; i < packetSize; i++ {
		data[i] = byte(i % 256)
	}

	totalRTT := time.Duration(0)
	successfulPackets := 0
	jitter := time.Duration(0)

	var lastRTT time.Duration

	startTime := time.Now()
	for i := 0; i < numPackets; i++ {
		sendTime := time.Now()
		_, err := conn.Write(data)
		if err != nil {
			continue
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(ct.Timeout))

		// 尝试读取响应
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			continue
		}

		receiveTime := time.Now()
		rtt := receiveTime.Sub(sendTime)
		totalRTT += rtt
		successfulPackets++

		// 计算抖动（RTT变化）
		if lastRTT != 0 {
			diff := rtt - lastRTT
			if diff < 0 {
				diff = -diff
			}
			jitter += diff
		}
		lastRTT = rtt
	}

	if successfulPackets > 0 {
		avgRTT := totalRTT / time.Duration(successfulPackets)
		var avgJitter time.Duration
		if successfulPackets > 1 {
			avgJitter = jitter / time.Duration(successfulPackets-1)
		}

		// 创建测试结果并保存
		result := utils.TestResult{
			Protocol:    "TCP",
			TestType:    "latency",
			Direction:   "round-trip",
			AvgRTT:      float64(avgRTT),
			AvgJitter:   float64(avgJitter),
			SuccessRate: float64(successfulPackets) / float64(numPackets),
			Duration:    time.Since(startTime).Seconds(),
		}
		ct.latencyResult = &result
	}

	return nil
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
func (ut *UDPTester) RunUDPTest(serverAddr string, threads int, duration int, targetBandwidth string, localIP string) error {
	fmt.Printf("UDP测试参数 - 目标: %s, 线程数: %d, 时长: %d秒", serverAddr, threads, duration)
	if targetBandwidth != "" {
		fmt.Printf(", 目标带宽: %s", targetBandwidth)
	}
	if localIP != "" {
		fmt.Printf(", 本地IP: %s", localIP)
	}
	fmt.Printf("\n")

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

	// 执行UDP延迟测试
	err = ut.runUDPLatencyTest(conn)
	if err != nil {
		fmt.Printf("UDP延迟测试失败: %v\n", err)
	}

	// 统一输出所有结果
	ut.printCombinedResults()

	return nil
}

// printCombinedResults 统一输出带宽和延迟测试结果
func (ut *UDPTester) printCombinedResults() {
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

// runUDPLatencyTest 执行UDP延迟测试
func (ut *UDPTester) runUDPLatencyTest(conn *net.UDPConn) error {

	packetSize := 64 // 64字节的数据包
	numPackets := 10 // 发送10个包来测试延迟

	data := make([]byte, packetSize)
	for i := 0; i < packetSize; i++ {
		data[i] = byte(i % 256)
	}

	startTime := time.Now()
	totalRTT := time.Duration(0)
	successfulPackets := 0
	jitter := time.Duration(0)

	var lastRTT time.Duration

	for i := 0; i < numPackets; i++ {
		sendTime := time.Now()

		// 发送UDP数据包
		_, err := conn.Write(data)
		if err != nil {
			continue
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(ut.Timeout))

		// 尝试读取响应
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			continue
		}

		receiveTime := time.Now()
		rtt := receiveTime.Sub(sendTime)
		totalRTT += rtt
		successfulPackets++

		// 计算抖动（RTT变化）
		if lastRTT != 0 {
			diff := rtt - lastRTT
			if diff < 0 {
				diff = -diff
			}
			jitter += diff
		}
		lastRTT = rtt
	}

	if successfulPackets > 0 {
		avgRTT := totalRTT / time.Duration(successfulPackets)
		var avgJitter time.Duration
		if successfulPackets > 1 {
			avgJitter = jitter / time.Duration(successfulPackets-1)
		}

		// 创建测试结果并保存
		result := utils.TestResult{
			Protocol:    "UDP",
			TestType:    "latency",
			Direction:   "round-trip",
			AvgRTT:      float64(avgRTT),
			AvgJitter:   float64(avgJitter),
			SuccessRate: float64(successfulPackets) / float64(numPackets),
			Duration:    time.Since(startTime).Seconds(),
		}
		ut.latencyResult = &result
	}

	return nil
}
