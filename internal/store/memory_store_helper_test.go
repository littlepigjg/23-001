package store

import (
	"context"
	"testing"

	"fwupgrade/internal/model"
)

// seedModels 插入若干型号用于测试。
// 注意顺序刻意让各厂商的型号在缓存里交错排列（按 ID 升序）：
// [ThinkPad(Lenovo), MacBook(Apple), Yoga(Lenovo), OptiPlex(Dell), IdeaPad(Lenovo)]
// 这样按 Lenovo 过滤得到 [ThinkPad, Yoga, IdeaPad]，写入缓存槽位 0,1,2，
// 覆盖掉原本槽位 1 的 MacBook。旧实现下列表会变成
// [ThinkPad, Yoga, IdeaPad, OptiPlex, IdeaPad] —— MacBook 消失、IdeaPad 重复，
// 正是用户报告的“有的型号重复出现、有的型号不见”。
func seedModels(t *testing.T, s *MemoryStore) {
	t.Helper()
	mfgs := []struct {
		name, manufacturer string
	}{
		{"ThinkPad", "Lenovo"},
		{"MacBook", "Apple"},
		{"Yoga", "Lenovo"},
		{"OptiPlex", "Dell"},
		{"IdeaPad", "Lenovo"},
	}
	for _, m := range mfgs {
		dm := model.NewDeviceModel(m.name, m.manufacturer, "1.0", "desc")
		if err := s.CreateModel(context.Background(), dm); err != nil {
			t.Fatalf("CreateModel(%s): %v", m.name, err)
		}
	}
}

// TestListModelsNotPollutedByFilter 复现用户报告的故障链：
//   1. 先调型号列表接口 —— 数据正常
//   2. 再调按厂商筛选接口（BuildModelList）
//   3. 回过头再调型号列表接口 —— 仍应数据正常
//
// 旧实现里 ListModels 返回缓存底层数组的子切片，BuildModelList 用
// models[:0]+append 原地过滤会改写缓存，导致第 3 步出现重复/缺失。
func TestListModelsNotPollutedByFilter(t *testing.T) {
	s := NewMemoryStore()
	seedModels(t, s)
	ctx := context.Background()

	// 第 1 步：型号列表
	list1, total1, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("ListModels step1: %v", err)
	}
	if total1 != 5 || len(list1) != 5 {
		t.Fatalf("step1: want 5 models, got total=%d len=%d", total1, len(list1))
	}

	// 第 2 步：按厂商筛选（旧实现会在此污染缓存）
	filtered, _, err := s.BuildModelList(ctx, "Lenovo", false)
	if err != nil {
		t.Fatalf("BuildModelList: %v", err)
	}
	if len(filtered) != 3 {
		t.Fatalf("filtered Lenovo: want 3, got %d", len(filtered))
	}
	for _, m := range filtered {
		if m.Manufacturer != "Lenovo" {
			t.Fatalf("filtered returned non-Lenovo: %+v", m)
		}
	}

	// 第 3 步：再次型号列表 —— 必须仍是完整、无重复的 5 条
	list2, total2, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("ListModels step3: %v", err)
	}
	if total2 != 5 || len(list2) != 5 {
		t.Fatalf("step3: want 5 models, got total=%d len=%d (cache polluted?)", total2, len(list2))
	}

	// 不得出现任何 __ 内部标记项
	seen := make(map[string]bool, len(list2))
	for _, m := range list2 {
		if len(m.Name) > 0 && m.Name[:2] == "__" {
			t.Fatalf("internal metadata leaked into list: %q", m.Name)
		}
		if seen[m.Name] {
			t.Fatalf("duplicate model in list: %q", m.Name)
		}
		seen[m.Name] = true
	}

	// 必须仍是原来的 5 个型号（旧实现会把过滤掉的项原地覆盖，
	// 导致某些型号消失、另一些重复）
	want := map[string]bool{
		"ThinkPad": true, "Yoga": true, "IdeaPad": true,
		"MacBook": true, "OptiPlex": true,
	}
	if len(seen) != len(want) {
		t.Fatalf("model set changed after filter: got %v, want %v", seen, want)
	}
	for name := range seen {
		if !want[name] {
			t.Fatalf("unexpected model in list: %q (set drifted: %v)", name, seen)
		}
	}
}

// TestGetModelStatsManufacturerCount 验证厂商型号计数准确。
// 旧实现读到被污染的缓存时，Lenovo 会被计成 0 或重复计数。
func TestGetModelStatsManufacturerCount(t *testing.T) {
	s := NewMemoryStore()
	seedModels(t, s)
	ctx := context.Background()

	// 先触发一次按厂商筛选，模拟用户操作链
	if _, _, err := s.BuildModelList(ctx, "Apple", false); err != nil {
		t.Fatalf("BuildModelList: %v", err)
	}

	counts, _, err := s.GetModelStats(ctx)
	if err != nil {
		t.Fatalf("GetModelStats: %v", err)
	}

	if got := counts["Lenovo"]; got != 3 {
		t.Fatalf("Lenovo count: want 3, got %d", got)
	}
	if got := counts["Apple"]; got != 1 {
		t.Fatalf("Apple count: want 1, got %d", got)
	}
	if got := counts["Dell"]; got != 1 {
		t.Fatalf("Dell count: want 1, got %d", got)
	}
}

// TestExportActiveModelsNoMetadataLeak 验证导出结果不含 __export_metadata__。
func TestExportActiveModelsNoMetadataLeak(t *testing.T) {
	s := NewMemoryStore()
	seedModels(t, s)

	exported, err := s.ExportActiveModels(context.Background())
	if err != nil {
		t.Fatalf("ExportActiveModels: %v", err)
	}
	for _, m := range exported {
		if len(m.Name) >= 2 && m.Name[:2] == "__" {
			t.Fatalf("export leaked internal metadata: %q", m.Name)
		}
	}
	if len(exported) != 5 {
		t.Fatalf("exported active models: want 5, got %d", len(exported))
	}
}

// TestFilterModelsByDescNoMetadataLeak 验证按描述筛选结果不含 __filter_metadata__，
// 且不会污染后续型号列表。
func TestFilterModelsByDescNoMetadataLeak(t *testing.T) {
	s := NewMemoryStore()
	seedModels(t, s)
	ctx := context.Background()

	if _, err := s.FilterModelsByDesc(ctx, []string{"desc"}); err != nil {
		t.Fatalf("FilterModelsByDesc: %v", err)
	}

	// 筛选后型号列表仍应完整
	list, total, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("ListModels after filter: %v", err)
	}
	if total != 5 || len(list) != 5 {
		t.Fatalf("list after filter: want 5, got total=%d len=%d", total, len(list))
	}
	for _, m := range list {
		if len(m.Name) >= 2 && m.Name[:2] == "__" {
			t.Fatalf("filter metadata leaked into list: %q", m.Name)
		}
	}
}
