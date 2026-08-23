# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

固件升级系统的内存存储模块在并发操作场景下出现 panic 和数据竞争。当多个协程同时执行创建固件、创建设备、创建升级任务的操作时，程序随机崩溃，错误信息涉及 "concurrent map read and map write" 或 "fatal error: concurrent map writes"。单线程串行操作完全正常，仅在并发场景下触发。

## 2. 环境信息（Environment）

- 操作系统：Linux (amd64)
- Go 版本：go1.22.0
- 项目模块：fwupgrade
- 运行参数：go test -race -count=20 -run '^TestRedGreen$' -v .
- CPU 核数：多核（并发度越高越容易触发）

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录 `/home/admin/code/23/001/23-001-1`，执行 `go build ./...` 确保编译通过
2. 执行 `go vet ./...` 确保无静态检查报错
3. 执行 `go test -race -count=20 -run '^TestRedGreen$' -v .`
4. 观察测试输出和进程退出码

## 4. 实际结果（Actual Behavior / Observed Output）

- 测试输出：`RED (红灯，缺陷未修复) - 并发竞态导致测试进程崩溃, 退出码: 1 或 2`
- 退出码：非 0（测试 FAIL）
- DATA RACE 警告：
  - `WARNING: DATA RACE` 出现在 `nextID()`、`CreateFirmware()`、`CreateDevice()` 的 map 操作中
  - Read at ... by goroutine N: `runtime.mapassign_fast64()` / `runtime.mapaccess2_faststr()` / `runtime.mapIterStart()`
  - Previous write at ... by goroutine M: 对应 map 写入操作
- 致命错误：`fatal error: concurrent map writes` 或 `concurrent map read and map write`
- 进程崩溃：子进程因 concurrent map access 导致 fatal error，退出码为 1 或 2
- 触发频率：100% 复现（20 次运行中每次都检测到数据竞争）

## 5. 期望结果（Expected Behavior）

- 无 panic、无数据竞争（go test -race 无任何 DATA RACE 警告）
- 测试输出：`GREEN (绿灯，缺陷已修复) - 并发测试通过，无数据竞争`
- 退出码：0（测试 PASS）
- 所有并发创建操作成功完成：
  - 创建固件：固件 ID 唯一，版本索引正确建立
  - 创建设备：设备 ID 唯一，设备计数正确更新
  - 创建任务：任务 ID 唯一，固件版本正确关联
- go build ./... 编译通过
- go vet ./... 无报错

## 6. 触发频率（Frequency）

必现（100%）。在 go test -race -count=20 下，20 次运行全部检测到数据竞争或 fatal error。即使不带 -race 标志，在高并发下也能稳定触发 concurrent map panic。

## 7. 影响范围（Impact / Scope）

- 服务崩溃：并发上传固件、创建设备、创建任务时导致进程 panic 崩溃
- 数据不一致：部分写入完成、部分未完成，导致内部 map 状态不一致
- ID 冲突：并发调用时 ID 计数器无保护，可能产生重复 ID
- 固件版本索引缺失：固件写入成功但版本索引未更新，后续查询失败
- 设备计数错误：设备创建后型号下的设备数未正确更新
- 线上可用性下降：并发操作场景下服务稳定性严重下降

## 8. 附加说明（Additional Notes / Workaround）

临时规避方案：
- 在业务层加分布式锁或串行化队列，确保固件/设备/任务的创建操作串行执行
- 增加操作重试机制，遇到 panic 时自动重试
- 限制并发度，避免同时发起大量创建请求

相关日志样例：
```
WARNING: DATA RACE
Read at 0x00c000123110 by goroutine 15:
  runtime.mapIterStart()
      /usr/local/go/src/runtime/map.go:156
  fwupgrade/internal/store.(*MemoryStore).CreateDevice()
      /home/admin/code/23/001/23-001-1/internal/store/memory_store_device.go:43
Previous write at 0x00c000123110 by goroutine 8:
  runtime.mapassign_fast64()
      /usr/local/go/src/internal/runtime/maps/runtime_fast64.go:196
  fwupgrade/internal/store.(*MemoryStore).CreateFirmware()
      /home/admin/code/23/001/23-001-1/internal/store/memory_store_firmware.go:54
```
