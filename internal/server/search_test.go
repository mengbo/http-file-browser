package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func searchURL(base, query string) string {
	return "/api/search?path=" + url.QueryEscape(base) + "&q=" + url.QueryEscape(query)
}

func decodeSearch(t *testing.T, recorder *httptest.ResponseRecorder) searchResponse {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d（body = %q）", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q，期望 JSON", contentType)
	}
	var body searchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, recorder.Body.String())
	}
	return body
}

func matchPaths(matches []searchMatch) []string {
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		paths = append(paths, match.Path)
	}
	return paths
}

// findMatch 按路径取命中条目。断言一律经它取而不是按下标：顺序契约由专门的
// ordering 用例钉住，形状断言不该把顺序写死第二遍。
func findMatch(t *testing.T, matches []searchMatch, path string) searchMatch {
	t.Helper()
	for _, match := range matches {
		if match.Path == path {
			return match
		}
	}
	t.Fatalf("命中列表中没有路径为 %q 的条目，实际为 %v", path, matchPaths(matches))
	return searchMatch{}
}

// rawMatches 把命中列表解成 map 而不是结构体，用来看清字段「是否出现」
// （与 browse_test.go 的 rawEntries 同一动机）。
func rawMatches(t *testing.T, recorder *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var raw struct {
		Matches []map[string]any `json:"matches"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v", err)
	}
	return raw.Matches
}

// topLevelFieldNames 给出响应体顶层实际出现的字段名，按名称排序以便与字面量列表比对。
func topLevelFieldNames(t *testing.T, body string) []string {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, body)
	}
	fields := make([]string, 0, len(raw))
	for field := range raw {
		fields = append(fields, field)
	}
	slices.Sort(fields)
	return fields
}

// --- Result ordering ---

// TestMatchesAreArrangedInTreeOrder 对应 Scenario Matches are arranged in tree order
// 与 Matches within the same directory follow listing order：基准位置自身的命中全部先于
// 任何子树命中，每个子目录子树中的命中连续排列，且目录内顺序与浏览该目录一致
// （目录先于文件、组内按大小写折叠后的名称排序）。
func TestMatchesAreArrangedInTreeOrder(t *testing.T) {
	handler, root := newAPI(t)
	// 两个基准子目录：Other 名字不命中但子树内有两个命中（验证连续性且先于 reports），
	// reports 名字命中、子树内有一个命中。目录内顺序按 sortEntries：目录在前、
	// 文件组内折叠排序（daily < report-a < Report-z）。
	mkdir(t, root, "Other")
	writeFile(t, filepath.Join(root, "Other", "report-draft.md"))
	writeFile(t, filepath.Join(root, "Other", "report-notes.txt"))
	mkdir(t, root, "reports")
	writeFile(t, filepath.Join(root, "reports", "2024-report.txt"))
	writeFile(t, filepath.Join(root, "report-a.txt"))
	writeFile(t, filepath.Join(root, "Report-z.txt"))
	writeFile(t, filepath.Join(root, "daily-report.md"))

	body := decodeSearch(t, get(t, handler, searchURL("", "report")))

	want := []string{
		"reports",                // 基准位置自身的命中：目录条目先于文件条目
		"daily-report.md",        // 文件组内按大小写折叠后的名称排序
		"report-a.txt",
		"Report-z.txt",
		"Other/report-draft.md",  // 子树按目录顺序展开：Other 在 reports 之前，命中连续排列
		"Other/report-notes.txt",
		"reports/2024-report.txt",
	}
	if got := matchPaths(body.Matches); !slices.Equal(got, want) {
		t.Errorf("命中顺序 = %v，期望树序 %v", got, want)
	}
}

// TestRepeatedSearchesReturnTheSameOrder 对应 Scenario Repeated searches return the
// same order：同一基准与查询词的重复请求在子树未变化时顺序完全相同。
func TestRepeatedSearchesReturnTheSameOrder(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "Sub")
	mkdir(t, root, "src")
	writeFile(t, filepath.Join(root, "Sub", "report.md"))
	writeFile(t, filepath.Join(root, "src", "notes.txt"))
	writeFile(t, filepath.Join(root, "TODO"))

	first := decodeSearch(t, get(t, handler, searchURL("", "")))
	second := decodeSearch(t, get(t, handler, searchURL("", "")))

	if !slices.Equal(matchPaths(first.Matches), matchPaths(second.Matches)) {
		t.Errorf("两次请求的命中顺序不一致：%v 与 %v",
			matchPaths(first.Matches), matchPaths(second.Matches))
	}
	if !slices.Equal(first.Matches, second.Matches) {
		t.Errorf("两次请求的命中条目不一致：%v 与 %v", first.Matches, second.Matches)
	}
}

// --- Name matching semantics ---

// TestAQueryIsASubstringOfAName 对应 Scenario A query is a substring of a name。
func TestAQueryIsASubstringOfAName(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "reports.txt"))
	writeFile(t, filepath.Join(root, "summary.md"))

	body := decodeSearch(t, get(t, handler, searchURL("", "port")))

	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"reports.txt"}) {
		t.Errorf("命中 = %v，期望只有名称含连续子串的 [reports.txt]", got)
	}
}

// TestAQueryDiffersFromANameOnlyInLetterCase 对应 Scenario A query differs from a name
// only in letter case：名称比较不区分大小写。
func TestAQueryDiffersFromANameOnlyInLetterCase(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "reports.txt"))

	body := decodeSearch(t, get(t, handler, searchURL("", "REPORTS")))

	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"reports.txt"}) {
		t.Errorf("命中 = %v，期望大小写不同的查询词仍命中 [reports.txt]", got)
	}
}

// TestAQueryAppearsOnlyInTheLocationPath 对应 Scenario A query appears only in the
// location path：只以条目自身名称匹配，所在目录路径不参与。
func TestAQueryAppearsOnlyInTheLocationPath(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "reports")
	writeFile(t, filepath.Join(root, "reports", "summary.txt"))

	body := decodeSearch(t, get(t, handler, searchURL("", "reports")))

	// reports 自身名称命中；summary.txt 的名称不含查询词，尽管其所在路径含 reports，
	// 也不是命中。
	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"reports"}) {
		t.Errorf("命中 = %v，期望只有 [reports]（路径中的查询词不产生命中）", got)
	}
}

// TestQueryIsMissingOrEmpty 对应 Scenario Query is missing or empty：全部条目都是命中。
func TestQueryIsMissingOrEmpty(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "docs", "readme.md"))
	writeFile(t, filepath.Join(root, "top.txt"))

	// 缺失（不带 q 参数）与空值（q=）同语义：空查询退化为无过滤的递归列表。
	// 树序：基准层命中 docs、top.txt 按列表顺序在前，子树命中 docs/readme.md 在后。
	want := []string{"docs", "top.txt", "docs/readme.md"}
	for _, target := range []string{"/api/search", "/api/search?path=", "/api/search?q=", searchURL("", "")} {
		t.Run(target, func(t *testing.T) {
			body := decodeSearch(t, get(t, handler, target))

			if got := matchPaths(body.Matches); !slices.Equal(got, want) {
				t.Errorf("命中 = %v，期望子树内全部条目 %v", got, want)
			}
		})
	}
}

// TestNothingMatches 对应 Scenario Nothing matches：无命中是空列表的成功响应，不是错误。
func TestNothingMatches(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "notes.txt"))

	body := decodeSearch(t, get(t, handler, searchURL("", "no-such-name")))

	if body.Matches == nil {
		t.Error("matches = null，期望空列表而不是错误")
	}
	if len(body.Matches) != 0 {
		t.Errorf("matches = %v，期望空列表", matchPaths(body.Matches))
	}
}

// --- Traversal boundaries ---

// TestASymbolicLinkDirectoryIsMatchedButNotDescended 对应 Scenario A symbolic link
// directory is matched but not descended：软链条目自身按名称参与匹配，指向的子树不产生
// 命中。目标放在根目录之外，其内容只有经软链才可达，命中与否完全由「跟进与否」决定。
func TestASymbolicLinkDirectoryIsMatchedButNotDescended(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "linked-payload")
	writeFile(t, filepath.Join(outside, "report-inside.txt"))

	link := filepath.Join(root, "link-report")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	recorder := get(t, handler, searchURL("", "report"))
	body := decodeSearch(t, recorder)

	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"link-report"}) {
		t.Fatalf("命中 = %v，期望只有软链条目自身 [link-report]", got)
	}
	// 软链指向的目录内容不得出现在结果里。
	if strings.Contains(recorder.Body.String(), "report-inside.txt") {
		t.Errorf("遍历跟进了符号链接：%q", recorder.Body.String())
	}
}

// TestADanglingSymbolicLinkCanMatch 对应 Scenario A dangling symbolic link can match：
// 悬空软链按名称参与匹配，且搜索不因此失败。遍历不取条目元信息、不跟进软链，
// 悬空目标无 stat 可失败。
func TestADanglingSymbolicLinkCanMatch(t *testing.T) {
	handler, root := newAPI(t)

	link := filepath.Join(root, "dangling-report.txt")
	if err := os.Symlink(filepath.Join(root, "vanished"), link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	body := decodeSearch(t, get(t, handler, searchURL("", "report")))

	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"dangling-report.txt"}) {
		t.Errorf("命中 = %v，期望悬空软链照常命中 [dangling-report.txt]", got)
	}
}

// TestAnUnreadableSubdirectoryIsSkipped 对应 Scenario An unreadable subdirectory is
// skipped：无权限子目录自身按名称照常参与匹配（它出现在上级目录的 readdir 里），
// 其子树不产生命中，其余命中照常返回，整体不失败。
func TestAnUnreadableSubdirectoryIsSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("以 root 身份运行时 0o000 目录仍可读，用例会假失败")
	}

	handler, root := newAPI(t)
	locked := mkdir(t, root, "locked-report-dir")
	writeFile(t, filepath.Join(locked, "hidden-report.txt"))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("设置目录权限失败：%v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o700) })
	writeFile(t, filepath.Join(root, "open-report.txt"))

	recorder := get(t, handler, searchURL("", "report"))
	body := decodeSearch(t, recorder)

	// 目录条目自身命中在前（目录先于文件），无权限子树零命中，其余命中照常。
	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"locked-report-dir", "open-report.txt"}) {
		t.Fatalf("命中 = %v，期望 [locked-report-dir open-report.txt]", got)
	}
	if strings.Contains(recorder.Body.String(), "hidden-report.txt") {
		t.Errorf("无权限子树产生了命中：%q", recorder.Body.String())
	}
}

// --- Search endpoint ---

// TestASubtreeIsSearched 对应 Scenario A subtree is searched：响应携带基准位置的规范化
// 相对路径、回显的查询词与命中条目列表，且只在基准位置的子树内搜索。
func TestASubtreeIsSearched(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "docs", "readme.md"))
	// 基准子树外的同名条目：能命中它就说明搜索范围越过了基准位置。
	writeFile(t, filepath.Join(root, "readme.md"))
	mkdir(t, root, "other")
	writeFile(t, filepath.Join(root, "other", "readme.md"))

	body := decodeSearch(t, get(t, handler, searchURL("docs", "readme")))

	if body.Path != "docs" {
		t.Errorf("path = %q，期望基准位置的规范化相对路径 %q", body.Path, "docs")
	}
	if body.Query != "readme" {
		t.Errorf("query = %q，期望回显 %q", body.Query, "readme")
	}
	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"docs/readme.md"}) {
		t.Errorf("命中 = %v，期望只有基准子树内的 [docs/readme.md]", got)
	}
}

// TestSearchBaseParameterIsMissingOrEmpty 对应 Scenario Base parameter is missing or
// empty：以根目录为基准执行搜索，响应中的基准路径为空字符串。
func TestSearchBaseParameterIsMissingOrEmpty(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "docs", "readme.md"))

	for _, target := range []string{"/api/search?q=readme", "/api/search?path=&q=readme"} {
		t.Run(target, func(t *testing.T) {
			body := decodeSearch(t, get(t, handler, target))

			if body.Path != "" {
				t.Errorf("path = %q，期望空字符串（根目录）", body.Path)
			}
			// 命中的是嵌套条目，证明搜索确实以根目录为基准展开。
			if got := matchPaths(body.Matches); !slices.Equal(got, []string{"docs/readme.md"}) {
				t.Errorf("命中 = %v，期望 [docs/readme.md]", got)
			}
		})
	}
}

// TestSearchBaseParameterContainsRedundantSegments 对应 Scenario Base parameter
// contains redundant segments：按规范化后的基准位置执行搜索。
func TestSearchBaseParameterContainsRedundantSegments(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "a", "b")
	writeFile(t, filepath.Join(root, "a", "b", "c-report.txt"))

	for _, input := range []string{"a/b", "./a/b", "a//b", "a/./b", "a/b/", "./a//./b/"} {
		t.Run(input, func(t *testing.T) {
			body := decodeSearch(t, get(t, handler, searchURL(input, "report")))

			if body.Path != "a/b" {
				t.Errorf("path = %q，期望规范化后的 %q", body.Path, "a/b")
			}
			if got := matchPaths(body.Matches); !slices.Equal(got, []string{"a/b/c-report.txt"}) {
				t.Errorf("命中 = %v，期望 [a/b/c-report.txt]", got)
			}
		})
	}
}

// --- Base path access failures ---

// TestTheSearchBasePathDoesNotExist 对应 Scenario The base path does not exist。
func TestTheSearchBasePathDoesNotExist(t *testing.T) {
	handler, _ := newAPI(t)

	expectFailure(t, get(t, handler, searchURL("nope/deeper", "x")), codeNotFound, http.StatusNotFound)
}

// TestTheSearchBasePathIsNotADirectory 对应 Scenario The base path is not a directory。
func TestTheSearchBasePathIsNotADirectory(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "file.txt"))

	expectFailure(t, get(t, handler, searchURL("file.txt", "x")), codeNotADirectory, http.StatusBadRequest)
}

// TestTheSearchBasePathCannotBeRead 对应 Scenario The base path cannot be read。
func TestTheSearchBasePathCannotBeRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("以 root 身份运行时 0o000 目录仍可读，用例会假失败")
	}

	handler, root := newAPI(t)
	locked := mkdir(t, root, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("设置目录权限失败：%v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o700) })

	expectFailure(t, get(t, handler, searchURL("locked", "x")), codePermissionDenied, http.StatusForbidden)
}

// --- Search scope confinement ---

// TestASearchBaseEscapesTheRootByParentReferences 对应 Scenario A base escapes the root
// by parent references：规范化后指向根目录之外的基准位置被拒绝，不返回任何命中。
func TestASearchBaseEscapesTheRootByParentReferences(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "outside")
	writeFile(t, filepath.Join(outside, "secret-report.txt"))

	for _, input := range []string{"..", "../", "../outside", "a/../../outside"} {
		t.Run(input, func(t *testing.T) {
			recorder := get(t, handler, searchURL(input, "report"))

			expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
			if strings.Contains(recorder.Body.String(), "secret-report.txt") {
				t.Errorf("越界搜索返回了根目录外的命中：%q", recorder.Body.String())
			}
		})
	}
}

// TestASearchBaseTraversesASymbolicLinkOutward 对应 Scenario A base traverses a
// symbolic link outward：字面位置在根内、解析后指向外部的基准位置按物理判定拒绝。
func TestASearchBaseTraversesASymbolicLinkOutward(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "linked-target")
	writeFile(t, filepath.Join(outside, "visible-report.txt"))

	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	recorder := get(t, handler, searchURL("link", "report"))

	expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
	if strings.Contains(recorder.Body.String(), "visible-report.txt") {
		t.Errorf("越界搜索返回了根目录外的命中：%q", recorder.Body.String())
	}
}

// TestASearchBaseResolvesViaASymbolicLinkToInsideTheRoot 对应 Scenario A base resolves
// via a symbolic link to inside the root：解析后仍落在物理根之内的基准位置照常搜索。
func TestASearchBaseResolvesViaASymbolicLinkToInsideTheRoot(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "real")
	writeFile(t, filepath.Join(root, "real", "readme.md"))

	link := filepath.Join(root, "alias")
	if err := os.Symlink(filepath.Join(root, "real"), link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	body := decodeSearch(t, get(t, handler, searchURL("alias", "readme")))

	if body.Path != "alias" {
		t.Errorf("path = %q，期望 %q（不视为越界）", body.Path, "alias")
	}
	if got := matchPaths(body.Matches); !slices.Equal(got, []string{"alias/readme.md"}) {
		t.Errorf("命中 = %v，期望以软链指向的位置为基准的 [alias/readme.md]", got)
	}
}

// --- Match entry shape ---

// TestAMatchEntryCarriesNameTypeAndFullPath 对应 Scenario A match entry carries name,
// type, and full path：命中条目给出名称、类型与相对根目录的完整路径。
func TestAMatchEntryCarriesNameTypeAndFullPath(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "docs", "readme.md"))

	recorder := get(t, handler, searchURL("docs", "readme"))
	body := decodeSearch(t, recorder)

	match := findMatch(t, body.Matches, "docs/readme.md")
	if match.Name != "readme.md" {
		t.Errorf("name = %q，期望 %q", match.Name, "readme.md")
	}
	if match.Type != entryFile {
		t.Errorf("type = %q，期望 %q", match.Type, entryFile)
	}
	if match.Path != "docs/readme.md" {
		t.Errorf("path = %q，期望相对根目录的完整路径 %q", match.Path, "docs/readme.md")
	}

	// 字段集合恰为 name/path/type：结构之外的额外字段（尤其承载元信息的那类）不得出现。
	raw := rawMatches(t, recorder)
	if len(raw) != 1 {
		t.Fatalf("matches = %v，期望恰好一个命中", raw)
	}
	if got, want := fieldNames(raw[0]), []string{"name", "path", "type"}; !slices.Equal(got, want) {
		t.Errorf("命中条目字段 = %v，期望恰为 %v", got, want)
	}
}

// TestMatchEntriesContainNoContent 对应 Scenario Match entries contain no content：
// 命中条目不携带条目内容，也不携带 size/modified_at 之类的元信息字段。
func TestMatchEntriesContainNoContent(t *testing.T) {
	handler, root := newAPI(t)
	const content = "SEARCHCONTENTMARKER"
	writeContent(t, filepath.Join(root, "notes.txt"), content)

	recorder := get(t, handler, searchURL("", "notes"))
	decodeSearch(t, recorder)

	if strings.Contains(recorder.Body.String(), content) {
		t.Errorf("搜索响应带回了文件内容：%q", recorder.Body.String())
	}
	for _, candidate := range rawMatches(t, recorder) {
		if got, want := fieldNames(candidate), []string{"name", "path", "type"}; !slices.Equal(got, want) {
			t.Errorf("命中条目字段 = %v，期望恰为 %v（不含内容与元信息字段）", got, want)
		}
	}
}

// TestSearchResponseFieldsDoNotLeakAcrossEndpoints 钉住两个端点的响应分区：
// search 顶层字段恰为 matches/path/query（不含 list 的 entries/parent），
// list 顶层字段恰为 entries/parent/path（不含 search 的 matches/query）。
func TestSearchResponseFieldsDoNotLeakAcrossEndpoints(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "notes.txt"))

	searchRecorder := get(t, handler, searchURL("", "notes"))
	if got, want := topLevelFieldNames(t, searchRecorder.Body.String()), []string{"matches", "path", "query"}; !slices.Equal(got, want) {
		t.Errorf("search 响应顶层字段 = %v，期望恰为 %v", got, want)
	}

	listRecorder := get(t, handler, listURL(""))
	if got, want := topLevelFieldNames(t, listRecorder.Body.String()), []string{"entries", "parent", "path"}; !slices.Equal(got, want) {
		t.Errorf("list 响应顶层字段 = %v，期望恰为 %v", got, want)
	}
}
