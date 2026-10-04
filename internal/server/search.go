package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// searchMatch 是命中条目的瘦条目（add-file-search design D4）：搜索是「定位」不是
// 「展示」，只带定位所需的名称、类型与完整路径；size/mtime 的取值规则是
// directory-browsing 的 Entry metadata 的领地，不在此复述也不交叉引用。
type searchMatch struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Path 是相对根目录的完整路径而非相对基准的短路径：点击命中即导航，
	// 客户端无需再拼基准（add-file-search design D2）。
	Path string `json:"path"`
}

type searchResponse struct {
	// Path 是规范化后的基准位置相对根目录路径，基准为根目录时是空字符串。
	Path string `json:"path"`
	// Query 是回显的查询词：响应自描述，与 list 回显规范化路径同一动机（design D4）。
	Query   string        `json:"query"`
	Matches []searchMatch `json:"matches"`
}

func (b *browser) handleSearch(w http.ResponseWriter, r *http.Request) {
	result, apiErr := b.search(r.URL.Query().Get("path"), r.URL.Query().Get("q"))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// search 在基准位置 rel 的子树内按名称递归匹配，rel 为空表示根目录。
//
// 基准位置自身的失败不走降级：resolve 的字面 + 物理越界判定照常把门，Stat 与首层
// ReadDir 的失败经 classify 落到既有错误码，零新增词汇（add-file-search design D7）。
// 查询词折叠一次供全程复用；缺失或为空的查询折叠结果为空串，Contains(name, "") 恒真，
// 空查询自然退化为无过滤的递归列表，实现零特判（design D3）。
func (b *browser) search(rel, query string) (*searchResponse, *apiError) {
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

	folded := strings.ToLower(query)
	matches := make([]searchMatch, 0)
	// 基准目录读取失败按基准路径失败分类，不静默降级；子树内的失败才由
	// searchTree 内部跳过（design D7：基准与子树的失败语义不同）。
	if err := b.searchTree(abs, normalized, folded, &matches); err != nil {
		return nil, classify(err)
	}
	return &searchResponse{Path: normalized, Query: query, Matches: matches}, nil
}

// searchTree 手写深度优先遍历（add-file-search design D5）：当前目录 os.ReadDir 后经
// sortEntries 恢复与浏览该目录一致的列表顺序；先一段输出本目录全部命中，再二段按同一
// 目录顺序依次递归每个子目录子树。两段而不是一个循环边输出边递归：spec 的树序要求
// 本目录的命中全部先于任何子树命中，单循环会让先排到的子目录把它的子树命中插进来。
// 不用 filepath.WalkDir：它按纯字典序访问条目且自行决定下钻时机，表达不出这套顺序契约。
//
// 目录判定用 DirEntry.IsDir（lstat 语义）：符号链接天然不是目录、不被下钻（design D6），
// 软链条目自身按名字参与匹配，悬空软链无 Info/Stat 可失败、照常命中——本函数从头到尾
// 不取条目元信息，正是瘦条目（D4）让遍历天然免疫这些边角。
//
// 子树内读取失败静默跳过：该子目录所属子树不产生命中，其余命中照常（design D7）；
// 错误只向上返回到基准目录那一层，由调用方 classify。
func (b *browser) searchTree(dir, rel, foldedQuery string, matches *[]searchMatch) error {
	items, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	entries := make([]listEntry, 0, len(items))
	for _, item := range items {
		entry := listEntry{Name: item.Name(), Type: entryFile}
		if item.IsDir() {
			entry.Type = entryDirectory
		}
		entries = append(entries, entry)
	}
	sortEntries(entries)

	for _, entry := range entries {
		// 命中条目的 path 是相对根目录的完整路径（design D2）；rel 与条目名都已规范化，
		// path.Join 兼顾根目录 rel 为空串的情形。
		childRel := path.Join(rel, entry.Name)
		// 只以条目自身名称匹配（design D3）：大小写折叠与 sortEntries 的排序折叠同源。
		if strings.Contains(strings.ToLower(entry.Name), foldedQuery) {
			*matches = append(*matches, searchMatch{Name: entry.Name, Type: entry.Type, Path: childRel})
		}
	}

	// 第二段才递归：本目录的命中已全部输出，子树命中按目录顺序依次接在后面。
	for _, entry := range entries {
		if entry.Type != entryDirectory {
			continue
		}
		// 子树内的失败按局部降级处理：跳过该子目录，其余照常（design D7）。
		_ = b.searchTree(filepath.Join(dir, entry.Name), path.Join(rel, entry.Name), foldedQuery, matches)
	}
	return nil
}
