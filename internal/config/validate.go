package config

import (
	"fmt"
	"strings"
)

// ValidatePortRange 验证端口范围
func ValidatePortRange(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	return nil
}

// ValidateConfigValid 验证配置完整性
func ValidateConfigValid(c *Config) error {
	if err := ValidatePortRange(c.Server.Port); err != nil {
		return err
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

// ValidateExtList 验证文件扩展名列表
func ValidateExtList(exts string) []string {
	allowed := strings.Split(exts, ",")
	var result []string
	for _, ext := range allowed {
		ext = strings.TrimSpace(strings.ToLower(ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
	}
	result = append(result, ext)
	}
	return result
}

// IsValidLogLevel 验证日志级别是否合法
func IsValidLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error", "fatal":
		return true
	default:
		return false
	}
}
