// Package handler 提供 HTTP 请求处理器
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/response"
)

// Router 路由注册器
type Router struct {
	mux    *http.ServeMux
	config *config.Config
}

// NewRouter 创建路由注册器
func NewRouter(cfg *config.Config) *Router {
	return &Router{
		mux:    http.NewServeMux(),
		config: cfg,
	}
}

// Handler 获取 HTTP Handler
func (r *Router) Handler() http.Handler {
	var handler http.Handler = r.mux

	// 添加中间件
	handler = r.loggingMiddleware(handler)
	handler = r.corsMiddleware(handler)
	handler = r.recoveryMiddleware(handler)

	return handler
}

// HandleFunc 注册路由处理函数
func (r *Router) HandleFunc(pattern string, handlerFunc http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handlerFunc)
}

// loggingMiddleware 日志中间件
func (r *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以获取状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 处理请求
		next.ServeHTTP(wrapped, req)

		// 记录日志
		duration := time.Since(start)
		logger.Debug("HTTP request",
			"method", req.Method,
			"path", req.URL.Path,
			"status", wrapped.statusCode,
			"duration", duration,
			"remote_addr", req.RemoteAddr,
		)
	})
}

// corsMiddleware CORS 中间件
func (r *Router) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.config.Server.EnableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// 预检请求直接返回
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

// recoveryMiddleware 恢复中间件
func (r *Router) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered", "error", err, "path", req.URL.Path)
				response.InternalError(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, req)
	})
}

// responseWriter 包装 http.ResponseWriter 以获取状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写 WriteHeader 以捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ParsePathID 从路径中解析 ID
func ParsePathID(path, prefix string) (int64, error) {
	path = strings.TrimPrefix(path, prefix)
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.ParseInt(parts[0], 10, 64)
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// ReadJSON 读取 JSON 请求体
func ReadJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// GetIDFromPath 从路径获取最后一段作为 ID
func GetIDFromPath(r *http.Request) (string, error) {
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("empty path")
	}
	return parts[len(parts)-1], nil
}

// GetIntIDFromPath 从路径获取整型 ID
func GetIntIDFromPath(r *http.Request) (int64, error) {
	idStr, err := GetIDFromPath(r)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(idStr, 10, 64)
}
