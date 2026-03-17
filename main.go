package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/TigoTin/perfgo/pkg/utils"
	"github.com/TigoTin/perfgo/work/client"
	"github.com/TigoTin/perfgo/work/server"
)

func main() {
	app := &cli.App{
		Name:    "perfgo",
		Usage:   "网络性能测试工具",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:    "server",
				Aliases: []string{"s"},
				Usage:   "启动服务器模式",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Value:   "5432",
						Usage:   "服务器端口",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:  "bind",
						Value: "",
						Usage: "绑定IP地址 (可选)",
					},
				},
				Action: func(cCtx *cli.Context) error {
					return serverAction(cCtx)
				},
			},
			{
				Name:    "interface",
				Aliases: []string{"i"},
				Usage:   "检测网络接口信息（网卡名称、IP、NAT类型）",
				Action: func(cCtx *cli.Context) error {
					return interfaceInfoAction(cCtx)
				},
			},
			{
				Name:    "tcp",
				Aliases: []string{"t"},
				Usage:   "执行TCP网络测试（带宽和延迟）",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Value: "",
						Usage: "服务器主机地址 (单个服务器)",
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "5432",
						Usage:   "服务器端口",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:  "servers",
						Value: "",
						Usage: "多服务端地址，格式: host1:port1,host2:port2 (可选，可指定带宽 host:port/bandwidth 如 1.2.3.4:5432/10M)",
					},
					&cli.IntFlag{
						Name:    "connections",
						Value:   1,
						Usage:   "并发连接数（每个连接独立测试）",
						Aliases: []string{"c", "threads"},
					},
					&cli.IntFlag{
						Name:  "duration",
						Value: 10,
						Usage: "测试持续时间（秒）",
					},
					&cli.StringFlag{
						Name:  "localip",
						Value: "",
						Usage: "本地IP地址 (可选，用于指定源IP进行测试)",
					},
					&cli.StringFlag{
						Name:    "interface",
						Value:   "",
						Usage:   "网络接口名称 (可选，用于指定源接口进行测试；使用 'all' 对所有在线接口进行测试)",
						Aliases: []string{"iface"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					return tcpTestAction(cCtx)
				},
			},
			{
				Name:    "udp",
				Aliases: []string{"u"},
				Usage:   "执行UDP网络测试（带宽和延迟）",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Value: "",
						Usage: "服务器主机地址 (单个服务器)",
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "5432",
						Usage:   "服务器端口",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:  "servers",
						Value: "",
						Usage: "多服务端地址，格式: host1:port1,host2:port2 (可选，可指定带宽 host:port/bandwidth 如 1.2.3.4:5432/10M)",
					},
					&cli.IntFlag{
						Name:    "connections",
						Value:   1,
						Usage:   "并发连接数",
						Aliases: []string{"c", "threads"},
					},
					&cli.IntFlag{
						Name:  "duration",
						Value: 10,
						Usage: "测试持续时间（秒）",
					},
					&cli.StringFlag{
						Name:    "bandwidth",
						Value:   "",
						Usage:   "目标带宽 (例如: 10M, 100K, 1G)，用于UDP带宽测试",
						Aliases: []string{"b"},
					},
					&cli.StringFlag{
						Name:  "localip",
						Value: "",
						Usage: "本地IP地址 (可选，用于指定源IP进行测试)",
					},
					&cli.StringFlag{
						Name:    "interface",
						Value:   "",
						Usage:   "网络接口名称 (可选，用于指定源接口进行测试；使用 'all' 对所有在线接口进行测试)",
						Aliases: []string{"iface"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					return udpTestAction(cCtx)
				},
			},
			{
				Name:    "loss",
				Aliases: []string{"l"},
				Usage:   "执行UDP丢包率测试",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "host",
						Value:    "localhost",
						Usage:    "服务器主机地址",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "5432",
						Usage:   "服务器端口",
						Aliases: []string{"p"},
					},
					&cli.IntFlag{
						Name:    "packets",
						Value:   100,
						Usage:   "发送的数据包数量",
						Aliases: []string{"n"},
					},
					&cli.IntFlag{
						Name:  "size",
						Value: 1024,
						Usage: "数据包大小（字节）",
					},
					&cli.StringFlag{
						Name:  "localip",
						Value: "",
						Usage: "本地IP地址 (可选，用于指定源IP进行测试)",
					},
				},
				Action: func(cCtx *cli.Context) error {
					return packetLossTestAction(cCtx)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func serverAction(cCtx *cli.Context) error {
	port := cCtx.String("port")
	bindIP := cCtx.String("bind")

	fmt.Printf("启动服务器模式，端口: %s\n", port)

	// 创建服务器上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := server.NewServer(port, bindIP)
	return server.Start(ctx)
}

func tcpTestAction(cCtx *cli.Context) error {
	serversStr := cCtx.String("servers")
	connections := cCtx.Int("connections")
	duration := cCtx.Int("duration")
	localIP := cCtx.String("localip")
	interfaceName := cCtx.String("interface")

	var servers []utils.ServerConfig
	if serversStr != "" {
		servers = parseServers(serversStr)
	} else {
		host := cCtx.String("host")
		port := cCtx.String("port")
		if host == "" {
			return fmt.Errorf("请指定服务端地址 (--host 或 --servers)")
		}
		if port == "" {
			port = "5432"
		}
		servers = []utils.ServerConfig{{Addr: fmt.Sprintf("%s:%s", host, port)}}
	}

	if len(servers) == 0 {
		return fmt.Errorf("服务端地址不能为空")
	}

	tester := client.NewTCPTester()

	if interfaceName == "all" {
		return tester.RunTCPTestOnAllInterfaces(servers, connections, duration)
	}

	config := utils.TestConfig{
		Servers:     servers,
		LocalIPs:    []string{localIP},
		Duration:    duration,
		Concurrency: connections,
		TestType:    utils.TestTypeTCP,
	}

	result, err := client.RunTest(config)
	if err != nil {
		return err
	}

	printInterfaceResults(result.Results)
	return nil
}

func parseServers(serversStr string) []utils.ServerConfig {
	var servers []utils.ServerConfig
	parts := strings.Split(serversStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var server utils.ServerConfig
		bandwidthParts := strings.Split(part, "/")
		addrPart := bandwidthParts[0]

		if len(bandwidthParts) > 1 {
			server.Bandwidth = bandwidthParts[1]
		}

		hostPort := strings.Split(addrPart, ":")
		if len(hostPort) == 2 {
			server.Addr = addrPart
		} else {
			server.Addr = addrPart + ":5432"
		}

		servers = append(servers, server)
	}
	return servers
}

func printInterfaceResults(results []utils.InterfaceResult) {
	for _, r := range results {
		if !r.Success {
			fmt.Printf("接口 %s: 测试失败 - %s\n", r.LocalIP, r.Error)
			continue
		}
		fmt.Printf("\n========== %s 测试结果 ==========\n", r.InterfaceName)
		fmt.Printf("本地 IP: %s\n", r.LocalIP)
		fmt.Printf("NAT 类型: %s\n", r.NATType)
		fmt.Printf("公网 IP: %s\n", r.PublicIP)
		fmt.Printf("吞吐量: %.2f MB/s (%.2f Mbps)\n", r.Throughput/1024/1024, r.ThroughputMbps)
		fmt.Printf("平均延迟: %.2f ms\n", r.AvgRTT)
		fmt.Printf("平均抖动: %.2f ms\n", r.AvgJitter)
		fmt.Printf("丢包率: %.2f%%\n", r.PacketLoss)
		fmt.Printf("总传输: %d bytes\n", r.TotalBytes)
		fmt.Printf("测试时长: %.2f 秒\n", r.Duration)
	}
}

func udpTestAction(cCtx *cli.Context) error {
	serversStr := cCtx.String("servers")
	connections := cCtx.Int("connections")
	duration := cCtx.Int("duration")
	bandwidth := cCtx.String("bandwidth")
	localIP := cCtx.String("localip")
	interfaceName := cCtx.String("interface")

	var servers []utils.ServerConfig
	if serversStr != "" {
		servers = parseServers(serversStr)
	} else {
		host := cCtx.String("host")
		port := cCtx.String("port")
		if host == "" {
			return fmt.Errorf("请指定服务端地址 (--host 或 --servers)")
		}
		if port == "" {
			port = "5432"
		}
		servers = []utils.ServerConfig{{Addr: fmt.Sprintf("%s:%s", host, port), Bandwidth: bandwidth}}
	}

	if len(servers) == 0 {
		return fmt.Errorf("服务端地址不能为空")
	}

	tester := client.NewUDPTester()

	if interfaceName == "all" {
		return tester.RunUDPTestOnAllInterfaces(servers, connections, duration)
	}

	config := utils.TestConfig{
		Servers:     servers,
		LocalIPs:    []string{localIP},
		Duration:    duration,
		Concurrency: connections,
		TestType:    utils.TestTypeUDP,
	}

	result, err := client.RunTest(config)
	if err != nil {
		return err
	}

	printInterfaceResults(result.Results)
	return nil
}

func interfaceInfoAction(cCtx *cli.Context) error {
	fmt.Println("正在检测所有网络接口...")
	onlineInterfaces, err := utils.GetOnlineNetworkInterfaces()
	if err != nil {
		log.Fatalf("获取在线网络接口失败: %v", err)
	}

	fmt.Println("\n已联网的网络接口信息:")
	utils.PrintNetworkInterfaceInfo(onlineInterfaces)

	return nil
}

func packetLossTestAction(cCtx *cli.Context) error {
	host := cCtx.String("host")
	port := cCtx.String("port")
	packets := cCtx.Int("packets")
	packetSize := cCtx.Int("size")
	localIP := cCtx.String("localip")

	serverAddr := fmt.Sprintf("%s:%s", host, port)

	if localIP != "" {
		fmt.Printf("使用本地 IP: %s\n", localIP)
	}

	fmt.Printf("\n========== UDP 丢包率测试 ==========\n")
	fmt.Printf("目标地址: %s\n", serverAddr)
	fmt.Printf("数据包数量: %d\n", packets)
	fmt.Printf("数据包大小: %d 字节\n", packetSize)
	fmt.Println("=====================================\n")

	result, err := utils.TestUDPPacketLoss(serverAddr, packets, packetSize, 3*time.Second)
	if err != nil {
		return fmt.Errorf("丢包率测试失败：%v", err)
	}

	fmt.Printf("发送数据包: %d\n", result.PacketsSent)
	fmt.Printf("接收数据包: %d\n", result.PacketsReceived)
	fmt.Printf("丢包率: %.2f%%\n", result.PacketLoss)
	fmt.Printf("平均延迟: %.2f ms\n", result.AvgLatency)
	fmt.Printf("最小延迟: %.2f ms\n", result.MinLatency)
	fmt.Printf("最大延迟: %.2f ms\n", result.MaxLatency)
	fmt.Printf("抖动: %.2f ms\n", result.Jitter)

	return nil
}
