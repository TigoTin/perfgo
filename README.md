# Perfgo - 网络质量测试工具

Perfgo是一个基于Go语言开发的网络质量测试工具，支持带宽、延迟、抖动、丢包率等多种网络性能指标测试，所有测试结果均以中文显示。

## 功能特性

- 带宽测试 (上传/下载速度)
- 延迟测试 (ping测试)
- 抖动测试 (延迟变化)
- 丢包率测试
- UDP带宽和延迟测试
- 支持多线程并发测试
- 支持自定义测试时长
- 中文界面和结果输出

## 快速开始

### 服务器端

```bash
# 运行服务器模式
go run main.go -mode=server -port=5432
```

### 客户端

```bash
# 上传速度测试
go run main.go -mode=client -host=localhost -port=5432 -test=bandwidth-upload -threads=4

# 下载速度测试
go run main.go -mode=client -host=localhost -port=5432 -test=bandwidth-download -threads=4

# 延迟测试
go run main.go -mode=client -host=localhost -port=5432 -test=latency-ping

# 抖动测试
go run main.go -mode=client -host=localhost -port=5432 -test=latency-jitter

# 丢包率测试
go run main.go -mode=client -host=localhost -port=5432 -test=packetloss

# UDP带宽测试
go run main.go -mode=client -host=localhost -port=5432 -test=udp-bandwidth

# UDP带宽测试（指定目标带宽，类似iperf的-b参数）
go run main.go -mode=client -host=localhost -port=5432 -test=udp-bandwidth -b 10M

# UDP延迟测试
go run main.go -mode=client -host=localhost -port=5432 -test=udp-latency
```

## 参数说明

- `-mode` 运行模式: server 或 client (默认: client)
- `-host` 服务器主机地址 (客户端模式) (默认: localhost)
- `-port` 服务器端口 (默认: 5432)
- `-test` 测试类型:
  - `bandwidth-upload`: 上传速度测试
  - `bandwidth-download`: 下载速度测试
  - `latency-ping`: 延迟测试
  - `latency-jitter`: 抖动测试
  - `packetloss`: 丢包率测试
  - `udp-bandwidth`: UDP带宽测试
  - `udp-latency`: UDP延迟测试
- `-threads` 并发线程数 (用于带宽和UDP测试) (默认: 1)
- `-localip` 本地IP地址 (可选，用于指定源IP进行测试)
- `-duration` 测试持续时间 (秒) (默认: 10)
- `-b` 目标带宽 (例如: 10M, 100K, 1G)，用于UDP带宽测试，类似iperf的-b参数
- `-help` 显示帮助信息

## 构建

运行以下命令构建可执行文件：
```bash
go build -o perfgo .
```

## 依赖

- Go 1.16+