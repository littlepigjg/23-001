package store

import (
	"context"
	"sync"
	"testing"

	"fwupgrade/internal/model"
)

// newConcurrentTestStore 构造带预置记录的内存存储，用于并发读取测试。
func newConcurrentTestStore() *MemoryStore {
	s := NewMemoryStore()
	for i := 0; i < 50; i++ {
		devID := "device-A"
		if i%2 == 0 {
			devID = "device-B"
		}
		rec := &model.UpgradeRecord{
			ID:       model.ID(i + 1),
			DeviceID: devID,
			Status:   model.UpgradeSuccess,
		}
		if i%3 == 0 {
			rec.Status = model.UpgradeFailed
		}
		s.records[model.ID(i+1)] = rec
	}
	s.idCounter = 50
	return s
}

// TestListRecordsByDevice_NoSharedBufferCorruption 并发调用多个 List 方法，
// 确保返回的 slice 不复用共享底层数组、数据不互相串流。
func TestListRecordsByDevice_NoSharedBufferCorruption(t *testing.T) {
	s := newConcurrentTestStore()
	ctx := context.Background()

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for g := 0; g < goroutines; g++ {
		// 读路径一：获取最近记录
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				recs, err := s.GetRecentRecords(ctx, 10)
				if err != nil {
					t.Errorf("GetRecentRecords: %v", err)
					return
				}
				if len(recs) > 10 {
					t.Errorf("GetRecentRecords returned %d, want <= 10", len(recs))
				}
				// 返回的每条记录 DeviceID 必须是预置的合法值
				for _, r := range recs {
					if r.DeviceID != "device-A" && r.DeviceID != "device-B" {
						t.Errorf("corrupted DeviceID after GetRecent: %q", r.DeviceID)
						return
					}
				}
			}
		}()

		// 读路径二：按设备查询历史
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				recs, err := s.ListRecordsByDevice(ctx, "device-A")
				if err != nil {
					t.Errorf("ListRecordsByDevice: %v", err)
					return
				}
				for _, r := range recs {
					if r.DeviceID != "device-A" {
						t.Errorf("corrupted DeviceID in ListRecordsByDevice: %q", r.DeviceID)
						return
					}
				}
			}
		}()

		// 读路径三：全量列表分页
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				recs, _, err := s.ListRecords(ctx, 1, 20, "")
				if err != nil {
					t.Errorf("ListRecords: %v", err)
					return
				}
				for _, r := range recs {
					if r.DeviceID != "device-A" && r.DeviceID != "device-B" {
						t.Errorf("corrupted DeviceID in ListRecords: %q", r.DeviceID)
						return
					}
				}
			}
		}()
	}

	wg.Wait()
}

// TestListRecordsByDevice_BoundaryNoCorruption 验证 ListRecordsByDevice 只返回该设备记录，
// 且切片长度稳定（device-A 共 25 条），不受并发 GetRecentRecords 影响。
func TestListRecordsByDevice_BoundaryNoCorruption(t *testing.T) {
	s := newConcurrentTestStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			if _, err := s.GetRecentRecords(ctx, 5); err != nil {
				t.Errorf("GetRecentRecords: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			recs, err := s.ListRecordsByDevice(ctx, "device-B")
			if err != nil {
				t.Errorf("ListRecordsByDevice: %v", err)
				return
			}
			if len(recs) != 25 {
				t.Errorf("device-B count = %d, want 25", len(recs))
				return
			}
			for _, r := range recs {
				if r.DeviceID != "device-B" {
					t.Errorf("corrupted DeviceID in device-B list: %q", r.DeviceID)
					return
				}
			}
		}
	}()

	wg.Wait()
}
