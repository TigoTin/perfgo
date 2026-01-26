package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"perfgo/internal/client"
	"perfgo/internal/server"
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
						Name:  "threads",
						Value: 1,
						Usage: "并发线程数",
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
						Name:  "threads",
						Value: 1,
						Usage: "并发线程数",
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
	threads := cCtx.Int("threads")
	duration := cCtx.Int("duration")
	localIP := cCtx.String("localip")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tester := client.NewTCPTester()

	return tester.RunTCPTest(serverAddr, threads, duration, localIP)
}

func udpTestAction(cCtx *cli.Context) error {
	host := cCtx.String("host")
	port := cCtx.String("port")
	threads := cCtx.Int("threads")
	duration := cCtx.Int("duration")
	bandwidth := cCtx.String("bandwidth")
	localIP := cCtx.String("localip")

	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tester := client.NewUDPTester()

	return tester.RunUDPTest(serverAddr, threads, duration, bandwidth, localIP)
}
