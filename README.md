# Perfgo

Perfgo 是一个现代化的网络性能测试工具，支持 TCP 和 UDP 协议的带宽和延迟测试。

## 功能特性

- 支持 TCP/UDP 协议测试
- 同时测试带宽和延迟指标
- 多并发连接支持
- 多网卡测试支持（自动识别网卡信息）
- NAT 类型检测
- 结构化输出测试结果（JSON 格式）
- 交叉编译支持（Windows/Linux/macOS）
- Docker 容器化部署
- 支持作为库集成到其他项目

## 快速开始

### 服务器端

```bash
# 运行服务器模式
go run main.go server -p 5432
```

### TCP 客户端测试

```bash
go run main.go tcp --host=localhost --port=5432 --threads=4 --duration=10
```

### UDP 客户端测试

```bash
go run main.go udp --host=localhost --port=5432 --threads=4 --duration=10 --bandwidth=10M
```

### 查看网络接口信息

```bash
go run main.go interface
```

## 命令说明

### server 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| --port | -p | 5432 | 服务器端口 |
| --bind | | | 绑定 IP 地址（可选）|

### tcp 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| --host | | | 服务器主机地址（必需）|
| --port | -p | 5432 | 服务器端口 |
| --connections | -c | 1 | 并发连接数 |
| --duration | | 10 | 测试持续时间（秒）|
| --localip | | | 本地 IP 地址（可选）|
| --interface | -i | all | 网络接口名称（可选）|

### udp 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| --host | | | 服务器主机地址（必需）|
| --port | -p | 5432 | 服务器端口 |
| --connections | -c | 1 | 并发连接数 |
| --duration | | 10 | 测试持续时间（秒）|
| --bandwidth | -b | | 目标带宽（如 10M）|
| --localip | | | 本地 IP 地址（可选）|
| --interface | -i | | 网络接口名称（可选）|

### interface 命令

查看本地网络接口信息（网卡名称、IP、NAT 类型）。

## 作为库使用

Perfgo 可以作为库集成到其他 Go 项目中。

### 安装

```go
import "perfgo/internal/client"
```

### 示例代码

```go
package main

import (
    "encoding/json"
    "fmt"
    "perfgo/internal/client"
)

func main() {
    config := client.TestConfig{
        ServerAddr:  "192.168.1.100:5432",
        LocalIPs:    []string{
            "192.168.1.10",
            "192.168.2.10",
        },
        Duration:    10,
        Concurrency: 4,
        TestType:    client.TestTypeTCP,
    }

    result, err := client.RunTest(config)
    if err != nil {
        fmt.Printf("测试失败: %v\n", err)
        return
    }

    // 输出 JSON 格式结果
    jsonData, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(jsonData))
}
```

### 数据结构

**TestConfig** - 测试配置

| 字段 | 类型 | 说明 |
|------|------|------|
| ServerAddr | string | 服务端地址 (IP:端口) |
| LocalIPs | []string | 本地网卡 IP 列表 |
| Duration | int | 测试持续时间（秒）|
| Concurrency | int | 并发连接数 |
| TestType | TestType | tcp 或 udp |
| Bandwidth | string | UDP 目标带宽（如 "10M"）|

**TestResult** - 测试结果

| 字段 | 类型 | 说明 |
|------|------|------|
| Results | []InterfaceResult | 每个网卡的测试结果 |

**InterfaceResult** - 单网卡测试结果

| 字段 | 类型 | 说明 |
|------|------|------|
| LocalIP | string | 本地网卡 IP |
| InterfaceName | string | 网卡名称 |
| NATType | string | NAT 类型 |
| PublicIP | string | 公网 IP |
| Success | bool | 测试是否成功 |
| Error | string | 错误信息 |
| Throughput | float64 | 吞吐量 (bytes/s) |
| ThroughputMbps | float64 | 吞吐量 (Mbps) |
| AvgRTT | float64 | 平均延迟 (ms) |
| AvgJitter | float64 | 平均抖动 (ms) |
| TotalBytes | int64 | 总传输字节数 |
| Duration | float64 | 测试持续时间 (秒) |

## 构建

```bash
# 构建当前平台
go build -o perfgo .

# 使用 Makefile
make build
```

## 交叉编译

```bash
# Windows 64位
GOOS=windows GOARCH=amd64 go build -o perfgo.exe .

# Linux 64位
GOOS=linux GOARCH=amd64 go build -o perfgo .

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o perfgo-linux-arm64 .

# macOS
GOOS=darwin GOARCH=amd64 go build -o perfgo-darwin-amd64 .
```

或使用 Makefile：

```bash
make build-linux-amd64
make build-windows-amd64
make build-darwin-amd64
```

## Docker 部署

```bash
# 构建镜像
docker build -t perfgo .

# 运行服务器
docker run -d -p 5432:5432 perfgo server
```

或使用 docker-compose：

```bash
docker-compose up -d
```

## 依赖

- Go 1.21+
- github.com/urfave/cli/v2
- github.com/cheynewallace/tabby
- github.com/ccding/go-stun
- github.com/vishvananda/netlink

## 测试结果示例

```json
{
  "results": [
    {
      "local_ip": "192.168.1.10",
      "interface_name": "eth0",
      "nat_type": "Full Cone",
      "public_ip": "1.2.3.4",
      "success": true,
      "throughput": 12500000,
      "throughput_mbps": 100.00,
      "avg_rtt_ms": 1.25,
      "avg_jitter_ms": 0.05,
      "total_bytes": 125000000,
      "duration_sec": 10
    }
  ]
}
```

## License

MIT
