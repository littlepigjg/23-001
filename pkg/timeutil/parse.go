package timeutil

import (
	"time"
)

// ParseRFC3339 解析RFC3339格式时间
func ParseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// Sleep 睡眠指定时间
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// UntilNext 计算距离下一个指定时刻的持续时间
func UntilNext(hour, minute int) time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
