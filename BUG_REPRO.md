# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

固件升级系统在高并发场景下出现配置偶发不一致问题。当多个 goroutine 并发执行配置热加载与固件/灰度配置读取操作时，读取到的配置快照中各字段（max_file_size、require_md5、upload_dir 等）来自不同的配置版本，导致配置一致性校验失败、业务逻辑判断错误。

## 2. 环境信息（Environment）

- 操作系统：Linux (amd64)
- Go 版本：go1.21.x
- 项目模块：fwupgrade
- 运行参数：go test -race -count=3 -run '^TestRedGreen$' .
- 并发数量：8 个配置重载 goroutine + 8 个服务读取 goroutine，持续 3 秒
- CPU 核数：多核（并发缺陷与核数正相关）

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go vet ./...` 确保静态检查无报错
3. 执行 `go test -c -o /dev/null .` 确保测试编译通过
4. 执行 `go test -race -count=3 -run '^TestRedGreen$' .` 运行验证测试
5. 观察测试输出中的 RED/GREEN 判定结果和 DATA RACE 警告

## 4. 实际结果（Actual Behavior / Observed Output）

- 3 次运行均 FAIL，判定为 RED（红灯，检测到缺陷）
- 检测到 463362~549647 次配置不一致读写
- 多次报告 `WARNING: DATA RACE`，涉及配置字段的并发读写冲突
- 典型不一致输出：
  ```
  当前配置: max_size=52428800, require_md5=true, upload_dir=/tmp/.../uploads1 - 来自不同配置源，违反一致性
  ```
  其中 max_size=52428800 对应配置 B，而 upload_dir 包含 uploads1 对应配置 A，表明一次读取操作中各字段来自不同配置版本
- 退出码非零（1）

## 5. 期望结果（Expected Behavior）

- 无 DATA RACE 警告（go test -race 无输出）
- 判定为 GREEN（绿灯，缺陷已修复）
- 配置快照中所有字段来自同一版本，无混合配置
- 并发压测下所有配置读写操作正确完成，配置一致性校验通过
- go build ./... 与 go vet ./... 全部通过

## 6. 触发频率（Frequency）

- 必现（100%）：在 8+8 并发 goroutine 持续 3 秒的场景下，每次运行均能复现
- 每次运行检测到数十万次配置不一致
- 在低并发或单线程场景下不会触发

## 7. 影响范围（Impact / Scope）

- 配置热加载与读取并发场景下，配置字段不一致导致固件上传/灰度决策逻辑判断错误
- 可能导致固件被错误拒绝（max_file_size 校验使用了错误版本）
- 可能导致灰度比例计算错误（min/max ratio 来自不同配置版本）
- 可能导致上传路径错误（upload_dir 指向旧配置目录）
- 线上高并发场景下可能出现间歇性业务异常，难以排查

## 8. 附加说明（Additional Notes / Workaround）

- 临时规避方法：在低并发或单 goroutine 环境下运行，或在配置加载完成后再开始读取操作
- 根本解决方案：确保配置读写的原子性，在读取和写入配置字段时都使用适当的同步机制
