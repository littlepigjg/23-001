package logger

import (
	"fmt"
	"strings"
	"time"
)

// TimestampFormat 时间戳格式常量
const (
	// TimestampRFC3339 RFC3339 格式
	TimestampRFC3339 = "2006-01-02T15:04:05.000Z07:00"
	// TimestampSimple 简单日期时间格式
	TimestampSimple = "2006-01-02 15:04:05"
)

// FormatTimestamp 格式化时间戳
func FormatTimestamp(t time.Time) string {
	return t.Format(TimestampRFC3339)
}

// FormatDuration 格式化持续时间
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// FormatFields 格式化字段为字符串
func FormatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}
	var parts []string
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, " ")
}

// PadLevel 对其日志级别字符串
func PadLevel(level Level) string {
	s := level.String()
	padLen := 5 - len(s)
	if padLen > 0 {
		s += strings.Repeat(" ", padLen)
	}
	return s
}
