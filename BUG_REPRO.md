# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
固件升级系统中，新创建的升级任务在未启动状态下查询进度时，返回了错误的进度信息（Progress=100, Status=Running），而非预期的 Progress=0, Status=Pending。系统内部实际上发生了 nil 指针解引用 panic，但被错误地 recover 后返回了伪造的"缓存数据"，导致业务数据严重失真。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22.0
- 项目模块：fwupgrade
- 项目路径：/home/admin/code/23/001/23-001-12
- 存储类型：memory（内存存储）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录：`cd /home/admin/code/23/001/23-001-12`
2. 确保编译通过：`go build ./...`
3. 运行复现测试：`go test -v -count=1 -run '^TestRedGreen$' .`
4. 观察测试输出和日志

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出 RED（红灯），测试失败
- 日志中出现警告：`Progress calculation encountered issue, returning cached data`
- recover 信息：`runtime error: invalid memory address or nil pointer dereference`
- GetTaskProgress 返回 Progress=100、Status=TaskRunning（完全错误的数据）
- 测试退出码为 1

典型输出：
```
RED (红灯，缺陷未修复) - New task progress should be 0, but got 100 (panic recovery returned incorrect data) | New task status should be pending, but got running (panic recovery changed status)
--- FAIL: TestRedGreen (0.00s)
FAIL
```

## 5. 期望结果（Expected Behavior）
- 无 panic、无 nil 指针解引用
- GetTaskProgress 正确返回 Progress=0、Status=TaskPending（新建任务的初始状态）
- 日志中不应出现 "Progress calculation encountered issue" 警告
- go build ./... 编译通过
- go vet ./... 无报错
- 测试输出 GREEN（绿灯），退出码为 0

## 6. 触发频率（Frequency）
必现（100%）。只要对一个刚创建未启动的任务（CompletedAt=nil）调用进度查询接口，必然触发。

## 7. 影响范围（Impact / Scope）
- 所有新建未启动的任务查询进度时返回完全错误的数据（进度100%、状态运行中）
- 前端页面展示严重失真，误导操作人员
- 统计数据被污染，可能影响运营决策
- panic 被静默 recover 掩盖，排查困难

## 8. 附加说明（Additional Notes / Workaround）
临时规避方案：在任务创建后手动设置 CompletedAt 为未来某个时间点，但这只是治标不治本，且会破坏任务的时间统计逻辑。正确修复需要定位 nil 指针解引用的根因并加以解决。
