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

// 内容读取独有的三个失败原因（design D8、D10）。前四个的语义绑定目录列表，
// 由 directory-browsing 的 Directory access failures 承诺，取值一字不改。
const (
	// codeNotText 是路径存在、可读、但不被视为可读文本的文件：名字的扩展名不在白名单内。
	codeNotText = "not_text"
	// codeTooLarge 是超过可提供内容的最大字节数。
	codeTooLarge = "too_large"
	// codeNotARegularFile 是路径存在但不是普通文件（命名管道、Socket、设备）。
	// 它是唯一能避免「打开命名管道永久阻塞」的标识，而先 Stat 后判定让它的成本接近零。
	codeNotARegularFile = "not_a_regular_file"
)

// codeStatus 把错误标识映射到 HTTP 状态码。
// outside_root 用 400 而非 403：越界是请求路径本身不合法，不是身份受限。
// 新增三个同样是 400：它们描述的都是「这个请求的内容端点给不出」，不是身份问题。
var codeStatus = map[string]int{
	codeNotFound:         http.StatusNotFound,
	codeNotADirectory:    http.StatusBadRequest,
	codePermissionDenied: http.StatusForbidden,
	codeOutsideRoot:      http.StatusBadRequest,
	codeNotText:          http.StatusBadRequest,
	codeTooLarge:         http.StatusBadRequest,
	codeNotARegularFile:  http.StatusBadRequest,
}

// browser 持有命令行指定的根目录，目录列表端点以它为唯一的越界判定基准。
type browser struct {
	root string
	// entryInfo 是条目元信息的取数函数，默认 DirEntry.Info（lstat 语义，design D4）。
	//
	// 这是一个显式声明的注入 seam（design D9），不是顺手加的参数：DirEntry.Info
	// 只在「条目在 readdir 之后消失」时失败，那条竞态无法确定性构造，用
	// 「建一堆文件 + 后台删」去撞只会得到 flaky 测试——而 flaky 测试会训练团队忽略红色。
	// 保留这个字段让 Entry metadata 的「元信息不可得」场景有确定性覆盖，
	// 它防的是将来有人把取数错误顺手 continue 掉（丢弃条目）或让整个列表失败。
	entryInfo func(fs.DirEntry) (fs.FileInfo, error)
}

// newBrowser 构造生产路径上的 browser，entryInfo 取 DirEntry.Info。
func newBrowser(root string) *browser {
	return &browser{
		root: root,
		entryInfo: func(entry fs.DirEntry) (fs.FileInfo, error) {
			return entry.Info()
		},
	}
}

// NewAPIHandler 返回 /api/ 区域专用的 handler。root 是已校验的根目录绝对路径。
// 该区域的所有响应（含错误）都是 JSON，不产生 HTML 错误页或纯文本响应。
func NewAPIHandler(root string) http.Handler {
	return apiHandler(newBrowser(root))
}

// apiHandler 按给定 browser 装配路由。生产路径走 NewAPIHandler，
// 包内测试可自建 browser 替换 entryInfo 后调用它（design D9 的 seam）。
func apiHandler(b *browser) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", b.handleHealth)
	mux.HandleFunc("/api/list", b.handleList)
	// 内容走独立端点，列表响应继续不携带任何条目内容（design D1）。
	mux.HandleFunc("/api/content", b.handleContent)
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
