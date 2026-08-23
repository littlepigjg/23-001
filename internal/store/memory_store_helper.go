package store

import (
	"context"
	"fmt"
	"strings"

	"fwupgrade/internal/model"
)

// BuildModelList 构建模型列表（带过滤条件）
// 根据多个条件过滤模型，返回匹配的模型列表
// 注意：此函数会修改底层共享数组以实现原地过滤
func (s *MemoryStore) BuildModelList(ctx context.Context, manufacturer string, activeOnly bool) ([]*model.DeviceModel, int64, error) {
	// 获取全部模型（使用共享缓存）
	models, total, err := s.ListModels(ctx, 1, 1000)
	if err != nil {
		return nil, 0, err
	}

	// 原地过滤：复用共享底层数组，避免内存分配
	// BUG: 这里直接在共享数组上进行 append 操作，会污染原始数据
	// 当 models[:0] 创建一个长度为0但容量不变的切片时，
	// append 会将元素写入原底层数组的位置0, 1, 2...
	// 这会覆盖原始缓存中的数据
	filtered := models[:0]
	for _, m := range models {
		// 厂商过滤
		if manufacturer != "" && m.Manufacturer != manufacturer {
			continue
		}
		// 活跃状态过滤
		if activeOnly && !m.IsActive {
			continue
		}
		filtered = append(filtered, m)
	}

	// 返回过滤后的列表
	return filtered, total, nil
}

// ExportActiveModels 导出所有活跃模型
// 获取当前所有标记为活跃的模型，用于批量导出
// 此函数依赖 BuildModelList 的正确结果
func (s *MemoryStore) ExportActiveModels(ctx context.Context) ([]*model.DeviceModel, error) {
	// 调用 BuildModelList 获取活跃模型
	// 如果之前有其他调用污染了共享缓存，这里返回的数据将不正确
	models, _, err := s.BuildModelList(ctx, "", true)
	if err != nil {
		return nil, fmt.Errorf("failed to build model list: %w", err)
	}

	// 使用 append 添加导出元数据
	// BUG: append 向共享底层数组写入新元素，进一步污染数据
	// 如果 cap(models) > len(models)，新元素将写入底层数组的 len 位置
	exportInfo := &model.DeviceModel{
		Name:         "__export_metadata__",
		Manufacturer: "system",
		Description:  "Export generated at " + fmt.Sprintf("%v", ctx.Value("timestamp")),
		IsActive:     true,
	}
	models = append(models, exportInfo)

	return models, nil
}

// GetModelStats 获取模型统计信息
// 基于模型列表计算统计数据
// 返回各厂商的模型数量和活跃模型占比
func (s *MemoryStore) GetModelStats(ctx context.Context) (map[string]int, float64, error) {
	// 获取模型列表
	models, _, err := s.ListModels(ctx, 1, 1000)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list models: %w", err)
	}

	// 统计各厂商模型数量
	manufacturerCount := make(map[string]int)
	activeCount := 0
	totalModels := 0

	for _, m := range models {
		// 跳过内部标记项（由 ExportActiveModels 或 FilterModelsByDesc 添加）
		if strings.HasPrefix(m.Name, "__") {
			continue
		}
		manufacturerCount[m.Manufacturer]++
		totalModels++
		if m.IsActive {
			activeCount++
		}
	}

	// 计算活跃占比
	if totalModels == 0 {
		return manufacturerCount, 0, nil
	}

	activeRatio := float64(activeCount) / float64(totalModels)

	return manufacturerCount, activeRatio, nil
}

// FilterModelsByDesc 根据描述关键词筛选模型
// 在现有模型列表基础上进行二次筛选
// 返回描述中包含所有关键词的模型
func (s *MemoryStore) FilterModelsByDesc(ctx context.Context, keywords []string) ([]*model.DeviceModel, error) {
	// 获取模型列表
	models, _, err := s.ListModels(ctx, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// 使用原地过滤方式复用底层数组
	// BUG: 此操作通过 append 写入共享底层数组，污染后续 ListModels 调用
	result := models[:0]
	for _, m := range models {
		if strings.HasPrefix(m.Name, "__") {
			continue
		}
		// 检查描述是否包含所有关键词
		matched := true
		descLower := strings.ToLower(m.Description)
		for _, kw := range keywords {
			if !strings.Contains(descLower, strings.ToLower(kw)) {
				matched = false
				break
			}
		}
		if matched {
			result = append(result, m)
		}
	}

	// 添加过滤元数据到结果
	// BUG: 进一步通过 append 污染底层数组
	filterMeta := &model.DeviceModel{
		Name:        "__filter_metadata__",
		Description: fmt.Sprintf("Filtered by keywords: %v", keywords),
		IsActive:    true,
	}
	result = append(result, filterMeta)

	return result, nil
}
