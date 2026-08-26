# 设备固件升级管理服务

## 项目简介

设备固件升级管理服务（Device Firmware Upgrade Management Service）是一个用 Go 语言开发的后端系统，用于管理 IoT 设备的固件版本、创建升级任务、执行灰度发布、跟踪升级进度并统计升级效果。

**技术栈：** Go 1.22 标准库，无第三方依赖

**核心功能：**
- 设备型号管理（增删改查）
- 固件版本管理（上传、下载、MD5校验）
- 设备注册与在线状态上报
- 升级任务创建（全量/灰度/指定设备三种模式）
- 设备轮询接口（获取升级指令）
- 升级进度上报与历史记录
- 灰度策略计算与自动推进
- 升级统计与仪表盘

## 目录结构

```
fwupgrade/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── handler/
│   │   ├── router.go            # 路由与中间件
│   │   ├── handlers.go          # 基础处理器
│   │   ├── api_handler.go      # API 处理器（固件/任务/进度等）
│   │   └── setup.go             # 路由配置
│   ├── model/
│   │   ├── device.go            # 数据模型定义
│   │   └── request.go           # 请求/响应结构体
│   ├── service/
│   │   ├── device_service.go    # 设备业务逻辑
│   │   ├── firmware_service.go  # 固件业务逻辑
│   │   ├── task_service.go      # 任务业务逻辑
│   │   ├── grayscale_service.go # 灰度策略
│   │   ├── poll_service.go      # 轮询服务
│   │   ├── progress_service.go  # 进度服务
│   │   ├── history_service.go   # 历史记录
│   │   └── stats_service.go     # 统计服务
│   └── store/
│       ├── store.go             # 存储接口定义
│       ├── memory_store.go      # 内存存储基础实现
│       ├── memory_store_impl.go # 固件/任务/记录存储实现
│       └── file_store.go        # 文件持久化存储
├── pkg/
│   ├── logger/
│   │   └── logger.go            # 结构化日志
│   ├── response/
│   │   └── response.go          # 统一响应格式
│   ├── md5util/
│   │   └── md5.go               # MD5 工具
│   ├── timeutil/
│   │   └── time.go              # 时间工具
│   └── fileutil/
│       └── file.go              # 文件工具
├── web/                         # 前端静态文件
│   ├── index.html
│   ├── css/style.css
│   └── js/app.js
├── go.mod
├── BUG_CATALOG.md               # 缺陷候选清单
├── benzhi.Dockerfile            # Docker 构建文件
├── build_benzhi_docker.sh       # Docker 构建脚本
├── BENZHI_README.md             # 本文档
└── .dockerignore
```

## API 文档

### 健康检查

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查（返回服务状态、版本、运行时间） |
| GET | `/ready` | 就绪检查（返回各依赖服务状态） |

### 设备型号管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/models?page=1&page_size=20` | 获取型号列表（分页） |
| POST | `/api/models` | 创建设备型号 |
| GET | `/api/models/{id}` | 获取单个型号详情 |
| PUT | `/api/models/{id}` | 更新型号信息 |
| DELETE | `/api/models/{id}` | 删除型号 |

**创建型号请求体：**
```json
{
  "name": "SmartSensor-X1",
  "manufacturer": "TechCorp",
  "hardware_version": "1.0",
  "description": "智能传感器 X1"
}
```

### 设备管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/devices?page=1&page_size=20&model_id=1&status=online` | 获取设备列表 |
| POST | `/api/devices` | 创建设备 |
| GET | `/api/devices/{id}` | 获取设备详情 |
| PUT | `/api/devices/{id}` | 更新设备信息 |
| DELETE | `/api/devices/{id}` | 删除设备 |
| POST | `/api/devices/register` | 设备注册/上报 |
| GET | `/api/devices/search?keyword=xxx` | 搜索设备 |
| POST | `/api/devices/batch` | 批量创建设备 |
| GET | `/api/devices/status` | 获取设备状态统计 |

**设备注册请求体：**
```json
{
  "device_id": "sensor-001",
  "model_id": 1,
  "name": "传感器 #001",
  "firmware_version": "1.0.0",
  "ip_address": "192.168.1.101",
  "serial_number": "SN0000000001",
  "status": "online"
}
```

### 固件管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/firmware?page=1&page_size=20&model_id=1` | 获取固件列表 |
| POST | `/api/firmware` | 上传固件（multipart form） |
| GET | `/api/firmware/{id}` | 获取固件详情 |
| PUT | `/api/firmware/{id}` | 更新固件信息 |
| DELETE | `/api/firmware/{id}` | 删除固件 |
| GET | `/api/firmware/{id}/download` | 下载固件文件 |

**上传固件表单字段：**
- `model_id`: 型号ID（必填）
- `version`: 版本号（必填）
- `md5`: MD5校验值（可选，留空自动计算）
- `changelog`: 更新日志
- `file`: 固件文件（必填，支持 .bin/.hex/.img 等格式）

### 升级任务管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/tasks?page=1&page_size=20&status=running` | 获取任务列表 |
| POST | `/api/tasks` | 创建升级任务 |
| GET | `/api/tasks/{id}` | 获取任务详情 |
| PUT | `/api/tasks/{id}` | 更新任务（仅待执行状态） |
| DELETE | `/api/tasks/{id}` | 删除任务 |
| POST | `/api/tasks/{id}/start` | 启动任务 |
| POST | `/api/tasks/{id}/cancel` | 取消任务 |
| GET | `/api/tasks/{id}/progress` | 获取任务进度 |

**创建任务请求体：**
```json
{
  "name": "X1 固件升级 v1.1.0",
  "model_id": 1,
  "firmware_id": 2,
  "task_type": "grayscale",
  "grayscale_ratio": 10,
  "description": "灰度升级 10% 设备",
  "created_by": "admin"
}
```

**任务类型：**
- `full`: 全量升级（所有设备）
- `grayscale`: 灰度升级（按比例选取设备）
- `targeted`: 指定设备升级

### 设备轮询接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/poll` | 单设备轮询（获取升级指令） |
| POST | `/api/poll/batch` | 批量设备轮询 |

**轮询请求体：**
```json
{
  "device_id": "sensor-001",
  "model_id": 1,
  "current_version": "1.0.0",
  "device_status": "online"
}
```

**轮询响应体：**
```json
{
  "code": 0,
  "data": {
    "should_upgrade": true,
    "task_id": 1,
    "firmware_id": 2,
    "firmware_version": "1.1.0",
    "firmware_url": "/api/firmware/2/download",
    "grayscale_ratio": 10
  }
}
```

### 进度上报接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/progress` | 上报单个设备升级进度 |
| POST | `/api/progress/batch` | 批量上报进度 |
| GET | `/api/progress?device_id=xxx` | 查询设备升级进度 |
| GET | `/api/progress/task/{id}` | 查询任务中所有设备的进度 |

**上报进度请求体：**
```json
{
  "device_id": "sensor-001",
  "task_id": 1,
  "progress": 50,
  "status": "in_progress",
  "error_message": ""
}
```

### 升级历史

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/history?page=1&page_size=20` | 获取历史记录列表 |
| GET | `/api/history/{id}` | 获取单条记录详情 |
| DELETE | `/api/history/{id}` | 删除记录 |
| GET | `/api/history/recent?limit=10` | 获取最近记录 |
| GET | `/api/history/device/{device_id}` | 获取设备的升级历史 |
| GET | `/api/history/task/{task_id}` | 获取任务的升级历史 |

### 统计分析

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/stats/dashboard` | 仪表盘数据（概览） |
| GET | `/api/stats/statistics` | 详细统计数据 |
| GET | `/api/stats/versions` | 各版本设备分布 |
| GET | `/api/stats/models` | 各型号设备分布 |
| GET | `/api/stats/tasks` | 任务状态汇总 |
| GET | `/api/stats/devices` | 设备状态汇总 |

### 数据初始化

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/init/sample` | 初始化示例数据（创建测试型号/设备/固件） |

## 本地运行

### 前置条件
- Go 1.22+
- 操作系统：Linux/macOS/Windows

### 运行步骤

1. **克隆项目**
```bash
cd /home/admin/code/23/001/23-001
```

2. **直接运行**
```bash
go run ./cmd/server
```

3. **或编译后运行**
```bash
go build -o fwupgrade ./cmd/server
./fwupgrade
```

4. **自定义端口运行**
```bash
go run ./cmd/server --port 9090
```

5. **使用配置文件运行**
```bash
go run ./cmd/server --config ./config.ini
```

6. **查看帮助**
```bash
go run ./cmd/server --help
```

### 环境变量配置

| 变量 | 描述 | 默认值 |
|------|------|--------|
| SERVER_HOST | 监听地址 | 0.0.0.0 |
| SERVER_PORT | 端口号 | 8080 |
| STORAGE_TYPE | 存储类型（memory/file） | memory |
| DATA_DIR | 数据存储目录 | data |
| UPLOAD_DIR | 固件上传目录 | uploads |
| LOG_LEVEL | 日志级别（debug/info/warn/error） | info |
| MAX_FILE_SIZE | 最大固件文件大小（字节） | 104857600 |

### 初始化示例数据

启动服务后，执行以下命令初始化测试数据：
```bash
curl -X POST http://localhost:8080/api/init/sample
```

## Docker 构建与运行

### 使用构建脚本（推荐）

```bash
# 赋予脚本执行权限
chmod +x build_benzhi_docker.sh

# 使用默认参数构建
./build_benzhi_docker.sh

# 自定义镜像名、标签和平台
./build_benzhi_docker.sh my-fwupgrade v2.0.0 linux/arm64
```

### 手动 Docker 命令

```bash
# 构建镜像
docker build -f benzhi.Dockerfile -t fwupgrade:latest .

# 运行容器（基本）
docker run -d -p 8080:8080 fwupgrade:latest

# 运行容器（挂载数据卷）
docker run -d -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/uploads:/app/uploads \
  fwupgrade:latest

# 运行容器（使用环境变量）
docker run -d -p 8080:8080 \
  -e SERVER_PORT=9090 \
  -e LOG_LEVEL=debug \
  fwupgrade:latest

# 查看日志
docker logs -f <container_id>

# 停止容器
docker stop <container_id>
```

## 测试命令

### API 测试

```bash
# 健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready

# 创建设备型号
curl -X POST http://localhost:8080/api/models \
  -H "Content-Type: application/json" \
  -d '{"name":"TestModel","manufacturer":"TestMfg","hardware_version":"1.0","description":"测试型号"}'

# 创建设备
curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{"device_id":"test-001","model_id":1,"name":"测试设备","ip_address":"192.168.1.1","serial_number":"SN001"}'

# 查询设备列表
curl http://localhost:8080/api/devices?page=1&page_size=20

# 仪表盘
curl http://localhost:8080/api/stats/dashboard
```

### 代码质量检查

```bash
# 编译检查
go build ./...

# 代码静态分析
go vet ./...

# 竞态检测
go test -race -count=1 ./...

# 格式化检查
gofmt -l .
```

### 压力测试示例

```bash
# 使用 curl 循环测试并发
for i in {1..100}; do
  curl -s http://localhost:8080/api/devices?page=1 > /dev/null &
done
wait

# 或使用 ab 命令
ab -n 1000 -c 10 http://localhost:8080/health
```

## 缺陷注入

详细的缺陷候选清单请参见 [BUG_CATALOG.md](./BUG_CATALOG.md)。

## 许可证

本项目仅供学习和研究使用。
