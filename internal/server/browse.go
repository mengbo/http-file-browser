package server

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// 条目类型取值。
const (
	entryDirectory = "directory"
	entryFile      = "file"
)

type listEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Size 是条目自身占用的字节数；符号链接条目取链接自身的长度而非目标内容的大小。
	// 目录条目与元信息不可得的条目不出现该字段（design D5：缺省而非 null）。
	Size *int64 `json:"size,omitempty"`
	// ModifiedAt 是自 Unix 纪元起的整秒，不用可读字符串：取值必须与时区和 locale
	// 无关地逐字节可复现（design D6）。元信息不可得时同样不出现该字段。
	ModifiedAt *int64 `json:"modified_at,omitempty"`
}

type listResponse struct {
	// Path 是规范化后的相对根目录路径，根目录为空字符串。
	Path string `json:"path"`
	// Parent 是上级相对路径，根目录为空字符串。
	Parent  string      `json:"parent"`
	Entries []listEntry `json:"entries"`
}

// apiError 是带机器可读错误标识的失败。
type apiError struct {
	code    string
	message string
}

func (e *apiError) Error() string { return e.message }

func fail(code, message string) *apiError {
	return &apiError{code: code, message: message}
}

func (b *browser) handleList(w http.ResponseWriter, r *http.Request) {
	result, apiErr := b.list(r.URL.Query().Get("path"))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// list 返回相对路径 rel 所指目录的列表，rel 为空表示根目录。
func (b *browser) list(rel string) (*listResponse, *apiError) {
	abs, normalized, apiErr := b.resolve(rel)
	if apiErr != nil {
		return nil, apiErr
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, classify(err)
	}
	if !info.IsDir() {
		return nil, fail(codeNotADirectory, "目标不是目录")
	}

	items, err := os.ReadDir(abs)
	if err != nil {
		return nil, classify(err)
	}

	entries := make([]listEntry, 0, len(items))
	for _, item := range items {
		entry := listEntry{Name: item.Name(), Type: entryFile}
		if item.IsDir() {
			entry.Type = entryDirectory
		}
		b.fillMetadata(&entry, item)
		entries = append(entries, entry)
	}
	sortEntries(entries)

	return &listResponse{Path: normalized, Parent: parentOf(normalized), Entries: entries}, nil
}

// fillMetadata 用 b.entryInfo 补上条目的大小与最后修改时间（design D4、D5、D6、D7）。
//
// 取数走 DirEntry.Info()（lstat 语义）而不是 os.Stat：后者跟随符号链接，会让根目录内
// 一个指向外部的软链报出根目录之外那个目标的真实大小，而同一响应里的越界判定还在说
// 「这在根目录内」。检测不是解析，因此 lstat 不触碰 design D3 的任何承诺。
//
// 目录条目只给修改时间、不给大小。取数失败时两个字段都省略，但条目本身保留——
// DirEntry.Info 文档明确写了条目可能在 readdir 之后消失，那在构建产物或正在下载的
// 目录里会偶发触发，此时整个列表失败是不可接受的（design D7）。
func (b *browser) fillMetadata(entry *listEntry, item fs.DirEntry) {
	info, err := b.entryInfo(item)
	if err != nil {
		return
	}

	// 直接取 Unix 整秒，不做任何时区或本地化格式化：格式化会让同一个文件在不同
	// 运行环境下返回不同字符串，与「顺序可复现」的同一价值观冲突（design D6）。
	modified := info.ModTime().Unix()
	entry.ModifiedAt = &modified

	if entry.Type != entryDirectory {
		size := info.Size()
		entry.Size = &size
	}
}

// resolve 把相对根目录的浏览位置解析为绝对路径与规范化后的相对路径。
//
// 越界判定分两层（improve-root-confinement design D2）。第一层是字面检查：绝对路径
// 拒绝、Clean/Join、Rel 越界判定，继续拦截 `..` 与兄弟前缀这类字面逃逸。第二层是
// 物理判定：对拼接出的绝对路径执行 EvalSymlinks，与启动时解析好的物理根做 Rel，
// `..` 开头即越界——字面位置在根内、经符号链接指向外部的路径不再提供内容。两侧都
// 解析（根与路径各一次）保证了判定基准一致：`/tmp` → `/private/tmp` 这类环境不会
// 产生假越界。
func (b *browser) resolve(rel string) (abs string, normalized string, apiErr *apiError) {
	// 浏览位置的契约是「相对根目录的路径」，绝对路径不在契约内，直接按越界拒绝，
	// 而不是被 Join 悄悄重解释成根目录下的同名子路径。
	if filepath.IsAbs(filepath.FromSlash(rel)) {
		return "", "", fail(codeOutsideRoot, "浏览位置必须是相对根目录的路径")
	}

	abs = filepath.Join(b.root, filepath.Clean(filepath.FromSlash(rel)))

	// 用 filepath.Rel 判定而不是 strings.HasPrefix：根目录为 a/b 时 a/bc 的字符串前缀
	// 也会匹配，裸前缀判断会把兄弟目录放进根目录内（design D4）。
	fromRoot, err := filepath.Rel(b.root, abs)
	if err != nil || fromRoot == ".." || strings.HasPrefix(fromRoot, ".."+string(filepath.Separator)) {
		return "", "", fail(codeOutsideRoot, "路径超出根目录范围")
	}

	// 物理判定：请求路径解析其全部符号链接后，物理位置落在物理根之内才提供内容。
	if physical, err := filepath.EvalSymlinks(abs); err == nil {
		fromPhysical, relErr := filepath.Rel(b.physicalRoot, physical)
		if relErr != nil || fromPhysical == ".." || strings.HasPrefix(fromPhysical, ".."+string(filepath.Separator)) {
			return "", "", fail(codeOutsideRoot, "路径超出根目录范围")
		}
	}
	// EvalSymlinks 失败（ENOENT / EACCES / ELOOP）时不做越界判定，放行到端点既有的
	// Stat 分类：悬空软链得 not_found，无权限得 permission_denied，成环落入 classify
	// 的 default 分支（improve-root-confinement design D3）。同一组件序列下 Stat 也必然
	// 失败，解析不出来的路径一个字节都提供不出去，回落不重开洞，分类词汇不变。
	//
	// 已知限制（improve-root-confinement design D5）：本判定与端点后续的 Stat/Open 之间
	// 存在 TOCTOU 窗口，本机进程在此窗口内替换符号链接可竞赢判定。本 Change 防御的
	// 对象是远程网络请求，不是本机进程——后者本就能直接读文件系统，服务器不给它任何
	// 增益。内核级封堵（Linux openat2 + RESOLVE_BENEATH）是 Linux-only，与 ADR-0001
	// 的跨平台交叉编译价值冲突，不做。

	normalized = ""
	if fromRoot != "." {
		normalized = filepath.ToSlash(fromRoot)
	}
	return abs, normalized, nil
}

func parentOf(normalized string) string {
	if normalized == "" {
		return ""
	}
	parent := path.Dir(normalized)
	if parent == "." || parent == "/" {
		return ""
	}
	return parent
}

// sortEntries 用 (非目录在前, 大小写折叠后的名称, 原名) 三元组排序（design D5）。
// 大小与最后修改时间不参与排序：同一目录内容未变时排序键必须只由名称决定，
// 否则一个只改了修改时间的文件会在两次请求之间跳位。第三段是必需的：sort.Slice
// 不稳定，缺了它 README/readme 这类仅大小写不同的条目每次请求的相对顺序都会抖动。
//
// 用 strings.ToLower 做简单大小写折叠而非 locale 排序：locale 排序依赖运行环境语言，
// 与「顺序可复现」的 Scenario 冲突。代价是中文名按码位而非拼音排列。
func sortEntries(entries []listEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Type != b.Type {
			return a.Type == entryDirectory
		}
		if folded, foldedB := strings.ToLower(a.Name), strings.ToLower(b.Name); folded != foldedB {
			return folded < foldedB
		}
		return a.Name < b.Name
	})
}

// classify 把文件系统错误映射到机器可读错误标识。
func classify(err error) *apiError {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return fail(codePermissionDenied, "没有读取该位置的权限")
	case errors.Is(err, syscall.ENOTDIR):
		// 路径的中间某段是文件，例如 a/b 中 a 是文件。
		return fail(codeNotADirectory, "目标不是目录")
	case errors.Is(err, fs.ErrNotExist):
		return fail(codeNotFound, "位置不存在")
	default:
		return fail(codeNotFound, "位置不可用")
	}
}
