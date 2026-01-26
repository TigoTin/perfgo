# Perfgo

Perfgo 是一个现代化的网络性能测试工具，使用urfave/cli提供专业的命令行界面，支持TCP和UDP协议的带宽和延迟测试。

## 功能特性

- 使用urfave/cli提供专业的命令行界面
- 支持TCP和UDP协议测试
- 同时测试带宽和延迟指标
- 结构化输出测试结果
- 支持多线程并发测试
- 支持自定义测试时长
- 中文界面和结果输出
- 交叉编译支持
- Docker容器化部署

## 快速开始

### 服务器端

```bash
# 运行服务器模式
go run main.go server -p 5432
# 或者
go run main.go server --port 5432
```

### TCP客户端测试

```bash
# TCP带宽和延迟测试
go run main.go tcp --host=localhost --port=5432 --threads=4 --duration=10
```

### UDP客户端测试

```bash
# UDP带宽和延迟测试
go run main.go udp --host=localhost --port=5432 --threads=4 --duration=10 --bandwidth=10M
```

## 命令说明

### server 命令
- `--port, -p`: 服务器端口 (默认: 5432)
- `--bind`: 绑定IP地址 (可选)

### tcp 命令
- `--host`: 服务器主机地址 (必需)
- `--port, -p`: 服务器端口 (默认: 5432)
- `--threads`: 并发线程数 (默认: 1)
- `--duration`: 测试持续时间 (秒，默认: 10)
- `--localip`: 本地IP地址 (可选，用于指定源IP进行测试)

### udp 命令
- `--host`: 服务器主机地址 (必需)
- `--port, -p`: 服务器端口 (默认: 5432)
- `--threads`: 并发线程数 (默认: 1)
- `--duration`: 测试持续时间 (秒，默认: 10)
- `--bandwidth, -b`: 目标带宽 (例如: 10M, 100K, 1G)，用于UDP带宽测试
- `--localip`: 本地IP地址 (可选，用于指定源IP进行测试)

## 构建

运行以下命令构建可执行文件：

```bash
go build -o perfgo .
```

## 交叉编译

使用Makefile进行交叉编译：

```bash
# 构建所有平台版本
make build-all

# 构建特定平台版本
make build-windows-amd64    # Windows 64位
make build-linux-amd64      # Linux 64位
make build-linux-arm64      # Linux ARM64
make build-darwin-amd64     # macOS 64位
make build-darwin-arm64     # macOS ARM64
```

## 依赖

- Go 1.25.0+
- github.com/urfave/cli/v2

## 测试结果

所有测试结果都以结构化格式输出，包括：
- 协议类型 (TCP/UDP)
- 测试类型 (bandwidth/latency)
- 传输方向 (uplink/downlink/round-trip)
- 吞吐量 (带宽测试)
- 平均RTT (延迟测试)
- 平均抖动
- 成功率
- 传输总量
- 测试时长