package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func newAPI(t *testing.T) (http.Handler, string) {
	t.Helper()
	root := t.TempDir()
	handler, err := NewAPIHandler(root)
	if err != nil {
		t.Fatalf("以根目录 %s 构造服务失败：%v", root, err)
	}
	return handler, root
}

// newAPIWithBrowserInfo 走 design D9 的 seam：替换条目元信息的取数函数后再装配路由。
// 生产路径上的默认实现是 DirEntry.Info，被替换的只有这一个字段。
func newAPIWithBrowserInfo(t *testing.T, entryInfo func(fs.DirEntry) (fs.FileInfo, error)) (http.Handler, string) {
	t.Helper()
	root := t.TempDir()
	b, err := newBrowser(root)
	if err != nil {
		t.Fatalf("以根目录 %s 构造 browser 失败：%v", root, err)
	}
	b.entryInfo = entryInfo
	return apiHandler(b), root
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

func writeContent(t *testing.T, name, content string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}
	return name
}

// setModTime 把条目的修改时间固定到给定的时刻，让 modified_at 的断言不依赖运行时刻。
func setModTime(t *testing.T, name string, stamp time.Time) {
	t.Helper()
	if err := os.Chtimes(name, stamp, stamp); err != nil {
		t.Fatalf("设置 %s 的修改时间失败：%v", name, err)
	}
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

// findEntry 按名称取条目。用例里的断言一律经它取，而不是按下标——
// 排序键是名称，按下标断言等于把顺序写死两遍。
func findEntry(t *testing.T, entries []listEntry, name string) listEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("列表中没有名为 %q 的条目，实际为 %v", name, names(entries))
	return listEntry{}
}

// rawEntries 把响应解成 map 而不是结构体，用来看清字段「是否出现」。
// 结构体解不出来与字段不存在是同一种结果，无法区分这两种情况，
// 而 Entry metadata 承诺的正是缺省而非 null（design D5）。
func rawEntries(t *testing.T, recorder *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var raw struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v", err)
	}
	return raw.Entries
}

// fieldNames 给出某个条目实际出现的字段名，按名称排序以便与字面量列表比对。
func fieldNames(entry map[string]any) []string {
	fields := make([]string, 0, len(entry))
	for field := range entry {
		fields = append(fields, field)
	}
	slices.Sort(fields)
	return fields
}

// --- Directory listing response ---

func TestADirectoryIsListed(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	// 内容用一个在响应里绝不该出现的标记串，这样「响应不含条目内容」才是一条能失败的断言。
	const content = "READMECONTENTMARKER"
	writeContent(t, filepath.Join(root, "docs", "readme.md"), content)

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

	// 条目字段恰为 name/type/size/modified_at：结构之外的额外字段（尤其是承载内容的那类）都不得出现。
	recorder := get(t, handler, listURL("docs"))
	raw := rawEntries(t, recorder)
	if len(raw) != 1 {
		t.Fatalf("entries = %v，期望恰好一个条目", raw)
	}
	if got, want := fieldNames(raw[0]), []string{"modified_at", "name", "size", "type"}; !slices.Equal(got, want) {
		t.Errorf("条目字段 = %v，期望恰为 %v", got, want)
	}

	// Scenario `A listed directory contains readable files`：文件只以元信息出现。
	if strings.Contains(recorder.Body.String(), content) {
		t.Errorf("列表响应带回了文件内容：%q", recorder.Body.String())
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

// --- Entry metadata ---

func TestAFileEntryIsListed(t *testing.T) {
	handler, root := newAPI(t)
	path := writeContent(t, filepath.Join(root, "notes.txt"), "hello")
	stamp := time.Unix(1758000000, 0)
	setModTime(t, path, stamp)

	body := decodeList(t, get(t, handler, listURL("")))

	entry := findEntry(t, body.Entries, "notes.txt")
	if entry.Size == nil {
		t.Fatal("文件条目的 size 缺省，期望给出该文件自身的大小")
	}
	if want := int64(len("hello")); *entry.Size != want {
		t.Errorf("size = %d，期望该文件自身的字节数 %d", *entry.Size, want)
	}
	if entry.ModifiedAt == nil {
		t.Fatal("文件条目的 modified_at 缺省，期望给出最后修改时间")
	}
	if *entry.ModifiedAt != stamp.Unix() {
		t.Errorf("modified_at = %d，期望 Unix 整秒 %d", *entry.ModifiedAt, stamp.Unix())
	}
}

func TestADirectoryEntryIsListed(t *testing.T) {
	handler, root := newAPI(t)
	sub := mkdir(t, root, "docs")
	writeFile(t, filepath.Join(sub, "readme.md"))
	// 先写内容再定时间：新建条目会改写目录自身的修改时间。
	stamp := time.Unix(1758000000, 0)
	setModTime(t, sub, stamp)

	recorder := get(t, handler, listURL(""))
	body := decodeList(t, recorder)

	entry := findEntry(t, body.Entries, "docs")
	if entry.ModifiedAt == nil {
		t.Fatal("目录条目的 modified_at 缺省，期望给出最后修改时间")
	}
	if *entry.ModifiedAt != stamp.Unix() {
		t.Errorf("modified_at = %d，期望 Unix 整秒 %d", *entry.ModifiedAt, stamp.Unix())
	}

	// 目录没有「自身占用字节数」这个概念，因此不出现 size。断言字段集合而不是只看
	// 结构体取值：后者无法区分「size 为 0」与「size 不存在」。
	raw := rawEntries(t, recorder)
	var fields []string
	for _, candidate := range raw {
		if candidate["name"] == "docs" {
			fields = fieldNames(candidate)
		}
	}
	if fields == nil {
		t.Fatalf("原始响应里没有 docs 条目：%v", raw)
	}
	if want := []string{"modified_at", "name", "type"}; !slices.Equal(fields, want) {
		t.Errorf("目录条目字段 = %v，期望恰为 %v（目录不应给出 size）", fields, want)
	}
}

func TestAnEntryIsASymbolicLink(t *testing.T) {
	handler, root := newAPI(t)
	// 目标在根目录之外，且必须显著大于它的路径字符串，否则这条用例区分不出
	// lstat 与 Stat：两者在「目标很小」时会偶然取到同一个值。
	outside := mkdir(t, filepath.Dir(root), "linked-payload")
	payload := strings.Repeat("x", 64*1024)
	target := writeContent(t, filepath.Join(outside, "payload.bin"), payload)

	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	body := decodeList(t, get(t, handler, listURL("")))

	entry := findEntry(t, body.Entries, "link")
	if entry.Size == nil {
		t.Fatal("符号链接条目的 size 缺省，期望给出链接自身的长度")
	}
	if want := int64(len(target)); *entry.Size != want {
		t.Errorf("size = %d，期望链接自身的长度 %d（等于目标文件大小 %d 说明取数用了 os.Stat 而非 DirEntry.Info）",
			*entry.Size, want, len(payload))
	}
}

func TestModificationTimeFallsWithinTheSameSecond(t *testing.T) {
	handler, root := newAPI(t)
	base := time.Unix(1758000000, 0)
	early := writeContent(t, filepath.Join(root, "early.txt"), "a")
	late := writeContent(t, filepath.Join(root, "late.txt"), "b")
	// 同一秒内的两个不同亚秒时刻：向下取整到整秒后必须相同。
	setModTime(t, early, base.Add(120*time.Millisecond))
	setModTime(t, late, base.Add(880*time.Millisecond))

	body := decodeList(t, get(t, handler, listURL("")))

	earlyAt := findEntry(t, body.Entries, "early.txt").ModifiedAt
	lateAt := findEntry(t, body.Entries, "late.txt").ModifiedAt
	if earlyAt == nil || lateAt == nil {
		t.Fatalf("modified_at 缺省：early = %v，late = %v", earlyAt, lateAt)
	}
	if *earlyAt != *lateAt {
		t.Errorf("同一秒内的两个时刻给出不同取值：%d 与 %d", *earlyAt, *lateAt)
	}
	if *earlyAt != base.Unix() {
		t.Errorf("modified_at = %d，期望向下取整到整秒的 %d", *earlyAt, base.Unix())
	}
}

func TestModificationTimeDoesNotDependOnTheHostEnvironment(t *testing.T) {
	handler, root := newAPI(t)
	setModTime(t, writeContent(t, filepath.Join(root, "notes.txt"), "a"), time.Unix(1758000000, 0))
	mkdir(t, root, "docs")
	setModTime(t, filepath.Join(root, "docs"), time.Unix(1758000100, 0))

	// 每个时区取一份「条目名 -> modified_at」，跨时区逐条目比对。
	listed := make([]map[string]int64, 0, 2)
	for _, zone := range []string{"Asia/Tokyo", "America/New_York"} {
		t.Run(zone, func(t *testing.T) {
			t.Setenv("TZ", zone)
			body := decodeList(t, get(t, handler, listURL("")))
			snapshot := make(map[string]int64, len(body.Entries))
			for _, entry := range body.Entries {
				if entry.ModifiedAt == nil {
					t.Fatalf("%s 的 modified_at 缺省", entry.Name)
				}
				snapshot[entry.Name] = *entry.ModifiedAt
			}
			listed = append(listed, snapshot)
		})
	}

	for name, value := range listed[0] {
		if other, ok := listed[1][name]; !ok {
			t.Errorf("第二个时区下没有条目 %s", name)
		} else if other != value {
			t.Errorf("%s 的 modified_at 在两个时区下不同：%d 与 %d（说明取值经过了时区或本地化格式化）", name, value, other)
		}
	}
}

func TestAnEntrysMetadataCannotBeObtained(t *testing.T) {
	handler, root := newAPIWithBrowserInfo(t, func(entry fs.DirEntry) (fs.FileInfo, error) {
		// 只让一个条目取数失败，其余照常取——否则「整列表都缺元信息」也会让用例通过。
		if entry.Name() == "vanished.txt" {
			return nil, errors.New("条目在 readdir 之后消失")
		}
		return entry.Info()
	})
	writeFile(t, filepath.Join(root, "readable.txt"))
	writeFile(t, filepath.Join(root, "vanished.txt"))

	recorder := get(t, handler, listURL(""))
	body := decodeList(t, recorder)

	// 请求成功、条目仍在列表中：列表的首要性质是健壮，丢弃条目等于对存在性说谎。
	if got, want := names(body.Entries), []string{"readable.txt", "vanished.txt"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v，期望 %v（条目不得被丢弃）", got, want)
	}

	vanished := findEntry(t, body.Entries, "vanished.txt")
	if vanished.Size != nil {
		t.Errorf("取数失败的条目仍给出了 size = %d，期望省略", *vanished.Size)
	}
	if vanished.ModifiedAt != nil {
		t.Errorf("取数失败的条目仍给出了 modified_at = %d，期望省略", *vanished.ModifiedAt)
	}

	// 未失败的条目仍然带元信息，证明 seam 只影响了目标条目。
	readable := findEntry(t, body.Entries, "readable.txt")
	if readable.Size == nil || readable.ModifiedAt == nil {
		t.Errorf("可取元信息的条目缺字段：size = %v，modified_at = %v", readable.Size, readable.ModifiedAt)
	}

	// 缺省而不是 null：失败条目的字段集合恰为 [name type]。
	for _, candidate := range rawEntries(t, recorder) {
		if candidate["name"] != "vanished.txt" {
			continue
		}
		if got, want := fieldNames(candidate), []string{"name", "type"}; !slices.Equal(got, want) {
			t.Errorf("取数失败条目的字段 = %v，期望恰为 %v（不得用 null 表达缺省）", got, want)
		}
	}
}

func TestRepeatedListingsReturnTheSameMetadata(t *testing.T) {
	handler, root := newAPI(t)
	setModTime(t, writeContent(t, filepath.Join(root, "notes.txt"), "a"), time.Unix(1758000000, 0))
	setModTime(t, mkdir(t, root, "docs"), time.Unix(1758000100, 0))
	writeFile(t, filepath.Join(root, "docs", "readme.md"))

	first := decodeList(t, get(t, handler, listURL("")))
	second := decodeList(t, get(t, handler, listURL("")))

	if !slices.Equal(names(first.Entries), names(second.Entries)) {
		t.Fatalf("两次请求的条目不一致：%v 与 %v", names(first.Entries), names(second.Entries))
	}
	for i := range first.Entries {
		a, b := first.Entries[i], second.Entries[i]
		if a.ModifiedAt == nil || b.ModifiedAt == nil {
			t.Errorf("%s 的 modified_at 缺省：%v 与 %v", a.Name, a.ModifiedAt, b.ModifiedAt)
		} else if *a.ModifiedAt != *b.ModifiedAt {
			t.Errorf("%s 的两次 modified_at 不一致：%d 与 %d", a.Name, *a.ModifiedAt, *b.ModifiedAt)
		}
		// 目录条目本就没有 size（design D5），其余条目两次取值必须一致。
		if a.Type == entryDirectory {
			continue
		}
		if a.Size == nil || b.Size == nil {
			t.Errorf("%s 的 size 缺省：%v 与 %v", a.Name, a.Size, b.Size)
		} else if *a.Size != *b.Size {
			t.Errorf("%s 的两次 size 不一致：%d 与 %d", a.Name, *a.Size, *b.Size)
		}
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

	handler, err := NewAPIHandler(root)
	if err != nil {
		t.Fatalf("以根目录 %s 构造服务失败：%v", root, err)
	}
	recorder := get(t, handler, listURL("../bc"))

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

	recorder := get(t, handler, listURL("link"))

	// 物理判定：字面位置在根目录内，但解析符号链接后落在物理根之外，按越界拒绝，
	// 不返回该路径的列表。
	expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
	if strings.Contains(recorder.Body.String(), "visible.txt") {
		t.Errorf("越界请求返回了根目录外的内容：%q", recorder.Body.String())
	}
}

// TestAPathTraversesASymbolicLinkToAnotherLocationInsideTheRoot 对应 Scenario
// A path traverses a symbolic link to another location inside the root。
// 收紧不得过度阻塞：根内软链（目录内整理用链）继续可用。
func TestAPathTraversesASymbolicLinkToAnotherLocationInsideTheRoot(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "real")
	writeFile(t, filepath.Join(root, "real", "readme.md"))

	link := filepath.Join(root, "alias")
	if err := os.Symlink(filepath.Join(root, "real"), link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	body := decodeList(t, get(t, handler, listURL("alias")))

	// 解析后仍落在物理根之内：按该路径提供内容，不视为越界。
	if body.Path != "alias" {
		t.Errorf("path = %q，期望 %q", body.Path, "alias")
	}
	if got := names(body.Entries); !slices.Equal(got, []string{"readme.md"}) {
		t.Errorf("entries = %v，期望 [readme.md]", got)
	}
}

// TestTheRootPathItselfContainsASymbolicLink 对应 Scenario
// The root path itself contains a symbolic link。
// 用包住 t.TempDir 的软链作根，不依赖运行环境的路径巧合（design 风险条目：
// macOS /var/folders 是真目录而 /tmp 是软链，两边都必须能通过）。
func TestTheRootPathItselfContainsASymbolicLink(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "docs")
	writeFile(t, filepath.Join(root, "docs", "spec.md"))

	wrapped := filepath.Join(t.TempDir(), "wrapped-root")
	if err := os.Symlink(root, wrapped); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	handler, err := NewAPIHandler(wrapped)
	if err != nil {
		t.Fatalf("以软链路径 %s 作根构造服务失败：%v", wrapped, err)
	}

	// 根目录与其下子目录都正常：根与请求路径两侧都按物理位置解析，判定基准一致，
	// 字面路径与物理位置不同不产生假越界。
	rootBody := decodeList(t, get(t, handler, listURL("")))
	if got := names(rootBody.Entries); !slices.Equal(got, []string{"docs"}) {
		t.Errorf("entries = %v，期望 [docs]", got)
	}
	docsBody := decodeList(t, get(t, handler, listURL("docs")))
	if got := names(docsBody.Entries); !slices.Equal(got, []string{"spec.md"}) {
		t.Errorf("entries = %v，期望 [spec.md]", got)
	}
}

// TestAMiddleSegmentTraversesASymbolicLinkOutward：越界不只发生在最后一段——
// 路径中间段经软链出根时，终点无论字面上写成什么都一并拒绝。
func TestAMiddleSegmentTraversesASymbolicLinkOutward(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "linked-out")
	writeFile(t, filepath.Join(outside, "secret.txt"))
	mkdir(t, root, "a")

	link := filepath.Join(root, "a", "jump")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	for _, input := range []string{"a/jump", "a/jump/secret.txt"} {
		t.Run(input, func(t *testing.T) {
			expectFailure(t, get(t, handler, listURL(input)), codeOutsideRoot, http.StatusBadRequest)
		})
	}
}

// TestADanglingSymbolicLinkIsNotFound：悬空软链维持既有 not_found 行为——
// EvalSymlinks 失败时不做越界判定，回落到 Stat 分类（improve-root-confinement design D3）。
func TestADanglingSymbolicLinkIsNotFound(t *testing.T) {
	handler, root := newAPI(t)

	link := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "vanished"), link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	expectFailure(t, get(t, handler, listURL("dangling")), codeNotFound, http.StatusNotFound)
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
