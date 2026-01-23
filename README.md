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

### 简单构建
运行以下命令构建可执行文件：
```bash
go build -o perfgo .
```

### 使用Makefile构建
Perfgo提供了丰富的Makefile命令来简化构建过程：

```bash
# 构建默认平台版本
make build

# 构建特定平台版本
make build-windows-amd64    # Windows 64位
make build-linux-amd64      # Linux 64位
make build-linux-arm64      # Linux ARM64
make build-darwin-amd64     # macOS 64位
make build-darwin-arm64     # macOS ARM64

# 构建所有平台版本
make build-all

# 构建带版本信息的发布版
make build-release
```

### 交叉编译
Perfgo支持跨平台编译，可以为不同操作系统和架构构建二进制文件：

**Linux/macOS:**
```bash
# 使用构建脚本
chmod +x scripts/build-cross-platform.sh
./scripts/build-cross-platform.sh v1.1.0
```

**Windows:**
```powershell
# 使用PowerShell脚本
powershell -ExecutionPolicy Bypass -File scripts\build-cross-platform.ps1 -Version v1.1.0
```

构建的二进制文件将放在 `dist/` 目录中。

### Docker部署
Perfgo还支持Docker容器化部署：

```bash
# 构建Docker镜像
docker build -t perfgo .

# 以服务器模式运行
docker run -d --name perfgo-server -p 5432:5432 perfgo -mode=server

# 以客户端模式运行
docker run --rm perfgo -mode=client -host=<server-ip> -test=bandwidth-upload
```

使用docker-compose快速部署：
```bash
# 启动服务器和客户端
docker-compose up -d

# 查看日志
docker-compose logs -f

# 只启动服务器
docker-compose up -d perfgo-server
```

## 依赖

- Go 1.16+