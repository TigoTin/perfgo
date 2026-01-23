package utils

import (
	"fmt"
	"strconv"
	"time"
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

// FormatSpeedDetailed 将速度格式化为人类可读的详细格式，同时显示MB/s和Mbps
func FormatSpeedDetailed(bytesPerSec float64) string {
	basicFormat := FormatSpeed(bytesPerSec)
	mbpsFormat := FormatSpeedMbps(bytesPerSec)
	return fmt.Sprintf("%s (%s)", basicFormat, mbpsFormat)
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
