# Perfgo - Go语言性能测试工具

Perfgo是一个基于Go语言开发的性能测试和监控工具。

## 功能特性

- HTTP服务器性能测试
- 基准测试功能
- 性能指标收集与展示

## 快速开始

1. 克隆项目
2. 运行 `go run main.go`
3. 访问 `http://localhost:8080` 查看服务状态
4. 访问 `http://localhost:8080/api/performance` 执行性能测试

## 构建

运行以下命令构建可执行文件：
```bash
go build -o perfgo .
```

## 依赖

- Go 1.16+