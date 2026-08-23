// Package response 提供统一的 HTTP 响应格式封装。
// 所有 API 接口应使用此包的函数返回标准化的 JSON 响应。
package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"fwupgrade/internal/config"
	"fwupgrade/internal/store"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PageData 分页数据结构
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 返回成功响应
func Success(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessMessage 返回带消息的成功响应
func SuccessMessage(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Created 返回创建成功响应
func Created(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// NoContent 返回无内容响应
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// BadRequest 返回请求错误
func BadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, Response{
		Code:    400,
		Message: "bad request",
		Error:   message,
	})
}

// BadRequestWithError 返回带详细错误的请求错误
func BadRequestWithError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, Response{
		Code:    400,
		Message: "bad request",
		Error:   err.Error(),
	})
}

// NotFound 返回资源不存在
func NotFound(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusNotFound, Response{
		Code:    404,
		Message: "not found",
		Error:   message,
	})
}

// Conflict 返回冲突错误
func Conflict(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusConflict, Response{
		Code:    409,
		Message: "conflict",
		Error:   message,
	})
}

// InternalError 返回服务器内部错误
func InternalError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, Response{
		Code:    500,
		Message: "internal error",
		Error:   message,
	})
}

// InternalErrorWithError 返回带错误详情的服务器内部错误
func InternalErrorWithError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, Response{
		Code:    500,
		Message: "internal error",
		Error:   err.Error(),
	})
}

// WriteError 按错误类型将 service 错误映射为合适的 HTTP 响应。
// 已知的可识别错误类型（not found / conflict / 配置文件缺失）返回对应语义的状态码，
// 其余一律按 500 内部错误处理。避免"出任何错都 500、只能笼统提示"的问题。
//
// 映射规则：
//   - store.ErrNotFound / config.ErrConfigFileNotFound -> 404
//   - store.ErrConflict                              -> 409
//   - 其它                                          -> 500
func WriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, config.ErrConfigFileNotFound):
		NotFound(w, err.Error())
	case errors.Is(err, store.ErrConflict):
		Conflict(w, err.Error())
	default:
		InternalError(w, err.Error())
	}
}

// Paginated 返回分页成功响应
func Paginated(w http.ResponseWriter, list interface{}, total int64, page, pageSize int) {
	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// writeJSON 将响应写入 HTTP 响应
func writeJSON(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// DecodeBody 从请求体解析 JSON 到目标结构体
func DecodeBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// GetQueryMap 获取所有查询参数
func GetQueryMap(r *http.Request) map[string][]string {
	return r.URL.Query()
}

// GetIntParam 从查询参数获取整型参数
func GetIntParam(r *http.Request, key string, defaultVal int) int {
	q := r.URL.Query()
	val := q.Get(key)
	if val == "" {
		return defaultVal
	}
	var result int
	_, err := fmt.Sscanf(val, "%d", &result)
	if err != nil {
		return defaultVal
	}
	return result
}

// GetStringParam 从查询参数获取字符串参数
func GetStringParam(r *http.Request, key, defaultVal string) string {
	q := r.URL.Query()
	val := q.Get(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetBoolParam 从查询参数获取布尔参数
func GetBoolParam(r *http.Request, key string, defaultVal bool) bool {
	q := r.URL.Query()
	val := q.Get(key)
	if val == "" {
		return defaultVal
	}
	return val == "true" || val == "1"
}

// GetInt64Param 从查询参数获取 int64 参数
func GetInt64Param(r *http.Request, key string, defaultVal int64) int64 {
	q := r.URL.Query()
	val := q.Get(key)
	if val == "" {
		return defaultVal
	}
	var result int64
	_, err := fmt.Sscanf(val, "%d", &result)
	if err != nil {
		return defaultVal
	}
	return result
}
