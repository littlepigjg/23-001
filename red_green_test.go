// Package fwupgrade_test 缺陷注入测试 - Slice 底层数组共享导致数据污染
//
// RED/GREEN 测试说明：
//   RED: 缺陷存在时测试失败（数据污染可观测）
//   GREEN: 缺陷修复后测试通过
//
// 缺陷描述：
//   ListModels() 使用共享缓存 modelCache，返回的 slice 与缓存共享底层数组。
//   当调用方通过 append 或原地过滤修改返回的 slice 时，会污染共享缓存，
//   导致后续 ListModels() 调用返回错误数据。
//
// 根因文件: memory_store.go - ListModels() 返回共享底层数组的子切片
// 触发文件: memory_store_helper.go - BuildModelList(), ExportActiveModels(), FilterModelsByDesc()
// 触发条件: 调用方对返回的 slice 使用 append 或原地过滤（slice[:0] + append）
package fwupgrade_test

import (
	"context"
	"testing"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

// setupTestStore 创建带有测试数据的 MemoryStore
func setupTestStore() *store.MemoryStore {
	s := store.NewMemoryStore()
	ctx := context.Background()

	// 通过公开 API 创建设备型号
	models := []*model.DeviceModel{
		model.NewDeviceModel("Dell R740", "Dell", "2.0", ""),
		model.NewDeviceModel("HPE DL380", "HPE", "3.1", ""),
		model.NewDeviceModel("Lenovo X3650", "Lenovo", "1.5", ""),
		model.NewDeviceModel("Dell R640", "Dell", "1.8", ""),
		model.NewDeviceModel("HPE DL360", "HPE", "4.0", ""),
	}

	// Lenovo 型号设为不活跃
	models[2].IsActive = false

	for _, m := range models {
		if err := s.CreateModel(ctx, m); err != nil {
			panic(err)
		}
	}

	return s
}

// TestRedGreen_OriginalListWorks 验证 ListModels 正常工作
// 这是 GREEN 基线测试 - 验证未被污染的数据是正确的
func TestRedGreen_OriginalListWorks(t *testing.T) {
	s := setupTestStore()
	ctx := context.Background()

	// 第一次调用 ListModels - 应返回所有5个模型
	models, total, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected total=5, got %d", total)
	}
	if len(models) != 5 {
		t.Errorf("Expected 5 models, got %d", len(models))
	}

	// 验证模型按 ID 排序
	for i := 1; i < len(models); i++ {
		if models[i-1].ID > models[i].ID {
			t.Errorf("Models not sorted by ID: %d > %d", models[i-1].ID, models[i].ID)
		}
	}
}

// TestRedGreen_BuildModelListPollutesCache 验证 BuildModelList 污染缓存
// RED: 缺陷存在时，第二次 ListModels 返回被污染的数据
// GREEN: 缺陷修复后，数据不受影响
func TestRedGreen_BuildModelListPollutesCache(t *testing.T) {
	s := setupTestStore()
	ctx := context.Background()

	// 第一次 ListModels - 验证初始数据
	models1, _, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("First ListModels failed: %v", err)
	}
	initialCount := len(models1)
	initialNames := make([]string, len(models1))
	for i, m := range models1 {
		initialNames[i] = m.Name
	}

	// 调用 BuildModelList - 这会通过 append 污染共享缓存
	filtered, _, err := s.BuildModelList(ctx, "Dell", false)
	if err != nil {
		t.Fatalf("BuildModelList failed: %v", err)
	}

	// 验证过滤结果正确
	if len(filtered) < 2 {
		t.Errorf("Expected at least 2 Dell models, got %d", len(filtered))
	}

	// 第二次 ListModels - 如果缺陷存在，这里返回的数据已被污染
	models2, total2, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("Second ListModels failed: %v", err)
	}

	// RED 检查：数据污染导致总数或内容不正确
	if total2 != int64(initialCount) {
		t.Errorf("RED FAIL: Data pollution detected! Expected total=%d, got %d", initialCount, total2)
		t.Log("  Root cause: BuildModelList's append corrupted the shared backing array")
		t.Log("  Location: memory_store.go ListModels() + memory_store_helper.go BuildModelList()")
		t.FailNow()
	}

	// RED 检查：数量是否正确
	if len(models2) != initialCount {
		t.Errorf("RED FAIL: Slice count changed from %d to %d (data pollution)", initialCount, len(models2))
		t.FailNow()
	}

	// RED 检查：内容是否一致
	for i, m := range models2 {
		if m.Name != initialNames[i] {
			t.Errorf("RED FAIL: Model[%d] changed from '%s' to '%s'", i, initialNames[i], m.Name)
			t.FailNow()
		}
	}

	t.Log("GREEN PASS: No data pollution detected")
}

// TestRedGreen_ExportActiveModelsCorruptsStats 验证导出功能污染统计
// RED: 缺陷存在时，GetModelStats 返回错误数据
// GREEN: 缺陷修复后，统计数据正确
func TestRedGreen_ExportActiveModelsCorruptsStats(t *testing.T) {
	s := setupTestStore()
	ctx := context.Background()

	// 记录初始统计数据
	// 有 5 个模型，4 个活跃，1 个不活跃
	// Dell: 2个 (都活跃), HPE: 2个 (都活跃), Lenovo: 1个 (不活跃)
	initialStats, initialRatio, err := s.GetModelStats(ctx)
	if err != nil {
		t.Fatalf("GetModelStats failed: %v", err)
	}
	if initialStats["Dell"] != 2 {
		t.Errorf("Expected 2 Dell models initially, got %d", initialStats["Dell"])
	}
	if initialStats["HPE"] != 2 {
		t.Errorf("Expected 2 HPE models initially, got %d", initialStats["HPE"])
	}
	if initialStats["Lenovo"] != 1 {
		t.Errorf("Expected 1 Lenovo model initially, got %d", initialStats["Lenovo"])
	}
	if initialRatio < 0.75 {
		t.Errorf("Expected active ratio >= 0.75, got %f", initialRatio)
	}

	// 调用 ExportActiveModels - 会先调用 BuildModelList（污染缓存），
	// 然后 append 添加 exportInfo（进一步污染）
	exported, err := s.ExportActiveModels(ctx)
	if err != nil {
		t.Fatalf("ExportActiveModels failed: %v", err)
	}

	// 验证导出结果包含 exportInfo
	hasMarker := false
	for _, m := range exported {
		if m.Name == "__export_metadata__" {
			hasMarker = true
			break
		}
	}
	if !hasMarker {
		t.Error("Export should contain __export_metadata__ marker")
	}

	// 再次获取统计 - 如果缺陷存在，数据已被污染
	stats2, ratio2, err := s.GetModelStats(ctx)
	if err != nil {
		t.Fatalf("Second GetModelStats failed: %v", err)
	}

	// RED 检查：统计数据不应因为导出而改变
	if stats2["Dell"] != initialStats["Dell"] {
		t.Errorf("RED FAIL: Dell count changed from %d to %d after export (data pollution)",
			initialStats["Dell"], stats2["Dell"])
		t.Log("  Root cause: ExportActiveModels append corrupted the shared backing array")
		t.Log("  The __export_metadata__ entry was written into the shared cache")
		t.FailNow()
	}
	if stats2["HPE"] != initialStats["HPE"] {
		t.Errorf("RED FAIL: HPE count changed from %d to %d after export (data pollution)",
			initialStats["HPE"], stats2["HPE"])
		t.FailNow()
	}
	if stats2["Lenovo"] != initialStats["Lenovo"] {
		t.Errorf("RED FAIL: Lenovo count changed from %d to %d after export (data pollution)",
			initialStats["Lenovo"], stats2["Lenovo"])
		t.FailNow()
	}
	if ratio2 != initialRatio {
		t.Errorf("RED FAIL: Active ratio changed from %f to %f (data pollution)",
			initialRatio, ratio2)
		t.FailNow()
	}

	t.Log("GREEN PASS: Export did not corrupt statistics")
}

// TestRedGreen_FilterModelsByDescCorruptsList 验证 FilterModelsByDesc 污染列表
// RED: 缺陷存在时，过滤后的元数据残留在缓存中
// GREEN: 缺陷修复后，后续 ListModels 返回干净数据
func TestRedGreen_FilterModelsByDescCorruptsList(t *testing.T) {
	s := setupTestStore()
	ctx := context.Background()

	// 记录初始 ListModels 结果
	models1, _, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("First ListModels failed: %v", err)
	}
	initialCount := len(models1)

	// 调用 FilterModelsByDesc - 会污染缓存并添加 __filter_metadata__
	filtered, err := s.FilterModelsByDesc(ctx, []string{"server"})
	if err != nil {
		t.Fatalf("FilterModelsByDesc failed: %v", err)
	}

	// 验证过滤结果包含 filter metadata
	hasFilterMeta := false
	for _, m := range filtered {
		if m.Name == "__filter_metadata__" {
			hasFilterMeta = true
			break
		}
	}
	if !hasFilterMeta {
		t.Error("Filter result should contain __filter_metadata__")
	}

	// 再次调用 ListModels
	models2, total2, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("Second ListModels failed: %v", err)
	}

	// RED 检查：如果污染严重，total 可能变化
	if total2 != int64(initialCount) {
		t.Errorf("RED FAIL: Total changed from %d to %d after filter (data pollution)",
			initialCount, total2)
		t.FailNow()
	}

	// RED 检查：不应在正常列表中看到元数据标记
	for _, m := range models2 {
		if m.Name == "__filter_metadata__" || m.Name == "__export_metadata__" {
			t.Errorf("RED FAIL: Metadata marker '%s' leaked into model list (data pollution)", m.Name)
			t.FailNow()
		}
	}

	// RED 检查：每个模型都是有效模型
	for _, m := range models2 {
		if m.ID < 1 || m.Manufacturer == "" {
			t.Errorf("RED FAIL: Invalid model data detected: ID=%d, Manufacturer='%s'",
				m.ID, m.Manufacturer)
			t.FailNow()
		}
	}

	t.Log("GREEN PASS: Filter did not corrupt subsequent list operations")
}

// TestRedGreen_SequentialOperations 验证连续操作后的累积污染
// RED: 多次触发后数据完全混乱
// GREEN: 无论多少次操作，数据始终正确
func TestRedGreen_SequentialOperations(t *testing.T) {
	s := setupTestStore()
	ctx := context.Background()

	// 执行多次可能触发污染的操作
	operations := []struct {
		name string
		fn   func() error
	}{
		{"BuildModelList(Dell, false)", func() error {
			_, _, err := s.BuildModelList(ctx, "Dell", false)
			return err
		}},
		{"ExportActiveModels", func() error {
			_, err := s.ExportActiveModels(ctx)
			return err
		}},
		{"FilterModelsByDesc(['server'])", func() error {
			_, err := s.FilterModelsByDesc(ctx, []string{"server"})
			return err
		}},
		{"BuildModelList(HPE, true)", func() error {
			_, _, err := s.BuildModelList(ctx, "HPE", true)
			return err
		}},
		{"ExportActiveModels (again)", func() error {
			_, err := s.ExportActiveModels(ctx)
			return err
		}},
	}

	for _, op := range operations {
		if err := op.fn(); err != nil {
			t.Fatalf("Operation %s failed: %v", op.name, err)
		}
	}

	// 最终检查：ListModels 应返回正确的5个模型
	models, total, err := s.ListModels(ctx, 1, 100)
	if err != nil {
		t.Fatalf("Final ListModels failed: %v", err)
	}

	// RED 检查 1: 总数是否正确
	if total != 5 {
		t.Errorf("RED FAIL: After all operations, expected total=5, got %d", total)
		t.Log("  The shared backing array was corrupted by sequential append operations")
		t.FailNow()
	}

	// RED 检查 2: 数量是否正确
	if len(models) != 5 {
		t.Errorf("RED FAIL: Expected 5 models, got %d after sequential operations", len(models))
		t.FailNow()
	}

	// RED 检查 3: 是否有无效数据泄漏
	for _, m := range models {
		if m.Name == "" || m.ID < 1 || m.Manufacturer == "" {
			t.Errorf("RED FAIL: Corrupted model detected: ID=%d, Name='%s', Manufacturer='%s'",
				m.ID, m.Name, m.Manufacturer)
			t.FailNow()
		}
		// 检查是否有标记泄漏
		if len(m.Name) > 2 && m.Name[:2] == "__" {
			t.Errorf("RED FAIL: Metadata marker leaked: %s", m.Name)
			t.FailNow()
		}
	}

	// RED 检查 4: GetModelStats 应返回正确结果
	stats, ratio, err := s.GetModelStats(ctx)
	if err != nil {
		t.Fatalf("GetModelStats failed: %v", err)
	}
	if stats["Dell"] != 2 || stats["HPE"] != 2 || stats["Lenovo"] != 1 {
		t.Errorf("RED FAIL: Stats corrupted: Dell=%d(expect 2), HPE=%d(expect 2), Lenovo=%d(expect 1)",
			stats["Dell"], stats["HPE"], stats["Lenovo"])
		t.FailNow()
	}
	if ratio < 0.75 {
		t.Errorf("RED FAIL: Active ratio corrupted: got %f, expected >= 0.75", ratio)
		t.FailNow()
	}

	t.Log("GREEN PASS: All sequential operations completed without data corruption")
}
