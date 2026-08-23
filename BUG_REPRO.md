# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

固件升级管理系统的型号管理模块存在数据一致性问题。在连续调用型号列表相关接口后，返回的数据会出现错乱，包括型号条目重复、部分型号丢失、出现明显不属于业务数据的元数据条目（如 `__export_metadata__`、`__filter_metadata__`），以及统计接口返回的计数不准确。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：go1.22.0
- 项目模块：fwupgrade
- 运行参数：无特殊参数，使用默认配置
- 硬件信息：不涉及并发/性能相关硬件特性

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -v -run "TestRedGreen" ./internal/store/` 运行验收测试
3. 观察测试结果

手动复现路径：
1. 初始化数据存储，添加 5 个设备型号（2个 Dell、2个 HPE、1个 Lenovo）
2. 调用型号列表接口，确认返回 5 个型号，数据正确
3. 调用按厂商筛选接口，过滤 Dell 型号，返回 2 个结果
4. 再次调用型号列表接口，观察返回数据

## 4. 实际结果（Actual Behavior / Observed Output）

- 型号列表接口返回的数据中出现重复条目（如 Dell R640 出现两次）
- 部分型号丢失（如 HPE DL380 不见）
- 返回列表中包含非法元数据条目（如 `__export_metadata__`、`__filter_metadata__`）
- 统计接口返回的厂商计数不准确（如 Lenovo 计数为 0）

验收测试结果（RED 状态）：
```
=== RUN   TestRedGreen_BuildModelListPollutesCache
    RED FAIL: Model[1] changed from 'HPE DL380' to 'Dell R640'
--- FAIL: TestRedGreen_BuildModelListPollutesCache (0.00s)
=== RUN   TestRedGreen_ExportActiveModelsCorruptsStats
    RED FAIL: Lenovo count changed from 1 to 0 after export (data pollution)
--- FAIL: TestRedGreen_ExportActiveModelsCorruptsStats (0.00s)
=== RUN   TestRedGreen_FilterModelsByDescCorruptsList
    RED FAIL: Metadata marker '__filter_metadata__' leaked into model list (data pollution)
--- FAIL: TestRedGreen_FilterModelsByDescCorruptsList (0.00s)
=== RUN   TestRedGreen_SequentialOperations
    RED FAIL: Corrupted model detected: ID=0, Name='__export_metadata__', Manufacturer='system'
--- FAIL: TestRedGreen_SequentialOperations (0.00s)
FAIL
退出码：1
```

## 5. 期望结果（Expected Behavior）

- 型号列表接口返回的型号数据完整、无重复、无非法条目
- 连续调用任意接口组合后，数据保持一致性
- 统计接口返回准确的厂商型号计数和活跃型号占比
- 验收测试全部通过（GREEN 状态）
- `go build ./...` 通过
- `go vet ./...` 无报错

## 6. 触发频率（Frequency）

必现（100%）。只要调用涉及型号列表过滤/导出的接口，再调用型号列表接口，必然触发数据污染。

## 7. 影响范围（Impact / Scope）

- 型号管理页面显示错误数据，影响运维人员判断
- 统计报表数据不准确，影响容量规划和设备管理决策
- 批量导出功能可能包含错误数据，导致下游系统异常
- 重启服务后数据恢复正常，但操作后会再次出现，严重影响用户体验

## 8. 附加说明（Additional Notes / Workaround）

临时规避方法：在需要获取准确数据时，重启服务或重新初始化数据存储。此方法治标不治本，且在生产环境中不现实。建议尽快修复根因。
