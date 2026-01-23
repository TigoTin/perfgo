package bandwidth

import (
	"fmt"
	"net"
	"time"

	"perfgo/pkg/protocol"
)

// ClientHandler 客户端处理带宽测试请求（支持自定义持续时间）
func ClientHandler(conn net.Conn, testType string, threads int, duration int) error {
	msg := &protocol.Message{
		Type:     protocol.TypeTestRequest,
		TestType: "bandwidth",
		Payload: map[string]interface{}{
			"test_type": testType,
			"threads":   threads,
			"duration":  duration,
		},
	}

	err := msg.Send(conn)
	if err != nil {
		return fmt.Errorf("failed to send bandwidth test request: %v", err)
	}

	bwTester := NewBandwidthTester()
	bwTester.Duration = time.Duration(duration) * time.Second

	switch testType {
	case "upload":
		return bwTester.UploadTest(conn, threads)
	case "download":
		return bwTester.DownloadTest(conn, threads)
	default:
		return fmt.Errorf("unknown test type: %s", testType)
	}
}
