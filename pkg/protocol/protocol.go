package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// MessageType 定义协议消息类型
type MessageType int

const (
	TypeTestRequest  MessageType = iota // 测试请求
	TypeTestResponse                    // 测试响应
	TypeData                            // 数据传输
	TypeHeartbeat                       // 心跳
	TypeClose                           // 关闭连接
)

// Message 协议消息结构
type Message struct {
	Type      MessageType            `json:"type"`
	TestType  string                 `json:"test_type,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	ID        string                 `json:"id,omitempty"`
}

// Send 发送消息
func (m *Message) Send(conn net.Conn) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	// 添加换行符作为消息分隔符
	data = append(data, '\n')

	_, err = conn.Write(data)
	return err
}

// Receive 接收消息
func Receive(conn net.Conn) (*Message, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// 去除换行符
	line = strings.TrimSpace(line)

	var msg Message
	err = json.Unmarshal([]byte(line), &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v", err)
	}

	return &msg, nil
}

// UnmarshalMessage 从字节数组反序列化消息
func UnmarshalMessage(data []byte, msg *Message) error {
	return json.Unmarshal(data, msg)
}
