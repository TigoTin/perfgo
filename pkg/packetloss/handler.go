package packetloss

import (
	"fmt"
	"net"

	"perfgo/pkg/protocol"
)

// ClientHandler 客户端处理丢包率测试请求
func ClientHandler(conn net.Conn) error {
	msg := &protocol.Message{
		Type:     protocol.TypeTestRequest,
		TestType: "packetloss",
		Payload: map[string]interface{}{
			"test_type": "test",
		},
	}

	err := msg.Send(conn)
	if err != nil {
		return fmt.Errorf("failed to send packet loss test request: %v", err)
	}

	packetLossTester := NewPacketLossTester()
	return packetLossTester.Test(conn)
}
