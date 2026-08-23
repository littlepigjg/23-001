package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/response"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	startTime time.Time
	config    *config.Config
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(cfg *config.Config) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		config:    cfg,
	}
}

// Health 健康检查端点
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	resp := &model.HealthResponse{
		Status:    "ok",
		Version:   "1.0.0",
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	response.Success(w, resp)
}

// Ready 就绪检查端点
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	serverCfg := h.config.Get().Server
	storageCfg := h.config.GetStorageConfig()
	dependencies := map[string]string{
		"database":     "ok",
		"storage":      storageCfg.Type,
		"server":       serverCfg.Host + ":" + strconv.Itoa(serverCfg.Port),
		"firmware_dir": storageCfg.UploadDir,
	}

	resp := &model.ReadyResponse{
		Status:       "ready",
		Dependencies: dependencies,
	}

	response.Success(w, resp)
}

// DeviceHandler 设备处理器
type DeviceHandler struct {
	deviceService *service.DeviceService
	config        *config.Config
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(ds *service.DeviceService, cfg *config.Config) *DeviceHandler {
	return &DeviceHandler{
		deviceService: ds,
		config:        cfg,
	}
}

// Create 创建设备
func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDeviceRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	device, err := h.deviceService.CreateDevice(r.Context(), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Created(w, device)
}

// Get 获取设备
func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	device, err := h.deviceService.GetDevice(r.Context(), model.ID(id))
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.Success(w, device)
}

// List 列出设备
func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	modelID, _ := strconv.ParseInt(r.URL.Query().Get("model_id"), 10, 64)
	status := model.DeviceStatus(r.URL.Query().Get("status"))

	devices, total, err := h.deviceService.ListDevices(r.Context(), page, pageSize, model.ID(modelID), status)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Paginated(w, devices, total, page, pageSize)
}

// Update 更新设备
func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	var req model.UpdateDeviceRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	device, err := h.deviceService.UpdateDevice(r.Context(), model.ID(id), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, device)
}

// Delete 删除设备
func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.deviceService.DeleteDevice(r.Context(), model.ID(id)); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.SuccessMessage(w, "device deleted", nil)
}

// Register 设备注册
func (h *DeviceHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterDeviceRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	device, err := h.deviceService.RegisterDevice(r.Context(), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, device)
}

// Search 搜索设备
func (h *DeviceHandler) Search(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	devices, total, err := h.deviceService.SearchDevices(r.Context(), keyword, page, pageSize)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Paginated(w, devices, total, page, pageSize)
}

// BatchCreate 批量创建设备
func (h *DeviceHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req model.BatchCreateDeviceRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	result, err := h.deviceService.BatchCreateDevices(r.Context(), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, result)
}

// GetStatus 获取设备状态统计
func (h *DeviceHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	stats, err := h.deviceService.GetDeviceStatusStats(r.Context())
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, stats)
}

// registerHandlers 注册处理器到路由
func registerHandlers(ctx context.Context, router *Router, cfg *config.Config) {
	logger.Info("Registering HTTP handlers")

	// 创建服务实例
	modelService := service.NewDeviceModelService(nil, cfg) // 将在实际初始化时注入
	_ = modelService
}

// 占位：完整的处理器注册将在 main.go 中完成
