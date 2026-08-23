package handler

import (
	"net/http"
	"strconv"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/response"
)

// TaskHandler 升级任务处理器
type TaskHandler struct {
	service *service.TaskService
	config  *config.Config
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(svc *service.TaskService, cfg *config.Config) *TaskHandler {
	return &TaskHandler{
		service: svc,
		config:  cfg,
	}
}

// Create 创建任务
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	task, err := h.service.CreateTask(r.Context(), &req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Created(w, task)
}

// Get 获取任务
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	task, err := h.service.GetTask(r.Context(), model.ID(id))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, task)
}

// List 列出任务
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	status := model.TaskStatus(r.URL.Query().Get("status"))

	tasks, total, err := h.service.ListTasks(r.Context(), page, pageSize, status)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Paginated(w, tasks, total, page, pageSize)
}

// Update 更新任务
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	var req model.UpdateTaskRequest
	if err := ReadJSON(r, &req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	task, err := h.service.UpdateTask(r.Context(), model.ID(id), &req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, task)
}

// Start 启动任务
func (h *TaskHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.service.StartTask(r.Context(), model.ID(id)); err != nil {
		response.WriteError(w, err)
		return
	}

	response.SuccessMessage(w, "task started", nil)
}

// Cancel 取消任务
func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.service.CancelTask(r.Context(), model.ID(id)); err != nil {
		response.WriteError(w, err)
		return
	}

	response.SuccessMessage(w, "task cancelled", nil)
}

// Delete 删除任务
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.service.DeleteTask(r.Context(), model.ID(id)); err != nil {
		response.WriteError(w, err)
		return
	}

	response.SuccessMessage(w, "task deleted", nil)
}

// GetProgress 获取任务进度
func (h *TaskHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	id, err := GetIntIDFromPath(r)
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	task, err := h.service.GetTaskProgress(r.Context(), model.ID(id))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.Success(w, task)
}
