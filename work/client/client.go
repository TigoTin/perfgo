package client

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/TigoTin/perfgo/pkg/protocol"
	"github.com/TigoTin/perfgo/pkg/utils"
)

const (
	DefaultTimeout    = 30 * time.Second
	DefaultBufferSize = 32 * 1024
	DefaultPingCount  = 10
	channelBufferSize = 1
)

type connResult struct {
	bandwidth *utils.TestResult
	latency   *utils.TestResult
	err       error
}

type TCPTester struct {
	Timeout time.Duration
}

func NewTCPTester() *TCPTester {
	return &TCPTester{
		Timeout: DefaultTimeout,
	}
}

func (ct *TCPTester) RunTCPTest(serverAddr string, connections int, duration int, localIP string, interfaceName string) error {
	if interfaceName == "all" {
		return ct.runTestOnAllInterfacesAggregated(serverAddr, connections, duration, ct.runTCPInterfaceTest)
	}

	result, err := ct.RunTCPTestWithResult(serverAddr, connections, duration, localIP)
	if err != nil {
		return err
	}

	printTCPTestResult(result, connections)
	return nil
}

func printTCPTestResult(result *TCPTestResult, connections int) {
	fmt.Printf("\n========== 聚合测试结果 (%d 个连接) ==========\n", connections)
	fmt.Printf("吞吐量：%.2f MB/s (%.2f Mbps)\n", result.Throughput/1024/1024, result.ThroughputMbps)
	fmt.Printf("平均延迟：%.2f ms\n", result.AvgRTT)
	fmt.Printf("平均抖动：%.2f ms\n", result.AvgJitter)
	fmt.Printf("丢包率：%.2f%%\n", result.PacketLoss)
	fmt.Printf("总传输字节：%d bytes\n", result.TotalBytes)
	fmt.Printf("测试时长：%.2f 秒\n", result.Duration)
}

func (ct *TCPTester) RunTCPTestWithResult(serverAddr string, connections int, duration int, localIP string) (*TCPTestResult, error) {
	resultChan := make(chan connResult, connections)
	var wg sync.WaitGroup

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			resultChan <- ct.runSingleConnection(serverAddr, localIP, duration)
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return ct.aggregateResults(resultChan)
}

func (ct *TCPTester) runSingleConnection(serverAddr, localIP string, duration int) connResult {
	conn, err := dialTCP(serverAddr, localIP)
	if err != nil {
		return connResult{err: fmt.Errorf("建立连接失败：%v", err)}
	}
	defer conn.Close()

	msg := &protocol.Message{
		Type:     protocol.TypeTestRequest,
		TestType: "tcp",
		Payload: map[string]interface{}{
			"test_type": "tcp",
			"threads":   1,
			"duration":  duration,
		},
	}

	if err := msg.Send(conn); err != nil {
		return connResult{err: fmt.Errorf("发送测试请求失败：%v", err)}
	}

	return ct.runParallelTests(conn, extractHost(serverAddr), duration)
}

func (ct *TCPTester) runParallelTests(conn net.Conn, targetHost string, duration int) connResult {
	bwResultChan := make(chan *utils.TestResult, channelBufferSize)
	latResultChan := make(chan *utils.TestResult, channelBufferSize)
	errChan := make(chan error, 2)

	go func() {
		bwResult, err := ct.runBandwidthTest(conn, duration)
		if err != nil {
			errChan <- err
		}
		bwResultChan <- bwResult
	}()

	go func() {
		latResult, err := ct.runLatencyTest(targetHost)
		if err != nil {
			errChan <- err
		}
		latResultChan <- latResult
	}()

	var bwResult, latResult *utils.TestResult
	for i := 0; i < 2; i++ {
		select {
		case res := <-bwResultChan:
			bwResult = res
		case res := <-latResultChan:
			latResult = res
		case err := <-errChan:
			return connResult{err: err}
		}
	}

	return connResult{bandwidth: bwResult, latency: latResult}
}

func (ct *TCPTester) aggregateResults(resultChan <-chan connResult) (*TCPTestResult, error) {
	var totalBytes int64
	var totalDuration float64
	var latencies, jitters, packetLosses []float64
	successCount := 0

	for result := range resultChan {
		if result.err != nil {
			continue
		}
		if result.bandwidth != nil {
			totalBytes += result.bandwidth.TotalBytes
			totalDuration += result.bandwidth.Duration
			successCount++
		}
		if result.latency != nil {
			latencies = append(latencies, result.latency.AvgRTT)
			jitters = append(jitters, result.latency.AvgJitter)
			packetLosses = append(packetLosses, result.latency.PacketLoss)
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("所有连接均失败")
	}

	var avgPacketLoss float64
	if len(packetLosses) > 0 {
		avgPacketLoss = sumFloat64(packetLosses) / float64(len(packetLosses))
	}

	return buildTCPTestResult(totalBytes, totalDuration, float64(successCount), latencies, jitters, avgPacketLoss), nil
}

func buildTCPTestResult(totalBytes int64, totalDuration, successCount float64, latencies, jitters []float64, packetLoss float64) *TCPTestResult {
	avgDuration := totalDuration / successCount
	throughput := float64(totalBytes) / avgDuration

	var avgRTT, avgJitter float64
	if len(latencies) > 0 {
		avgRTT = sumFloat64(latencies) / float64(len(latencies))
		avgJitter = sumFloat64(jitters) / float64(len(jitters))
	}

	return &TCPTestResult{
		Throughput:     throughput,
		ThroughputMbps: throughput * 8 / 1000 / 1000,
		AvgRTT:         avgRTT,
		AvgJitter:      avgJitter,
		PacketLoss:     packetLoss,
		TotalBytes:     totalBytes,
		Duration:       avgDuration,
	}
}

func sumFloat64(slice []float64) float64 {
	var sum float64
	for _, v := range slice {
		sum += v
	}
	return sum
}

type TCPTestResult struct {
	Throughput     float64
	ThroughputMbps float64
	AvgRTT         float64
	AvgJitter      float64
	PacketLoss     float64
	TotalBytes     int64
	Duration       float64
}

func (ct *TCPTester) runBandwidthTest(conn net.Conn, duration int) (*utils.TestResult, error) {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)
	conn.SetWriteDeadline(endTime)

	var totalBytes int64
	data := make([]byte, DefaultBufferSize)
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

	return &utils.TestResult{
		Protocol:   "TCP",
		TestType:   "bandwidth",
		Direction:  "uplink",
		Throughput: float64(totalBytes) / elapsed.Seconds(),
		TotalBytes: totalBytes,
		Duration:   elapsed.Seconds(),
	}, nil
}

func (ct *TCPTester) runLatencyTest(target string) (*utils.TestResult, error) {
	pingResult, err := utils.PingTarget(target, DefaultPingCount)
	if err != nil {
		return nil, fmt.Errorf("ping 测试失败：%v", err)
	}

	if !pingResult.Success {
		return nil, fmt.Errorf("ping 测试无响应")
	}

	return &utils.TestResult{
		Protocol:    "PING",
		TestType:    "latency",
		Direction:   "round-trip",
		AvgRTT:      pingResult.Latency,
		AvgJitter:   pingResult.Jitter,
		SuccessRate: (100 - pingResult.PacketLoss) / 100,
		Duration:    pingResult.Latency * DefaultPingCount,
		PacketLoss:  pingResult.PacketLoss,
	}, nil
}

func (ct *TCPTester) runTestOnAllInterfacesAggregated(serverAddr string, connections int, duration int,
	testFunc func(string, string, int, int) ([]connResult, error)) error {

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

		connResults, err := testFunc(iface.IP, serverAddr, connections, duration)
		if err != nil {
			fmt.Printf("接口 %s 测试失败：%v\n", iface.Name, err)
			results = append(results, createInterfaceResult(iface.Name, iface.NATType, utils.TestResult{}, err))
			continue
		}

		result := aggregateConnectionResults(connResults)
		fmt.Printf("接口 %s 测试完成\n", iface.Name)
		results = append(results, createInterfaceResult(iface.Name, iface.NATType, result, nil))
		fmt.Println()
	}

	utils.PrintStructuredInterfaceResult(results)
	return nil
}

func (ct *TCPTester) RunTCPTestOnAllInterfaces(servers []utils.ServerConfig, connections int, duration int) error {
	fmt.Println("正在获取所有在线网络接口...")
	interfaces, err := utils.GetOnlineNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("获取在线网络接口失败：%v", err)
	}

	if len(interfaces) == 0 {
		return fmt.Errorf("未找到任何在线网络接口")
	}

	fmt.Printf("发现 %d 个在线网络接口，%d 个服务端，开始测试:\n\n", len(interfaces), len(servers))

	for i, iface := range interfaces {
		fmt.Printf("=== 测试第 %d/%d 个网络接口：%s (IP: %s, NAT 类型：%s) ===\n",
			i+1, len(interfaces), iface.Name, iface.IP, iface.NATType)

		config := utils.TestConfig{
			Servers:     servers,
			LocalIPs:    []string{iface.IP},
			Duration:    duration,
			Concurrency: connections,
			TestType:    utils.TestTypeTCP,
		}

		result, err := RunTest(config)
		if err != nil {
			fmt.Printf("接口 %s 测试失败：%v\n", iface.Name, err)
			continue
		}

		if len(result.Results) > 0 {
			r := result.Results[0]
			fmt.Printf("吞吐量: %.2f Mbps, 延迟: %.2f ms, 抖动: %.2f ms, 丢包率: %.2f%%\n",
				r.ThroughputMbps, r.AvgRTT, r.AvgJitter, r.PacketLoss)
		}
		fmt.Println()
	}

	return nil
}

func (ct *TCPTester) runTCPInterfaceTest(localIP, serverAddr string, connections, duration int) ([]connResult, error) {
	resultChan := make(chan connResult, connections)
	var wg sync.WaitGroup

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			resultChan <- ct.runSingleConnection(serverAddr, localIP, duration)
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []connResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results, nil
}

func aggregateConnectionResults(connResults []connResult) utils.TestResult {
	var totalBytes int64
	var totalDuration float64
	var latencies, jitters []float64
	successCount := 0

	for _, res := range connResults {
		if res.err != nil {
			continue
		}
		if res.bandwidth != nil {
			totalBytes += res.bandwidth.TotalBytes
			totalDuration += res.bandwidth.Duration
			successCount++
		}
		if res.latency != nil {
			latencies = append(latencies, res.latency.AvgRTT)
			jitters = append(jitters, res.latency.AvgJitter)
		}
	}

	if successCount == 0 {
		return utils.TestResult{}
	}

	avgDuration := totalDuration / float64(successCount)
	throughput := float64(totalBytes) / avgDuration

	var avgRTT, avgJitter float64
	if len(latencies) > 0 {
		avgRTT = sumFloat64(latencies) / float64(len(latencies))
		avgJitter = sumFloat64(jitters) / float64(len(jitters))
	}

	return utils.TestResult{
		Protocol:   "TCP",
		TestType:   "combined",
		Direction:  "uplink",
		Throughput: throughput,
		TotalBytes: totalBytes,
		AvgRTT:     avgRTT,
		AvgJitter:  avgJitter,
		Duration:   avgDuration,
	}
}

func createInterfaceResult(name, natType string, result utils.TestResult, err error) utils.InterfaceTestResult {
	return utils.InterfaceTestResult{
		TestResult:    result,
		InterfaceName: name,
		NATType:       natType,
		Error:         err,
	}
}

func dialTCP(serverAddr, localIP string) (net.Conn, error) {
	if localIP != "" {
		localAddr, err := net.ResolveTCPAddr("tcp", localIP+":0")
		if err != nil {
			return nil, fmt.Errorf("解析本地 TCP 地址失败：%v", err)
		}
		remoteAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
		if err != nil {
			return nil, fmt.Errorf("解析远程 TCP 地址失败：%v", err)
		}
		dialer := net.Dialer{LocalAddr: localAddr}
		return dialer.Dial("tcp", remoteAddr.String())
	}
	return net.Dial("tcp", serverAddr)
}

func extractHost(serverAddr string) string {
	if idx := strings.LastIndex(serverAddr, ":"); idx != -1 {
		return serverAddr[:idx]
	}
	return serverAddr
}
