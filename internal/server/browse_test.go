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

func newAPI(t *testing.T) (http.Handler, string) {
	t.Helper()
	root := t.TempDir()
	return NewAPIHandler(root), root
}

func get(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func listURL(path string) string {
	return "/api/list?path=" + url.QueryEscape(path)
}

func decodeList(t *testing.T, recorder *httptest.ResponseRecorder) listResponse {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d（body = %q）", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q，期望 JSON", contentType)
	}
	var body listResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, recorder.Body.String())
	}
	return body
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) (errorDetail, int) {
	t.Helper()
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q，期望 JSON（/api/ 区域不产生 HTML 错误页或纯文本响应）", contentType)
	}
	if strings.Contains(recorder.Body.String(), "<html") {
		t.Fatalf("响应体是 HTML 页面，期望 JSON 错误信封：%q", recorder.Body.String())
	}
	var body errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, recorder.Body.String())
	}
	return body.Error, recorder.Code
}

func expectFailure(t *testing.T, recorder *httptest.ResponseRecorder, code string, status int) {
	t.Helper()
	detail, gotStatus := decodeError(t, recorder)
	if detail.Code != code {
		t.Errorf("error.code = %q，期望 %q", detail.Code, code)
	}
	if gotStatus != status {
		t.Errorf("状态码 = %d，期望 %d", gotStatus, status)
	}
	if detail.Message == "" {
		t.Error("错误响应缺少人类可读的说明（error.message）")
	}
}

func mkdir(t *testing.T, elements ...string) string {
	t.Helper()
	dir := filepath.Join(elements...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("准备目录 %s 失败：%v", dir, err)
	}
	return dir
}

func writeFile(t *testing.T, name string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}
	return name
}

func names(entries []listEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}
	return out
}

func typeOf(entries []listEntry, name string) string {
	for _, entry := range entries {
		if entry.Name == name {
			return entry.Type
		}
	}
	return ""
}

// --- Directory listing response ---

func TestADirectoryIsListed(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "docs", "readme.md"))

	body := decodeList(t, get(t, handler, listURL("docs")))

	if body.Path != "docs" {
		t.Errorf("path = %q，期望 %q", body.Path, "docs")
	}
	if body.Parent != "" {
		t.Errorf("parent = %q，期望空（上级为根目录）", body.Parent)
	}
	if got := names(body.Entries); !slices.Equal(got, []string{"readme.md"}) {
		t.Errorf("entries = %v，期望 [readme.md]", got)
	}

	// 条目只带 name 与 type：结构大小写之外的额外字段（大小、修改时间、内容）都不得出现。
	var raw struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(get(t, handler, listURL("docs")).Body.Bytes(), &raw); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v", err)
	}
	if len(raw.Entries) != 1 {
		t.Fatalf("entries = %v，期望恰好一个条目", raw.Entries)
	}
	keys := make([]string, 0, len(raw.Entries[0]))
	for key := range raw.Entries[0] {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"name", "type"}) {
		t.Errorf("条目字段 = %v，期望恰为 [name type]", keys)
	}
}

func TestPathParameterIsMissingOrEmpty(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "top.txt"))

	for _, target := range []string{"/api/list", "/api/list?path=", listURL(".")} {
		t.Run(target, func(t *testing.T) {
			body := decodeList(t, get(t, handler, target))

			if body.Path != "" {
				t.Errorf("path = %q，期望空字符串（根目录）", body.Path)
			}
			if body.Parent != "" {
				t.Errorf("parent = %q，期望空字符串（根目录没有上级）", body.Parent)
			}
			if got := names(body.Entries); !slices.Equal(got, []string{"docs", "top.txt"}) {
				t.Errorf("entries = %v，期望根目录的条目 [docs top.txt]", got)
			}
		})
	}
}

func TestADirectoryHasNoEntries(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "empty")

	body := decodeList(t, get(t, handler, listURL("empty")))

	if body.Entries == nil {
		t.Error("entries = null，期望空列表而不是错误")
	}
	if len(body.Entries) != 0 {
		t.Errorf("entries = %v，期望空列表", names(body.Entries))
	}
}

func TestPathParameterContainsRedundantSegments(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "a", "b")
	writeFile(t, filepath.Join(root, "a", "b", "c.txt"))

	for _, input := range []string{"a/b", "./a/b", "a//b", "a/./b", "a/b/", "./a//./b/"} {
		t.Run(input, func(t *testing.T) {
			body := decodeList(t, get(t, handler, listURL(input)))

			if body.Path != "a/b" {
				t.Errorf("path = %q，期望规范化后的 %q", body.Path, "a/b")
			}
			if body.Parent != "a" {
				t.Errorf("parent = %q，期望 %q", body.Parent, "a")
			}
			if got := names(body.Entries); !slices.Equal(got, []string{"c.txt"}) {
				t.Errorf("entries = %v，期望 [c.txt]", got)
			}
		})
	}
}

// --- Entry type distinction ---

func TestEntryTypeIsDistinguishedInAMixedDirectory(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "dir")
	writeFile(t, filepath.Join(root, "file.txt"))

	body := decodeList(t, get(t, handler, listURL("")))

	if got := typeOf(body.Entries, "dir"); got != entryDirectory {
		t.Errorf("子目录条目 dir 的 type = %q，期望 %q", got, entryDirectory)
	}
	if got := typeOf(body.Entries, "file.txt"); got != entryFile {
		t.Errorf("文件条目 file.txt 的 type = %q，期望 %q", got, entryFile)
	}
}

// --- List ordering ---

func TestDirectoriesPrecedeFilesInAMixedDirectory(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "zdir")
	mkdir(t, root, "adir")
	writeFile(t, filepath.Join(root, "zzz.txt"))
	writeFile(t, filepath.Join(root, "aaa.txt"))

	body := decodeList(t, get(t, handler, listURL("")))

	want := []string{"adir", "zdir", "aaa.txt", "zzz.txt"}
	if got := names(body.Entries); !slices.Equal(got, want) {
		t.Errorf("entries = %v，期望目录全部排在文件之前的 %v", got, want)
	}
}

func TestEntriesWithinTheSameGroupAreSortedByName(t *testing.T) {
	handler, root := newAPI(t)
	for _, name := range []string{"Banana", "apple", "Cherry"} {
		writeFile(t, filepath.Join(root, name))
	}
	mkdir(t, root, "zebra")
	mkdir(t, root, "Ant")

	body := decodeList(t, get(t, handler, listURL("")))

	// 名称比较不区分大小写：Apple < banana < cherry。
	want := []string{"Ant", "zebra", "apple", "Banana", "Cherry"}
	if got := names(body.Entries); !slices.Equal(got, want) {
		t.Errorf("entries = %v，期望 %v", got, want)
	}
}

func TestNamesDifferingOnlyInLetterCaseAreNotInverted(t *testing.T) {
	handler, root := newAPI(t)
	want := []string{"README", "ReadMe", "readme"}
	// macOS 默认的大小写不敏感文件系统会把仅大小写不同的名字折叠成同一个条目，
	// 该 Scenario 在这类文件系统上无法构造，跳过而不是给出一个必然通过的空断言。
	if !canHostCaseDistinctNames(t, root) {
		t.Skip("当前文件系统无法在同一目录创建仅大小写不同的条目")
	}

	body := decodeList(t, get(t, handler, listURL("")))

	// 三者折叠后同键：顺序由原名 tiebreak 决定，且不因大小写被颠倒或抖动。
	if got := names(body.Entries); !slices.Equal(got, want) {
		t.Errorf("entries = %v，期望 %v", got, want)
	}
}

func canHostCaseDistinctNames(t *testing.T, dir string) bool {
	t.Helper()
	names := []string{"README", "ReadMe", "readme"}
	for _, name := range names {
		writeFile(t, filepath.Join(dir, name))
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", dir, err)
	}
	return len(items) == len(names)
}

// TestSortEntriesFallsBackToNameForCaseOnlyDifferences 直接断言排序比较键的第三段。
// API 层版本受限于文件系统大小写敏感性（见上一个用例的 skip），
// 但 tiebreak 缺失会让顺序在请求间抖动，因此在这里脱离文件系统钉住它。
func TestSortEntriesFallsBackToNameForCaseOnlyDifferences(t *testing.T) {
	entries := []listEntry{
		{Name: "readme", Type: entryFile},
		{Name: "ReadMe", Type: entryFile},
		{Name: "README", Type: entryFile},
	}

	sortEntries(entries)

	want := []string{"README", "ReadMe", "readme"}
	if got := names(entries); !slices.Equal(got, want) {
		t.Errorf("排序结果 = %v，期望 %v", got, want)
	}
}

func TestRepeatedListingsOfTheSameDirectoryReturnTheSameOrder(t *testing.T) {
	handler, root := newAPI(t)
	// 仅大小写不同的条目是 tiebreak 的用武之地；大小写不敏感的文件系统上它们会被折叠为
	// 同一个条目，用例仍然成立，只是退化为对普通条目的稳定性断言。
	canHostCaseDistinctNames(t, root)
	for _, name := range []string{"notes.txt", "TODO", "Alpha"} {
		writeFile(t, filepath.Join(root, name))
	}
	mkdir(t, root, "Sub")
	mkdir(t, root, "src")

	first := decodeList(t, get(t, handler, listURL("")))
	second := decodeList(t, get(t, handler, listURL("")))

	if !slices.Equal(names(first.Entries), names(second.Entries)) {
		t.Errorf("两次请求的顺序不一致：%v 与 %v", names(first.Entries), names(second.Entries))
	}
}

// --- Position representation and reproducibility ---

func TestAPositionIsReopened(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs", "api")
	writeFile(t, filepath.Join(root, "docs", "api", "spec.md"))

	first := decodeList(t, get(t, handler, listURL("docs/api")))
	second := decodeList(t, get(t, handler, listURL("docs/api")))

	if second.Path != first.Path {
		t.Errorf("第二次的 path = %q，期望与首次相同 %q", second.Path, first.Path)
	}
	if !slices.Equal(names(second.Entries), names(first.Entries)) {
		t.Errorf("第二次的 entries = %v，期望与首次相同 %v", names(second.Entries), names(first.Entries))
	}
}

func TestPositionIsResolvedFromAPathContainingAParentReference(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "a", "b", "c")
	writeFile(t, filepath.Join(root, "a", "b", "target.txt"))

	body := decodeList(t, get(t, handler, listURL("a/c/../b")))

	if body.Path != "a/b" {
		t.Errorf("path = %q，期望规范化后的 %q", body.Path, "a/b")
	}
	// a/b 下有子目录 c 与文件 target.txt：目录在前。
	if got := names(body.Entries); !slices.Equal(got, []string{"c", "target.txt"}) {
		t.Errorf("entries = %v，期望 [c target.txt]", got)
	}
}

// --- Parent directory reference ---

func TestParentReferenceIsGivenForADirectoryInsideTheRoot(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "a", "b", "c")
	mkdir(t, root, "top")

	for _, input := range []struct {
		path string
		want string
	}{
		{"a/b/c", "a/b"},
		// 根目录的一级子目录，其上级相对路径就是根目录本身，即空字符串。
		// 因此客户端要靠 path 是否为空来区分「已在根目录」与「上级是根目录」。
		{"top", ""},
	} {
		t.Run(input.path, func(t *testing.T) {
			body := decodeList(t, get(t, handler, listURL(input.path)))
			if body.Parent != input.want {
				t.Errorf("parent = %q，期望 %q", body.Parent, input.want)
			}
		})
	}
}

func TestParentReferenceIsAbsentForTheRootDirectory(t *testing.T) {
	handler, _ := newAPI(t)

	body := decodeList(t, get(t, handler, listURL("")))

	if body.Parent != "" {
		t.Errorf("parent = %q，期望空字符串（根目录没有上级）", body.Parent)
	}
}

// --- Root directory confinement ---

func TestAPathEscapesTheRootByParentReferences(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "outside")
	writeFile(t, filepath.Join(outside, "secret.txt"))

	for _, input := range []string{"..", "../", "../outside", "a/../../outside"} {
		t.Run(input, func(t *testing.T) {
			recorder := get(t, handler, listURL(input))

			expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
			if strings.Contains(recorder.Body.String(), "secret.txt") {
				t.Errorf("越界请求返回了根目录外的内容：%q", recorder.Body.String())
			}
		})
	}
}

func TestASiblingDirectorySharesTheRootPathPrefix(t *testing.T) {
	// 根目录为 a/b，兄弟目录 a/bc 的字符串前缀与根目录相同。
	// 裸 strings.HasPrefix 判定会把 a/bc 放进根目录内，因此这里必须用 filepath.Rel 判定。
	base := t.TempDir()
	root := mkdir(t, base, "a", "b")
	sibling := mkdir(t, base, "a", "bc")
	writeFile(t, filepath.Join(sibling, "secret.txt"))

	recorder := get(t, NewAPIHandler(root), listURL("../bc"))

	expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
	if strings.Contains(recorder.Body.String(), "secret.txt") {
		t.Errorf("兄弟目录的内容被返回：%q", recorder.Body.String())
	}
}

func TestAPathInsideTheRootTraversesASymbolicLinkOutward(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "linked-target")
	writeFile(t, filepath.Join(outside, "visible.txt"))

	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	body := decodeList(t, get(t, handler, listURL("link")))

	// 判定链路不解析符号链接：字面位置在根目录内，因此按该路径提供内容，不视为越界。
	if body.Path != "link" {
		t.Errorf("path = %q，期望 %q", body.Path, "link")
	}
	if got := names(body.Entries); !slices.Equal(got, []string{"visible.txt"}) {
		t.Errorf("entries = %v，期望 [visible.txt]", got)
	}
}

// --- Directory access failures ---

func TestTheRequestedPathIsOutsideTheRoot(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(filepath.Dir(root), "outside.txt"))

	for _, input := range []string{"../outside.txt", "../..", "/etc"} {
		t.Run(input, func(t *testing.T) {
			// 浏览位置的契约是相对根目录的路径，因此越界与绝对路径都归为 outside_root。
			expectFailure(t, get(t, handler, listURL(input)), codeOutsideRoot, http.StatusBadRequest)
		})
	}
}

func TestTheRequestedPathDoesNotExist(t *testing.T) {
	handler, _ := newAPI(t)

	expectFailure(t, get(t, handler, listURL("nope/deeper")), codeNotFound, http.StatusNotFound)
}

func TestTheRequestedPathIsNotADirectory(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "file.txt"))

	expectFailure(t, get(t, handler, listURL("file.txt")), codeNotADirectory, http.StatusBadRequest)
	// 路径的中间某段是文件，同样是「不是目录」；子路径无需真实存在即可命中该判定。
	expectFailure(t, get(t, handler, listURL("file.txt/child.txt")), codeNotADirectory, http.StatusBadRequest)
}

func TestTheRequestedDirectoryCannotBeRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("以 root 身份运行时 0o000 目录仍可读，用例会假失败")
	}

	handler, root := newAPI(t)
	locked := mkdir(t, root, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("设置目录权限失败：%v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o700) })

	expectFailure(t, get(t, handler, listURL("locked")), codePermissionDenied, http.StatusForbidden)
}

func TestFailureCausesAreDistinguishableFromTheResponseBody(t *testing.T) {
	handler, root := newAPI(t)
	writeFile(t, filepath.Join(root, "file.txt"))

	missing, _ := decodeError(t, get(t, handler, listURL("nope")))
	notADir, _ := decodeError(t, get(t, handler, listURL("file.txt")))
	outside, _ := decodeError(t, get(t, handler, listURL("../..")))

	causes := []struct {
		name   string
		detail errorDetail
	}{
		{"不存在", missing},
		{"不是目录", notADir},
		{"越界", outside},
	}
	for _, cause := range causes {
		if cause.detail.Code == "" {
			t.Errorf("%s 的响应缺少机器可读错误标识", cause.name)
		}
		if cause.detail.Message == "" {
			t.Errorf("%s 的响应缺少人类可读说明", cause.name)
		}
		for _, other := range causes {
			if cause.name == other.name {
				continue
			}
			if cause.detail.Code == other.detail.Code {
				t.Errorf("%s 与 %s 的 code 相同（%q），调用方无法区分失败原因", cause.name, other.name, cause.detail.Code)
			}
			if cause.detail.Message == other.detail.Message {
				t.Errorf("%s 与 %s 的 message 相同（%q），说明未表达各自的失败原因", cause.name, other.name, cause.detail.Message)
			}
		}
	}
}
