package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
)

// TestSentinelErrors_Distinct 验证两个哨兵错误互不相同且非 nil，
// 防止后续重构把它们合并或改成同一变量。
func TestSentinelErrors_Distinct(t *testing.T) {
	if ErrNotFound == nil || ErrConflict == nil {
		t.Fatal("sentinel errors must be non-nil")
	}
	if errors.Is(ErrNotFound, ErrConflict) || errors.Is(ErrConflict, ErrNotFound) {
		t.Fatal("ErrNotFound and ErrConflict must be distinct")
	}
}

// TestMemoryStore_ModelNotFound 验证按 id/name 查询不存在的型号时，
// 错误可被 errors.Is(err, ErrNotFound) 识别。
func TestMemoryStore_ModelNotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if _, err := s.GetModelByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetModelByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
	if _, err := s.GetModelByName(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetModelByName: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
}

// TestMemoryStore_ModelCreateConflict 验证重复创建同名型号返回 ErrConflict。
func TestMemoryStore_ModelCreateConflict(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	m1 := model.NewDeviceModel("A", "vendor", "1.0", "")
	if err := s.CreateModel(ctx, m1); err != nil {
		t.Fatalf("first CreateModel: %v", err)
	}
	m2 := model.NewDeviceModel("A", "vendor", "1.0", "")
	if err := s.CreateModel(ctx, m2); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate CreateModel: errors.Is(err, ErrConflict) = false; err=%v", err)
	}
}

// TestMemoryStore_FirmwareNotFoundAndConflict 验证固件的 not-found 与版本重复冲突。
func TestMemoryStore_FirmwareNotFoundAndConflict(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	// 先建一个型号，CreateFirmware 的版本索引不依赖型号存在，但为了贴近真实用法先建。
	m := model.NewDeviceModel("M", "vendor", "1.0", "")
	_ = s.CreateModel(ctx, m)

	if _, err := s.GetFirmwareByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetFirmwareByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}

	fw := model.NewFirmware(m.ID, m.Name, "1.0.0", "md5", 10, "/tmp/fw.bin", time.Now(), "")
	if err := s.CreateFirmware(ctx, fw); err != nil {
		t.Fatalf("first CreateFirmware: %v", err)
	}
	dup := model.NewFirmware(m.ID, m.Name, "1.0.0", "md5", 10, "/tmp/fw.bin", time.Now(), "")
	if err := s.CreateFirmware(ctx, dup); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate CreateFirmware: errors.Is(err, ErrConflict) = false; err=%v", err)
	}
}

// TestMemoryStore_DeviceNotFound 验证设备 not-found。
func TestMemoryStore_DeviceNotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if _, err := s.GetDeviceByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetDeviceByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
	if _, err := s.GetDeviceByDeviceID(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetDeviceByDeviceID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
}

// TestMemoryStore_TaskAndRecordNotFound 验证任务/记录 not-found。
func TestMemoryStore_TaskAndRecordNotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if _, err := s.GetTaskByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetTaskByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
	if _, err := s.GetRecordByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetRecordByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
}

// TestFileStore_DelegatesSentinels 验证 FileStore 透传 MemoryStore 的哨兵错误
// （FileStore 所有方法委托给内部 MemoryStore，错误链不应断）。
func TestFileStore_DelegatesSentinels(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = t.TempDir()
	s := NewFileStore(cfg)
	ctx := context.Background()

	if _, err := s.GetModelByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("FileStore.GetModelByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
	if _, err := s.GetFirmwareByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("FileStore.GetFirmwareByID: errors.Is(err, ErrNotFound) = false; err=%v", err)
	}
}
