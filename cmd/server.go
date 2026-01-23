package cmd

import (
	"fmt"
	"net"
	"strings"
	"time"

	"perfgo/pkg/bandwidth"
	"perfgo/pkg/latency"
	"perfgo/pkg/packetloss"
	"perfgo/pkg/protocol"
	"perfgo/pkg/udp"
)

// Server 启动服务器模式
func Server(port string, bindIP string) error {
	// 确定TCP服务器绑定地址
	tcpAddr := ":" + port
	if bindIP != "" {
		tcpAddr = bindIP + ":" + port
	}
	// 启动TCP服务器
	tcpListener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server on %s: %v", tcpAddr, err)
	}
	defer tcpListener.Close()

	// 确定UDP服务器绑定地址
	udpAddrStr := ":" + port
	if bindIP != "" {
		udpAddrStr = bindIP + ":" + port
	}
	// 启动UDP服务器
	udpAddr, err := net.ResolveUDPAddr("udp", udpAddrStr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server on %s: %v", udpAddrStr, err)
	}
	defer udpListener.Close()

	fmt.Printf("Server listening on port %s (TCP and UDP)...\n", port)

	// 启动UDP处理协程
	go handleUDPConnection(udpListener)

	// TCP服务器主循环
	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			fmt.Printf("Error accepting TCP connection: %v\n", err)
			continue
		}

		fmt.Printf("New TCP connection from %s\n", conn.RemoteAddr())

		// 为每个TCP连接启动处理协程
		go handleConnection(conn)
	}
}

// handleConnection 处理单个TCP连接
func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// 接收消息
		msg, err := protocol.Receive(conn)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				// 连接已被关闭
				return
			}
			fmt.Printf("Error receiving message: %v\n", err)
			return
		}

		// 根据消息类型处理
		switch msg.Type {
		case protocol.TypeTestRequest:
			err = handleTestRequest(conn, msg)
			if err != nil {
				fmt.Printf("Error handling test request: %v\n", err)
				return
			}
		case protocol.TypeClose:
			fmt.Println("Client requested close")
			return
		default:
			fmt.Printf("Unknown message type: %v\n", msg.Type)
			return
		}
	}
}

// handleUDPConnection 处理UDP连接
func handleUDPConnection(udpConn *net.UDPConn) {
	fmt.Println("UDP server started, listening for UDP connections...")

	buf := make([]byte, 4096)
	for {
		// 设置读取超时，避免无限阻塞
		udpConn.SetReadDeadline(time.Now().Add(5 * time.Second))

		n, clientAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时是正常的，继续循环
				continue
			}
			fmt.Printf("Error reading UDP message: %v\n", err)
			continue
		}

		// 尝试解析协议消息
		var msg protocol.Message
		err = protocol.UnmarshalMessage(buf[:n], &msg)
		if err != nil {
			fmt.Printf("Error unmarshaling UDP message: %v\n", err)
			continue
		}

		fmt.Printf("Received UDP message from %s: type=%v, testType=%s\n", clientAddr, msg.Type, msg.TestType)

		// 根据消息类型处理
		switch msg.Type {
		case protocol.TypeTestRequest:
			err = handleUDPTestRequest(udpConn, &msg)
			if err != nil {
				fmt.Printf("Error handling UDP test request: %v\n", err)
			}
		default:
			fmt.Printf("Unknown UDP message type: %v\n", msg.Type)
		}
	}
}

// handleTestRequest 处理TCP测试请求
func handleTestRequest(conn net.Conn, msg *protocol.Message) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("missing test_type in payload")
	}

	fmt.Printf("Received %s test request\n", testType)

	switch msg.TestType {
	case "bandwidth":
		// 检查是否提供了测试持续时间
		duration, ok := msg.Payload["duration"].(float64)
		if !ok {
			duration = 10 // 默认10秒
		}
		// 创建一个新的BandwidthTester实例，使用从客户端接收的持续时间
		bwTester := &bandwidth.BandwidthTester{
			BufferSize: bandwidth.DefaultBufferSize,
			Duration:   time.Duration(duration) * time.Second,
		}
		return bwTester.ServerHandle(conn, msg)
	case "latency":
		latencyTester := latency.NewLatencyTester()
		return latencyTester.ServerHandle(conn, msg)
	case "packetloss":
		plt := packetloss.NewPacketLossTester()
		return plt.ServerHandle(conn, msg)
	default:
		return fmt.Errorf("unknown test type: %s", msg.TestType)
	}
}

// handleUDPTestRequest 处理UDP测试请求
func handleUDPTestRequest(udpConn *net.UDPConn, msg *protocol.Message) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("missing test_type in payload")
	}

	fmt.Printf("Received UDP %s test request\n", testType)

	switch msg.TestType {
	case "udp":
		ut := udp.NewUDPTester()
		return ut.ServerHandle(udpConn, msg)
	default:
		return fmt.Errorf("unknown UDP test type: %s", msg.TestType)
	}
}
