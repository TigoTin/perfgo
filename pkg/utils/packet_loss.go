package utils

import (
	"fmt"
	"time"
)

type PacketLossResult struct {
	PacketsSent     int     // 发送数据包数
	PacketsReceived int     // 接收数据包数
	PacketLoss      float64 // 丢包率 (%)
	AvgLatency      float64 // 平均延迟 (ms)
	MinLatency      float64 // 最小延迟 (ms)
	MaxLatency      float64 // 最大延迟 (ms)
	Jitter          float64 // 抖动 (ms)
}

func TestUDPPacketLoss(targetAddr string, packets int, packetSize int, timeout time.Duration) (*PacketLossResult, error) {
	host := extractHost(targetAddr)

	pingResult, err := PingTarget(host, packets)
	if err != nil {
		return nil, fmt.Errorf("丢包率测试失败：%v", err)
	}

	var avgLatency, minLatency, maxLatency, jitter float64
	if pingResult.Success {
		avgLatency = pingResult.Latency
		minLatency = pingResult.MinRTT
		maxLatency = pingResult.MaxRTT
		jitter = pingResult.Jitter
	}

	return &PacketLossResult{
		PacketsSent:     packets,
		PacketsReceived: int(float64(packets) * (100 - pingResult.PacketLoss) / 100),
		PacketLoss:      pingResult.PacketLoss,
		AvgLatency:      avgLatency,
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
		Jitter:          jitter,
	}, nil
}

func extractHost(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
