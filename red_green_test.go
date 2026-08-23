package fwupgrade

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	m := model.NewDeviceModel("ModelX", "MfrX", "HW1", "Test model")
	if err := memStore.CreateModel(context.Background(), m); err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	task := model.NewUpgradeTask(
		"TestTask", "Test description",
		m.ID, m.Name,
		99, "v1.0",
		model.TaskTypeFull, 100, nil, "admin",
	)
	if err := memStore.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	svc := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)

	_ = svc.CancelTask(context.Background(), task.ID)

	done := make(chan struct{}, 1)
	go func() {
		_ = memStore.UpdateTaskStatus(context.Background(), task.ID, model.TaskCancelled)
		done <- struct{}{}
	}()

	select {
	case <-done:
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	case <-time.After(time.Second * 2):
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
	}
}
