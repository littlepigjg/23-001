package handler

import (
	"net/http"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/response"
)

// PollHandler 轮询处理器
type PollHandler struct {
	service *service.PollService
	config  *config.Config
}

// NewPollHandler 创建轮询处理器
func NewPollHandler(ps *service.PollService, cfg *config.Config) *PollHandler {
	return &PollHandler{
		service: ps,
		config:  cfg,
	}
}

// Poll 设备轮询
func (h *PollHandler) Poll(w http.ResponseWriter, r *http.Request) {
	var req model.PollUpgradeRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	resp, err := h.service.PollDevice(r.Context(), &req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, resp)
}

// PollBatch 批量轮询
func (h *PollHandler) PollBatch(w http.ResponseWriter, r *http.Request) {
	var requests []*model.PollUpgradeRequest
	if err := ReadJSON(r, &requests); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	responses, err := h.service.PollMultipleDevices(r.Context(), requests)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, responses)
}
