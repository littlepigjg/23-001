# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
固件升级管理服务在执行优雅关闭时，进程无法正常退出。服务关闭接口被调用后，HTTP 请求持续超时，必须通过强制 kill 才能终止进程。通过 pprof 分析发现有大量后台 goroutine 阻塞在 `sync.WaitGroup.Wait()` 调用上，表明存在 goroutine 泄漏。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go version go1.22.x
- 项目模块：fwupgrade
- 运行参数：go test -race -count=20 -timeout 120s -run '^TestRedGreen$' .
- 硬件信息：CPU 4 核及以上

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go vet ./...` 确保无静态错误
3. 执行 `go test -race -count=20 -timeout 120s -run '^TestRedGreen$' .`
4. 观察测试输出和退出码

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出 RED（红灯）：
  ```
  red_green_test.go:62: RED: PollService goroutine leak detected - timeout waiting for goroutines to finish
  red_green_test.go:131: RED: TaskService goroutine leak detected - timeout waiting for goroutines to finish
  === RED（红灯，缺陷未修复）===
  ```
- 退出码：1（FAIL）
- 其他异常现象：
  - 日志中出现 "context cancelled: context canceled" 错误信息
  - 日志中出现 "Context cancelled during task completion" 错误信息
  - 每次运行都稳定复现，无需多次尝试
- go test -race 不报告 DATA RACE（此缺陷为功能逻辑缺陷，非数据竞争）

## 5. 期望结果（Expected Behavior）
- 无 goroutine 泄漏，所有后台 goroutine 在 context 取消后及时退出
- lifecycle.Stop() 在超时时间内（3秒）正常返回 nil
- 测试输出 GREEN（绿灯）：
  ```
  red_green_test.go:63: GREEN: PollService lifecycle shutdown clean
  red_green_test.go:136: GREEN: TaskService lifecycle shutdown clean
  === GREEN（绿灯，缺陷已修复）===
  ```
- 退出码：0（PASS）
- go build 与 go vet 全部通过
- go test -race -count=20 连续 20 次全部通过

## 6. 触发频率（Frequency）
100% 必现。只要服务启动了轮询调度或任务调度，并在 goroutine 执行期间触发 context 取消（如服务关闭），就会稳定触发 goroutine 泄漏。单次运行即可复现。

## 7. 影响范围（Impact / Scope）
- 服务无法优雅关闭，必须强制 kill
- 服务重启时可能导致端口占用、数据不一致
- goroutine 泄漏持续消耗系统资源
- 高并发场景下泄漏的 goroutine 数量增多，资源消耗加剧
- 影响运维操作，增加服务故障排查难度

## 8. 附加说明（Additional Notes / Workaround）
- 临时规避方法：在服务关闭前等待所有后台任务完成（不可行，因为任务本身就无法完成）
- 建议修复后重点回归：服务优雅关闭流程、轮询调度启停、任务调度启停
- 相关日志样例：
  ```
  ERROR: Poll batch failed error=context cancelled: context canceled
  ERROR: Context cancelled during task completion task_id=X error=context canceled
  ```
