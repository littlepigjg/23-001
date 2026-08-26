package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration 解析持续时间字符串（支持 s/m/h/d 后缀）
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// 尝试 Go 标准库解析 (如 "30s", "1h30m")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// 扩展格式：支持 "d" (天)
	var total time.Duration
	var currentNum strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' || ch == '.' {
			currentNum.WriteRune(ch)
			continue
		}
		if currentNum.Len() == 0 {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		val, err := strconv.ParseFloat(currentNum.String(), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration number: %s", currentNum.String())
		}
		currentNum.Reset()

		switch ch {
		case 'd', 'D':
			total += time.Duration(val * 24 * float64(time.Hour))
		case 'h', 'H':
			total += time.Duration(val * float64(time.Hour))
		case 'm', 'M':
			total += time.Duration(val * float64(time.Minute))
		case 's', 'S':
			total += time.Duration(val * float64(time.Second))
		default:
			return 0, fmt.Errorf("unknown duration unit: %c", ch)
		}
	}

	return total, nil
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

// NowUTC 获取当前UTC时间
func NowUTC() time.Time {
	return time.Now().UTC()
}
