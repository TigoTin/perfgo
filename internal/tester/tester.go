package tester

import (
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

// HandleTCPTest 处理TCP测试（带宽）
func (nt *NetworkTester) HandleTCPTest(conn net.Conn, threads int, duration int) error {
	// 更新测试持续时间
	nt.Duration = time.Duration(duration) * time.Second

	// 执行带宽测试
	bwResult, err := nt.tcpBandwidthTest(conn, threads)
	if err != nil {
		fmt.Printf("TCP带宽测试失败: %v\n", err)
	}

	// 输出结果
	if bwResult != nil {
		utils.PrintStructuredResult(*bwResult)
	}

	return nil
}

// HandleUDPTest 处理UDP测试（带宽）
func (nt *NetworkTester) HandleUDPTest(conn net.Conn, threads int, duration int, targetBandwidth string) error {
	// 更新测试持续时间
	nt.Duration = time.Duration(duration) * time.Second

	// 执行带宽测试
	bwResult, err := nt.udpBandwidthTest(conn, threads, targetBandwidth)
	if err != nil {
		fmt.Printf("UDP带宽测试失败: %v\n", err)
	}

	// 输出结果
	if bwResult != nil {
		utils.PrintStructuredResult(*bwResult)
	}

	return nil
}

// tcpBandwidthTest 执行TCP带宽测试
func (nt *NetworkTester) tcpBandwidthTest(conn net.Conn, threads int) (*utils.TestResult, error) {
	startTime := time.Now()
	endTime := startTime.Add(nt.Duration)

	var totalBytes int64

	// 使用更大的缓冲区提高效率
	buf := make([]byte, nt.BufferSize)

	// 设置总的读取超时时间，避免在循环中重复设置系统调用，这是提高压测准确性的关键
	conn.SetReadDeadline(endTime)

	// 在指定时间内接收客户端发送的数据
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err != nil {
			// 如果是超时（达到endTime）或连接关闭，则退出循环
			break
		}
	}

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Microsecond // 避免除以0
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

// udpBandwidthTest 执行UDP带宽测试
func (nt *NetworkTester) udpBandwidthTest(conn net.Conn, threads int, targetBandwidth string) (*utils.TestResult, error) {
	// 对于TCP连接上的UDP测试模拟，我们暂时跳过
	fmt.Printf("UDP带宽测试（通过TCP连接）: 目标带宽 %s\n", targetBandwidth)
	return nil, nil
}

// udpLatencyTest 执行UDP延迟测试
func (nt *NetworkTester) udpLatencyTest(conn net.Conn) (*utils.TestResult, error) {
	// 对于TCP连接上的UDP测试模拟，我们暂时跳过
	fmt.Println("UDP延迟测试（通过TCP连接）")
	return nil, nil
}

// HandleUDPTestUDP 处理UDP测试（通过UDP连接）
func (nt *NetworkTester) HandleUDPTestUDP(udpConn *net.UDPConn, clientAddr *net.UDPAddr, threads int, duration int, targetBandwidth string) error {
	// 更新测试持续时间
	nt.Duration = time.Duration(duration) * time.Second

	// 执行带宽测试
	bwResult, err := nt.udpBandwidthTestUDP(udpConn, clientAddr, threads, targetBandwidth)
	if err != nil {
		return err
	}

	// 输出结果
	if bwResult != nil {
		utils.PrintStructuredResult(*bwResult)
	}

	return nil
}

// udpBandwidthTestUDP 执行UDP带宽测试（通过UDP连接）
func (nt *NetworkTester) udpBandwidthTestUDP(udpConn *net.UDPConn, clientAddr *net.UDPAddr, threads int, targetBandwidth string) (*utils.TestResult, error) {
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

			// 使用 SetWriteDeadline 来控制时间，避免在循环中重复检查 time.Now()
			// 注意：UDP连接也可以设置 WriteDeadline
			udpConn.SetWriteDeadline(endTime)

			for {
				_, err := udpConn.WriteToUDP(data, clientAddr)
				if err != nil {
					break
				}

				threadBytes += int64(len(data))

				// 如果指定了目标带宽，则保持 sleep；否则尽可能快地发送
				if targetBandwidth != "" {
					time.Sleep(time.Millisecond * 10)
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
		elapsed = time.Microsecond
	}
	throughput := float64(totalBytes) / elapsed.Seconds()

	return &utils.TestResult{
		Protocol:   "UDP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}, nil
}
