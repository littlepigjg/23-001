#!/bin/bash
# build_benzhi_docker.sh - 设备固件升级管理服务 Docker 构建脚本
# 用法: ./build_benzhi_docker.sh [镜像名] [标签] [平台]

set -e

# 默认参数
IMAGE_NAME="${1:-fwupgrade}"
TAG="${2:-latest}"
PLATFORM="${3:-linux/amd64}"
DOCKERFILE="benzhi.Dockerfile"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 打印彩色信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否可用
check_docker() {
    if ! command -v docker &> /dev/null; then
        error "Docker 未安装或不在 PATH 中"
        echo "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! docker info &> /dev/null 2>&1; then
        error "Docker 守护进程未运行"
        echo "请启动 Docker: sudo systemctl start docker"
        exit 1
    fi

    info "Docker 检查通过"
    docker version --format '{{.Client.Version}}' 2>/dev/null || echo "unknown"
}

# 检查 Dockerfile 是否存在
check_dockerfile() {
    if [ ! -f "$DOCKERFILE" ]; then
        error "Dockerfile 文件不存在: $DOCKERFILE"
        exit 1
    fi
    info "Dockerfile 检查通过: $DOCKERFILE"
}

# 构建镜像
build_image() {
    local image_tag="${IMAGE_NAME}:${TAG}"

    info "开始构建镜像..."
    info "  镜像名: $IMAGE_NAME"
    info "  标签: $TAG"
    info "  平台: $PLATFORM"
    info "  Dockerfile: $DOCKERFILE"
    info "  完整标签: $image_tag"

    # 设置平台
    export DOCKER_BUILDKIT=1
    export BUILDKIT_PLATFORM="$PLATFORM"

    docker build \
        -f "$DOCKERFILE" \
        -t "$image_tag" \
        --platform "$PLATFORM" \
        --no-cache \
        .

    if [ $? -eq 0 ]; then
        info "镜像构建成功: $image_tag"
    else
        error "镜像构建失败"
        exit 1
    fi
}

# 显示运行示例
show_run_examples() {
    local image_tag="${IMAGE_NAME}:${TAG}"

    echo ""
    info "============================================"
    info "构建完成！镜像: $image_tag"
    info "============================================"
    echo ""
    info "运行示例:"
    echo ""

    # 基本运行
    echo "  # 基本运行（使用默认配置）"
    echo "  docker run -d -p 8080:8080 $image_tag"
    echo ""

    # 挂载数据卷
    echo "  # 挂载数据卷（数据持久化）"
    echo "  docker run -d -p 8080:8080 \\"
    echo "    -v \$(pwd)/data:/app/data \\"
    echo "    -v \$(pwd)/uploads:/app/uploads \\"
    echo "    $image_tag"
    echo ""

    # 环境变量配置
    echo "  # 使用环境变量配置"
    echo "  docker run -d -p 8080:8080 \\"
    echo "    -e SERVER_HOST=0.0.0.0 \\"
    echo "    -e SERVER_PORT=8080 \\"
    echo "    -e LOG_LEVEL=info \\"
    echo "    $image_tag"
    echo ""

    # 开发模式（挂载本地代码）
    echo "  # 开发模式（挂载本地代码，热更新）"
    echo "  docker run -d -p 8080:8080 \\"
    echo "    -v \$(pwd):/app \\"
    echo "    $image_tag"
    echo ""

    # 查看日志
    echo "  # 查看日志"
    echo "  docker logs -f \$(docker ps -q -f ancestor=$image_tag)"
    echo ""

    # 停止容器
    echo "  # 停止容器"
    echo "  docker stop \$(docker ps -q -f ancestor=$image_tag)"
    echo ""
}

# 主函数
main() {
    echo ""
    echo "============================================"
    echo " 设备固件升级管理服务 - Docker 构建脚本"
    echo "============================================"
    echo ""

    check_docker
    check_dockerfile
    build_image
    show_run_examples
}

# 执行主函数
main
