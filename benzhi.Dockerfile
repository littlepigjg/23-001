# benzhi.Dockerfile - 设备固件升级管理服务
# 基于 golang:1.22 官方镜像，非多阶段构建

FROM golang:1.22-alpine

# 安装 CGO 依赖（race detector 需要）
RUN apk add --no-cache gcc musl-dev

# 设置工作目录
WORKDIR /app

# 复制源代码
COPY . /app

# 预先下载依赖
RUN go mod download

# 预先编译验证
RUN go build ./...

# 创建必要目录
RUN mkdir -p /app/data /app/uploads /app/logs

# 暴露端口
EXPOSE 8080

# 启动服务
CMD ["go", "run", "./cmd/server"]
