// Package logger 提供结构化日志工具，基于 Go 标准库实现。
// 支持日志级别（DEBUG/INFO/WARN/ERROR/FATAL）和关键字段的结构化输出。
package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level 定义日志级别类型
type Level int

const (
	// LevelDebug 调试级别
	LevelDebug Level = iota
	// LevelInfo 信息级别
	LevelInfo
	// LevelWarn 警告级别
	LevelWarn
	// LevelError 错误级别
	LevelError
	// LevelFatal 致命错误级别
	LevelFatal
)

// String 返回日志级别字符串表示
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger 是结构化日志记录器
type Logger struct {
	mu       sync.Mutex
	level    Level
	output   io.Writer
	prefix   string
	fields   map[string]interface{}
	fieldMu  sync.RWMutex
}

// defaultLogger 全局默认日志实例
var defaultLogger = NewLogger(LevelInfo, os.Stdout)

// NewLogger 创建一个新的日志记录器
func NewLogger(level Level, output io.Writer) *Logger {
	return &Logger{
		level:  level,
		output: output,
		fields: make(map[string]interface{}),
	}
}

// Default 返回全局默认日志实例
func Default() *Logger {
	return defaultLogger
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithField 添加一个默认字段（并发安全）
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.fieldMu.Lock()
	defer l.fieldMu.Unlock()
	l.fields[key] = value
	return l
}

// WithFields 添加多个默认字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.fieldMu.Lock()
	defer l.fieldMu.Unlock()
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

// Debug 输出调试日志
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info 输出信息日志
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error 输出错误日志
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal 输出致命错误日志并退出程序
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
	os.Exit(1)
}

// Debugf 格式化输出调试日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LevelDebug, fmt.Sprintf(format, args...))
}

// Infof 格式化输出信息日志
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...))
}

// Warnf 格式化输出警告日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, fmt.Sprintf(format, args...))
}

// Errorf 格式化输出错误日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...))
}

// Fatalf 格式化输出致命错误日志
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(LevelFatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// log 执行实际的日志输出
func (l *Logger) log(level Level, msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	l.fieldMu.RLock()
	mergedFields := make(map[string]interface{}, len(l.fields)+len(args)/2)
	for k, v := range l.fields {
		mergedFields[k] = v
	}
	l.fieldMu.RUnlock()

	// 将 args 视为 key-value 对
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		mergedFields[key] = args[i+1]
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")

	var fieldStr string
	for k, v := range mergedFields {
		fieldStr += fmt.Sprintf(" %s=%v", k, v)
	}

	line := fmt.Sprintf("[%s] %s: %s%s\n", timestamp, level, msg, fieldStr)
	fmt.Fprint(l.output, line)
}

// 包级别便捷函数

// Debug 调试日志
func Debug(msg string, args ...interface{}) { defaultLogger.Debug(msg, args...) }

// Info 信息日志
func Info(msg string, args ...interface{}) { defaultLogger.Info(msg, args...) }

// Warn 警告日志
func Warn(msg string, args ...interface{}) { defaultLogger.Warn(msg, args...) }

// Error 错误日志
func Error(msg string, args ...interface{}) { defaultLogger.Error(msg, args...) }

// Fatal 致命错误日志
func Fatal(msg string, args ...interface{}) { defaultLogger.Fatal(msg, args...) }

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) { defaultLogger.Debugf(format, args...) }

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) { defaultLogger.Infof(format, args...) }

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) { defaultLogger.Warnf(format, args...) }

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) { defaultLogger.Errorf(format, args...) }

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) { defaultLogger.Fatalf(format, args...) }

// SetLevel 设置默认日志级别
func SetLevel(level Level) { defaultLogger.SetLevel(level) }

// WithField 为默认日志添加字段
func WithField(key string, value interface{}) *Logger { return defaultLogger.WithField(key, value) }
