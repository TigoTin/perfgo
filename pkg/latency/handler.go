package latency

import (
	"fmt"
	"net"

	"perfgo/pkg/protocol"
)

// ClientHandler 客户端处理延迟测试请求
func ClientHandler(conn net.Conn, testType string) error {
	msg := &protocol.Message{
		Type:     protocol.TypeTestRequest,
		TestType: "latency",
		Payload: map[string]interface{}{
			"test_type": testType,
		},
	}

	err := msg.Send(conn)
	if err != nil {
		return fmt.Errorf("failed to send latency test request: %v", err)
	}

	latencyTester := NewLatencyTester()

	switch testType {
	case "ping":
		return latencyTester.PingTest(conn)
	case "jitter":
		return latencyTester.JitterTest(conn)
	default:
		return fmt.Errorf("unknown test type: %s", testType)
	}
}
