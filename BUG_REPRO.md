# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
固件升级系统在对同一台设备进行并发进度上报时，最终持久化的升级进度值会出现异常：本应达到的最大进度值被较小的进度值覆盖，导致进度回退或跳变；同时在带 -race 的并发测试下会报告 DATA RACE。批量压测场景中还会伴随大量"进度回退"类错误，严重影响升级过程的数据一致性与可观测性。

## 2. 环境信息（Environment）
- 操作系统：Linux（x86_64）
- Go 版本：go1.22.x（详见 `go version` 输出）
- 项目模块：fwupgrade（见 go.mod）
- 运行参数：`go test -race -count=5`（必须带 -race 与 -count=N 以稳定捕获偶发竞态）
- 硬件建议：多核 CPU（4 核及以上）以便并发充分交错

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保项目编译通过。
2. 执行 `go test -race -count=5 -run '^(TestRedGreen|TestProgressMonotonicity|TestConcurrentProgressQuery)$' .` 运行并发验证用例。
3. 观察测试输出与进程退出码。
4. （可选）单独运行 `go test -race -count=20 -run '^TestRedGreen$' .` 以更高次数捕获偶发竞态。

## 4. 实际结果（Actual Behavior / Observed Output）
测试输出出现以下典型片段：
- `RED (红灯，缺陷未修复): progress regressed from 100 to 96, data race detected`
- `RED (红灯，缺陷未修复): progress dropped from 300 to 50 due to non-atomic read-write race`
- `RED (红灯，缺陷未修复): concurrent read-write caused progress loss (100 -> 40)`
- `testing.go: race detected during execution of test`
- 测试进程以退出码 1 结束。

其他异常现象：
- 并发请求中会出现大量 `progress regression detected` / `progress rollback not allowed` 错误，但实际上报的 progress 是单调递增的。
- 在不同机器/不同并发度下，最终进度回退到的值不固定，呈概率性分布。

## 5. 期望结果（Expected Behavior）
- 并发上报完成后，设备的 UpgradeProgress 严格等于所有上报中的最大值（100），不出现任何回退。
- `go test -race -count=5 ...` 全部通过，-race 下不再报告 DATA RACE，退出码为 0。
- 测试输出打印 `GREEN (绿灯，缺陷已修复)`。
- 并发混合读写下，所有读到的进度值都在 [0, 100] 合法区间内，且最终值等于所有写入的最大值。
- `go build ./...` 与 `go vet ./...` 均通过。

## 6. 触发频率（Frequency）
缺陷为并发场景下的偶发竞态：单次普通测试（-count=1）未必立即触发；使用 `-race -count=5` 及 100 goroutine 并发时可稳定复现；提高到 `-count=20` 或更高能 100% 捕获。

## 7. 影响范围（Impact / Scope）
- 批量升级任务进度统计失真，任务整体完成度被低估。
- 设备升级状态可能停留在"升级中"无法正确流转到"成功"，影响后续业务流程（如触发告警、自动下线等）。
- 接口返回错误（如 progress regression / rollback not allowed）导致上层业务中断。
- 数据竞争在高并发下可能引发更隐蔽的崩溃或状态污染，属于典型的生产级并发隐患。

## 8. 附加说明（Additional Notes / Workaround）
- 临时规避方案（仅供参考）：在调用方以串行方式上报进度或加外部串行队列，但无法根治并发写冲突。
- 正式修复需要从并发读写路径入手，保证同一设备上的进度校验与写入具备原子性。
