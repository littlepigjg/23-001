# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

固件升级系统在灰度发布场景下，设备 ID 列表出现数据损坏现象。执行灰度升级任务时，原本应该被升级的设备出现丢失，部分设备 ID 被重复写入，导致灰度分组结果与预期严重不符。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：go 1.22.0
- 项目模块：fwupgrade
- 依赖：标准库 + fwupgrade/internal/config、fwupgrade/internal/model、fwupgrade/internal/service、fwupgrade/internal/store
- 运行参数：无需特殊参数

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 初始化一个 MemoryStore，创建设备型号和固件
3. 创建 10 台测试设备（deviceID 分别为 dev001 ~ dev010）
4. 创建 GrayscaleService 和 TaskService，将 GrayscaleService 注入 TaskService
5. 创建一个灰度升级任务（TaskType 为 grayscale，GrayscaleRatio 为 50%）
6. 调用 TaskService.StartTask 启动任务
7. 观察被选中的设备 ID 列表

```bash
cd /path/to/fwupgrade
go test -v -count=1 -run '^TestRedGreen$' .
```

## 4. 实际结果（Actual Behavior / Observed Output）

执行上述测试命令后，观察到以下异常：

- 原始输入的设备 ID 列表 `[dev001, dev002, dev003, dev004, dev005, dev006, dev007, dev008, dev009, dev010]` 在灰度分组后被篡改
- 部分设备 ID 消失，如 dev001、dev002 不见了
- 后续设备 ID 被重复出现，如 dev007、dev008、dev009、dev010 多次出现
- 对简单 5 元素输入 `[a, b, c, d, e]`，结果变成 `[b, d, e, d, e]`，第一个元素丢失，后两个元素重复

RED/GREEN 判定结果：RED（红灯）

```
RED（红灯，缺陷未修复）: grayGroup=[dev003 dev006 dev007 dev009] waitGroup=[dev003 dev006 dev007 dev009 dev008 dev010] input_corrupted ids=[dev003 dev006 dev007 dev009 dev008 dev010 dev007 dev008 dev009 dev010] | test2: input=[a b c d e] gray=[b d e] wait=[b d] input_corrupted testIDs=[b d e d e] | StartTask_ok |
--- FAIL: TestRedGreen (0.00s)
FAIL
```

## 5. 期望结果（Expected Behavior）

灰度分组后应满足：

- 原始输入的设备 ID 列表保持完整不变，无丢失、无重复
- 灰度组和等待组包含所有设备，且元素数量之和等于输入总数
- 被选中的灰度设备符合预期的灰度比例
- RED/GREEN 判定结果：GREEN（绿灯）
- go build ./... 编译通过
- go vet ./... 无静态分析错误

## 6. 触发频率（Frequency）

必现（100%）。每次调用灰度分组功能均能稳定复现，无需特殊条件或多次重复。

## 7. 影响范围（Impact / Scope）

- 灰度升级任务的设备选择结果错误，导致部分设备被跳过或重复升级
- 设备 ID 丢失可能导致后续升级记录与实际设备不匹配
- 多次灰度策略调用后设备 ID 丢失累积，影响范围逐步扩大
- 影响灰度升级的准确性和可靠性，可能造成部分设备无法按时完成升级

## 8. 附加说明（Additional Notes / Workaround）

目前无有效 workaround。如需临时规避，可在灰度升级前对设备 ID 列表创建独立副本后再传入灰度分组逻辑，但这只是治标不治本的临时方案。建议从根本上修复该数据污染问题。
