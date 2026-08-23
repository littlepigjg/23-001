# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
设备管理系统中，当使用不同大小写的设备ID（如 "DEVICE001" vs "device001"）通过创建设备接口（CreateDevice）注册时，系统未能正确识别重复，导致同一设备被创建多次。同时，设备查询接口（GetDeviceByDeviceID）对大小写敏感，使用与创建时不同的大小写查询会返回"设备不存在"。而注册接口（RegisterDevice）在相同场景下却能正确阻止重复，表现出接口间行为不一致。

## 2. 环境信息（Environment）
- 操作系统：Linux (amd64)
- Go 版本：go1.26.5 linux/amd64
- 项目模块：fwupgrade
- 运行参数：无特殊参数，直接运行 go test
- 硬件信息：CPU 与缺陷触发无关，任意 CPU 核数均可复现

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test . -count=1 -run '^TestRedGreen$' -v` 运行验证测试
3. 观察测试输出结果

或手动复现：
1. 启动应用并确保已有设备型号（Model）数据
2. 调用 CreateDevice 接口，传入 DeviceID = "DEVICE001"，创建设备 A
3. 再次调用 CreateDevice 接口，传入 DeviceID = "device001"（仅大小写不同），观察是否创建成功
4. 调用 GetDeviceByDeviceID 接口，分别用 "device001"、"DEVICE001"、"Device001" 查询，观察结果
5. 调用 RegisterDevice 接口，先用 "SENSOR001" 注册，再用 "sensor001" 注册，观察第二次是否被拦截

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试运行结果：RED（红灯，缺陷未修复）
- 退出码：1
- 关键失败信息：
  - CaseInsensitiveCreate: second create with different case should have failed but succeeded
  - CaseInsensitiveGet: lookup with mixed case failed: device not found: device_id=Device001
- 手动复现观察：
  - CreateDevice("DEVICE001") → 成功创建设备 A
  - CreateDevice("device001") → 成功创建设备 B（应失败但未失败）
  - GetDeviceByDeviceID("device001") → 查到设备 B
  - GetDeviceByDeviceID("DEVICE001") → 查到设备 A
  - GetDeviceByDeviceID("Device001") → 返回 "device not found"
  - RegisterDevice("SENSOR001") → 成功
  - RegisterDevice("sensor001") → 正确拦截，返回"已存在"错误

## 5. 期望结果（Expected Behavior）
- RED/GREEN 判定结果应为 GREEN（修复后）
- CreateDevice("DEVICE001") 成功后，CreateDevice("device001") 应返回 "device already exists" 错误
- GetDeviceByDeviceID 使用任意大小写变体（"device001"、"DEVICE001"、"Device001"）均应查到同一设备
- RegisterDevice 与 CreateDevice 对大小写的处理行为一致
- BatchCreateDevices 与 CreateDevice 对大小写的处理行为一致
- go build ./... 全部通过
- go vet ./... 无报错
- go test -c -o /dev/null . 编译通过

## 6. 触发频率（Frequency）
必现（100%）：只要使用 CreateDevice 接口以不同大小写的 ID 创建设备，必定产生重复；查询接口使用混合大小写查询必定失败。

## 7. 影响范围（Impact / Scope）
- 设备数据重复：同一物理设备可能被多次录入系统，导致设备数量统计失真
- 数据不一致：不同接口对设备ID的唯一性判定标准不同，造成数据混乱
- 查询不可靠：用户可能因输入大小写不同而无法查到已存在的设备
- 批量操作风险：BatchCreateDevices 与单次 CreateDevice 行为不一致，可能导致批量导入时绕过重复检查
- 运维困难：设备索引条目混乱，部分查询路径失效，增加运维排障难度

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：
- 统一使用 RegisterDevice 接口代替 CreateDevice 接口进行设备注册，该接口能正确处理大小写不敏感的重复检查
- 在调用 CreateDevice 前，手动将设备ID转换为统一大小写（如全大写或全小写）
- 查询时确保使用与创建时完全一致的设备ID大小写
