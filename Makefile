# Makefile for Perfgo

# 默认目标
.PHONY: all
all: build

# 构建应用
.PHONY: build
build:
	go build -o bin/perfgo .

# 运行应用
.PHONY: run
run:
	go run main.go

# 运行测试
.PHONY: test
test:
	go test -v ./...

# 运行基准测试
.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

# 格式化代码
.PHONY: fmt
fmt:
	go fmt ./...

# 更新依赖
.PHONY: tidy
tidy:
	go mod tidy

# 清理构建产物
.PHONY: clean
clean:
	rm -rf bin/

# 安装应用
.PHONY: install
install:
	go install .

# 运行竞态检测
.PHONY: race
race:
	go run -race main.go

# 交叉编译 - Windows 64位
.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -o bin/perfgo-windows-amd64.exe .

# 交叉编译 - Linux 64位
.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/perfgo-linux-amd64 .

# 交叉编译 - Linux ARM64
.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/perfgo-linux-arm64 .

# 交叉编译 - macOS 64位
.PHONY: build-darwin-amd64
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o bin/perfgo-darwin-amd64 .

# 交叉编译 - macOS ARM64
.PHONY: build-darwin-arm64
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/perfgo-darwin-arm64 .

# 交叉编译 - 所有平台
.PHONY: build-all
build-all: build-windows-amd64 build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

# 简化的构建命令，带版本信息
.PHONY: build-release
build-release:
	@mkdir -p bin
	go build -ldflags="-X main.Version=`git describe --tags --abbrev=0 2>/dev/null || echo 'v1.0.0'` -X main.BuildTime=`date -u +%Y-%m-%dT%H:%M:%SZ`" -o bin/perfgo .