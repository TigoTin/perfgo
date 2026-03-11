package client

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/TigoTin/perfgo/pkg/utils"
)

type UDPTester struct {
	Timeout         time.Duration
	bandwidthResult *utils.TestResult
	latencyResult   *utils.TestResult
}

func NewUDPTester() *UDPTester {
	return &UDPTester{
		Timeout: DefaultTimeout,
	}
}

type UDPTestResult struct {
	Throughput     float64
	ThroughputMbps float64
	AvgRTT         float64
	AvgJitter      float64
	TotalBytes     int64
	Duration       float64
}

func (ut *UDPTester) RunUDPTest(serverAddr string, threads int, duration int, targetBandwidth string, localIP string, interfaceName string) error {
	if interfaceName == "all" {
		return ut.runTestOnAllInterfacesAggregated(serverAddr, threads, duration, targetBandwidth)
	}

	result, err := ut.RunUDPTestWithResult(serverAddr, threads, duration, targetBandwidth, localIP)
	if err != nil {
		return err
	}

	printUDPTestResult(result)
	return nil
}

func printUDPTestResult(result *UDPTestResult) {
	fmt.Printf("\n========== UDP 测试结果 ==========\n")
	fmt.Printf("吞吐量：%.2f MB/s (%.2f Mbps)\n", result.Throughput/1024/1024, result.ThroughputMbps)
	fmt.Printf("平均延迟：%.2f ms\n", result.AvgRTT)
	fmt.Printf("平均抖动：%.2f ms\n", result.AvgJitter)
	fmt.Printf("总传输字节：%d bytes\n", result.TotalBytes)
	fmt.Printf("测试时长：%.2f 秒\n", result.Duration)
}

func (ut *UDPTester) RunUDPTestWithResult(serverAddr string, threads int, duration int, targetBandwidth string, localIP string) (*UDPTestResult, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("解析 UDP 地址失败：%v", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("创建 UDP 连接失败：%v", err)
	}
	defer conn.Close()

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)
	conn.SetWriteDeadline(endTime)

	var totalBytes int64
	data := make([]byte, 32*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	for {
		n, err := conn.Write(data)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err != nil {
			break
		}
	}

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second
	}

	throughput := float64(totalBytes) / elapsed.Seconds()

	targetHost := extractHost(serverAddr)
	pingResult, _ := utils.PingTarget(targetHost, 10)

	result := &UDPTestResult{
		Throughput:     throughput,
		ThroughputMbps: throughput * 8 / 1000 / 1000,
		TotalBytes:     totalBytes,
		Duration:       elapsed.Seconds(),
	}

	if pingResult != nil && pingResult.Success {
		result.AvgRTT = pingResult.Latency
		result.AvgJitter = pingResult.Jitter
	}

	return result, nil
}

func (ut *UDPTester) runTestOnAllInterfacesAggregated(serverAddr string, threads int, duration int, targetBandwidth string) error {
	fmt.Println("正在获取所有在线网络接口...")
	interfaces, err := utils.GetOnlineNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("获取在线网络接口失败：%v", err)
	}

	if len(interfaces) == 0 {
		return fmt.Errorf("未找到任何在线网络接口")
	}

	fmt.Printf("发现 %d 个在线网络接口，开始逐个测试并聚合结果:\n\n", len(interfaces))

	var results []utils.InterfaceTestResult
	for i, iface := range interfaces {
		fmt.Printf("=== 测试第 %d/%d 个网络接口：%s (IP: %s, NAT 类型：%s) ===\n",
			i+1, len(interfaces), iface.Name, iface.IP, iface.NATType)

		tester := NewUDPTester()
		err := tester.RunUDPTest(serverAddr, threads, duration, targetBandwidth, iface.IP, "")
		if err != nil {
			fmt.Printf("接口 %s 测试失败：%v\n", iface.Name, err)
			results = append(results, createInterfaceResult(iface.Name, iface.NATType, utils.TestResult{}, err))
			fmt.Println()
			continue
		}

		result := buildUDPInterfaceResult(tester)
		fmt.Printf("接口 %s 测试完成\n", iface.Name)
		results = append(results, createInterfaceResult(iface.Name, iface.NATType, result, nil))
		fmt.Println()
	}

	utils.PrintStructuredInterfaceResult(results)
	return nil
}

func buildUDPInterfaceResult(tester *UDPTester) utils.TestResult {
	var result utils.TestResult

	if tester.bandwidthResult != nil && tester.latencyResult != nil {
		result = utils.TestResult{
			Protocol:   tester.bandwidthResult.Protocol,
			TestType:   "combined",
			Direction:  tester.bandwidthResult.Direction,
			Throughput: tester.bandwidthResult.Throughput,
			AvgRTT:     tester.latencyResult.AvgRTT,
			AvgJitter:  tester.latencyResult.AvgJitter,
			Duration:   tester.bandwidthResult.Duration,
		}
	} else if tester.bandwidthResult != nil {
		result = *tester.bandwidthResult
	} else if tester.latencyResult != nil {
		result = *tester.latencyResult
	}

	return result
}

func (ut *UDPTester) runUDPBandwidthTest(conn *net.UDPConn, threads int, duration int, targetBandwidth string) error {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	packetSize := 1024
	data := make([]byte, packetSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			threadBytes := int64(0)

			for {
				if time.Now().After(endTime) {
					break
				}

				_, err := conn.Write(data)
				if err != nil {
					break
				}

				threadBytes += int64(len(data))

				if targetBandwidth != "" {
					time.Sleep(time.Millisecond * 10)
				}
			}

			mu.Lock()
			totalBytes += threadBytes
			mu.Unlock()
		}()
	}

	time.Sleep(time.Duration(duration) * time.Second)
	wg.Wait()

	elapsed := time.Since(startTime)
	if elapsed.Seconds() == 0 {
		elapsed = time.Second
	}
	throughput := float64(totalBytes) / elapsed.Seconds()

	result := utils.TestResult{
		Protocol:   "UDP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}
	ut.bandwidthResult = &result

	return nil
}

func (ut *UDPTester) runUDPLatencyTest(serverAddr string) error {
	targetHost := serverAddr
	colonIndex := strings.LastIndex(serverAddr, ":")
	if colonIndex != -1 {
		targetHost = serverAddr[:colonIndex]
	}

	pingResult, err := utils.PingTarget(targetHost, 10)
	if err != nil {
		return fmt.Errorf("ping 测试失败：%v", err)
	}

	if !pingResult.Success {
		return fmt.Errorf("ping 测试无响应")
	}

	result := utils.TestResult{
		Protocol:    "PING",
		TestType:    "latency",
		Direction:   "round-trip",
		AvgRTT:      pingResult.Latency,
		AvgJitter:   pingResult.Jitter,
		SuccessRate: (100 - pingResult.PacketLoss) / 100,
		Duration:    pingResult.Latency * 10,
	}
	ut.latencyResult = &result

	return nil
}
