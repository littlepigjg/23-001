package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/response"
)

// Setup 配置所有路由和处理器，并启动后台调度器（轮询调度、任务调度）。
// lifecycle 与 rootCtx 由 main 持有，关闭时由 main 取消 rootCtx 并调用 lifecycle.Stop 排空后台 goroutine。
func Setup(router *Router, cfg *config.Config, appStore store.Store, lifecycle *service.ServiceLifecycle, rootCtx context.Context) {
	logger.Info("Setting up API routes")

	// 创建服务实例
	modelStore := appStore
	deviceStore := appStore
	firmwareStore := appStore
	taskStore := appStore
	recordStore := appStore

	// 初始化服务
	deviceModelService := service.NewDeviceModelService(modelStore, cfg)
	deviceService := service.NewDeviceService(deviceStore, modelStore, cfg)
	firmwareService := service.NewFirmwareService(firmwareStore, modelStore, cfg)
	taskService := service.NewTaskService(taskStore, deviceStore, firmwareStore, modelStore, recordStore, cfg)
	grayscaleService := service.NewGrayscaleService(taskStore, cfg)
	pollService := service.NewPollService(taskStore, deviceStore, firmwareStore, recordStore, grayscaleService)
	progressService := service.NewProgressService(deviceStore, recordStore, taskStore)
	historyService := service.NewHistoryService(recordStore)
	statsService := service.NewStatsService(deviceStore, modelStore, firmwareStore, taskStore, recordStore)

	// 绑定服务生命周期，并启动后台调度器
	pollService.SetLifecycle(lifecycle)
	taskService.SetLifecycle(lifecycle)

	// 轮询调度器：runManagedPolling 内部自 spawn 受管 goroutine 并返回
	pollService.SchedulePolling(rootCtx, time.Duration(cfg.Grayscale.DevicePollInterval)*time.Second)

	// 任务调度器：ProcessScheduledTasks 是一次性的，用 ticker 循环包装
	taskInterval := time.Duration(cfg.Grayscale.UpgradeInterval) * time.Second
	lifecycle.Add(1)
	go func() {
		defer lifecycle.Done()
		taskSchedulerLoop(rootCtx, taskService, taskInterval)
	}()

	// 初始化处理器
	healthHandler := NewHealthHandler(cfg)
	modelHandler := NewDeviceModelHandler(deviceModelService, cfg)
	deviceHandler := NewDeviceHandler(deviceService, cfg)
	firmwareHandler := NewFirmwareHandler(firmwareService, cfg)
	taskHandler := NewTaskHandler(taskService, cfg)
	pollHandler := NewPollHandler(pollService, cfg)
	progressHandler := NewProgressHandler(progressService, cfg)
	historyHandler := NewHistoryHandler(historyService, cfg)
	statsHandler := NewStatsHandler(statsService, cfg)

	// --- 健康检查路由 ---
	router.HandleFunc("/health", healthHandler.Health)
	router.HandleFunc("/ready", healthHandler.Ready)

	// --- 设备型号路由 ---
	router.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			modelHandler.List(w, r)
		case http.MethodPost:
			modelHandler.Create(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/models/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			modelHandler.Get(w, r)
		case http.MethodPut:
			modelHandler.Update(w, r)
		case http.MethodDelete:
			modelHandler.Delete(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 设备路由 ---
	router.HandleFunc("/api/devices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			deviceHandler.List(w, r)
		case http.MethodPost:
			deviceHandler.Create(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/devices/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		// /api/devices/batch
		if len(parts) >= 3 && parts[2] == "batch" {
			if r.Method == http.MethodPost {
				deviceHandler.BatchCreate(w, r)
				return
			}
		}

		// /api/devices/search
		if len(parts) >= 3 && parts[2] == "search" {
			if r.Method == http.MethodGet {
				deviceHandler.Search(w, r)
				return
			}
		}

		// /api/devices/status
		if len(parts) >= 3 && parts[2] == "status" {
			if r.Method == http.MethodGet {
				deviceHandler.GetStatus(w, r)
				return
			}
		}

		// /api/devices/register
		if len(parts) >= 3 && parts[2] == "register" {
			if r.Method == http.MethodPost {
				deviceHandler.Register(w, r)
				return
			}
		}

		// /api/devices/{id}
		switch r.Method {
		case http.MethodGet:
			deviceHandler.Get(w, r)
		case http.MethodPut:
			deviceHandler.Update(w, r)
		case http.MethodDelete:
			deviceHandler.Delete(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 固件路由 ---
	router.HandleFunc("/api/firmware", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			firmwareHandler.List(w, r)
		case http.MethodPost:
			firmwareHandler.Upload(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/firmware/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		// /api/firmware/{id}/download
		if len(parts) >= 3 && parts[2] == "download" {
			if r.Method == http.MethodGet {
				r.URL.Path = "/api/firmware/" + parts[1]
				firmwareHandler.Download(w, r)
				return
			}
		}

		switch r.Method {
		case http.MethodGet:
			firmwareHandler.Get(w, r)
		case http.MethodPut:
			firmwareHandler.Update(w, r)
		case http.MethodDelete:
			firmwareHandler.Delete(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 任务路由 ---
	router.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskHandler.List(w, r)
		case http.MethodPost:
			taskHandler.Create(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		// /api/tasks/{id}/start
		if len(parts) >= 3 && parts[2] == "start" {
			if r.Method == http.MethodPost {
				r.URL.Path = "/api/tasks/" + parts[1]
				taskHandler.Start(w, r)
				return
			}
		}

		// /api/tasks/{id}/cancel
		if len(parts) >= 3 && parts[2] == "cancel" {
			if r.Method == http.MethodPost {
				r.URL.Path = "/api/tasks/" + parts[1]
				taskHandler.Cancel(w, r)
				return
			}
		}

		// /api/tasks/{id}/progress
		if len(parts) >= 3 && parts[2] == "progress" {
			if r.Method == http.MethodGet {
				r.URL.Path = "/api/tasks/" + parts[1]
				taskHandler.GetProgress(w, r)
				return
			}
		}

		switch r.Method {
		case http.MethodGet:
			taskHandler.Get(w, r)
		case http.MethodPut:
			taskHandler.Update(w, r)
		case http.MethodDelete:
			taskHandler.Delete(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 轮询路由 ---
	router.HandleFunc("/api/poll", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			pollHandler.Poll(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/poll/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			pollHandler.PollBatch(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 进度路由 ---
	router.HandleFunc("/api/progress", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			progressHandler.Report(w, r)
		case http.MethodGet:
			progressHandler.GetDeviceProgress(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/progress/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			progressHandler.BatchReport(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/progress/task/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			progressHandler.GetTaskProgress(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 历史记录路由 ---
	router.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			historyHandler.List(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/history/recent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			historyHandler.GetRecent(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/history/device/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			historyHandler.GetDeviceHistory(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/history/task/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			historyHandler.GetTaskHistory(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/history/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			historyHandler.Get(w, r)
		case http.MethodDelete:
			historyHandler.Delete(w, r)
		default:
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 统计路由 ---
	router.HandleFunc("/api/stats/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.Dashboard(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/stats/statistics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.Statistics(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/stats/versions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.VersionDistribution(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/stats/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.ModelDistribution(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/stats/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.TaskStatusSummary(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})
	router.HandleFunc("/api/stats/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.DeviceStatusSummary(w, r)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 初始化示例数据路由 ---
	router.HandleFunc("/api/init/sample", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createSampleData(r, deviceModelService, deviceService, firmwareService, taskService, grayscaleService)
			response.SuccessMessage(w, "sample data created", nil)
		} else {
			response.BadRequest(w, "method not allowed")
		}
	})

	// --- 静态文件服务 ---
	webDir := cfg.Server.StaticDir
	if webDir != "" {
		logger.Info("Serving static files", "dir", webDir)
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, webDir+"/index.html")
				return
			}
			http.FileServer(http.Dir(webDir)).ServeHTTP(w, r)
		})
	}

	logger.Info("Routes setup complete")
}

// taskSchedulerLoop 周期性调用 ProcessScheduledTasks，尊重 rootCtx 取消以支持优雅关闭
func taskSchedulerLoop(ctx context.Context, ts *service.TaskService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Task scheduler started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Task scheduler stopped")
			return
		case <-ticker.C:
			if err := ts.ProcessScheduledTasks(ctx); err != nil {
				logger.Error("Scheduled task processing failed", "error", err)
			}
		}
	}
}
