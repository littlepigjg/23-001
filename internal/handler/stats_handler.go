package handler

import (
	"net/http"

	"fwupgrade/internal/config"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/response"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	service *service.StatsService
	config  *config.Config
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(ss *service.StatsService, cfg *config.Config) *StatsHandler {
	return &StatsHandler{
		service: ss,
		config:  cfg,
	}
}

// Dashboard 获取仪表盘
func (h *StatsHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.GetDashboard(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, dashboard)
}

// Statistics 获取详细统计
func (h *StatsHandler) Statistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStatistics(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, stats)
}

// VersionDistribution 获取版本分布
func (h *StatsHandler) VersionDistribution(w http.ResponseWriter, r *http.Request) {
	dist, err := h.service.GetVersionDistribution(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, dist)
}

// ModelDistribution 获取型号分布
func (h *StatsHandler) ModelDistribution(w http.ResponseWriter, r *http.Request) {
	dist, err := h.service.GetModelDistribution(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, dist)
}

// TaskStatusSummary 获取任务状态汇总
func (h *StatsHandler) TaskStatusSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetTaskStatusSummary(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, summary)
}

// DeviceStatusSummary 获取设备状态汇总
func (h *StatsHandler) DeviceStatusSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetDeviceStatusSummary(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, summary)
}
