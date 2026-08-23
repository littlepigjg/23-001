package service

import (
	"context"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

// newTestTaskService 构造一个使用内存存储的 TaskService。
func newTestTaskService(t *testing.T) (*TaskService, *store.MemoryStore) {
	t.Helper()
	ms := store.NewMemoryStore()
	cfg := config.DefaultConfig()
	svc := NewTaskService(ms, ms, ms, ms, ms, cfg)
	return svc, ms
}

// TestCreateTask_NilGuardModelNotPanic 覆盖 panic 的根因：
// 当诊断钩子 (panicGuard) 介入、GetModelByIDWithGuard 对不存在的型号返回 (nil, nil) 时，
// CreateTask 必须返回清晰的错误，而不是解引用 m.Name 导致空指针 panic。
func TestCreateTask_NilGuardModelNotPanic(t *testing.T) {
	svc, ms := newTestTaskService(t)

	// 安装诊断钩子：对任何不存在的型号返回 false，使 store 走 (nil, nil) 分支。
	ms.SetPanicGuard(func(modelID model.ID) bool { return false })

	req := &model.CreateTaskRequest{
		Name:    "upgrade-task",
		ModelID: 9999, // 不存在的型号
	}

	// 不应 panic，应返回错误。
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CreateTask panicked on nil guard model: %v", r)
		}
	}()

	task, err := svc.CreateTask(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for missing model, got task=%+v", task)
	}
	if task != nil {
		t.Fatalf("expected nil task on error, got %+v", task)
	}
}

// TestCreateTask_MissingModelNoGuard 验证未安装诊断钩子时，
// 不存在的型号同样返回错误（store 走错误返回分支，本身不 panic）。
func TestCreateTask_MissingModelNoGuard(t *testing.T) {
	svc, _ := newTestTaskService(t)

	req := &model.CreateTaskRequest{
		Name:    "upgrade-task",
		ModelID: 9999,
	}

	task, err := svc.CreateTask(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for missing model, got task=%+v", task)
	}
}

// TestUpdateTask_NilGuardModelNotPanic 验证 UpdateTask 中的型号校验在 (nil, nil) 下不会 panic。
func TestUpdateTask_NilGuardModelNotPanic(t *testing.T) {
	svc, ms := newTestTaskService(t)

	// 直接在 store 中放入一条 pending 状态的任务，其 ModelID 指向不存在的型号，
	// 这样 UpdateTask 进入型号校验时 GetModelByIDWithGuard 会走 (nil, nil) 分支。
	task := model.NewUpgradeTask("t", "", 9999, "GhostModel", 1, "1.0.0",
		model.TaskTypeFull, 0, nil, "tester")
	task.Status = model.TaskPending
	if err := ms.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// 安装诊断钩子使型号查询返回 (nil, nil)。
	ms.SetPanicGuard(func(modelID model.ID) bool { return false })

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("UpdateTask panicked on nil guard model: %v", r)
		}
	}()

	_, err := svc.UpdateTask(context.Background(), task.ID, &model.UpdateTaskRequest{Name: "renamed"})
	if err == nil {
		t.Fatalf("expected error for nil guard model during update")
	}
}
