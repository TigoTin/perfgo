package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"perfgo/cmd"
)

var (
	mode      = flag.String("mode", "client", "运行模式: server 或 client")
	host      = flag.String("host", "localhost", "服务器主机地址 (客户端模式)")
	port      = flag.String("port", "5432", "服务器端口")
	test      = flag.String("test", "bandwidth", "测试类型: bandwidth-upload, bandwidth-download, latency-ping, latency-jitter, packetloss, udp-bandwidth, udp-latency")
	threads   = flag.Int("threads", 1, "并发线程数 (仅用于带宽测试)")
	localIP   = flag.String("localip", "", "本地IP地址 (可选，用于指定源IP进行测试)")
	duration  = flag.Int("duration", 10, "测试持续时间 (秒)")
	bandwidth = flag.String("b", "", "目标带宽 (例如: 10M, 100K, 1G)，用于UDP带宽测试")
	help      = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	switch *mode {
	case "server":
		err := cmd.Server(*port, *localIP)
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "client":
		err := cmd.Client(*host, *port, *test, *threads, *localIP, *duration, *bandwidth)
		if err != nil {
			log.Fatalf("Client error: %v", err)
		}
	default:
		fmt.Printf("Invalid mode: %s. Use 'server' or 'client'.\n", *mode)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Perfgo - 网络质量测试工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  服务器模式: go run main.go -mode=server -port=5432 [-localip=0.0.0.0]")
	fmt.Println("  客户端模式: go run main.go -mode=client -host=localhost -port=5432 -test=bandwidth-upload [-localip=192.168.1.100]")
	fmt.Println("  参数说明:")
	fmt.Println("    -localip: 指定本地IP地址，用于多网卡环境下的源地址绑定")
	fmt.Println()
	fmt.Println("参数说明:")
	fmt.Println("  -mode     运行模式: server 或 client (默认: client)")
	fmt.Println("  -host     服务器主机地址 (客户端模式) (默认: localhost)")
	fmt.Println("  -port     服务器端口 (默认: 5432)")
	fmt.Println("  -test     测试类型:")
	fmt.Println("            bandwidth-upload      上传速度测试")
	fmt.Println("            bandwidth-download    下载速度测试")
	fmt.Println("            latency-ping          延迟测试")
	fmt.Println("            latency-jitter        抖动测试")
	fmt.Println("            packetloss            丢包率测试")
	fmt.Println("            udp-bandwidth         UDP带宽测试")
	fmt.Println("            udp-latency           UDP延迟测试")
	fmt.Println("  -localip  本地IP地址 (可选，用于指定源IP进行测试)")
	fmt.Println("  -duration 测试持续时间 (秒) (默认: 10)")
	fmt.Println("  -threads  并发线程数 (用于带宽和UDP测试) (默认: 1)")
	fmt.Println("  -b       目标带宽 (例如: 10M, 100K, 1G)，用于UDP带宽测试，类似iperf的-b参数")
	fmt.Println("  -help     显示此帮助信息")
}
