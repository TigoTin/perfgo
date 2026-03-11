package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cheynewallace/tabby"
)

// FormatBytes 将字节数格式化为人类可读的格式
func FormatBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "kMGTPE"[exp])
}

// FormatSpeed 将速度格式化为人类可读的格式 (B/s, KB/s, MB/s等)
func FormatSpeed(bytesPerSec float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	unitIndex := 0
	speed := bytesPerSec

	for speed >= 1000 && unitIndex < len(units)-1 {
		speed /= 1000
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", speed, units[unitIndex])
}

// FormatSpeedMbps 将速度格式化为Mbps（兆比特每秒）格式
func FormatSpeedMbps(bytesPerSec float64) string {
	// 将字节每秒转换为比特每秒，然后转换为Mbps
	mbps := bytesPerSec * 8 / 1000 / 1000
	return fmt.Sprintf("%.2f Mbps", mbps)
}

// FormatSpeedDetailed 将速度格式化为人类可读的详细格式，同时显示 MB/s 和 Mbps
func FormatSpeedDetailed(bytesPerSec float64) string {
	basicFormat := FormatSpeed(bytesPerSec)
	mbpsFormat := FormatSpeedMbps(bytesPerSec)
	return fmt.Sprintf("%s (%s)", basicFormat, mbpsFormat)
}

// PrintStructuredResult 打印结构化测试结果
func PrintStructuredResult(result TestResult) {
	table := tabby.New()

	// 如果同时有吞吐量和延迟数据，一起显示
	if result.Throughput > 0 && result.AvgRTT > 0 {
		throughputStr := FormatSpeedMbps(result.Throughput)
		table.AddHeader("网速", "延迟")
		table.AddLine(throughputStr, fmt.Sprintf("%.2f ms", result.AvgRTT))
	} else if result.Throughput > 0 {
		// 只有网速
		throughputStr := FormatSpeedMbps(result.Throughput)
		table.AddHeader("网速", "延迟")
		table.AddLine(throughputStr, "0")
	} else if result.AvgRTT > 0 {
		// 只有延迟
		table.AddHeader("网速", "延迟")
		table.AddLine("0", fmt.Sprintf("%.2f ms", result.AvgRTT))
	}
	table.Print()
}

// PrintStructuredResult 打印结构化测试结果
func PrintStructuredInterfaceResult(result []InterfaceTestResult) {
	table := tabby.New()
	table.AddHeader("接口名称", "NAT类型", "网速", "延迟")
	for _, result := range result {
		throughputStr := FormatSpeedMbps(result.Throughput)
		table.AddLine(result.InterfaceName, result.NATType, throughputStr, fmt.Sprintf("%.2f ms", result.AvgRTT))
	}
	table.Print()
}

// ParseDuration 解析持续时间字符串
func ParseDuration(durationStr string) (time.Duration, error) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// 如果没有单位，则默认为秒
		seconds, err := strconv.Atoi(durationStr)
		if err != nil {
			return 0, err
		}
		return time.Duration(seconds) * time.Second, nil
	}
	return duration, nil
}
