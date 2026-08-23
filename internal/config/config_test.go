package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFileToTest 是测试用的临时文件写入辅助。
func writeFileToTest(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestLoadFromFile_NotFound_PreservesErrorChain 验证对不存在的配置文件路径，
// 错误链同时携带 ErrConfigFileNotFound 与 os.ErrNotExist。
// 这正是用户反馈"字符串在但 errors.Is 返回 false"的场景，用测试锁定其正确行为，
// 防止回归（例如误把 %w 改成 %s/v、或换了不支持双 %w 的写法）。
func TestLoadFromFile_NotFound_PreservesErrorChain(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "definitely_does_not_exist.cfg")

	_, err := LoadFromFile(missing)
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}

	if !errors.Is(err, ErrConfigFileNotFound) {
		t.Errorf("errors.Is(err, ErrConfigFileNotFound) = false; err=%v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("errors.Is(err, os.ErrNotExist) = false; err=%v", err)
	}
}

// TestLoadFromFile_InvalidFormat_PreservesSentinel 验证格式错误的配置行
// 返回的 err 仍可被 errors.Is 识别为 ErrConfigInvalidFormat。
func TestLoadFromFile_InvalidFormat_PreservesSentinel(t *testing.T) {
	dir := t.TempDir()
	path := writeFileToTest(t, dir, "bad.cfg", "this line has no equals sign\n")

	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for malformed config file, got nil")
	}
	if !errors.Is(err, ErrConfigInvalidFormat) {
		t.Errorf("errors.Is(err, ErrConfigInvalidFormat) = false; err=%v", err)
	}
}

// TestLoadWithFallback_NotFound_PreservesErrorChain 验证经 LoadWithFallback 二次包裹后
// 两个哨兵错误仍可被识别（双重 %w 包裹在整条链上都成立）。
func TestLoadWithFallback_NotFound_PreservesErrorChain(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.cfg")

	_, err := LoadWithFallback(missing)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigFileNotFound) {
		t.Errorf("errors.Is(err, ErrConfigFileNotFound) = false; err=%v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("errors.Is(err, os.ErrNotExist) = false; err=%v", err)
	}
}

// TestTryLoadConfig_NotFound_PreservesErrorChain 验证 TryLoadConfig 的
// "all config paths failed" 包裹同样保持错误链。
func TestTryLoadConfig_NotFound_PreservesErrorChain(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.cfg")

	_, err := TryLoadConfig(missing)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigFileNotFound) {
		t.Errorf("errors.Is(err, ErrConfigFileNotFound) = false; err=%v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("errors.Is(err, os.ErrNotExist) = false; err=%v", err)
	}
}

// TestLoadConfigAndValidate_NotFound_PreservesErrorChain 验证严格加载入口
// 对不存在文件保持错误链。
func TestLoadConfigAndValidate_NotFound_PreservesErrorChain(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.cfg")

	_, err := LoadConfigAndValidate(missing)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigFileNotFound) {
		t.Errorf("errors.Is(err, ErrConfigFileNotFound) = false; err=%v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("errors.Is(err, os.ErrNotExist) = false; err=%v", err)
	}
}

// TestLoadConfigAndValidate_ValidationFailure_PreservesSentinel 验证 LoadConfigAndValidate
// 在配置值非法（如 max_file_size<=0）时返回 ErrConfigValidationFailed。
// 这覆盖了原先"%w 包裹 nil err"的 bug：之前这些分支返回的是不包裹任何东西的伪错误，
// 现在应能被 errors.Is 识别。
func TestLoadConfigAndValidate_ValidationFailure_PreservesSentinel(t *testing.T) {
	dir := t.TempDir()
	// 合法格式但 max_file_size 非法（<=0），触发严格校验失败分支。
	path := writeFileToTest(t, dir, "badmax.cfg", "firmware.max_file_size=0\n")

	_, err := LoadConfigAndValidate(path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, ErrConfigValidationFailed) {
		t.Errorf("errors.Is(err, ErrConfigValidationFailed) = false; err=%v", err)
	}
}
