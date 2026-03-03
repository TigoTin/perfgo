package server

import (
	"context"
	"fmt"
	"net"

	"github.com/TigoTin/perfgo/internal/tester"
	"github.com/TigoTin/perfgo/pkg/protocol"
)

// Server 网络测试服务器
type Server struct {
	port   string
	bindIP string
}

// NewServer 创建新的服务器实例
func NewServer(port, bindIP string) *Server {
	return &Server{
		port:   port,
		bindIP: bindIP,
	}
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	addr := ":" + s.port
	if s.bindIP != "" {
		addr = s.bindIP + ":" + s.port
	}

	// TCP监听
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("解析TCP地址失败: %v", err)
	}
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("启动TCP服务器失败: %v", err)
	}
	defer tcpListener.Close()

	// UDP监听
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("解析UDP地址失败: %v", err)
	}
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("启动UDP服务器失败: %v", err)
	}
	defer udpListener.Close()

	fmt.Printf("服务器启动，监听端口: %s (TCP 和 UDP)\n", s.port)

	// 启动UDP处理器
	go s.handleUDP(udpListener)

	// TCP服务器主循环
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := tcpListener.Accept()
			if err != nil {
				fmt.Printf("接受TCP连接失败: %v\n", err)
				continue
			}

			fmt.Printf("新TCP连接来自: %s\n", conn.RemoteAddr())

			// 为每个连接启动处理协程
			go s.handleConnection(conn)
		}
	}
}

// handleConnection 处理单个TCP连接
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		msg, err := protocol.Receive(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时，继续等待
				continue
			}
			// 静默处理连接错误
			return
		}

		switch msg.Type {
		case protocol.TypeTestRequest:
			err = s.handleTestRequest(conn, msg)
			if err != nil {
				// 静默处理错误
				return
			}
		case protocol.TypeClose:
			return
		default:
			return
		}
	}
}

// handleTestRequest 处理测试请求
func (s *Server) handleTestRequest(conn net.Conn, msg *protocol.Message) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("缺少 test_type 参数")
	}

	switch testType {
	case "tcp":
		threads, _ := msg.Payload["threads"].(float64)
		duration, _ := msg.Payload["duration"].(float64)
		fmt.Printf("TCP测试请求 - 线程数: %d, 时长: %d秒\n", int(threads), int(duration))

		tester := tester.NewNetworkTester()
		return tester.HandleTCPTest(conn, int(threads), int(duration))
	case "udp":
		threads, _ := msg.Payload["threads"].(float64)
		duration, _ := msg.Payload["duration"].(float64)
		bandwidth, _ := msg.Payload["bandwidth"].(string)
		if bandwidth != "" {
			fmt.Printf("UDP测试请求 - 线程数: %d, 时长: %d秒, 目标带宽: %s\n", int(threads), int(duration), bandwidth)
		} else {
			fmt.Printf("UDP测试请求 - 线程数: %d, 时长: %d秒\n", int(threads), int(duration))
		}

		tester := tester.NewNetworkTester()
		return tester.HandleUDPTest(conn, int(threads), int(duration), bandwidth)
	default:
		return fmt.Errorf("未知测试类型: %s", testType)
	}
}

// handleUDP 处理UDP连接
func (s *Server) handleUDP(udpConn *net.UDPConn) {
	fmt.Println("UDP服务器启动，监听UDP连接...")

	buf := make([]byte, 4096)
	for {
		n, clientAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("读取UDP消息失败: %v\n", err)
			continue
		}

		// 尝试解析协议消息
		var msg protocol.Message
		err = protocol.UnmarshalMessage(buf[:n], &msg)
		if err != nil {
			fmt.Printf("解析UDP消息失败: %v\n", err)
			continue
		}

		fmt.Printf("从 %s 接收UDP消息: type=%v, testType=%s\n", clientAddr, msg.Type, msg.TestType)

		// 处理UDP测试请求
		go s.handleUDPTestRequest(udpConn, &msg, clientAddr)
	}
}

// handleUDPTestRequest 处理UDP测试请求
func (s *Server) handleUDPTestRequest(udpConn *net.UDPConn, msg *protocol.Message, clientAddr *net.UDPAddr) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("缺少 test_type 参数")
	}

	fmt.Printf("收到UDP %s 测试请求\n", testType)

	switch testType {
	case "udp":
		threads, _ := msg.Payload["threads"].(float64)
		duration, _ := msg.Payload["duration"].(float64)
		bandwidth, _ := msg.Payload["bandwidth"].(string)

		tester := tester.NewNetworkTester()
		return tester.HandleUDPTestUDP(udpConn, clientAddr, int(threads), int(duration), bandwidth)
	default:
		return fmt.Errorf("未知UDP测试类型: %s", testType)
	}
}
