package handler

import (
	"net/http"
	"strconv"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/response"
)

// HistoryHandler 历史记录处理器
type HistoryHandler struct {
	service *service.HistoryService
	config  *config.Config
}

// NewHistoryHandler 创建历史记录处理器
func NewHistoryHandler(hs *service.HistoryService, cfg *config.Config) *HistoryHandler {
	return &HistoryHandler{
		service: hs,
		config:  cfg,
	}
}

// List 列出历史记录
func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	status := model.UpgradeStatus(r.URL.Query().Get("status"))

	records, total, err := h.service.ListRecords(r.Context(), page, pageSize, status)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Paginated(w, records, total, page, pageSize)
}

// Get 获取单条记录
func (h *HistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	record, err := h.service.GetRecord(r.Context(), model.ID(id))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, record)
}

// GetDeviceHistory 获取设备历史
func (h *HistoryHandler) GetDeviceHistory(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		response.BadRequest(w, "device_id is required")
		return
	}

	records, err := h.service.GetDeviceHistory(r.Context(), deviceID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, records)
}

// GetTaskHistory 获取任务历史
func (h *HistoryHandler) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	records, err := h.service.GetTaskHistory(r.Context(), model.ID(id))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, records)
}

// GetRecent 获取最近记录
func (h *HistoryHandler) GetRecent(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	records, err := h.service.GetRecentRecords(r.Context(), limit)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, records)
}

// Delete 删除记录
func (h *HistoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.service.DeleteRecord(r.Context(), model.ID(id)); err != nil {
		response.WriteError(w, err)
		return
	}

	response.SuccessMessage(w, "record deleted", nil)
}
