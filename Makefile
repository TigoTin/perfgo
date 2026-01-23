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