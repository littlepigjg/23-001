# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

固件升级系统的历史记录查询模块存在数据串扰缺陷：当连续调用不同的记录查询接口时（例如先获取最近记录，再按设备查询历史），前一次调用返回的结果数据会被后一次调用污染，导致显示的数据与实际不一致。该问题在快速连续操作或多页面快速切换时较易出现。

## 2. 环境信息（Environment）

- 操作系统：Linux amd64
- Go 版本：go1.26.5
- 项目模块：fwupgrade
- 依赖：仅标准库 + pkg/logger（项目内部包）
- 运行参数：go test . -count=1 -run '^TestRedGreen$'

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 创建 MemoryStore 实例并写入至少 3 条以上的升级记录（不同设备 ID、不同开始时间）
3. 调用获取最近记录的方法（如 GetRecentRecords），保存返回的切片引用
4. 再调用按条件查询记录的方法（如按设备 ID 查询历史）
5. 观察步骤 3 中保存的切片引用，检查其元素的字段值是否已被改变

## 4. 实际结果（Actual Behavior / Observed Output）

- 步骤 3 返回的切片在步骤 4 执行后，其元素的字段值（如 ID、DeviceID、StartedAt 等）被后续查询操作覆盖
- 测试输出：
  ```
  RED (红灯，缺陷未修复)
      red_green_test.go:64: recent records data corrupted after subsequent store call: recent[0].ID changed from 5 to 3, DeviceID changed from device-01 to device-03
  --- FAIL: TestRedGreen (0.00s)
  ```
- 退出码：1（FAIL）

## 5. 期望结果（Expected Behavior）

- 步骤 3 返回的切片数据在步骤 4 执行后保持不变，每次查询返回的结果应为独立的数据副本
- 测试输出应为：
  ```
  GREEN (绿灯，缺陷已修复)
  --- PASS: TestRedGreen (0.00s)
  PASS
  ```
- 退出码：0（PASS）
- go build ./... 与 go vet ./... 均通过

## 6. 触发频率（Frequency）

必现（100%）：只要在持有前一次查询返回切片引用的情况下调用另一个使用共享缓冲区的查询函数，数据污染必然发生。

## 7. 影响范围（Impact / Scope）

- 历史记录页面数据展示错误：用户看到的记录信息与实际不符
- 前端页面快速切换时出现数据串扰：从历史列表点击进入设备详情后返回，列表数据可能被篡改
- 任何依赖"先获取结果再进行额外查询"的业务逻辑均可能受影响
- 自动化流程中如果连续调用不同查询接口，可能产生错误的业务判断

## 8. 附加说明（Additional Notes / Workaround）

临时规避方法：每次查询后不要跨接口持有返回切片的引用，或在需要持久化时自行创建副本。根本修复需要确保每次查询返回独立的数据副本而非共享底层数组的子切片。
