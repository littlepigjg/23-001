package handler

import (
	"net/http"
	"strconv"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/response"
)

// DeviceModelHandler 设备型号处理器
type DeviceModelHandler struct {
	service *service.DeviceModelService
	config  *config.Config
}

// NewDeviceModelHandler 创建设备型号处理器
func NewDeviceModelHandler(svc *service.DeviceModelService, cfg *config.Config) *DeviceModelHandler {
	return &DeviceModelHandler{
		service: svc,
		config:  cfg,
	}
}

// Create 创建型号
func (h *DeviceModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateModelRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	m, err := h.service.CreateModel(r.Context(), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Created(w, m)
}

// Get 获取型号
func (h *DeviceModelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	m, err := h.service.GetModel(r.Context(), model.ID(id))
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.Success(w, m)
}

// List 列表型号
func (h *DeviceModelHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	models, total, err := h.service.ListModels(r.Context(), page, pageSize)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Paginated(w, models, total, page, pageSize)
}

// Update 更新型号
func (h *DeviceModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	var req model.UpdateModelRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	m, err := h.service.UpdateModel(r.Context(), model.ID(id), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, m)
}

// Delete 删除型号
func (h *DeviceModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.service.DeleteModel(r.Context(), model.ID(id)); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.SuccessMessage(w, "model deleted", nil)
}
