# 缺陷复现报告

## 缺陷概述
- **缺陷ID**: fwupgrade-concur-006
- **缺陷类型**: concurrency
- **缺陷描述**: 文件存储层 saveToFile 方法在未获取 memStore 读锁的情况下遍历 map 字段收集数据快照，与并发写操作产生竞态条件，导致数据不一致或 panic

## 复现环境
- **操作系统**: Linux
- **Go 版本**: 1.22.0
- **项目路径**: /home/admin/code/23/001/23-001-6

## 复现步骤

### 1. 编译检查
```bash
go build ./...
```

### 2. 运行红绿色测试
```bash
go test -race -count=20 -timeout=30s ./internal/store/ -run '^TestConcurrentSaveToFile$'
```

### 3. 预期结果（缺陷存在时）
- 测试输出 `RED（红灯，缺陷未修复）`
- 检测到 `DATA RACE` 警告
- Panic guard 触发次数 > 0
- 可能出现 `data consistency check failed` 错误

## 缺陷位置
- **文件1**: internal/store/file_store.go
  - saveToFile 方法（第118行）：收集数据时未持有 memStore 读锁
  - autoSave 方法（第90行）：仅使用 RLock 保护 dirty 标志

- **文件2**: internal/store/memory_store.go
  - CreateModel/UpdateModel/DeleteModel 等方法：持有写锁修改 map 数据

## 竞态条件分析

### 时序图
1. Goroutine A: autoSave 触发 → saveToFile 开始遍历 memStore.models
2. Goroutine B: CreateModel 调用 → 获取 memStore 写锁 → 修改 models map
3. 结果: Go runtime 检测到 concurrent map read and write → panic

### 竞态窗口
- saveToFile 的数据收集循环
- CreateModel/CreateDevice 的写操作
- autoSave 的 ticker 轮询

## 验证结果
```
go test -race -count=5 -timeout=30s ./internal/store/ -run '^TestConcurrentSaveToFile$'

WARNING: DATA RACE
Write at 0x... by goroutine 7: SetPanicGuard()
Previous read at 0x... by goroutine 8: saveToFile()

TestConcurrentSaveToFile: RED（红灯，缺陷未修复）
Panic guard 触发次数: 76
操作错误次数: 0
```

## 修复后预期
- 测试输出 `GREEN（绿灯，缺陷已修复）`
- 无 DATA RACE 警告
- 所有并发操作正确持久化
- 型号数量与预期一致（>=30）