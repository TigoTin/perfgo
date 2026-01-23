package udp

import (
	"fmt"
	"net"
	"time"
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
	fmt.Printf("  速度: %.2f 字节/秒 (%.2f Mbps)\n", throughput, throughput*8/1024/1024)

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

			fmt.Printf("  线程 %d 完成: 时长: %.2fs, 数据包数: %d, 速度: %.2f 字节/秒 (%.2f Mbps)\n", threadID, elapsed.Seconds(), sentPackets, throughput, throughput*8/1024/1024)
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
