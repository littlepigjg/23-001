// Package timeutil 提供时间处理工具函数
package timeutil

import (
	"time"
)

// TimeFormat 标准时间格式
const TimeFormat = "2006-01-02T15:04:05Z07:00"

// DateFormat 日期格式
const DateFormat = "2006-01-02"

// DateTimeFormat 日期时间格式
const DateTimeFormat = "2006-01-02 15:04:05"

// Now 获取当前时间
func Now() time.Time {
	return time.Now()
}

// NowString 获取当前时间的字符串表示
func NowString() string {
	return time.Now().Format(TimeFormat)
}

// NowDateString 获取当前日期字符串
func NowDateString() string {
	return time.Now().Format(DateFormat)
}

// NowDateTimeString 获取当前日期时间字符串
func NowDateTimeString() string {
	return time.Now().Format(DateTimeFormat)
}

// ParseTime 解析时间字符串
func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeFormat, s)
}

// ParseDate 解析日期字符串
func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateFormat, s)
}

// ParseDateTime 解析日期时间字符串
func ParseDateTime(s string) (time.Time, error) {
	return time.Parse(DateTimeFormat, s)
}

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// FormatDate 格式化日期
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// FormatDateTime 格式化日期时间
func FormatDateTime(t time.Time) string {
	return t.Format(DateTimeFormat)
}

// StartOfDay 获取指定时间的当天开始时间
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取指定时间的当天结束时间
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek 获取指定时间所在周的开始时间（周一）
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysSinceMonday := weekday - 1
	return StartOfDay(t.AddDate(0, 0, -daysSinceMonday))
}

// EndOfWeek 获取指定时间所在周的结束时间（周日）
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth 获取指定时间所在月的开始时间
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取指定时间所在月的结束时间
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, -1)
}

// DaysBetween 计算两个日期之间的天数差
func DaysBetween(t1, t2 time.Time) int {
	duration := t2.Sub(t1)
	return int(duration.Hours() / 24)
}

// HoursBetween 计算两个时间之间的小时差
func HoursBetween(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Hours()
}

// MinutesBetween 计算两个时间之间的分钟差
func MinutesBetween(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Minutes()
}

// IsExpired 检查时间是否已过期
func IsExpired(expireAt time.Time) bool {
	return time.Now().After(expireAt)
}

// IsFuture 检查时间是否在未来
func IsFuture(t time.Time) bool {
	return time.Now().Before(t)
}

// IsPast 检查时间是否在过去
func IsPast(t time.Time) bool {
	return time.Now().After(t)
}

// WithinDuration 检查指定时间是否在给定持续时间内
func WithinDuration(t time.Time, d time.Duration) bool {
	return time.Since(t) < d
}

// AddDuration 返回指定时间加上持续时间
func AddDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

// DaysAgo 返回 N 天前的时间
func DaysAgo(n int) time.Time {
	return time.Now().AddDate(0, 0, -n)
}

// DaysLater 返回 N 天后的时间
func DaysLater(n int) time.Time {
	return time.Now().AddDate(0, 0, n)
}

// HoursAgo 返回 N 小时前的时间
func HoursAgo(n int) time.Time {
	return time.Now().Add(-time.Duration(n) * time.Hour)
}

// MinutesAgo 返回 N 分钟前的时间
func MinutesAgo(n int) time.Time {
	return time.Now().Add(-time.Duration(n) * time.Minute)
}

// UnixToTime 将 Unix 时间戳转换为 time.Time
func UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

// TimeToUnix 将 time.Time 转换为 Unix 时间戳
func TimeToUnix(t time.Time) int64 {
	return t.Unix()
}
