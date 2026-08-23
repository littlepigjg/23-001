# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
固件升级管理系统中，新注册设备若上报空的固件版本号，当设备参与升级轮询或管理员查看固件列表时，服务触发 runtime panic 崩溃，错误信息为 "index out of range [0] with length 0"。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go1.22.0
- 项目模块：fwupgrade
- 测试存储：MemoryStore（内存存储）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 创建设备型号（DeviceModel），获取型号 ID
3. 注册一台新设备，固件版本字段留空（FirmwareVer: ""）
4. 创建一个升级任务（UpgradeTask），将任务状态设为运行中
5. 调用设备轮询接口 PollDevice，传入该空版本设备的 DeviceID 和当前版本（空字符串）
6. 或直接调用固件列表接口 ListFirmwares，传入包含空版本固件的型号 ID

## 4. 实际结果（Actual Behavior / Observed Output）
- 错误信息：`runtime error: index out of range [0] with length 0`
- 判定结果：RED（红灯，缺陷未修复）
- 受影响操作：
  - PollDevice: panic 崩溃
  - ListFirmwares: panic 崩溃
  - shouldSkipUpgrade: panic 崩溃
- 服务直接崩溃，无法返回有效响应

## 5. 期望结果（Expected Behavior）
- 空版本设备能正常参与轮询，PollDevice 返回 ShouldUpgrade=false
- 固件列表能正常排序展示，空版本固件应有合理的排序位置
- 版本比较逻辑对空字符串有正确的默认处理，不触发 panic
- RED/GREEN 判定结果为 GREEN
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）：只要设备固件版本为空，触发轮询或列表操作即必定 panic。

## 7. 影响范围（Impact / Scope）
- 设备轮询服务崩溃，所有设备的升级检查停止
- 固件管理页面无法加载
- 新注册设备如果上报空版本号，会导致整个服务不可用
- 服务需要重启才能恢复，影响线上可用性

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：确保所有设备注册时固件版本号不为空，在设备端强制校验版本号格式。
