package model

import (
	"fmt"
	"strings"
)

// ValidateVersionFormat 校验版本号格式
func ValidateVersionFormat(version string) error {
	if version == "" {
		return fmt.Errorf("version is required")
	}

	if len(version) > 50 {
		return fmt.Errorf("version too long: %d characters", len(version))
	}

	for i, ch := range version {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '.' ||
			ch == '-' ||
			ch == '/') {
			return fmt.Errorf("version contains invalid character '%c' at position %d", ch, i)
		}
	}

	return nil
}

// SanitizeVersion 清理版本号中的特殊字符
func SanitizeVersion(version string) string {
	if version == "" {
		return version
	}

	sanitized := strings.ReplaceAll(version, " ", "")
	sanitized = strings.ReplaceAll(sanitized, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")

	return sanitized
}
