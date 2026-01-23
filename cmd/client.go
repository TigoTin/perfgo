package cmd

import (
	"fmt"
	"net"

	"perfgo/pkg/bandwidth"
	"perfgo/pkg/latency"
	"perfgo/pkg/packetloss"
	"perfgo/pkg/udp"
)

// isValidLocalIP 检查给定的IP地址是否是本机的有效IP地址
func isValidLocalIP(ip string) (bool, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.IP.String() == ip {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// isLocalhost 检查给定的地址是否是本地回环地址
func isLocalhost(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// 如果无法分割端口，直接使用原字符串
		host = addr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// 如果不是有效的IP格式，尝试解析为主机名
		ips, err := net.LookupIP(host)
		if err != nil {
			return false
		}
		for _, lookupIP := range ips {
			if lookupIP.IsLoopback() {
				return true
			}
		}
		return false
	}
	return ip.IsLoopback()
}

// Client 启动客户端模式并执行指定测试
func Client(host, port, testType string, threads int, localIP string, duration int, targetBandwidth string) error {
	serverAddr := host + ":" + port
	fmt.Printf("Connecting to server at %s\n", serverAddr)

	var conn net.Conn
	var err error
	if localIP != "" {
		// 验证本地IP地址是否属于本机
		validIP, err := isValidLocalIP(localIP)
		if err != nil || !validIP {
			return fmt.Errorf("invalid local IP address %s: %v", localIP, err)
		}
		// 检查目标地址是否是本地回环地址
		if isLocalhost(serverAddr) {
			fmt.Printf("Warning: Target address %s is localhost, ignoring localIP specification\n", serverAddr)
			conn, err = net.Dial("tcp", serverAddr)
			if err != nil {
				return fmt.Errorf("failed to connect to server: %v", err)
			}
		} else {
			// 如果指定了有效的本地IP，则创建带有本地地址绑定的TCP连接
			localAddr, err := net.ResolveTCPAddr("tcp", localIP+":0")
			if err != nil {
				return fmt.Errorf("failed to resolve local TCP address: %v", err)
			}
			remoteAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
			if err != nil {
				return fmt.Errorf("failed to resolve remote TCP address: %v", err)
			}
			dialer := net.Dialer{
				LocalAddr: localAddr,
			}
			conn, err = dialer.Dial("tcp", remoteAddr.String())
			if err != nil {
				return fmt.Errorf("failed to connect to server with local IP %s: %v", localIP, err)
			}
		}
	} else {
		conn, err = net.Dial("tcp", serverAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %v", err)
		}
	}
	defer conn.Close()

	fmt.Printf("Connected to server. Running %s test...\n", testType)

	switch testType {
	case "bandwidth-upload":
		return bandwidth.ClientHandler(conn, "upload", threads, duration)
	case "bandwidth-download":
		return bandwidth.ClientHandler(conn, "download", threads, duration)
	case "latency-ping":
		return latency.ClientHandler(conn, "ping")
	case "latency-jitter":
		return latency.ClientHandler(conn, "jitter")
	case "packetloss":
		return packetloss.ClientHandler(conn)
	case "udp-bandwidth":
		// 对于UDP测试，使用指定的持续时间和目标带宽
		return udp.ClientHandlerWithBandwidth(serverAddr, "bandwidth", localIP, threads, duration, targetBandwidth)
	case "udp-latency":
		// 对于UDP测试，使用指定的持续时间
		return udp.ClientHandler(serverAddr, "latency", localIP, threads, duration)
	default:
		return fmt.Errorf("unknown test type: %s", testType)
	}
}
