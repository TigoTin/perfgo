package utils

// TestType 测试类型
type TestType string

const (
	TestTypeTCP TestType = "tcp"
	TestTypeUDP TestType = "udp"
)

// TestResult 测试结果结构
type TestResult struct {
	Protocol    string  `json:"protocol"`     // 协议类型 (TCP/UDP)
	TestType    string  `json:"test_type"`    // 测试类型 (bandwidth/latency)
	Direction   string  `json:"direction"`    // 方向 (uplink/downlink)
	Throughput  float64 `json:"throughput"`   // 吞吐量 (bytes/s)
	AvgRTT      float64 `json:"avg_rtt"`      // 平均往返时间 (ms)
	AvgJitter   float64 `json:"avg_jitter"`   // 平均抖动 (ms)
	SuccessRate float64 `json:"success_rate"` // 成功率
	PacketLoss  float64 `json:"packet_loss"`  // 丢包率 (%)
	TotalBytes  int64   `json:"total_bytes"`  // 总字节数
	Duration    float64 `json:"duration"`     // 持续时间 (秒)
}

// InterfaceTestResult 带有接口信息的测试结果结构
type InterfaceTestResult struct {
	TestResult           // 嵌入基本测试结果
	InterfaceName string // 网络接口名称
	NATType       string // NAT类型
	Error         error  // 错误信息
}

// TestConfig 测试配置结构
type TestConfig struct {
	ServerAddr  string   `json:"server_addr"` // 服务端地址 (IP:端口)
	LocalIPs    []string `json:"local_ips"`    // 本地网卡IP列表 (支持多个)
	Duration    int      `json:"duration"`     // 测试持续时间 (秒)
	Concurrency int      `json:"concurrency"`  // 并发连接数
	TestType    TestType `json:"test_type"`    // 测试类型: tcp 或 udp
	Bandwidth   string   `json:"bandwidth"`    // 仅UDP有效，目标带宽如 "10M"
}

// InterfaceResult 单个接口的测试结果
type InterfaceResult struct {
	LocalIP        string  `json:"local_ip"`        // 本地网卡IP
	InterfaceName  string  `json:"interface_name"`  // 网卡名称
	NATType        string  `json:"nat_type"`        // NAT类型
	PublicIP       string  `json:"public_ip"`       // 公网IP
	Success        bool    `json:"success"`         // 测试是否成功
	Error          string  `json:"error,omitempty"` // 错误信息
	Throughput     float64 `json:"throughput"`      // 吞吐量 (bytes/s)
	ThroughputMbps float64 `json:"throughput_mbps"` // 吞吐量 (Mbps)
	AvgRTT         float64 `json:"avg_rtt_ms"`      // 平均延迟 (ms)
	AvgJitter      float64 `json:"avg_jitter_ms"`   // 平均抖动 (ms)
	PacketLoss     float64 `json:"packet_loss"`     // 丢包率 (%)
	TotalBytes     int64   `json:"total_bytes"`     // 总传输字节数
	Duration       float64 `json:"duration_sec"`    // 测试持续时间 (秒)
}

// TestResultSummary 测试结果汇总
type TestResultSummary struct {
	Results []InterfaceResult `json:"results"` // 每个网卡的测试结果
}
