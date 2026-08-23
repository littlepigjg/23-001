# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
统计仪表盘接口在多次调用或长时间运行后， goroutine 数量持续增长不释放。每次调用统计接口都会泄漏一个 goroutine，随着请求量增加，服务器 goroutine 数不断攀升，最终可能导致内存占用过高和服务响应变慢。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22.0
- 项目模块：fwupgrade
- 运行参数：go test -race -count=5 -v -run '^TestRedGreen$'
- CPU 核数：与并发相关，建议 2 核以上

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 go build ./... 确保编译通过
2. 执行 go test -race -count=5 -v -run '^TestRedGreen$' 运行测试
3. 观察测试输出与 goroutine 数量变化

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出：RED (红灯，缺陷未修复)
- goroutine 增长：每次运行增长 30 个 goroutine（baseline=2, current=32, growth=30）
- 5 次运行累积：goroutine 从 2 增长到 152
- 退出码：1（FAIL）
- go test -race 报告：无 DATA RACE（并发缺陷为 goroutine 阻塞泄漏，非常规数据竞争）
- 其他异常现象：每次调用统计接口后，内部 goroutine 阻塞在 channel send 操作，无法自行退出

## 5. 期望结果（Expected Behavior）
- 测试输出：GREEN (绿灯，缺陷已修复)
- goroutine 增长：连续 5 次运行 goroutine 数量稳定，无持续增长（growth < 5）
- 退出码：0（PASS）
- go test -race 报告：无 DATA RACE
- go build ./... 与 go vet ./... 全部通过
- 统计接口正常返回数据，内部 goroutine 正常退出无泄漏

## 6. 触发频率（Frequency）
必现（100%）：每次调用统计接口均泄漏 1 个 goroutine，调用次数越多泄漏越多。

## 7. 影响范围（Impact / Scope）
- goroutine 泄漏导致内存持续增长
- 长时间运行后服务器响应变慢
- 高并发场景下可能导致 goroutine 数超限
- 统计后台定时任务每 5 秒触发一次，持续产生泄漏
- 最终可能导致服务 OOM 或不可用

## 8. 附加说明（Additional Notes / Workaround）
当前无有效临时规避方法。可通过重启服务暂时恢复，但问题会再次出现。建议尽快修复。