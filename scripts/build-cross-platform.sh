#!/bin/bash

# Perfgo 交叉编译脚本
# 支持构建多个平台的二进制文件

set -e

# 版本信息
VERSION=${1:-"v1.0.0"}
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "")

# 创建输出目录
mkdir -p dist

echo "开始构建 Perfgo ${VERSION} (构建时间: ${BUILD_TIME})"

# 定义要构建的平台
PLATFORMS=(
    "windows/amd64"
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# 遍历平台进行构建
for platform in "${PLATFORMS[@]}"; do
    GOOS=$(echo "$platform" | cut -d'/' -f1)
    GOARCH=$(echo "$platform" | cut -d'/' -f2)
    
    echo "正在构建 ${GOOS}/${GOARCH}..."
    
    OUTPUT_NAME="dist/perfgo-${VERSION}-${GOOS}-${GOARCH}"
    
    # 根据操作系统添加扩展名
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    # 设置环境变量并构建
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT:0:7}" \
        -o "$OUTPUT_NAME" .
        
    echo "  -> 已生成: $(basename "$OUTPUT_NAME")"
done

echo ""
echo "交叉编译完成！生成的文件位于 dist/ 目录下："
ls -la dist/