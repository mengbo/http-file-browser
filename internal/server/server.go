package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
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

// codeNotAnImage 是图片内容端点独有的失败原因（image-preview design D2）：
// 名字的扩展名不在图片白名单内即拒绝，不读内容、不验魔数；无扩展名不嗅探。
const (
	// codeNotAnImage 是路径存在、可读、但不被视为图片文件的文件。
	codeNotAnImage = "not_an_image"
)

// codeStatus 把错误标识映射到 HTTP 状态码。
// outside_root 用 400 而非 403：越界是请求路径本身不合法，不是身份受限。
// 其余 400 的各条描述的都是「这个请求的目标给不出所请求的呈现」，不是身份问题。
var codeStatus = map[string]int{
	codeNotFound:         http.StatusNotFound,
	codeNotADirectory:    http.StatusBadRequest,
	codePermissionDenied: http.StatusForbidden,
	codeOutsideRoot:      http.StatusBadRequest,
	codeNotText:          http.StatusBadRequest,
	codeTooLarge:         http.StatusBadRequest,
	codeNotARegularFile:  http.StatusBadRequest,
	codeNotAnImage:       http.StatusBadRequest,
}

// browser 持有命令行指定的根目录，目录列表端点以它为唯一的越界判定基准。
type browser struct {
	root string
	// physicalRoot 是根目录解析全部符号链接后的物理位置（improve-root-confinement
	// design D2）：启动时解析一次，此后每个请求不再重复解析根，越界的物理判定以它
	// 为基准。构造失败（根不可解析）则无边界可言，服务不得启动。
	physicalRoot string
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
// 根目录的物理位置在此解析一次（improve-root-confinement design D2）；根不可解析时
// 返回错误，由启动路径报错退出——根没有物理边界，越界判定就无从谈起。
func newBrowser(root string) (*browser, error) {
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("根目录 %q 无法解析物理位置：%w", root, err)
	}
	return &browser{
		root:         root,
		physicalRoot: physicalRoot,
		entryInfo: func(entry fs.DirEntry) (fs.FileInfo, error) {
			return entry.Info()
		},
	}, nil
}

// NewAPIHandler 返回 /api/ 区域专用的 handler。root 是已校验的根目录绝对路径。
// 该区域的所有响应（含错误）都是 JSON，不产生 HTML 错误页或纯文本响应。
// 根目录无法解析物理位置时返回错误，调用方应以启动错误处理，不得用 nil handler 继续服务。
func NewAPIHandler(root string) (http.Handler, error) {
	b, err := newBrowser(root)
	if err != nil {
		return nil, err
	}
	return apiHandler(b), nil
}

// apiHandler 按给定 browser 装配路由。生产路径走 NewAPIHandler，
// 包内测试可自建 browser 替换 entryInfo 后调用它（design D9 的 seam）。
func apiHandler(b *browser) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", b.handleHealth)
	mux.HandleFunc("/api/list", b.handleList)
	// 内容走独立端点，列表响应继续不携带任何条目内容（design D1）。
	mux.HandleFunc("/api/content", b.handleContent)
	// 图片内容端点：/api/ 分区里唯一的非 JSON 成功响应（service-startup MODIFIED 的开口，
	// image-preview design D1）；错误响应仍走 JSON 信封。
	mux.HandleFunc("/api/image", b.handleImage)
	// 搜索端点：基准位置子树内按名称递归定位条目（add-file-search）。
	mux.HandleFunc("/api/search", b.handleSearch)
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
// 根目录无法解析物理位置时返回错误，由启动路径报错退出。
func NewHandler(root string, assets fs.FS) (http.Handler, error) {
	api, err := NewAPIHandler(root)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return mux, nil
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
