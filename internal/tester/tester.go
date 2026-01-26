package tester

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"perfgo/pkg/utils"
)

const (
	// DefaultBufferSize 默认缓冲区大小
	DefaultBufferSize = 64 * 1024 // 64KB
	// TestDuration 默认测试持续时间
	TestDuration = 10 * time.Second
	// DataChunkSize 数据块大小
	DataChunkSize = 32 * 1024 // 32KB
)

// NetworkTester 网络测试器
type NetworkTester struct {
	BufferSize int
	Duration   time.Duration
	Timeout    time.Duration
}

// NewNetworkTester 创建新的网络测试器
func NewNetworkTester() *NetworkTester {
	return &NetworkTester{
		BufferSize: DefaultBufferSize,
		Duration:   TestDuration,
		Timeout:    30 * time.Second,
	}
}

// HandleTCPTest 处理TCP测试（带宽和延迟）
func (nt *NetworkTester) HandleTCPTest(conn net.Conn, threads int, duration int) error {
	// 更新测试持续时间
	nt.Duration = time.Duration(duration) * time.Second

	// 分别执行带宽测试和延迟测试
	err := nt.tcpBandwidthTest(conn, threads)
	if err != nil {
		fmt.Printf("TCP带宽测试失败: %v\n", err)
	}

	err = nt.tcpLatencyTest(conn)
	if err != nil {
		fmt.Printf("TCP延迟测试失败: %v\n", err)
	}

	return nil
}

// HandleUDPTest 处理UDP测试（带宽和延迟）
func (nt *NetworkTester) HandleUDPTest(conn net.Conn, threads int, duration int, targetBandwidth string) error {
	// 更新测试持续时间
	nt.Duration = time.Duration(duration) * time.Second

	// 分别执行带宽测试和延迟测试
	err := nt.udpBandwidthTest(conn, threads, targetBandwidth)
	if err != nil {
		fmt.Printf("UDP带宽测试失败: %v\n", err)
	}

	err = nt.udpLatencyTest(conn)
	if err != nil {
		fmt.Printf("UDP延迟测试失败: %v\n", err)
	}

	return nil
}

// tcpBandwidthTest 执行TCP带宽测试
// tcpBandwidthTestResult 存储TCP带宽测试结果
var tcpBandwidthTestResult utils.TestResult
var tcpBandwidthTestResultMutex sync.Mutex

func (nt *NetworkTester) tcpBandwidthTest(conn net.Conn, threads int) error {
	startTime := time.Now()
	endTime := startTime.Add(nt.Duration)

	var totalBytes int64
	var mu sync.Mutex

	// 使用更大的缓冲区提高效率
	buf := make([]byte, 64*1024) // 64KB

	// 在指定时间内接收客户端发送的数据
	for {
		if time.Now().After(endTime) {
			break
		}
		conn.SetReadDeadline(time.Now().Add(nt.Timeout))
		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时，继续等待
				continue
			}
			// 其他错误，退出
			break
		}
		// 累加接收到的字节数
		mu.Lock()
		totalBytes += int64(n)
		mu.Unlock()
	}

	elapsed := time.Since(startTime)
	throughput := float64(totalBytes) / elapsed.Seconds()

	// 存储带宽测试结果
	tcpBandwidthTestResultMutex.Lock()
	tcpBandwidthTestResult = utils.TestResult{
		Protocol:   "TCP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}
	tcpBandwidthTestResultMutex.Unlock()

	return nil
}

// tcpLatencyTestResult 存储TCP延迟测试结果
var tcpLatencyTestResult utils.TestResult
var tcpLatencyTestResultMutex sync.Mutex

// tcpLatencyTest 执行TCP延迟测试
func (nt *NetworkTester) tcpLatencyTest(conn net.Conn) error {
	startTime := time.Now()

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

	for i := 0; i < numPackets; i++ {
		sendTime := time.Now()
		_, err := conn.Write(data)
		if err != nil {
			continue
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(nt.Timeout))

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
		avgJitter := jitter / time.Duration(successfulPackets-1)

		// 存储延迟测试结果
		tcpLatencyTestResultMutex.Lock()
		tcpLatencyTestResult = utils.TestResult{
			Protocol:    "TCP",
			TestType:    "latency",
			Direction:   "round-trip",
			AvgRTT:      float64(avgRTT),
			AvgJitter:   float64(avgJitter),
			SuccessRate: float64(successfulPackets) / float64(numPackets),
			Duration:    time.Since(startTime).Seconds(),
		}
		tcpLatencyTestResultMutex.Unlock()
	}

	// 如果两个结果都有，则合并输出
	tcpBandwidthTestResultMutex.Lock()
	tcpLatencyTestResultMutex.Lock()
	if tcpBandwidthTestResult.Throughput > 0 && tcpLatencyTestResult.AvgRTT > 0 {
		// 合并结果并输出
		combinedResult := utils.TestResult{
			Protocol:   "TCP",
			TestType:   "combined",
			Direction:  "uplink",
			Throughput: tcpBandwidthTestResult.Throughput,
			AvgRTT:     tcpLatencyTestResult.AvgRTT,
			AvgJitter:  tcpLatencyTestResult.AvgJitter,
			Duration:   tcpBandwidthTestResult.Duration,
		}
		utils.PrintStructuredResult(combinedResult)
	} else {
		// 分别输出可用的结果
		if tcpBandwidthTestResult.Throughput > 0 {
			utils.PrintStructuredResult(tcpBandwidthTestResult)
		}
		if tcpLatencyTestResult.AvgRTT > 0 {
			utils.PrintStructuredResult(tcpLatencyTestResult)
		}
	}
	tcpLatencyTestResultMutex.Unlock()
	tcpBandwidthTestResultMutex.Unlock()

	return nil
}

// udpBandwidthTest 执行UDP带宽测试
func (nt *NetworkTester) udpBandwidthTest(conn net.Conn, threads int, targetBandwidth string) error {
	// 对于TCP连接上的UDP测试模拟，我们暂时跳过
	fmt.Printf("UDP带宽测试（通过TCP连接）: 目标带宽 %s\n", targetBandwidth)
	return nil
}

// udpLatencyTest 执行UDP延迟测试
func (nt *NetworkTester) udpLatencyTest(conn net.Conn) error {
	// 对于TCP连接上的UDP测试模拟，我们暂时跳过
	fmt.Println("UDP延迟测试（通过TCP连接）")
	return nil
}

// HandleUDPTestUDP 处理UDP测试（通过UDP连接）
func (nt *NetworkTester) HandleUDPTestUDP(udpConn *net.UDPConn, clientAddr *net.UDPAddr, threads int, duration int, targetBandwidth string) error {
	// 更新测试持续时间
	nt.Duration = time.Duration(duration) * time.Second

	// 分别执行带宽测试和延迟测试
	err := nt.udpBandwidthTestUDP(udpConn, clientAddr, threads, targetBandwidth)
	if err != nil {
		return err
	}

	err = nt.udpLatencyTestUDP(udpConn, clientAddr)
	if err != nil {
		return err
	}

	return nil
}

// udpBandwidthTestResult 存储UDP带宽测试结果
var udpBandwidthTestResult utils.TestResult
var udpBandwidthTestResultMutex sync.Mutex

// udpBandwidthTestUDP 执行UDP带宽测试（通过UDP连接）
func (nt *NetworkTester) udpBandwidthTestUDP(udpConn *net.UDPConn, clientAddr *net.UDPAddr, threads int, targetBandwidth string) error {
	startTime := time.Now()
	endTime := startTime.Add(nt.Duration)

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 准备数据块
	packetSize := 1024 // UDP数据包大小
	data := make([]byte, packetSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// 多线程UDP带宽测试
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			threadBytes := int64(0)
			ctx, cancel := context.WithTimeout(context.Background(), nt.Duration)
			defer cancel()

			for {
				select {
				case <-ctx.Done():
					goto finish
				default:
					_, err := udpConn.WriteToUDP(data, clientAddr)
					if err != nil {
						goto finish
					}

					threadBytes += int64(len(data))

					// 检查是否已达到结束时间
					if time.Now().After(endTime) {
						goto finish
					}

					// 短暂延迟以避免过度占用网络
					time.Sleep(time.Millisecond * 10)
				}
			}

		finish:
			mu.Lock()
			totalBytes += threadBytes
			mu.Unlock()
		}()
	}

	// 等待所有协程完成
	wg.Wait()

	elapsed := time.Since(startTime)
	throughput := float64(totalBytes) / elapsed.Seconds()

	// 存储带宽测试结果
	udpBandwidthTestResultMutex.Lock()
	udpBandwidthTestResult = utils.TestResult{
		Protocol:   "UDP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}
	udpBandwidthTestResultMutex.Unlock()

	return nil
}

// udpLatencyTestResult 存储UDP延迟测试结果
var udpLatencyTestResult utils.TestResult
var udpLatencyTestResultMutex sync.Mutex

// udpLatencyTestUDP 执行UDP延迟测试（通过UDP连接）
func (nt *NetworkTester) udpLatencyTestUDP(udpConn *net.UDPConn, clientAddr *net.UDPAddr) error {
	startTime := time.Now()

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

	for i := 0; i < numPackets; i++ {
		sendTime := time.Now()

		// 发送UDP数据包
		_, err := udpConn.WriteToUDP(data, clientAddr)
		if err != nil {
			continue
		}

		// 设置读取超时
		udpConn.SetReadDeadline(time.Now().Add(nt.Timeout))

		// 尝试读取响应（这里模拟响应）
		buf := make([]byte, 1024)
		_, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		// 确保响应来自正确的客户端
		if addr.String() != clientAddr.String() {
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
		avgJitter := jitter / time.Duration(successfulPackets-1)

		// 存储延迟测试结果
		udpLatencyTestResultMutex.Lock()
		udpLatencyTestResult = utils.TestResult{
			Protocol:    "UDP",
			TestType:    "latency",
			Direction:   "round-trip",
			AvgRTT:      float64(avgRTT),
			AvgJitter:   float64(avgJitter),
			SuccessRate: float64(successfulPackets) / float64(numPackets),
			Duration:    time.Since(startTime).Seconds(),
		}
		udpLatencyTestResultMutex.Unlock()
	}

	// 如果两个结果都有，则合并输出
	udpBandwidthTestResultMutex.Lock()
	udpLatencyTestResultMutex.Lock()
	if udpBandwidthTestResult.Throughput > 0 && udpLatencyTestResult.AvgRTT > 0 {
		// 合并结果并输出
		combinedResult := utils.TestResult{
			Protocol:   "UDP",
			TestType:   "combined",
			Direction:  "uplink",
			Throughput: udpBandwidthTestResult.Throughput,
			AvgRTT:     udpLatencyTestResult.AvgRTT,
			AvgJitter:  udpLatencyTestResult.AvgJitter,
			Duration:   udpBandwidthTestResult.Duration,
		}
		utils.PrintStructuredResult(combinedResult)
	} else {
		// 分别输出可用的结果
		if udpBandwidthTestResult.Throughput > 0 {
			utils.PrintStructuredResult(udpBandwidthTestResult)
		}
		if udpLatencyTestResult.AvgRTT > 0 {
			utils.PrintStructuredResult(udpLatencyTestResult)
		}
	}
	udpLatencyTestResultMutex.Unlock()
	udpBandwidthTestResultMutex.Unlock()

	return nil
}
