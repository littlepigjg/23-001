package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// loadDefault 加载默认配置的辅助函数
func loadDefault() *Config {
	return &Config{
		Server:    NewDefaultServerConfig(),
		Storage:   NewDefaultStorageConfig(),
		Firmware:  NewDefaultFirmwareConfig(),
		Log:       NewDefaultLogConfig(),
		Grayscale: NewDefaultGrayscaleConfig(),
	}
}

// LoadFromEnvWithPrefix 从环境变量加载配置（支持前缀）
func LoadFromEnvWithPrefix(prefix string) *Config {
	cfg := loadDefault()

	if v := os.Getenv(prefix + "_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv(prefix + "_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv(prefix + "_STORAGE_TYPE"); v != "" {
		cfg.Storage.Type = v
	}
	if v := os.Getenv(prefix + "_DATA_DIR"); v != "" {
		cfg.Storage.DataDir = v
	}
	if v := os.Getenv(prefix + "_UPLOAD_DIR"); v != "" {
		cfg.Storage.UploadDir = v
	}
	if v := os.Getenv(prefix + "_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv(prefix + "_MAX_FILE_SIZE"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Firmware.MaxFileSize = size
		}
	}

	return cfg
}

// MergeConfig 合并两个配置（覆盖）
func MergeConfig(base, override *Config) *Config {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}
	merged := &Config{
		Server:    base.Server,
		Storage:   base.Storage,
		Firmware:  base.Firmware,
		Log:       base.Log,
		Grayscale: base.Grayscale,
	}
	if override.Server.Host != "" {
		merged.Server.Host = override.Server.Host
	}
	if override.Server.Port != 0 {
		merged.Server.Port = override.Server.Port
	}
	if override.Storage.Type != "" {
		merged.Storage.Type = override.Storage.Type
	}
	if override.Storage.DataDir != "" {
		merged.Storage.DataDir = override.Storage.DataDir
	}
	return merged
}

// ParseBool 解析字符串为布尔值
func ParseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// Describe 输出配置摘要
func Describe(c *Config) string {
	return fmt.Sprintf("host=%s port=%d storage=%s log=%s",
		c.Server.Host, c.Server.Port, c.Storage.Type, c.Log.Level)
}
