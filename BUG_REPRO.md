# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
固件升级管理系统在分页查询功能上存在一个会导致服务崩溃的缺陷：当 API 调用方传入 page=0（页码为零）或者 page 参数值远超实际数据页数时，服务会因为切片越界而 panic 崩溃。该问题影响所有涉及分页的列表查询接口，包括固件列表、升级历史记录、设备列表、任务列表和设备型号列表。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go1.24.x（以 go.mod 声明为准）
- 项目模块：fwupgrade
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：与并发无关，任何 CPU 均可稳定复现

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 创建一个内存存储实例，向其中写入少量测试数据（如 3 条固件记录）
3. 通过服务层调用分页列表接口，传入 page=0、pageSize=10
4. 观察接口返回结果或日志输出
5. 同样的步骤下，将 page 参数改为远超实际页数的值（如 page=100），重复步骤 3-4

## 4. 实际结果（Actual Behavior / Observed Output）
- 调用分页接口传入 page=0 时，服务 panic 崩溃，错误信息：
  ```
  runtime error: slice bounds out of range [-10:]
  ```
- 调用分页接口传入 page=100（远超实际页数）时，同样 panic 崩溃，错误信息：
  ```
  runtime error: slice bounds out of range [N:M]
  ```
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 正常页码（page=1、page=2 等）查询功能正常
- 无数据竞争（非并发缺陷）

## 5. 期望结果（Expected Behavior）
- 传入 page=0 时，应被自动规范化为 page=1，返回首页数据
- 传入远超实际页数的 page 时，应返回空结果列表，而非 panic
- RED/GREEN 判定结果应为 GREEN
- 正常分页查询行为不受影响
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）。只要传入 page=0 或超过实际页数的 page 参数，必定触发 panic，无需重复或并发。

## 7. 影响范围（Impact / Scope）
- 所有分页列表接口均受影响（固件列表、升级记录、设备列表、任务列表、型号列表）
- 服务 panic 导致请求失败，用户无法获取任何数据
- 如果服务框架未正确恢复 panic，可能导致整个服务实例崩溃
- 恶意用户可通过构造 page=0 请求轻易触发服务不可用
- 影响所有依赖分页功能的前端页面和 API 调用方

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：在调用分页接口前，确保 page 参数值 >= 1，且不超过预期页数。前端应对用户输入的页码进行校验，禁止传入 page=0 或负数。
