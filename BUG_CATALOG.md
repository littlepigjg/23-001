# 缺陷候选清单 (BUG_CATALOG)

> 本清单包含 30 个可注入缺陷，每个缺陷跨至少 2 个文件。
> 项目缩写：fwupgrade (设备固件升级管理服务)

## 缺陷列表

| bug_id | bug_category | 缺陷描述 | 植入位置 | 预期表现 | 触发方式 | 缺陷难度 |
|--------|-------------|---------|----------|---------|---------|---------|
| fwupgrade-concur-001 | concurrency | 高并发下 map 写入未加锁导致 data race | internal/store/memory_store_impl.go.CreateFirmware, internal/store/memory_store_impl.go.CreateTask | `concurrent map read and map write` panic | 并发上传固件或创建任务时触发 race condition | 2⭐ |
| fwupgrade-concur-002 | concurrency | WaitGroup 计数器错配导致 goroutine 永远阻塞 | internal/service/poll_service.go.SchedulePolling, internal/service/task_service.go.ProcessScheduledTasks | 轮询调度和任务调度 goroutine 卡住不退出 | 关闭服务时发现 goroutine 泄露，用 pprof 检测 | 3⭐ |
| fwupgrade-concur-003 | concurrency | channel 未关闭导致消费者 goroutine 泄露 | internal/service/stats_service.go.GetDashboard, internal/handler/setup.go.Setup | goroutine 持续增长不释放 | 多次请求统计接口，用 goroutine dump 观察 | 3⭐ |
| fwupgrade-concur-004 | concurrency | 配置读写锁保护范围缺失，字段无锁访问 | internal/config/config.go.LoadFromFile, internal/service/firmware_service.go.GetLatestFirmware | 高并发读取配置时偶发不一致 | 多 goroutine 并发访问配置与固件接口 | 2⭐ |
| fwupgrade-concur-005 | concurrency | 进度更新时读写竞态导致数据不一致 | internal/service/progress_service.go.ReportProgress, internal/store/memory_store.go.UpdateDeviceProgress | 进度数值跳变或回退 | 设备同时上报进度与查询进度 | 2⭐ |
| fwupgrade-concur-006 | concurrency | 文件存储 saveToFile 与 autoSave 产生竞态 | internal/store/file_store.go.saveToFile, internal/store/file_store.go.autoSave | 写入临时文件时 panic 或数据损坏 | 长时间运行后检查数据完整性 | 3⭐ |
| fwupgrade-nil-001 | nil | 型号不存在时返回 nil 指针未检查 | internal/service/task_service.go.CreateTask, internal/service/grayscale_service.go.DecideGrayscale | nil pointer dereference panic | 创建任务时指定不存在的 model_id | 1⭐ |
| fwupgrade-nil-002 | nil | 设备固件版本为空时字符串操作 panic | internal/service/poll_service.go.PollDevice, internal/handler/api_handler.go.FirmwareHandler.List | `index out of range` 或空指针 | 新注册设备上报空版本号 | 1⭐ |
| fwupgrade-nil-003 | nil | 读取固件文件前未检查 nil 指针 | internal/service/firmware_service.go.GetFirmwareFile, internal/store/memory_store_impl.go.GetFirmwareByVersion | nil pointer dereference | 查询不存在的固件 ID | 1⭐ |
| fwupgrade-nil-004 | nil | 上下文取消时 ctx.Err() 返回 nil 被忽视 | internal/service/poll_service.go.runPollCycle, internal/service/history_service.go.GetDeviceHistory | 返回过期数据或 panic | 服务关闭过程中请求正在处理 | 4⭐ |
| fwupgrade-nil-005 | nil | 时间字段为零值时格式化 nil 指针 | internal/model/device.go.IsActive, internal/service/stats_service.go.calculateSuccessRate | 统计结果 NaN 或 -0001 日期 | 设备从未设置 LastSeenAt | 2⭐ |
| fwupgrade-nil-006 | nil | 任务启动时 CompletedAt 为 nil 被直接访问 | internal/service/task_service.go.GetTaskProgress, internal/model/device.go.CalculateProgress | nil pointer dereference | 任务刚创建未启动时查询进度 | 1⭐ |
| fwupgrade-slice-001 | slice | append 后共享底层数组导致数据污染 | internal/store/memory_store_impl.go.ListModels, internal/store/memory_store_impl.go.ListDevices | 返回数据被后续请求修改 | 并发请求列表接口时数据错乱 | 3⭐ |
| fwupgrade-slice-002 | slice | 灰度设备列表子切片写回污染原数组 | internal/service/grayscale_service.go.GenerateDeviceGroup, internal/service/task_service.go.calculateTargetDevices | 设备列表内容被意外修改 | 多次调用灰度策略后设备 ID 丢失 | 3⭐ |
| fwupgrade-slice-003 | slice | 容量估算错误导致切片越界 | internal/service/task_service.go.calculateTargetDevices, internal/service/progress_service.go.BatchReportProgress | index out of range panic | 目标设备数超过预分配容量 | 2⭐ |
| fwupgrade-slice-004 | slice | 分页计算时 start 偏移未做边界检查 | internal/store/memory_store_impl.go.ListFirmwares, internal/store/memory_store_impl.go.ListRecords | slice bounds out of range | 请求 page 参数超过实际页数 | 2⭐ |
| fwupgrade-slice-005 | slice | records 列表排序后返回时缺少拷贝 | internal/service/history_service.go.ListRecords, internal/store/memory_store_impl.go.GetRecentRecords | 并发下数据竞争 | 多个请求同时获取最近记录 | 3⭐ |
| fwupgrade-error-001 | error | %w 错误链丢失导致 errors.Is 失效 | internal/config/config.go.LoadFromFile, internal/service/firmware_service.go.UploadFirmware | 上层无法用 errors.Is 判断错误类型 | 尝试 errors.Is/errors.As 检查特定错误 | 3⭐ |
| fwupgrade-error-002 | error | err 变量被 := 遮蔽导致错误丢失 | internal/service/device_service.go.CreateDevice, internal/service/task_service.go.CreateTask | 错误被吞掉，用户看到成功但数据未入库 | 创建设备/任务时传入无效参数 | 2⭐ |
| fwupgrade-error-003 | error | 只比较错误字符串而非类型判断 | internal/service/stats_service.go.calculateSuccessRate, internal/handler/api_handler.go.TaskHandler.Create | 不同类型但相同文案的错误无法区分 | 网络抖动导致错误消息变化 | 3⭐ |
| fwupgrade-error-004 | error | 固件上传 MD5 验证失败后文件未清理 | internal/service/firmware_service.go.UploadFirmware, internal/store/memory_store_impl.go.CreateFirmware | 磁盘残留未完成的固件文件 | 上传大文件时网络中断重试 | 2⭐ |
| fwupgrade-error-005 | error | 进度更新错误被忽略导致状态不一致 | internal/service/progress_service.go.ReportProgress, internal/store/memory_store.go.UpdateDeviceStatus | 设备状态未更新但进度已记录 | 设备上报进度到 100% 时 | 2⭐ |
| fwupgrade-error-006 | error | 存储层错误直接返回未封装上下文 | internal/store/file_store.go.Init, internal/config/config.go.LoadFromFile | 错误信息缺少操作上下文 | 文件权限错误时排查困难 | 2⭐ |
| fwupgrade-context-001 | context | context 取消未传播到下游 goroutine | internal/service/poll_service.go.SchedulePolling, internal/service/task_service.go.ProcessScheduledTasks | goroutine 在 ctx cancel 后仍运行 | 关闭服务时轮询/任务调度未停止 | 3⭐ |
| fwupgrade-context-002 | context | context 存入结构体后复用导致并发问题 | internal/service/task_service.go.StartTask, internal/service/progress_service.go.ReportProgress | 使用过期 context 的请求 panic | 长任务运行中 context 被复用 | 4⭐ |
| fwupgrade-context-003 | context | HTTP 请求 context 未传递给存储层 | internal/handler/api_handler.go.DeviceHandler.List, internal/service/device_service.go.ListDevices | 请求超时后存储操作仍执行 | 客户端断开连接后数据库仍被访问 | 3⭐ |
| fwupgrade-context-004 | context | 忽略 ctx.Err() 检查导致已取消请求继续 | internal/service/stats_service.go.GetDashboard, internal/store/memory_store.go.ListModels | 取消的请求仍返回数据 | 客户端快速断开后收到数据 | 3⭐ |
| fwupgrade-defer-001 | defer | 循环内 defer 直到函数返回才执行导致文件句柄泄露 | internal/service/firmware_service.go.UploadFirmware, internal/store/file_store.go.loadFromFile | 大量文件句柄泄露，系统资源耗尽 | 循环上传固件文件 | 3⭐ |
| fwupgrade-defer-002 | defer | defer 修改命名返回值导致错误被覆盖 | internal/service/history_service.go.DeleteRecord, internal/store/memory_store.go.DeleteDevice | 返回 nil 但实际删除失败 | 删除不存在的记录/设备 | 2⭐ |
| fwupgrade-defer-003 | defer | error 分支跳过 defer 释放导致锁未释放 | internal/store/memory_store_impl.go.GetLatestFirmware, internal/service/task_service.go.CancelTask | 永久锁死导致后续请求超时 | 并发获取最新固件时触发 | 3⭐ |
| fwupgrade-other-001 | other | 固件版本号未校验格式导致非法版本入库 | internal/model/device.go.Validate, internal/service/firmware_service.go.UploadFirmware | 非法版本号如 "../../../etc/passwd" 入库 | 创建任务时指定恶意固件版本 | 2⭐ |
| fwupgrade-other-002 | other | 灰度比例使用整数截断导致设备分组不均 | internal/service/grayscale_service.go.isInGrayGroup, internal/service/task_service.go.calculateTargetDevices | 10% 灰度实际只分到 0% 设备 | 创建灰度任务比例为 10% 但设备数少时 | 3⭐ |
| fwupgrade-other-003 | other | 设备 ID 大小写敏感导致重复注册绕过 | internal/service/device_service.go.RegisterDevice, internal/store/memory_store_impl.go.GetDeviceByDeviceID | 同一设备以不同大小写 ID 重复注册 | 设备上报时随机改变 ID 大小写 | 2⭐ |
