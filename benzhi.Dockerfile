# benzhi.Dockerfile - 设备固件升级管理服务
# 基于 golang:1.22 官方镜像（Debian 版本自带 gcc，支持 CGO/race 检测）

FROM golang:1.22

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
