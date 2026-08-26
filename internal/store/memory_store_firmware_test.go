package store

import (
	"context"
	"testing"
	"time"

	"fwupgrade/internal/model"
)

// TestCreateFirmware_DuplicateVersionDoesNotSkipID 验证上传重复版本的固件时，
// 自增 ID 不应被浪费，后续成功上传的固件 ID 应当连续。
func TestCreateFirmware_DuplicateVersionDoesNotSkipID(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	release := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 首个固件 -> ID 1
	fw1 := model.NewFirmware(1, "M1", "1.0.0", "md5-1", 100, "/tmp/f1.bin", release, "")
	if err := s.CreateFirmware(ctx, fw1); err != nil {
		t.Fatalf("create fw1: %v", err)
	}
	if fw1.ID != 1 {
		t.Fatalf("fw1 ID = %d, want 1", fw1.ID)
	}

	// 重复版本 -> 应失败
	dup := model.NewFirmware(1, "M1", "1.0.0", "md5-dup", 100, "/tmp/fdup.bin", release, "")
	if err := s.CreateFirmware(ctx, dup); err == nil {
		t.Fatal("expected duplicate version error, got nil")
	}

	// 下一个成功的固件 ID 应为 2，而不是被跳过变成 3
	fw2 := model.NewFirmware(1, "M1", "2.0.0", "md5-2", 100, "/tmp/f2.bin", release, "")
	if err := s.CreateFirmware(ctx, fw2); err != nil {
		t.Fatalf("create fw2: %v", err)
	}
	if fw2.ID != 2 {
		t.Fatalf("fw2 ID = %d, want 2 (duplicate must not skip ID)", fw2.ID)
	}
}
