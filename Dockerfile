# Perfgo Dockerfile
# Multi-stage build for perfgo

# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装git（用于获取版本信息）
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制go模文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o perfgo .

# 运行阶段
FROM alpine:latest

# 安装ca-certificates
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从builder阶段复制二进制文件
COPY --from=builder /app/perfgo .

# 暴露端口（如果作为服务器运行）
EXPOSE 5432

# 启动命令
CMD ["./perfgo"]