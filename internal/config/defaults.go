package config

// 默认值常量
const (
	// 默认服务器配置
	DefaultHost         = "0.0.0.0"
	DefaultPort         = 8080
	DefaultReadTimeout  = 30
	DefaultWriteTimeout = 30
	DefaultIdleTimeout  = 120
	DefaultShutdownTimeout = 10
	DefaultMaxHeaderBytes = 1 << 20
	DefaultEnableCORS   = true
	DefaultStaticDir    = "web"

	// 默认存储配置
	DefaultStorageType  = "memory"
	DefaultDataDir      = "data"
	DefaultUploadDir    = "uploads"

	// 默认固件配置
	DefaultMaxFileSize  = 100 * 1024 * 1024 // 100MB
	DefaultAllowedExts  = ".bin,.hex,.img,.firmware,.fw"
	DefaultBackupDir    = "backups"
	DefaultKeepVersions = 10
	DefaultMaxVersionAge = 365

	// 默认日志配置
	DefaultLogLevel      = "info"
	DefaultLogFormat     = "text"
	DefaultLogOutput     = "stdout"
	DefaultLogFilePath   = "logs/app.log"
	DefaultLogMaxSize    = 100
	DefaultLogMaxBackups = 3

	// 默认灰度配置
	DefaultGrayscaleRatio          = 10.0
	DefaultMaxRatio                = 100.0
	DefaultMinRatio                = 0.0
	DefaultRatioIncrement          = 10.0
	DefaultUpgradeInterval         = 30
	DefaultProgressUpdateInterval  = 5
	DefaultRollbackThreshold       = 20
	DefaultMaxConcurrentTasks      = 10
	DefaultDevicePollInterval      = 60
)

// NewDefaultServerConfig 返回默认服务器配置
func NewDefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:            DefaultHost,
		Port:            DefaultPort,
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,
		IdleTimeout:     DefaultIdleTimeout,
		ShutdownTimeout: DefaultShutdownTimeout,
		MaxHeaderBytes:  DefaultMaxHeaderBytes,
		EnableCORS:      DefaultEnableCORS,
		StaticDir:       DefaultStaticDir,
	}
}

// NewDefaultStorageConfig 返回默认存储配置
func NewDefaultStorageConfig() StorageConfig {
	return StorageConfig{
		Type:      DefaultStorageType,
		DataDir:   DefaultDataDir,
		UploadDir: DefaultUploadDir,
	}
}

// NewDefaultFirmwareConfig 返回默认固件配置
func NewDefaultFirmwareConfig() FirmwareConfig {
	return FirmwareConfig{
		MaxFileSize:    DefaultMaxFileSize,
		AllowedExts:    DefaultAllowedExts,
		AutoBackup:     false,
		BackupDir:      DefaultBackupDir,
		KeepVersions:   DefaultKeepVersions,
		RequireMD5:     true,
		MaxVersionAge:  DefaultMaxVersionAge,
	}
}

// NewDefaultLogConfig 返回默认日志配置
func NewDefaultLogConfig() LogConfig {
	return LogConfig{
		Level:      DefaultLogLevel,
		Format:     DefaultLogFormat,
		Output:     DefaultLogOutput,
		FilePath:   DefaultLogFilePath,
		MaxSize:    DefaultLogMaxSize,
		MaxBackups: DefaultLogMaxBackups,
	}
}

// NewDefaultGrayscaleConfig 返回默认灰度配置
func NewDefaultGrayscaleConfig() GrayscaleConfig {
	return GrayscaleConfig{
		DefaultRatio:          DefaultGrayscaleRatio,
		MaxRatio:              DefaultMaxRatio,
		MinRatio:              DefaultMinRatio,
		RatioIncrement:        DefaultRatioIncrement,
		UpgradeInterval:       DefaultUpgradeInterval,
		ProgressUpdateInterval: DefaultProgressUpdateInterval,
		EnableAutoRollback:    true,
		RollbackThreshold:     DefaultRollbackThreshold,
		MaxConcurrentTasks:    DefaultMaxConcurrentTasks,
		DevicePollInterval:    DefaultDevicePollInterval,
	}
}
