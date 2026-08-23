// Package main 设备固件升级管理服务入口
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/handler"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// 版本号
const version = "1.0.0"

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	port := flag.Int("port", 0, "端口号（覆盖配置）")
	host := flag.String("host", "", "监听地址（覆盖配置）")
	logLevel := flag.String("log-level", "", "日志级别（debug/info/warn/error）")
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// 应用命令行参数覆盖
	if *port > 0 {
		cfg.Server.Port = *port
	}
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *logLevel != "" {
		applyLogLevel(*logLevel)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}

	logger.Info("Starting Firmware Upgrade Management Service",
		"version", version,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"storage", cfg.Storage.Type,
	)

	// 初始化存储
	var appStore store.Store
	if cfg.Storage.Type == "file" {
		appStore = store.NewFileStore(cfg)
	} else {
		appStore = store.NewMemoryStore()
	}

	// 初始化存储上下文：可取消的根 context，关闭时级联取消 autoSave、调度器与在途 worker
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	if err := appStore.Init(rootCtx); err != nil {
		logger.Fatalf("Failed to initialize store: %v", err)
	}

	// 确保必要目录存在
	ensureDir(cfg.Storage.UploadDir)
	ensureDir(cfg.Server.StaticDir)

	// 创建服务生命周期管理器（main 持有句柄，关闭时调用 lifecycle.Stop 排空后台 goroutine）
	lifecycle := service.NewServiceLifecycle()

	// 创建路由和处理器，并启动后台调度器
	router := handler.NewRouter(cfg)
	handler.Setup(router, cfg, appStore, lifecycle, rootCtx)

	// 创建 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        router.Handler(),
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(cfg.Server.IdleTimeout) * time.Second,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// 启动 HTTP 服务器
	go func() {
		logger.Infof("HTTP server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 等待中断信号进行优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Infof("Received signal: %v, shutting down...", sig)

	// 1) 取消根 context：级联取消 autoSave、轮询调度、任务调度及在途 worker。
	//    关键：lifecycle.Stop 只取消生命周期内部的子 ctx（轮询侧），
	//    任务侧 worker 与 autoSave 持有的是 rootCtx，必须靠 rootCancel 级联取消。
	rootCancel()

	// 2) 排空受生命周期管理的后台 goroutine（轮询循环/worker、任务 worker、任务调度循环）。
	//    它们均尊重 ctx，~80ms 内退出；超时兜底为 ShutdownTimeout。
	if err := lifecycle.Stop(time.Duration(cfg.Server.ShutdownTimeout) * time.Second); err != nil {
		logger.Errorf("Lifecycle stop error: %v", err)
	}

	// 3) 优雅关闭 HTTP：此时不再有新的后台任务被调度，排空在途请求
	shutdownCtx, cancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("HTTP server shutdown error: %v", err)
	}

	// 4) 关闭存储刷盘：autoSave 已在步骤 1 被取消并退出
	if err := appStore.Close(); err != nil {
		logger.Errorf("Store close error: %v", err)
	}

	logger.Info("Server shut down gracefully")
}

// loadConfig 加载配置
func loadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		cfg, err := config.LoadFromFile(configPath)
		if err != nil {
			logger.Warnf("Failed to load config file, using defaults: %v", err)
			return config.DefaultConfig(), nil
		}
		return cfg, nil
	}

	// 尝试从环境变量加载
	cfg := config.LoadFromEnv()
	return cfg, nil
}

// applyLogLevel 应用日志级别
func applyLogLevel(level string) {
	switch level {
	case "debug":
		logger.SetLevel(logger.LevelDebug)
	case "info":
		logger.SetLevel(logger.LevelInfo)
	case "warn":
		logger.SetLevel(logger.LevelWarn)
	case "error":
		logger.SetLevel(logger.LevelError)
	}
}

// ensureDir 确保目录存在
func ensureDir(dir string) {
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Warnf("Failed to create directory %s: %v", dir, err)
	}
}

// init 初始化：确保必要目录存在
func init() {
	dirs := []string{"uploads", "web", "data", "logs"}
	for _, dir := range dirs {
		ensureDir(filepath.Join(".", dir))
	}
}
