package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
)

// NewAPIHandler 返回 /api/ 区域专用的 handler。
// 该区域的所有响应（含错误）都是 JSON，不产生 HTML 错误页或纯文本响应。
func NewAPIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/", handleAPINotFound)
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
}

func handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found: " + r.URL.Path})
}

// NewHandler 返回根 handler：/api/ 前缀交给 API 分区，其余路径交给内嵌前端静态资源。
// 两支各自独立构造，根 mux 按路径前缀分派，因此 /api/ 下的请求不会落到文件服务上。
func NewHandler(assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", NewAPIHandler())
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return mux
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
