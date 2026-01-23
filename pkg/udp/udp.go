package udp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"perfgo/pkg/utils"
)

// UDPPacket 表示UDP数据包
type UDPPacket struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Data      []byte    `json:"data"`
}

// UDPTester UDP测试器
type UDPTester struct {
	Timeout time.Duration
}

// NewUDPTester 创建新的UDP测试器
func NewUDPTester() *UDPTester {
	return &UDPTester{
		Timeout: 5 * time.Second,
	}
}

// UDPBandwidthTest 执行UDP带宽测试
func (ut *UDPTester) UDPBandwidthTest(conn *net.UDPConn) error {
	fmt.Println("开始UDP带宽测试...")

	// UDP带宽测试实现
	// 通过UDP连接发送大量数据包来测试带宽

	packetSize := 1024 // 1KB packets
	numPackets := 1000 // 发送1000个包

	data := make([]byte, packetSize)
	for i := 0; i < packetSize; i++ {
		data[i] = byte(i % 256)
	}

	startTime := time.Now()
	sentPackets := 0

	for i := 0; i < numPackets; i++ {
		packet := UDPPacket{
			// ID:        i,
			// Timestamp: time.Now(),
			Data: data,
		}

		// 发送数据包
		_, err := conn.Write(packet.Data)
		if err != nil {
			return fmt.Errorf("failed to send UDP packet: %v", err)
		}
		sentPackets++

		// 短暂休眠以避免过快发送
		time.Sleep(1 * time.Millisecond)
	}

	elapsed := time.Since(startTime)
	totalBytes := int64(sentPackets * packetSize)
	throughput := float64(totalBytes) / elapsed.Seconds()

	fmt.Printf("UDP带宽测试完成:\n")
	fmt.Printf("  测试时长: %.2fs\n", elapsed.Seconds())
	fmt.Printf("  发送数据包数: %d\n", sentPackets)
	fmt.Printf("  总字节数: %d\n", totalBytes)
	fmt.Printf("  速度: %s\n", utils.FormatSpeedDetailed(throughput))

	return nil
}

// UDPBandwidthTestWithThreads 执行UDP带宽测试（多线程）
func (ut *UDPTester) UDPBandwidthTestWithThreads(conn *net.UDPConn, threads int) error {
	return ut.UDPBandwidthTestWithDuration(conn, threads, 10) // 默认10秒
}

// UDPBandwidthTestWithDuration 执行UDP带宽测试（多线程，指定持续时间）
func (ut *UDPTester) UDPBandwidthTestWithDuration(conn *net.UDPConn, threads int, durationSeconds int) error {
	fmt.Printf("开始UDP带宽测试，线程数: %d，时长: %d秒...\n", threads, durationSeconds)

	if threads <= 0 {
		threads = 1
	}
	if durationSeconds <= 0 {
		durationSeconds = 10
	}

	// 创建通道用于收集各线程的结果
	results := make(chan error, threads)

	// 计算结束时间
	endTime := time.Now().Add(time.Duration(durationSeconds) * time.Second)

	// 启动多个goroutine进行并发测试
	for i := 0; i < threads; i++ {
		go func(threadID int) {
			packetSize := 1024 // 1KB packets

			data := make([]byte, packetSize)
			for j := 0; j < packetSize; j++ {
				data[j] = byte((threadID*1000 + j) % 256)
			}

			startTime := time.Now()
			sentPackets := 0

			for time.Now().Before(endTime) {
				packet := UDPPacket{
					// ID:        threadID*10000 + sentPackets,
					// Timestamp: time.Now(),
					Data: data,
				}

				// 发送数据包
				_, err := conn.Write(packet.Data)
				if err != nil {
					results <- fmt.Errorf("thread %d: failed to send UDP packet: %v", threadID, err)
					return
				}
				sentPackets++

				// 短暂休眠以避免过快发送
				time.Sleep(1 * time.Millisecond)
			}

			elapsed := time.Since(startTime)
			totalBytes := int64(sentPackets * packetSize)
			throughput := float64(totalBytes) / elapsed.Seconds()

			fmt.Printf("  线程 %d 完成: 时长: %.2fs, 数据包数: %d, 速度: %s\n", threadID, elapsed.Seconds(), sentPackets, utils.FormatSpeedDetailed(throughput))
			results <- nil
		}(i)
	}

	// 等待所有线程完成
	for i := 0; i < threads; i++ {
		err := <-results
		if err != nil {
			return err
		}
	}

	fmt.Printf("UDP带宽测试完成，线程数: %d，时长: %d秒\n", threads, durationSeconds)
	return nil
}

// UDPLatencyTestWithThreads 执行UDP延迟测试（多线程）
func (ut *UDPTester) UDPLatencyTestWithThreads(conn *net.UDPConn, threads int) error {
	return ut.UDPLatencyTestWithDuration(conn, threads, 10) // 默认10秒
}

// UDPLatencyTestWithDuration 执行UDP延迟测试（多线程，指定持续时间）
func (ut *UDPTester) UDPLatencyTestWithDuration(conn *net.UDPConn, threads int, durationSeconds int) error {
	fmt.Printf("开始UDP延迟测试，线程数: %d，时长: %d秒...\n", threads, durationSeconds)

	if threads <= 0 {
		threads = 1
	}
	if durationSeconds <= 0 {
		durationSeconds = 10
	}

	// 创建通道用于收集各线程的结果
	results := make(chan error, threads)

	// 计算结束时间
	endTime := time.Now().Add(time.Duration(durationSeconds) * time.Second)

	// 启动多个goroutine进行并发测试
	for i := 0; i < threads; i++ {
		go func(threadID int) {
			packetSize := 64 // 64字节的数据包

			data := make([]byte, packetSize)
			for j := 0; j < packetSize; j++ {
				data[j] = byte((threadID*100 + j) % 256)
			}

			totalRTT := time.Duration(0)
			successfulPackets := 0

			for time.Now().Before(endTime) {
				packet := UDPPacket{
					// ID:        threadID*1000 + successfulPackets,
					// Timestamp: time.Now(),
					Data: data,
				}

				sendTime := time.Now()
				_, err := conn.Write(packet.Data)
				if err != nil {
					fmt.Printf("线程 %d: 发送UDP数据包失败: %v\n", threadID, err)
					break
				}

				// 设置读取超时
				conn.SetReadDeadline(time.Now().Add(ut.Timeout))

				// 尝试读取响应
				buf := make([]byte, 1024)
				_, addr, err := conn.ReadFromUDP(buf)
				if err != nil {
					fmt.Printf("线程 %d: 接收响应失败: %v\n", threadID, err)
					// 如果读取失败，短暂等待再继续
					time.Sleep(10 * time.Millisecond)
					continue
				}

				receiveTime := time.Now()
				rtt := receiveTime.Sub(sendTime)
				totalRTT += rtt
				successfulPackets++

				fmt.Printf("  线程 %d, 数据包 %d: 往返时间 = %v, 服务器 = %s\n", threadID, successfulPackets, rtt, addr)

				// 短暂休眠以避免过快发送
				time.Sleep(10 * time.Millisecond)
			}

			if successfulPackets > 0 {
				avgRTT := totalRTT / time.Duration(successfulPackets)
				fmt.Printf("  线程 %d 完成: 平均往返时间: %v, 成功数: %d\n", threadID, avgRTT, successfulPackets)
			} else {
				fmt.Printf("  线程 %d: 未收到响应\n", threadID)
			}
			results <- nil
		}(i)
	}

	// 等待所有线程完成
	for i := 0; i < threads; i++ {
		err := <-results
		if err != nil {
			return err
		}
	}

	fmt.Printf("UDP延迟测试完成，线程数: %d，时长: %d秒\n", threads, durationSeconds)
	return nil
}

// UDPLatencyTest 执行UDP延迟测试
func (ut *UDPTester) UDPLatencyTest(conn *net.UDPConn) error {
	return ut.UDPLatencyTestWithBandwidth(conn, "")
}

// UDPLatencyTestOriginal 原来的UDP延迟测试实现
func (ut *UDPTester) UDPLatencyTestOriginal(conn *net.UDPConn) error {
	return ut.UDPLatencyTestInternal(conn)
}

// UDPBandwidthTestWithBandwidth 执行UDP带宽测试（支持目标带宽限制）
func (ut *UDPTester) UDPBandwidthTestWithBandwidth(conn *net.UDPConn, threads int, durationSeconds int, targetBandwidth string) error {
	if targetBandwidth == "" {
		// 如果没有指定目标带宽，则执行普通带宽测试
		return ut.UDPBandwidthTestWithDuration(conn, threads, durationSeconds)
	}

	// 解析目标带宽
	targetBps, err := parseBandwidth(targetBandwidth)
	if err != nil {
		return fmt.Errorf("invalid bandwidth format: %v", err)
	}

	fmt.Printf("开始UDP带宽测试，目标带宽: %s，线程数: %d，时长: %d秒...\n", targetBandwidth, threads, durationSeconds)

	if threads <= 0 {
		threads = 1
	}
	if durationSeconds <= 0 {
		durationSeconds = 10
	}

	// 创建通道用于收集各线程的结果
	results := make(chan error, threads)

	// 计算结束时间
	endTime := time.Now().Add(time.Duration(durationSeconds) * time.Second)

	// 计算每秒需要发送的字节数
	bytesPerSecond := int64(targetBps / 8)
	// 计算每个线程每秒应该发送的字节数
	bytesPerSecondPerThread := bytesPerSecond / int64(threads)
	// 使用100ms的时间窗口来分配数据发送
	windowSize := time.Millisecond * 100
	bytesPerWindowPerThread := bytesPerSecondPerThread / 10 // 10个窗口 = 1秒

	// 启动多个goroutine进行并发测试
	for i := 0; i < threads; i++ {
		go func(threadID int) {
			startTime := time.Now()
			sentPackets := 0
			sentBytes := int64(0)

			// 记录上次窗口时间
			lastWindow := startTime

			for time.Now().Before(endTime) {
				currentTime := time.Now()
				// 检查是否进入新的时间窗口
				if currentTime.Sub(lastWindow) >= windowSize {
					lastWindow = currentTime
				}

				// 计算当前窗口内还可以发送多少字节
				windowStart := lastWindow
				timeSinceWindowStart := time.Since(windowStart)
				// 如果仍在当前窗口内，计算已用时间比例
				if timeSinceWindowStart < windowSize {
					// 按比例计算可用字节数
					usedWindowRatio := float64(timeSinceWindowStart) / float64(windowSize)
					usedBytesInWindow := int64(float64(bytesPerWindowPerThread) * usedWindowRatio)
					availableBytes := bytesPerWindowPerThread - usedBytesInWindow

					if availableBytes > 0 {
						// 发送适当大小的数据包
						packetSize := int(availableBytes)
						// 限制单个数据包大小，避免过大
						if packetSize > 1472 { // UDP MTU建议值
							packetSize = 1472
						} else if packetSize < 64 { // 最小合理数据包大小
							packetSize = 64
						}

						data := make([]byte, packetSize)
						for j := 0; j < packetSize; j++ {
							data[j] = byte((threadID*1000 + sentPackets + j) % 256)
						}

						err := conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
						if err != nil {
							results <- fmt.Errorf("thread %d: failed to set write deadline: %v", threadID, err)
							return
						}

						n, err := conn.Write(data)
						if err != nil {
							results <- fmt.Errorf("thread %d: failed to send UDP packet: %v", threadID, err)
							return
						}

						sentPackets++
						sentBytes += int64(n)

						// 短暂休眠以避免过度占用CPU
						time.Sleep(time.Microsecond * 100)
					} else {
						// 当前窗口已满，等待下一个窗口
						time.Sleep(windowSize - time.Since(lastWindow))
					}
				} else {
					// 时间窗口已过，等待下一个
					time.Sleep(windowSize)
				}
			}

			elapsed := time.Since(startTime)
			throughput := float64(sentBytes) / elapsed.Seconds()

			fmt.Printf("  线程 %d 完成: 时长: %.2fs, 数据包数: %d, 实际速度: %s\n",
				threadID, elapsed.Seconds(), sentPackets, utils.FormatSpeedDetailed(throughput))
			results <- nil
		}(i)
	}

	// 等待所有线程完成
	for i := 0; i < threads; i++ {
		err := <-results
		if err != nil {
			return err
		}
	}

	fmt.Printf("UDP带宽测试完成，目标带宽: %s，线程数: %d，时长: %d秒\n", targetBandwidth, threads, durationSeconds)
	return nil
}

// parseBandwidth 解析带宽字符串，返回比特每秒
func parseBandwidth(bandwidth string) (int64, error) {
	// 移除空格
	bandwidth = strings.TrimSpace(bandwidth)

	// 获取数值和单位
	var numStr string
	var unit string
	for i, r := range bandwidth {
		if r >= '0' && r <= '9' || r == '.' {
			numStr += string(r)
		} else {
			unit = bandwidth[i:]
			break
		}
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in bandwidth: %v", err)
	}

	// 解析单位
	switch strings.ToUpper(unit) {
	case "B", "":
		return int64(num), nil // bytes per second
	case "K", "KB":
		return int64(num * 1000), nil // kilobytes per second
	case "M", "MB":
		return int64(num * 1000 * 1000), nil // megabytes per second
	case "G", "GB":
		return int64(num * 1000 * 1000 * 1000), nil // gigabytes per second
	case "KBIT", "Kb":
		return int64(num * 1000 / 8), nil // kilobits per second
	case "MBIT", "Mb":
		return int64(num * 1000 * 1000 / 8), nil // megabits per second
	case "GBIT", "Gb":
		return int64(num * 1000 * 1000 * 1000 / 8), nil // gigabits per second
	default:
		return 0, fmt.Errorf("unknown bandwidth unit: %s", unit)
	}
}

// UDPLatencyTestWithBandwidth 执行UDP延迟测试（支持带宽参数但忽略）
func (ut *UDPTester) UDPLatencyTestWithBandwidth(conn *net.UDPConn, targetBandwidth string) error {
	// 对于延迟测试，带宽参数被忽略
	return ut.UDPLatencyTestInternal(conn)
}

// UDPLatencyTestInternal 执行UDP延迟测试的内部实现
func (ut *UDPTester) UDPLatencyTestInternal(conn *net.UDPConn) error {
	fmt.Println("开始UDP延迟测试...")

	packetSize := 64 // 64字节的数据包
	numPackets := 10 // 发送10个包来测试延迟

	data := make([]byte, packetSize)
	for i := 0; i < packetSize; i++ {
		data[i] = byte(i % 256)
	}

	totalRTT := time.Duration(0)
	successfulPackets := 0

	for i := 0; i < numPackets; i++ {
		packet := UDPPacket{
			// ID:        i,
			// Timestamp: time.Now(),
			Data: data,
		}

		sendTime := time.Now()
		_, err := conn.Write(packet.Data)
		if err != nil {
			fmt.Printf("发送UDP数据包 %d 失败: %v\n", i, err)
			continue
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(ut.Timeout))

		// 尝试读取响应
		buf := make([]byte, 1024)
		_, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("接收数据包 %d 的响应失败: %v\n", i, err)
			continue
		}

		receiveTime := time.Now()
		rtt := receiveTime.Sub(sendTime)
		totalRTT += rtt
		successfulPackets++

		fmt.Printf("  数据包 %d: 往返时间 = %v, 服务器 = %s\n", i, rtt, addr)
	}

	if successfulPackets > 0 {
		avgRTT := totalRTT / time.Duration(successfulPackets)
		fmt.Printf("UDP延迟测试完成:\n")
		fmt.Printf("  平均往返时间: %v\n", avgRTT)
		fmt.Printf("  成功的数据包: %d/%d\n", successfulPackets, numPackets)
	} else {
		fmt.Println("UDP延迟测试失败: 未收到响应")
	}

	return nil
}
