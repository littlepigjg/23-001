// Package config 提供应用程序的配置管理功能
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config 应用程序配置结构
type Config struct {
	mu sync.RWMutex

	// 服务器配置
	Server ServerConfig `json:"server"`
	// 存储配置
	Storage StorageConfig `json:"storage"`
	// 固件配置
	Firmware FirmwareConfig `json:"firmware"`
	// 日志配置
	Log LogConfig `json:"log"`
	// 灰度配置
	Grayscale GrayscaleConfig `json:"grayscale"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	ReadTimeout     int    `json:"read_timeout"`      // 读取超时（秒）
	WriteTimeout    int    `json:"write_timeout"`     // 写入超时（秒）
	IdleTimeout     int    `json:"idle_timeout"`      // 空闲超时（秒）
	ShutdownTimeout int    `json:"shutdown_timeout"`  // 优雅关闭超时（秒）
	MaxHeaderBytes  int    `json:"max_header_bytes"`  // 请求头最大字节数
	EnableCORS      bool   `json:"enable_cors"`       // 是否启用跨域
	StaticDir       string `json:"static_dir"`        // 静态文件目录
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type      string `json:"type"`       // 存储类型：memory / file
	DataDir   string `json:"data_dir"`   // 数据目录
	UploadDir string `json:"upload_dir"` // 上传文件目录
}

// FirmwareConfig 固件配置
type FirmwareConfig struct {
	MaxFileSize    int64  `json:"max_file_size"`     // 最大文件大小（字节）
	AllowedExts    string `json:"allowed_exts"`      // 允许的文件扩展名（逗号分隔）
	AutoBackup     bool   `json:"auto_backup"`       // 上传时是否自动备份
	BackupDir      string `json:"backup_dir"`        // 备份目录
	KeepVersions   int    `json:"keep_versions"`     // 每个型号保留的版本数
	RequireMD5     bool   `json:"require_md5"`       // 是否要求验证 MD5
	MaxVersionAge  int    `json:"max_version_age"`   // 版本最大保留天数
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`       // 日志级别：debug / info / warn / error / fatal
	Format     string `json:"format"`      // 日志格式
	Output     string `json:"output"`      // 日志输出位置：stdout / 文件
	FilePath   string `json:"file_path"`   // 日志文件路径
	MaxSize    int    `json:"max_size"`    // 日志文件最大大小（MB）
	MaxBackups int    `json:"max_backups"` // 日志文件最大备份数
}

// GrayscaleConfig 灰度策略配置
type GrayscaleConfig struct {
	DefaultRatio        float64 `json:"default_ratio"`         // 默认灰度比例
	MaxRatio            float64 `json:"max_ratio"`             // 最大灰度比例
	MinRatio            float64 `json:"min_ratio"`             // 最小灰度比例
	RatioIncrement      float64 `json:"ratio_increment"`       // 每次增加比例
	UpgradeInterval     int     `json:"upgrade_interval"`      // 升级检查间隔（秒）
	ProgressUpdateInterval int   `json:"progress_update_interval"` // 进度更新间隔（秒）
	EnableAutoRollback  bool    `json:"enable_auto_rollback"`  // 是否启用自动回滚
	RollbackThreshold   int     `json:"rollback_threshold"`    // 回滚阈值（失败百分比）
	MaxConcurrentTasks  int     `json:"max_concurrent_tasks"`  // 最大并发升级任务数
	DevicePollInterval  int     `json:"device_poll_interval"`  // 设备轮询间隔（秒）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     30,
			WriteTimeout:    30,
			IdleTimeout:     120,
			ShutdownTimeout: 10,
			MaxHeaderBytes:  1 << 20, // 1MB
			EnableCORS:      true,
			StaticDir:       "web",
		},
		Storage: StorageConfig{
			Type:      "memory",
			DataDir:   "data",
			UploadDir: "uploads",
		},
		Firmware: FirmwareConfig{
			MaxFileSize:   100 * 1024 * 1024, // 100MB
			AllowedExts:   ".bin,.hex,.img,.firmware,.fw",
			AutoBackup:    false,
			BackupDir:     "backups",
			KeepVersions:  10,
			RequireMD5:    true,
			MaxVersionAge: 365,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			FilePath:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 3,
		},
		Grayscale: GrayscaleConfig{
			DefaultRatio:          10.0,
			MaxRatio:              100.0,
			MinRatio:              0.0,
			RatioIncrement:        10.0,
			UpgradeInterval:       30,
			ProgressUpdateInterval: 5,
			EnableAutoRollback:    true,
			RollbackThreshold:     20,
			MaxConcurrentTasks:    10,
			DevicePollInterval:    60,
		},
	}
}

// LoadFromFile 从配置文件加载配置（简化的 key=value 格式）
func LoadFromFile(filePath string) (*Config, error) {
	cfg := DefaultConfig()

	file, err := os.Open(filePath)
	if err != nil {
		return cfg, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if err := cfg.setByKey(key, value); err != nil {
			return nil, fmt.Errorf("config key %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("STORAGE_TYPE"); v != "" {
		cfg.Storage.Type = v
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.Storage.DataDir = v
	}
	if v := os.Getenv("UPLOAD_DIR"); v != "" {
		cfg.Storage.UploadDir = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("MAX_FILE_SIZE"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Firmware.MaxFileSize = size
		}
	}

	return cfg
}

// setByKey 根据 key 设置配置值
func (c *Config) setByKey(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch key {
	case "server.host":
		c.Server.Host = value
	case "server.port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
		c.Server.Port = port
	case "server.read_timeout":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Server.ReadTimeout = v
	case "server.write_timeout":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Server.WriteTimeout = v
	case "server.enable_cors":
		c.Server.EnableCORS = value == "true" || value == "1"
	case "server.static_dir":
		c.Server.StaticDir = value
	case "storage.type":
		c.Storage.Type = value
	case "storage.data_dir":
		c.Storage.DataDir = value
	case "storage.upload_dir":
		c.Storage.UploadDir = value
	case "firmware.max_file_size":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		c.Firmware.MaxFileSize = v
	case "firmware.allowed_exts":
		c.Firmware.AllowedExts = value
	case "firmware.require_md5":
		c.Firmware.RequireMD5 = value == "true" || value == "1"
	case "log.level":
		c.Log.Level = value
	case "log.output":
		c.Log.Output = value
	case "log.file_path":
		c.Log.FilePath = value
	case "grayscale.default_ratio":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		c.Grayscale.DefaultRatio = v
	case "grayscale.max_ratio":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		c.Grayscale.MaxRatio = v
	case "grayscale.enable_auto_rollback":
		c.Grayscale.EnableAutoRollback = value == "true" || value == "1"
	case "grayscale.min_ratio":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		c.Grayscale.MinRatio = v
	case "grayscale.rollback_threshold":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Grayscale.RollbackThreshold = v
	case "grayscale.max_concurrent_tasks":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Grayscale.MaxConcurrentTasks = v
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}

// Get 获取配置值（线程安全）
//
// 返回当前配置的快照副本，调用方持有快照期间不受热加载影响。
func (c *Config) Get() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshotLocked()
}

// snapshotLocked 在已持有读锁的前提下返回配置的快照副本。
// 调用方必须在外层持有 c.mu 的读锁。
func (c *Config) snapshotLocked() *Config {
	snap := &Config{
		Server:    c.Server,
		Storage:   c.Storage,
		Firmware:  c.Firmware,
		Log:       c.Log,
		Grayscale: c.Grayscale,
	}
	return snap
}

// GetStorageConfig 获取存储配置（线程安全，返回一致性快照）
func (c *Config) GetStorageConfig() StorageConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Storage
}

// GetFirmwareConfig 获取固件配置（线程安全，返回一致性快照）
func (c *Config) GetFirmwareConfig() FirmwareConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Firmware
}

// GetGrayscaleConfig 获取灰度配置（线程安全，返回一致性快照）
func (c *Config) GetGrayscaleConfig() GrayscaleConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Grayscale
}

// GetConfigSnapshot 获取配置快照（线程安全，返回一致性快照）
//
// 返回的三个字段来自同一次热加载后的配置，避免出现字段分属不同配置版本的情况。
func (c *Config) GetConfigSnapshot() (maxSize int64, requireMD5 bool, uploadDir string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Firmware.MaxFileSize, c.Firmware.RequireMD5, c.Storage.UploadDir
}

// ReloadConfig 重新加载配置（从文件读取后在写锁保护下原子合并到当前配置）
//
// 整个合并过程持有写锁，读取方要么看到合并前的完整配置，要么看到合并后的完整配置，
// 不会观察到字段被部分更新的中间状态。
func (c *Config) ReloadConfig(filePath string) error {
	newCfg, err := LoadFromFile(filePath)
	if err != nil {
		return fmt.Errorf("reload config: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if newCfg.Server.Host != "" {
		c.Server.Host = newCfg.Server.Host
	}
	if newCfg.Server.Port > 0 {
		c.Server.Port = newCfg.Server.Port
	}
	if newCfg.Storage.Type != "" {
		c.Storage.Type = newCfg.Storage.Type
	}
	if newCfg.Storage.DataDir != "" {
		c.Storage.DataDir = newCfg.Storage.DataDir
	}
	if newCfg.Storage.UploadDir != "" {
		c.Storage.UploadDir = newCfg.Storage.UploadDir
	}
	if newCfg.Firmware.MaxFileSize > 0 {
		c.Firmware.MaxFileSize = newCfg.Firmware.MaxFileSize
	}
	if newCfg.Firmware.AllowedExts != "" {
		c.Firmware.AllowedExts = newCfg.Firmware.AllowedExts
	}
	// 布尔字段无法用零值表示“不覆盖”，必须无条件应用加载值，否则 false 永远无法生效。
	c.Firmware.RequireMD5 = newCfg.Firmware.RequireMD5
	if newCfg.Log.Level != "" {
		c.Log.Level = newCfg.Log.Level
	}
	if newCfg.Grayscale.DefaultRatio > 0 {
		c.Grayscale.DefaultRatio = newCfg.Grayscale.DefaultRatio
	}
	if newCfg.Grayscale.MaxRatio > 0 {
		c.Grayscale.MaxRatio = newCfg.Grayscale.MaxRatio
	}
	return nil
}

// ValidateConfigConsistency 校验配置一致性（线程安全，基于一致性快照校验）
func (c *Config) ValidateConfigConsistency() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	maxSize := c.Firmware.MaxFileSize
	uploadDir := c.Storage.UploadDir
	grayscaleRatio := c.Grayscale.DefaultRatio
	requireMD5 := c.Firmware.RequireMD5
	maxRatio := c.Grayscale.MaxRatio

	if maxSize <= 0 {
		return fmt.Errorf("config inconsistent: max file size is zero")
	}
	if uploadDir == "" {
		return fmt.Errorf("config inconsistent: upload dir is empty")
	}
	if grayscaleRatio < 0 || grayscaleRatio > maxRatio {
		return fmt.Errorf("config inconsistent: grayscale ratio out of range")
	}
	_ = requireMD5
	return nil
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Firmware.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}

	if c.Grayscale.MaxRatio < c.Grayscale.MinRatio {
		return fmt.Errorf("max ratio must be >= min ratio")
	}

	if c.Grayscale.DefaultRatio < c.Grayscale.MinRatio || c.Grayscale.DefaultRatio > c.Grayscale.MaxRatio {
		return fmt.Errorf("default ratio must be between min and max")
	}

	return nil
}

// IsAllowedExt 检查文件扩展名是否在允许列表中
func (c *Config) IsAllowedExt(ext string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ext = strings.ToLower(ext)
	allowed := strings.Split(c.Firmware.AllowedExts, ",")
	for _, a := range allowed {
		if strings.TrimSpace(strings.ToLower(a)) == ext {
			return true
		}
	}
	return false
}
