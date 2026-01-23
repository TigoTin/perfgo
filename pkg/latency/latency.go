package latency

import (
	"fmt"
	"net"
	"time"

	"perfgo/pkg/protocol"
)

const (
	// DefaultPingCount 默认ping次数
	DefaultPingCount = 10
	// PingTimeout ping超时时间
	PingTimeout = 3 * time.Second
)

// LatencyTester 延迟测试器
type LatencyTester struct {
	PingCount int
	Timeout   time.Duration
}

// NewLatencyTester 创建新的延迟测试器
func NewLatencyTester() *LatencyTester {
	return &LatencyTester{
		PingCount: DefaultPingCount,
		Timeout:   PingTimeout,
	}
}

// PingTest 执行ping测试，测量往返时间
func (lt *LatencyTester) PingTest(conn net.Conn) error {
	fmt.Printf("开始ping测试 (共%d次)...\n", lt.PingCount)

	rtts := make([]time.Duration, 0, lt.PingCount)

	for i := 0; i < lt.PingCount; i++ {
		// 发送ping消息
		pingMsg := &protocol.Message{
			Type:      protocol.TypeHeartbeat,
			Timestamp: time.Now(),
			ID:        fmt.Sprintf("ping-%d", i),
		}

		sendTime := time.Now()
		err := pingMsg.Send(conn)
		if err != nil {
			return fmt.Errorf("failed to send ping: %v", err)
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(lt.Timeout))

		// 等待响应
		_, err = protocol.Receive(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("Ping %d: timeout\n", i+1)
				continue
			}
			return fmt.Errorf("failed to receive pong: %v", err)
		}

		recvTime := time.Now()
		rtt := recvTime.Sub(sendTime)
		rtts = append(rtts, rtt)

		fmt.Printf("Ping %d: %v\n", i+1, rtt)

		// 等待1秒再进行下次ping
		time.Sleep(time.Second)
	}

	if len(rtts) == 0 {
		fmt.Println("所有ping请求均超时!")
		return nil
	}

	// 计算统计信息
	minRtt := rtts[0]
	maxRtt := rtts[0]
	var totalRtt time.Duration
	var jitter time.Duration

	for _, rtt := range rtts {
		if rtt < minRtt {
			minRtt = rtt
		}
		if rtt > maxRtt {
			maxRtt = rtt
		}
		totalRtt += rtt
	}

	avgRtt := totalRtt / time.Duration(len(rtts))

	// 计算抖动 (平均差异)
	for _, rtt := range rtts {
		diff := rtt - avgRtt
		if diff < 0 {
			diff = -diff
		}
		jitter += diff
	}
	jitter = jitter / time.Duration(len(rtts))

	fmt.Println("\nping统计信息:")
	fmt.Printf("  发送的数据包: %d\n", lt.PingCount)
	fmt.Printf("  接收的数据包: %d\n", len(rtts))
	fmt.Printf("  丢包率: %.2f%%\n", float64(lt.PingCount-len(rtts))/float64(lt.PingCount)*100)
	fmt.Printf("  最小/平均/最大/抖动: %v/%v/%v/%v\n", minRtt, avgRtt, maxRtt, jitter)

	return nil
}

// JitterTest 执行抖动测试，测量延迟变化
func (lt *LatencyTester) JitterTest(conn net.Conn) error {
	fmt.Printf("开始抖动测试 (共%d次测量)...\n", lt.PingCount)

	rtts := make([]time.Duration, 0, lt.PingCount)

	for i := 0; i < lt.PingCount; i++ {
		// 发送心跳消息
		pingMsg := &protocol.Message{
			Type:      protocol.TypeHeartbeat,
			Timestamp: time.Now(),
			ID:        fmt.Sprintf("jitter-%d", i),
		}

		sendTime := time.Now()
		err := pingMsg.Send(conn)
		if err != nil {
			return fmt.Errorf("failed to send heartbeat: %v", err)
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(lt.Timeout))

		// 等待响应
		_, err = protocol.Receive(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("抖动测量 %d: 超时\n", i+1)
				continue
			}
			return fmt.Errorf("failed to receive heartbeat response: %v", err)
		}

		recvTime := time.Now()
		rtt := recvTime.Sub(sendTime)
		rtts = append(rtts, rtt)
	}

	if len(rtts) < 2 {
		fmt.Println("抖动计算的测量次数不足!")
		return nil
	}

	// 计算连续RTT之间的差异（抖动）
	var totalJitter time.Duration
	for i := 1; i < len(rtts); i++ {
		diff := rtts[i] - rtts[i-1]
		if diff < 0 {
			diff = -diff
		}
		totalJitter += diff
	}

	avgJitter := totalJitter / time.Duration(len(rtts)-1)

	fmt.Printf("抖动测试完成:\n")
	fmt.Printf("  测量次数: %d\n", len(rtts))
	fmt.Printf("  平均抖动: %v\n", avgJitter)

	return nil
}

// ServerHandle 处理来自客户端的延迟测试请求
func (lt *LatencyTester) ServerHandle(conn net.Conn, msg *protocol.Message) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("missing test_type in payload")
	}

	switch testType {
	case "ping":
		// 服务器只是转发心跳消息回客户端
		return lt.handlePingServer(conn)
	case "jitter":
		// 服务器只是转发心跳消息回客户端
		return lt.handleJitterServer(conn)
	default:
		return fmt.Errorf("unknown test type: %s", testType)
	}
}

// handlePingServer 处理ping测试的服务器端逻辑
func (lt *LatencyTester) handlePingServer(conn net.Conn) error {
	// 在ping测试中，服务器只需接收客户端的心跳消息并返回
	for i := 0; i < lt.PingCount; i++ {
		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(lt.Timeout))

		// 接收客户端的ping消息
		msg, err := protocol.Receive(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // 跳过超时的消息
			}
			return fmt.Errorf("failed to receive ping from client: %v", err)
		}

		// 返回相同的消息给客户端
		err = msg.Send(conn)
		if err != nil {
			return fmt.Errorf("failed to send pong to client: %v", err)
		}
	}

	return nil
}

// handleJitterServer 处理抖动测试的服务器端逻辑
func (lt *LatencyTester) handleJitterServer(conn net.Conn) error {
	// 在抖动测试中，服务器只需接收客户端的心跳消息并返回
	for i := 0; i < lt.PingCount; i++ {
		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(lt.Timeout))

		// 接收客户端的心跳消息
		msg, err := protocol.Receive(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // 跳过超时的消息
			}
			return fmt.Errorf("failed to receive heartbeat from client: %v", err)
		}

		// 返回相同的消息给客户端
		err = msg.Send(conn)
		if err != nil {
			return fmt.Errorf("failed to send heartbeat response to client: %v", err)
		}
	}

	return nil
}
