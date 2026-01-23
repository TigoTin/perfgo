package packetloss

import (
	"fmt"
	"net"
	"sync"
	"time"

	"perfgo/pkg/protocol"
)

const (
	// DefaultPacketCount 默认发送的数据包数量
	DefaultPacketCount = 100
	// PacketTimeout 数据包超时时间
	PacketTimeout = 2 * time.Second
)

// PacketLossTester 丢包率测试器
type PacketLossTester struct {
	PacketCount int
	Timeout     time.Duration
}

// NewPacketLossTester 创建新的丢包率测试器
func NewPacketLossTester() *PacketLossTester {
	return &PacketLossTester{
		PacketCount: DefaultPacketCount,
		Timeout:     PacketTimeout,
	}
}

// Test 执行丢包率测试
func (plt *PacketLossTester) Test(conn net.Conn) error {
	fmt.Printf("开始丢包率测试 (共%d个数据包)...\n", plt.PacketCount)

	// 发送数据包并记录ID
	sentPackets := make(map[int]bool)
	receivedPackets := make(map[int]bool)

	var mu sync.Mutex

	// 启动接收协程
	receiveDone := make(chan bool, 1)
	go func() {
		defer func() { receiveDone <- true }()

		for {
			conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100)) // 短暂超时以检查退出条件
			msg, err := protocol.Receive(conn)
			if err != nil {
				// 检查是否是超时错误，如果不是网络错误则可能测试已完成
				if netErr, ok := err.(net.Error); ok && !netErr.Timeout() {
					return // 非超时错误，退出
				}
				// 超时是正常的，继续等待
				continue
			}

			// 解析数据包ID
			var id int
			if n, err := fmt.Sscanf(msg.ID, "data-%d", &id); err == nil && n == 1 && id >= 0 {
				mu.Lock()
				receivedPackets[id] = true
				mu.Unlock()
			}
		}
	}()

	// 发送数据包
	for i := 0; i < plt.PacketCount; i++ {
		packetMsg := &protocol.Message{
			Type:      protocol.TypeData,
			Timestamp: time.Now(),
			ID:        fmt.Sprintf("data-%d", i),
			Payload: map[string]interface{}{
				"data": fmt.Sprintf("packet-data-%d", i),
			},
		}

		err := packetMsg.Send(conn)
		if err != nil {
			fmt.Printf("发送数据包 %d 失败: %v\n", i, err)
		} else {
			mu.Lock()
			sentPackets[i] = true
			mu.Unlock()
		}

		// 短暂延迟以避免过快发送
		time.Sleep(time.Millisecond * 10)
	}

	// 等待一段时间以接收回复的数据包
	time.Sleep(plt.Timeout)

	// 停止接收协程
	close(receiveDone)

	// 等待一小段时间确保接收协程结束
	time.Sleep(time.Millisecond * 100)

	// 计算结果
	mu.Lock()
	defer mu.Unlock()

	sentCount := len(sentPackets)
	receivedCount := len(receivedPackets)

	var lostPackets []int
	for i := 0; i < plt.PacketCount; i++ {
		if sentPackets[i] && !receivedPackets[i] {
			lostPackets = append(lostPackets, i)
		}
	}

	packetLossRate := float64(len(lostPackets)) / float64(sentCount) * 100

	fmt.Printf("\n丢包率测试结果:\n")
	fmt.Printf("  发送数据包数: %d\n", sentCount)
	fmt.Printf("  接收数据包数: %d\n", receivedCount)
	fmt.Printf("  丢失数据包数: %d\n", len(lostPackets))
	fmt.Printf("  丢包率: %.2f%%\n", packetLossRate)

	if len(lostPackets) > 0 {
		fmt.Printf("  丢失的数据包ID: %v\n", lostPackets[:min(10, len(lostPackets))]) // 只显示前10个丢失的包
		if len(lostPackets) > 10 {
			fmt.Printf("  ... 还有 %d 个\n", len(lostPackets)-10)
		}
	}

	return nil
}

// ServerHandle 处理来自客户端的丢包率测试请求
func (plt *PacketLossTester) ServerHandle(conn net.Conn, msg *protocol.Message) error {
	// 服务器只是接收客户端发送的数据包并原样返回
	receivedCount := 0

	for receivedCount < plt.PacketCount {
		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(plt.Timeout))

		// 接收客户端的数据包
		clientMsg, err := protocol.Receive(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时但还未收到足够数据包，继续等待
				continue
			}
			return fmt.Errorf("failed to receive data from client: %v", err)
		}

		// 回显数据包给客户端
		err = clientMsg.Send(conn)
		if err != nil {
			fmt.Printf("回显数据包到客户端失败: %v\n", err)
			continue
		}

		receivedCount++
	}

	return nil
}

// min 是一个辅助函数，返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
