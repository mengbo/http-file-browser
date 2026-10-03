package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
)

// 机器可读的错误标识（design D6）。
// 目录浏览是第一个需要按失败原因分支的消费者，因此错误信封携带 code 而非单一说明文本。
const (
	codeNotFound         = "not_found"
	codeNotADirectory    = "not_a_directory"
	codePermissionDenied = "permission_denied"
	codeOutsideRoot      = "outside_root"
)

// codeStatus 把错误标识映射到 HTTP 状态码。
// outside_root 用 400 而非 403：越界是请求路径本身不合法，不是身份受限。
var codeStatus = map[string]int{
	codeNotFound:         http.StatusNotFound,
	codeNotADirectory:    http.StatusBadRequest,
	codePermissionDenied: http.StatusForbidden,
	codeOutsideRoot:      http.StatusBadRequest,
}

// browser 持有命令行指定的根目录，目录列表端点以它为唯一的越界判定基准。
type browser struct {
	root string
}

// NewAPIHandler 返回 /api/ 区域专用的 handler。root 是已校验的根目录绝对路径。
// 该区域的所有响应（含错误）都是 JSON，不产生 HTML 错误页或纯文本响应。
func NewAPIHandler(root string) http.Handler {
	b := &browser{root: root}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", b.handleHealth)
	mux.HandleFunc("/api/list", b.handleList)
	mux.HandleFunc("/api/", b.handleAPINotFound)
	return mux
}

type healthResponse struct {
	Status string `json:"status"`
	// Root 让前端能显示绝对位置：浏览位置在 URL 与响应中都是相对根目录的路径。
	Root string `json:"root"`
}

func (b *browser) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Root: b.root})
}

func (b *browser) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, fail(codeNotFound, "未找到接口："+r.URL.Path))
}

// NewHandler 返回根 handler：/api/ 前缀交给 API 分区，其余路径交给内嵌前端静态资源。
// 两支各自独立构造，根 mux 按路径前缀分派，因此 /api/ 下的请求不会落到文件服务上。
func NewHandler(root string, assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", NewAPIHandler(root))
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return mux
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, apiErr *apiError) {
	status, ok := codeStatus[apiErr.code]
	if !ok {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, errorResponse{Error: errorDetail{Code: apiErr.code, Message: apiErr.message}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
