package utils

import (
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// PingResult ping测试结果结构
type PingResult struct {
	Target     string
	Latency    float64 // 延迟时间(毫秒)
	Success    bool
	Error      string
	PacketLoss float64
	Jitter     float64 // 抖动(毫秒)
	MinRTT     float64
	MaxRTT     float64
}

// PingTarget 对目标执行ping操作
func PingTarget(target string, count int) (*PingResult, error) {
	host := target
	// 如果输入的是域名，需要先解析IP地址
	if ip := net.ParseIP(target); ip == nil {
		// 是域名，需要解析
		ips, err := net.LookupIP(target)
		if err != nil {
			return nil, err
		}
		if len(ips) > 0 {
			// 优先使用IPv4
			for _, ip := range ips {
				if ip.To4() != nil {
					host = ip.String()
					break
				}
			}
			// 如果没找到IPv4，使用第一个IP
			if host == target {
				host = ips[0].String()
			}
		}
	}

	pinger := NewPinger(host)
	pinger.Count = count
	pinger.Timeout = 3 * time.Second

	err := pinger.Run()
	if err != nil {
		return nil, err
	}

	stats := pinger.Statistics()
	return &PingResult{
		Target:     target,
		Latency:    stats.AvgRtt.Seconds() * 1000,
		Success:    stats.PacketLoss < 100,
		PacketLoss: stats.PacketLoss,
		Jitter:     calculateJitter(pinger.rtts),
		MinRTT:     stats.MinRtt.Seconds() * 1000,
		MaxRTT:     stats.MaxRtt.Seconds() * 1000,
	}, nil
}

// calculateJitter 计算抖动
func calculateJitter(rtts []time.Duration) float64 {
	if len(rtts) < 2 {
		return 0
	}

	var totalDiff float64
	for i := 1; i < len(rtts); i++ {
		diff := rtts[i].Seconds() - rtts[i-1].Seconds()
		if diff < 0 {
			diff = -diff
		}
		totalDiff += diff
	}

	if len(rtts) <= 1 {
		return 0
	}

	return (totalDiff / float64(len(rtts)-1)) * 1000 // 转换为毫秒
}

// Pinger ping工具结构体
type Pinger struct {
	Target  string
	Count   int
	Timeout time.Duration
	rtts    []time.Duration
}

// NewPinger 创建新的ping实例
func NewPinger(target string) *Pinger {
	return &Pinger{
		Target:  target,
		Count:   4,
		Timeout: 3 * time.Second,
		rtts:    make([]time.Duration, 0),
	}
}

// Run 执行ping操作
func (p *Pinger) Run() error {
	host := p.Target
	if _, err := net.ResolveIPAddr("ip4", host); err != nil {
		return err
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	defer conn.Close()

	var seq int
	for i := 0; i < p.Count; i++ {
		seq = i
		if err := p.sendICMP(conn, seq); err != nil {
			continue
		}

		rtt, err := p.receiveICMP(conn, seq)
		if err != nil {
			continue
		}

		p.rtts = append(p.rtts, rtt)
		time.Sleep(100 * time.Millisecond) // 发包间隔
	}

	return nil
}

// sendICMP 发送ICMP包
func (p *Pinger) sendICMP(conn *icmp.PacketConn, seq int) error {
	body := &icmp.Echo{
		ID:   1234, // 固定ID
		Seq:  seq,
		Data: []byte("ping"),
	}

	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: body,
	}

	data, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	dst, err := net.ResolveIPAddr("ip4", p.Target)
	if err != nil {
		return err
	}

	_, err = conn.WriteTo(data, dst)
	return err
}

// receiveICMP 接收ICMP包
func (p *Pinger) receiveICMP(conn *icmp.PacketConn, seq int) (time.Duration, error) {
	start := time.Now()
	conn.SetReadDeadline(time.Now().Add(p.Timeout))

	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return 0, err
		}

		if addr.String() != p.Target {
			continue
		}

		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if err != nil {
			continue
		}

		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			echoReply := rm.Body.(*icmp.Echo)
			if echoReply.ID == 1234 && echoReply.Seq == seq {
				return time.Since(start), nil
			}
		}
	}
}

// Statistics 返回统计信息
func (p *Pinger) Statistics() *Statistics {
	if len(p.rtts) == 0 {
		return &Statistics{
			PacketLoss: 100,
		}
	}

	var total, min, max time.Duration
	min = p.rtts[0]
	max = p.rtts[0]

	for _, rtt := range p.rtts {
		total += rtt
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
	}

	avg := total / time.Duration(len(p.rtts))

	return &Statistics{
		PacketsSent: p.Count,
		PacketsRecv: len(p.rtts),
		PacketLoss:  float64(p.Count-len(p.rtts)) / float64(p.Count) * 100,
		MinRtt:      min,
		MaxRtt:      max,
		AvgRtt:      avg,
		Jitter:      calculateJitter(p.rtts),
	}
}

// Statistics ping统计信息
type Statistics struct {
	PacketsSent int
	PacketsRecv int
	PacketLoss  float64
	MinRtt      time.Duration
	MaxRtt      time.Duration
	AvgRtt      time.Duration
	Jitter      float64
}
