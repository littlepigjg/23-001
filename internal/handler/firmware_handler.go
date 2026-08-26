package handler

import (
	"net/http"
	"strconv"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/response"
)

// FirmwareHandler 固件处理器
type FirmwareHandler struct {
	service *service.FirmwareService
	config  *config.Config
}

// NewFirmwareHandler 创建固件处理器
func NewFirmwareHandler(svc *service.FirmwareService, cfg *config.Config) *FirmwareHandler {
	return &FirmwareHandler{
		service: svc,
		config:  cfg,
	}
}

// Upload 上传固件
func (h *FirmwareHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// 解析 multipart form
	if err := r.ParseMultipartForm(h.config.Firmware.MaxFileSize); err != nil {
		response.BadRequest(w, "failed to parse multipart form: "+err.Error())
		return
	}

	// 获取表单字段
	modelIDStr := r.FormValue("model_id")
	version := r.FormValue("version")
	md5 := r.FormValue("md5")
	changelog := r.FormValue("changelog")

	modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid model_id")
		return
	}

	// 获取上传的文件
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	// 读取文件内容
	fileData := make([]byte, fileHeader.Size)
	_, err = file.Read(fileData)
	if err != nil {
		response.InternalError(w, "failed to read file: "+err.Error())
		return
	}

	req := &model.UploadFirmwareRequest{
		ModelID:   model.ID(modelID),
		Version:   version,
		Md5:       md5,
		Changelog: changelog,
	}

	firmware, err := h.service.UploadFirmware(r.Context(), req, fileData, fileHeader.Filename)
	if err != nil {
		logger.Error("Failed to upload firmware", "error", err)
		response.InternalError(w, err.Error())
		return
	}

	response.Created(w, firmware)
}

// Get 获取固件
func (h *FirmwareHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	fw, err := h.service.GetFirmware(r.Context(), model.ID(id))
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	response.Success(w, fw)
}

// List 列出固件
func (h *FirmwareHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	modelID, _ := strconv.ParseInt(r.URL.Query().Get("model_id"), 10, 64)

	firmwares, total, err := h.service.ListFirmwares(r.Context(), page, pageSize, model.ID(modelID))
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Paginated(w, firmwares, total, page, pageSize)
}

// Update 更新固件
func (h *FirmwareHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	var req model.UpdateFirmwareRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	fw, err := h.service.UpdateFirmware(r.Context(), model.ID(id), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, fw)
}

// Delete 删除固件
func (h *FirmwareHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.service.DeleteFirmware(r.Context(), model.ID(id)); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.SuccessMessage(w, "firmware deleted", nil)
}

// Download 下载固件
func (h *FirmwareHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	data, fw, err := h.service.GetFirmwareFile(r.Context(), model.ID(id))
	if err != nil {
		response.NotFound(w, err.Error())
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=firmware_"+fw.Version+".bin")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	w.Header().Set("X-Firmware-MD5", fw.Md5)
	w.Header().Set("X-Firmware-Version", fw.Version)

	w.Write(data)
}
