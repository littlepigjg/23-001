# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

固件上传功能存在安全隐患：当用户上传固件时，如果使用包含特殊字符的版本号（如包含路径分隔符或目录遍历序列），系统会错误地接受这些非法版本号，而没有进行有效的格式校验。这可能导致文件被写入非预期的路径位置，存在潜在的路径遍历攻击风险。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：Go 1.22+
- 项目模块：fwupgrade
- 存储类型：memory
- 运行参数：无特殊要求

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录：
   ```bash
   cd /home/admin/code/23/001/23-001-28
   ```

2. 确保项目能正常编译：
   ```bash
   go build ./...
   ```

3. 执行缺陷验证测试：
   ```bash
   go test . -count=1 -run '^TestRedGreen$'
   ```

4. 观察测试输出结果

## 4. 实际结果（Actual Behavior / Observed Output）

测试运行后显示 RED（红灯，缺陷未修复），具体输出如下：

```
=== Test Results ===
PASS: Valid version - Normal semantic version should pass validation
PASS: Valid version with dashes - Version with dashes should pass validation
FAIL: Malicious version with path traversal - Version with path traversal should be rejected (version '1.0.0/../../../etc/passwd' should have been rejected but was accepted)
FAIL: Malicious version with dotslashes - Version with ../ should be rejected (version '../etc/shadow' should have been rejected but was accepted)
FAIL: Version with absolute path - Version with absolute path should be rejected (version '/etc/passwd' should have been rejected but was accepted)
FAIL: ValidateVersionFormat accepted malicious version '1.0.0/../../../etc/passwd'
FAIL: ValidateVersionFormat accepted malicious version '../etc/shadow'
FAIL: ValidateVersionFormat accepted malicious version '/etc/passwd'
FAIL: SanitizeVersion did not remove path traversal characters

RED（红灯，缺陷未修复）
```

退出码为 1，表示测试失败。

## 5. 期望结果（Expected Behavior）

修复后执行相同的测试命令，应该得到以下结果：

```
=== Test Results ===
PASS: Valid version - Normal semantic version should pass validation
PASS: Valid version with dashes - Version with dashes should pass validation
PASS: Malicious version with path traversal - Version with path traversal should be rejected
PASS: Malicious version with dotslashes - Version with ../ should be rejected
PASS: Version with absolute path - Version with absolute path should be rejected
PASS: ValidateVersionFormat rejected malicious version '1.0.0/../../../etc/passwd'
PASS: ValidateVersionFormat rejected malicious version '../etc/shadow'
PASS: ValidateVersionFormat rejected malicious version '/etc/passwd'
PASS: SanitizeVersion removed path traversal characters

GREEN（绿灯，缺陷已修复）
```

退出码为 0，表示测试通过。同时：
- go build ./... 编译通过
- go vet ./... 无警告

## 6. 触发频率（Frequency）

必现（100%）。使用包含路径遍历字符的版本号上传固件时，缺陷每次都能稳定复现。

## 7. 影响范围（Impact / Scope）

- 固件上传接口的安全性
- 版本号数据的完整性
- 文件系统操作的安全性
- 可能导致任意文件写入或路径遍历攻击

## 8. 附加说明（Additional Notes / Workaround）

目前没有临时的解决办法。建议尽快修复此安全缺陷，防止潜在的恶意攻击。
