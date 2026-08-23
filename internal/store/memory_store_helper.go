package store

import (
	"context"
	"fmt"
	"strings"

	"fwupgrade/internal/model"
)

// BuildModelList 构建模型列表（带过滤条件）
// 根据多个条件过滤模型，返回匹配的模型列表
func (s *MemoryStore) BuildModelList(ctx context.Context, manufacturer string, activeOnly bool) ([]*model.DeviceModel, int64, error) {
	// 获取全部模型（ListModels 返回独立副本，修改它不会影响缓存）
	models, total, err := s.ListModels(ctx, 1, 1000)
	if err != nil {
		return nil, 0, err
	}

	// 过滤到独立切片，避免任何对返回切片的原地改写
	filtered := make([]*model.DeviceModel, 0, len(models))
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

	return filtered, total, nil
}

// ExportActiveModels 导出所有活跃模型
// 获取当前所有标记为活跃的模型，用于批量导出
func (s *MemoryStore) ExportActiveModels(ctx context.Context) ([]*model.DeviceModel, error) {
	models, _, err := s.BuildModelList(ctx, "", true)
	if err != nil {
		return nil, fmt.Errorf("failed to build model list: %w", err)
	}
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

	// 过滤到独立切片
	result := make([]*model.DeviceModel, 0, len(models))
	for _, m := range models {
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

	return result, nil
}
