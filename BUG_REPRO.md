# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
固件升级管理系统存在三处关联异常：新注册设备（从未产生心跳）被错误标记为活跃状态；统计仪表盘的升级成功率在特定条件下显示异常数值；升级超时检测逻辑混乱，新近启动的升级任务被误判为超时，而真正卡死的任务未被识别。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22.0
- 项目模块：fwupgrade
- 运行参数：go test -v -run '^TestRedGreen$' .

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -v -run '^TestRedGreen$' .` 运行验证测试
3. 观察测试输出，查看三个子测试的判定结果
4. 也可以通过以下手动方式复现各子问题：

   **复现设备活跃状态异常：**
   1. 编写代码创建一个 Device 结构体实例，不设置 LastSeenAt 字段（保持零值）
   2. 调用该实例的 IsActive() 方法
   3. 观察返回值为 true（应为 false）

   **复现成功率 NaN 问题：**
   1. 创建 MemoryStore 实例并初始化
   2. 创建 5 条 UpgradeRecord，状态全部设为 UpgradeInProgress
   3. 通过 StatsService.GetDashboard() 获取统计数据
   4. 检查返回的 SuccessRate 字段，值为 NaN

   **复现超时检测异常：**
   1. 创建 MemoryStore 实例并初始化
   2. 创建两条 UpgradeRecord：一条 StartedAt 为当前时间（新近），一条为 2 小时前（已超时）
   3. 通过 ProgressService.TimeOutCheck() 检查超时记录，超时阈值设为 30 分钟
   4. 观察新近记录也被标记为超时，或真正超时的记录未被检测

## 4. 实际结果（Actual Behavior / Observed Output）
```
=== RUN   TestRedGreen/IsActive_with_zero_LastSeenAt
RED（红灯，缺陷未修复）: IsActive 对零值 LastSeenAt 返回 true
=== RUN   TestRedGreen/Success_rate_with_all_in-progress_records
RED（红灯，缺陷未修复）: 成功率计算产生 NaN（除零错误）
=== RUN   TestRedGreen/Timeout_detection_with_recent_records
RED（红灯，缺陷未修复）: 新近记录也被错误标记为超时
========================================
RED（红灯，缺陷未修复）
========================================
退出码：1
```

- 设备活跃状态判定：零值 LastSeenAt 的设备被判定为活跃（true）
- 成功率计算：当所有记录均为进行中状态时，SuccessRate 返回 NaN（0/0）
- 超时检测：新近启动的升级记录被错误标记为超时
- 测试判定：RED（红灯，缺陷未修复）

## 5. 期望结果（Expected Behavior）
- 设备活跃状态判定：零值 LastSeenAt 的设备应判定为不活跃（false）
- 成功率计算：当所有记录均为进行中状态时，SuccessRate 应返回 0
- 超时检测：只有 StartedAt 早于（当前时间 - 超时阈值）的记录才应被标记为超时
- 测试判定：GREEN（绿灯，缺陷已修复），退出码为 0
- go build ./... 编译通过
- go vet ./... 无警告

## 6. 触发频率（Frequency）
必现（100%）。只要满足以下条件即可稳定复现：
- 设备未设置 LastSeenAt 时调用 IsActive()
- 所有升级记录均为非终态（进行中或取消）时获取统计数据
- 存在新近创建的升级记录时执行超时检测

## 7. 影响范围（Impact / Scope）
- 设备管理模块：设备在线状态判断严重失真，可能触发错误的设备调度策略
- 统计面板：成功率显示 NaN 会导致前端渲染异常（显示空白或"NaN%"），影响运维决策
- 升级调度：超时检测错误会导致正在进行的升级被异常中断，或真正卡死的升级无法被恢复，影响升级任务的可靠性和设备固件安全

## 8. 附加说明（Additional Notes / Workaround）
目前无临时规避方案。建议在修复前，手动检查设备 LastSeenAt 字段是否有效，统计面板数据仅供参考，升级超时检测结果需人工二次确认。
