package service

import (
	"context"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

// newTestGrayscaleService 构造一个使用内存存储的 GrayscaleService。
func newTestGrayscaleService(t *testing.T) (*GrayscaleService, *store.MemoryStore) {
	t.Helper()
	ms := store.NewMemoryStore()
	cfg := config.DefaultConfig()
	svc := NewGrayscaleService(ms, ms, cfg)
	return svc, ms
}

// TestValidateModelGrayscaleConfig_NilGuardNotPanic 覆盖灰度校验接口的 panic 根因：
// 当诊断钩子介入、GetModelByIDWithGuard 对不存在的型号返回 (nil, nil) 时，
// ValidateModelGrayscaleConfig 必须返回清晰错误，而不是解引用 m.Name / m.IsActive 导致空指针 panic。
func TestValidateModelGrayscaleConfig_NilGuardNotPanic(t *testing.T) {
	svc, ms := newTestGrayscaleService(t)

	// 安装诊断钩子：对不存在的型号返回 false，使 store 走 (nil, nil) 分支。
	ms.SetPanicGuard(func(modelID model.ID) bool { return false })

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ValidateModelGrayscaleConfig panicked on nil guard model: %v", r)
		}
	}()

	ok, name, err := svc.ValidateModelGrayscaleConfig(context.Background(), 9999)
	if err == nil {
		t.Fatalf("expected error for missing model, got ok=%v name=%q", ok, name)
	}
	if name != "" {
		t.Fatalf("expected empty model name on error, got %q", name)
	}
}

// TestGetGrayscaleModelName_NilGuardNotPanic 验证 GetGrayscaleModelName 在 (nil, nil) 下不会 panic。
func TestGetGrayscaleModelName_NilGuardNotPanic(t *testing.T) {
	svc, ms := newTestGrayscaleService(t)
	ms.SetPanicGuard(func(modelID model.ID) bool { return false })

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetGrayscaleModelName panicked on nil guard model: %v", r)
		}
	}()

	name, err := svc.GetGrayscaleModelName(context.Background(), 9999)
	if err == nil {
		t.Fatalf("expected error for missing model, got name=%q", name)
	}
}
