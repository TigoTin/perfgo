package udp

import (
	"fmt"
	"net"
	"time"

	"perfgo/pkg/protocol"
)

// ServerHandle 处理来自客户端的UDP测试请求
func (ut *UDPTester) ServerHandle(conn *net.UDPConn, msg *protocol.Message) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("missing test_type in payload")
	}

	fmt.Printf("Received UDP %s test request\n", testType)

	// 检查是否提供了测试持续时间
	duration, ok := msg.Payload["duration"].(float64)
	if !ok {
		duration = 10 // 默认10秒
	}

	ut.Timeout = time.Duration(duration) * time.Second
	if ut.Timeout < 5*time.Second {
		ut.Timeout = 5 * time.Second // 最小5秒超时
	}

	// 根据测试类型响应客户端
	switch testType {
	case "bandwidth":
		return ut.UDPBandwidthServerHandler(conn)
	case "latency":
		return ut.UDPLatencyServerHandler(conn)
	default:
		return fmt.Errorf("unknown UDP test type: %s", testType)
	}
}

// ServerHandleUDPMessage 处理来自TCP连接的UDP测试请求
func (ut *UDPTester) ServerHandleUDPMessage(tcpConn net.Conn, msg *protocol.Message) error {
	_, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("missing test_type in payload")
	}

	return fmt.Errorf("TCP-initiated UDP tests not supported directly. Use UDP connection for UDP tests.")
}

// ClientHandler 客户端处理UDP测试请求（支持多线程和持续时间）
func ClientHandler(serverAddr string, testType string, localIP string, threads int, duration int) error {
	// 解析UDP服务器地址
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	// 如果指定了本地IP，则解析本地地址
	var localAddr *net.UDPAddr
	if localIP != "" {
		localAddr, err = net.ResolveUDPAddr("udp", localIP+":0")
		if err != nil {
			return fmt.Errorf("failed to resolve local UDP address: %v", err)
		}
	}
	// 连接到UDP服务器
	conn, err := net.DialUDP("udp", localAddr, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to UDP server: %v", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to UDP server at %s. Running %s test with %d threads for %d seconds...\n", serverAddr, testType, threads, duration)

	// 发送测试请求到服务器
	msg := &protocol.Message{
		Type:     protocol.TypeTestRequest,
		TestType: "udp",
		Payload: map[string]interface{}{
			"test_type": testType,
			"threads":   threads,
			"duration":  duration,
		},
	}

	err = msg.Send(conn)
	if err != nil {
		return fmt.Errorf("failed to send UDP test request: %v", err)
	}

	// 根据测试类型执行相应的UDP测试
	ut := NewUDPTester()
	switch testType {
	case "bandwidth":
		return ut.UDPBandwidthTestWithDuration(conn, threads, duration)
	case "latency":
		return ut.UDPLatencyTestWithDuration(conn, threads, duration)
	default:
		return fmt.Errorf("unknown UDP test type: %s", testType)
	}
}

// UDPBandwidthServerHandler 服务器端处理UDP带宽测试
func (ut *UDPTester) UDPBandwidthServerHandler(conn *net.UDPConn) error {
	fmt.Println("Handling UDP bandwidth test on server...")

	// 设置较短的初始读取超时
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// 读取来自客户端的数据
	buf := make([]byte, 2048)
	lastActivity := time.Now()
	maxIdleTime := 2 * time.Second

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 检查是否已经超过最大空闲时间
				if time.Since(lastActivity) >= maxIdleTime {
					fmt.Println("UDP bandwidth test completed on server (timeout)")
					return nil
				}
				// 继续等待更多数据
				continue
			}
			return fmt.Errorf("error reading UDP data: %v", err)
		}

		// 更新最后活动时间
		lastActivity = time.Now()

		// 回显数据到客户端
		_, err = conn.WriteToUDP(buf[:n], clientAddr)
		if err != nil {
			return fmt.Errorf("error echoing UDP data: %v", err)
		}
	}
}

// UDPLatencyServerHandler 服务器端处理UDP延迟测试
func (ut *UDPTester) UDPLatencyServerHandler(conn *net.UDPConn) error {
	fmt.Println("Handling UDP latency test on server...")

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(ut.Timeout))

	// 读取来自客户端的数据并回显
	buf := make([]byte, 2048)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时，延迟测试完成
				fmt.Println("UDP latency test completed on server")
				return nil
			}
			return fmt.Errorf("error reading UDP data: %v", err)
		}

		// 回显数据到客户端
		_, err = conn.WriteToUDP(buf[:n], clientAddr)
		if err != nil {
			return fmt.Errorf("error echoing UDP data: %v", err)
		}
	}
}
