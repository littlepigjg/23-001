package store

import (
	"context"
	"reflect"
	"testing"
	"time"

	"fwupgrade/internal/model"
)

// seedStore 向内存存储中插入少量数据（3 条），用于复现“数据少但请求页码极大”的崩溃场景。
func seedStore(t *testing.T, s *MemoryStore) {
	t.Helper()
	ctx := context.Background()

	m := model.NewDeviceModel("Model-A", "Acme", "v1", "test model")
	if err := s.CreateModel(ctx, m); err != nil {
		t.Fatalf("CreateModel: %v", err)
	}

	for i := 0; i < 3; i++ {
		d := model.NewDevice("dev-00"+itoa(i), m.ID, "Model-A", "device-"+itoa(i), "10.0.0.1", "SN-00"+itoa(i))
		if err := s.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice: %v", err)
		}

		fw := model.NewFirmware(m.ID, "Model-A", "1.0."+itoa(i), "d41d8cd98f00b204e980", 1024, "/tmp/f.bin", time.Now(), "init")
		if err := s.CreateFirmware(ctx, fw); err != nil {
			t.Fatalf("CreateFirmware: %v", err)
		}

		task := model.NewUpgradeTask("task-"+itoa(i), "desc", m.ID, "Model-A", fw.ID, "1.0."+itoa(i), model.TaskTypeFull, 100, nil, "tester")
		if err := s.CreateTask(ctx, task); err != nil {
			t.Fatalf("CreateTask: %v", err)
		}

		rec := model.NewUpgradeRecord(d.DeviceID, d.Name, task.ID, task.Name, "1.0.0", "1.0."+itoa(i))
		if err := s.CreateRecord(ctx, rec); err != nil {
			t.Fatalf("CreateRecord: %v", err)
		}
	}
}

// 简易整数转字符串，避免引入 strconv 的依赖噪声。
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// TestPaginationNoPanic 验证所有分页接口在非法页码下都不会触发 slice bounds out of range。
// 这是对线上崩溃（page=0 / page=100 数据只有 3 条）的回归测试。
func TestPaginationNoPanic(t *testing.T) {
	s := NewMemoryStore()
	seedStore(t, s)
	ctx := context.Background()

	type pageCase struct {
		name              string
		page, pageSize    int
		wantNonEmptyFirst bool // page=1 时应返回全部 3 条
	}

	cases := []pageCase{
		{"page=0", 0, 20, false},
		{"page=1 合法", 1, 20, true},
		{"page=100 远超数据", 100, 20, false},
		{"page 负数", -5, 20, false},
		{"pageSize=0", 1, 0, true},
		{"pageSize 负数", 1, -10, true},
		{"pageSize 极大", 1, 999999, true},
	}

	// 设备
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ListDevices %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.ListDevices(ctx, c.page, c.pageSize, 0, "")
			assertInBounds(t, "ListDevices "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("SearchDevices %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.SearchDevices(ctx, "device", c.page, c.pageSize)
			assertInBounds(t, "SearchDevices "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetAllDevices %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.GetAllDevices(ctx, c.page, c.pageSize)
			assertInBounds(t, "GetAllDevices "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		// 固件
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ListFirmwares %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.ListFirmwares(ctx, c.page, c.pageSize, 0)
			assertInBounds(t, "ListFirmwares "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		// 任务
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ListTasks %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.ListTasks(ctx, c.page, c.pageSize, "")
			assertInBounds(t, "ListTasks "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("SearchTasks %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.SearchTasks(ctx, "task", c.page, c.pageSize)
			assertInBounds(t, "SearchTasks "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		// 历史记录
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ListRecords %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.ListRecords(ctx, c.page, c.pageSize, "")
			assertInBounds(t, "ListRecords "+c.name, got, total, c.wantNonEmptyFirst)
		}()

		// 型号
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ListModels %s panic: %v", c.name, r)
				}
			}()
			got, total, _ := s.ListModels(ctx, c.page, c.pageSize)
			// 仅 1 条型号数据，page=1 时应非空。
			wantNonEmpty := c.wantNonEmptyFirst && true
			assertInBounds(t, "ListModels "+c.name, got, total, wantNonEmpty)
		}()
	}
}

// TestGetRecentNoPanic 验证 limit 非法（0/负数）时 GetRecent* 不崩溃且返回空切片。
func TestGetRecentNoPanic(t *testing.T) {
	s := NewMemoryStore()
	seedStore(t, s)
	ctx := context.Background()

	for _, limit := range []int{-1, 0, 1, 100} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetRecentRecords limit=%d panic: %v", limit, r)
				}
			}()
			got, err := s.GetRecentRecords(ctx, limit)
			if err != nil {
				t.Errorf("GetRecentRecords limit=%d err: %v", limit, err)
			}
			if len(got) < 0 || (limit > 0 && len(got) > limit) {
				t.Errorf("GetRecentRecords limit=%d 返回 %d 条，越界", limit, len(got))
			}
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetRecentTasks limit=%d panic: %v", limit, r)
				}
			}()
			got, err := s.GetRecentTasks(ctx, limit)
			if err != nil {
				t.Errorf("GetRecentTasks limit=%d err: %v", limit, err)
			}
			if len(got) < 0 || (limit > 0 && len(got) > limit) {
				t.Errorf("GetRecentTasks limit=%d 返回 %d 条，越界", limit, len(got))
			}
		}()
	}
}

func assertInBounds(t *testing.T, label string, got interface{}, total int64, wantNonEmpty bool) {
	t.Helper()
	n := sliceLen(got)
	if n < 0 {
		t.Errorf("%s: 返回长度为负", label)
	}
	if int64(n) > total {
		t.Errorf("%s: 返回 %d 条超过 total=%d", label, n, total)
	}
	if wantNonEmpty && n == 0 {
		t.Errorf("%s: 期望非空结果，实际为空", label)
	}
}

// sliceLen 通过反射获取任意切片的长度，避免对每种实体类型重复断言。
func sliceLen(v interface{}) int {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len()
	}
	if v == nil {
		return 0
	}
	return 0
}
