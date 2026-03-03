package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	host := cCtx.String("host")
	port := cCtx.String("port")
	connections := cCtx.Int("connections")
	duration := cCtx.Int("duration")
	localIP := cCtx.String("localip")
	interfaceName := cCtx.String("interface")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tester := client.NewTCPTester()

	return tester.RunTCPTest(serverAddr, connections, duration, localIP, interfaceName)
}

func udpTestAction(cCtx *cli.Context) error {
	host := cCtx.String("host")
	port := cCtx.String("port")
	connections := cCtx.Int("connections")
	duration := cCtx.Int("duration")
	bandwidth := cCtx.String("bandwidth")
	localIP := cCtx.String("localip")
	interfaceName := cCtx.String("interface")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tester := client.NewUDPTester()

	return tester.RunUDPTest(serverAddr, connections, duration, bandwidth, localIP, interfaceName)
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
