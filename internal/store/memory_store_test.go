package store

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"fwupgrade/internal/model"
)

// TestGetFirmwareByID_NotFound 锁定契约：未命中时返回 (nil, err) 且 err 包装 ErrNotFound。
func TestGetFirmwareByID_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	fw, err := s.GetFirmwareByID(ctx, model.ID(999999))
	if fw != nil {
		t.Fatalf("expected nil firmware for missing id, got %+v", fw)
	}
	if err == nil {
		t.Fatal("expected error for missing firmware, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected error to wrap ErrNotFound, got %v", err)
	}
}

// TestGetFirmwareByVersion_NotFound 锁定契约：未命中时返回 (nil, err) 且 err 包装 ErrNotFound。
func TestGetFirmwareByVersion_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	fw, err := s.GetFirmwareByVersion(ctx, model.ID(1), "v9.9.9")
	if fw != nil {
		t.Fatalf("expected nil firmware for missing version, got %+v", fw)
	}
	if err == nil {
		t.Fatal("expected error for missing firmware version, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected error to wrap ErrNotFound, got %v", err)
	}
}

// TestGetDeviceByID_NotFound 锁定契约：未命中时返回 (nil, err) 且 err 包装 ErrNotFound。
func TestGetDeviceByID_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	d, err := s.GetDeviceByID(ctx, model.ID(999999))
	if d != nil {
		t.Fatalf("expected nil device for missing id, got %+v", d)
	}
	if err == nil {
		t.Fatal("expected error for missing device, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected error to wrap ErrNotFound, got %v", err)
	}
}

// TestGetDeviceByDeviceID_NotFound 锁定契约：未命中时返回 (nil, err) 且 err 包装 ErrNotFound。
func TestGetDeviceByDeviceID_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	d, err := s.GetDeviceByDeviceID(ctx, "no-such-device")
	if d != nil {
		t.Fatalf("expected nil device for missing device_id, got %+v", d)
	}
	if err == nil {
		t.Fatal("expected error for missing device, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected error to wrap ErrNotFound, got %v", err)
	}
}

// TestDeleteMissing_ReturnsNotFound 锁定删除不存在记录时的契约：
// 固件与设备删除未命中时都应返回包装了 ErrNotFound 的错误（驱动 handler 返回 404）。
func TestDeleteMissing_ReturnsNotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.DeleteFirmware(ctx, model.ID(999999)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteFirmware missing: expected ErrNotFound, got %v", err)
	}
	if err := s.DeleteDevice(ctx, model.ID(999999)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteDevice missing: expected ErrNotFound, got %v", err)
	}
}

// TestIsNotFound 断言哨兵判定：哨兵错误（含其包装）判为 true，无关错误判为 false。
func TestIsNotFound(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Error("IsNotFound(ErrNotFound) should be true")
	}
	if !IsNotFound(fmt.Errorf("%w: id=1", ErrNotFound)) {
		t.Error("IsNotFound on wrapped ErrNotFound should be true")
	}
	if IsNotFound(errors.New("some other error")) {
		t.Error("IsNotFound on unrelated error should be false")
	}
}
