# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

设备固件升级管理服务在处理涉及设备型号查询的操作时，当请求中指定的 model_id 在系统中不存在，服务会直接崩溃（panic），而不是返回合理的错误提示。受影响的操作包括：创建升级任务、更新升级任务、校验型号灰度配置、获取灰度型号名称等。

## 2. 环境信息（Environment）

- 操作系统：Linux (amd64)
- Go 版本：Go 1.22
- 项目模块：fwupgrade
- 运行参数：默认配置（存储类型：memory）
- 硬件信息：CPU 核数与并发缺陷无关

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 启动服务：`go run ./cmd/server`
3. 调用接口创建一个有效的设备型号（model_id 会自动分配，假设为 1）
4. 调用接口创建一个固件，关联到一个不存在的型号（指定 model_id 为 999）
5. 调用创建升级任务接口（POST /api/tasks），请求体中 model_id 设为不存在的值（如 999），firmware_id 使用上一步创建的固件 ID
6. 观察服务是否崩溃或返回错误

或者使用测试文件快速复现：
```
go test -v -run "TestNilPointer|TestCreateTask_Integration"
```

## 4. 实际结果（Actual Behavior / Observed Output）

- 具体的 panic 信息：
  ```
  runtime error: invalid memory address or nil pointer dereference
  [recovered from panic]
  
  goroutine 1 [running]:
  fwupgrade/internal/service.(*TaskService).CreateTask(...)
      .../task_service.go:70
  ```
- RED/GREEN 判定结果：RED（红灯，缺陷存在）
- 所有四个相关函数（CreateTask、UpdateTask、ValidateModelGrayscaleConfig、GetGrayscaleModelName）在触发条件下均会 panic
- go test -race 不涉及（本缺陷为确定性 nil 指针问题，非竞态条件）

## 5. 期望结果（Expected Behavior）

- 无 panic 发生
- 当 model_id 不存在时，应返回明确的错误信息，如 "model not found: id=999"
- RED/GREEN 判定结果应为 GREEN
- 正常型号查询（model_id 存在）功能不受影响，仍正常返回数据
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）

必现（100%）：只要传入不存在的 model_id 且 PanicGuard 钩子设置为返回 false，必定触发 nil pointer dereference panic。无需重复多次。

## 7. 影响范围（Impact / Scope）

- 服务崩溃：任何涉及无效 model_id 的请求都会导致服务 panic，影响稳定性
- 功能不可用：创建任务、更新任务、灰度配置校验等核心功能在特定输入下完全不可用
- 用户体验差：调用方收到的是 500 内部错误或连接断开，无法得知具体原因
- 数据一致性：本缺陷不会直接导致数据损坏，但可能导致部分写入操作半途而废

## 8. 附加说明（Additional Notes / Workaround）

临时规避方法：
1. 在调用相关接口前，先验证 model_id 是否存在
2. 确保所有前端表单在提交前已校验型号 ID 的有效性
3. 如果使用文件存储（file 模式），同样会触发此问题，因为 FileStore 委托给 MemoryStore 处理

相关测试命令：
```bash
# 运行全部缺陷验证测试
go test -v -run "TestNilPointer|TestNormal|TestPanicGuard|TestCreateTask_Integration"

# 单独验证 CreateTask 缺陷
go test -v -run "TestNilPointerPanice_InTaskService_CreateTask"

# 单独验证 UpdateTask 缺陷
go test -v -run "TestNilPointerPanice_InTaskService_UpdateTask"

# 单独验证 GrayscaleService 缺陷
go test -v -run "TestNilPointerPanice_InGrayscaleService"
```
