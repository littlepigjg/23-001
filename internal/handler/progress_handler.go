package handler

import (
	"net/http"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/response"
)

// ProgressHandler 进度处理器
type ProgressHandler struct {
	service *service.ProgressService
	config  *config.Config
}

// NewProgressHandler 创建进度处理器
func NewProgressHandler(ps *service.ProgressService, cfg *config.Config) *ProgressHandler {
	return &ProgressHandler{
		service: ps,
		config:  cfg,
	}
}

// Report 上报进度
func (h *ProgressHandler) Report(w http.ResponseWriter, r *http.Request) {
	var req model.ReportProgressRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if err := h.service.ReportProgress(r.Context(), &req); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.SuccessMessage(w, "progress reported", nil)
}

// BatchReport 批量上报
func (h *ProgressHandler) BatchReport(w http.ResponseWriter, r *http.Request) {
	var reports []*model.ReportProgressRequest
	if err := ReadJSON(r, &reports); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	results := h.service.BatchReportProgress(r.Context(), reports)
	response.Success(w, results)
}

// GetDeviceProgress 获取设备进度
func (h *ProgressHandler) GetDeviceProgress(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		response.BadRequest(w, "device_id is required")
		return
	}

	device, err := h.service.GetDeviceProgress(r.Context(), deviceID)
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.Success(w, device)
}

// GetTaskProgress 获取任务中所有设备的进度
func (h *ProgressHandler) GetTaskProgress(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	records, err := h.service.GetTaskProgress(r.Context(), model.ID(id))
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, records)
}
