package server

import (
	"bytes"
	"encoding/json"
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

func contentURL(path string) string {
	return "/api/content?path=" + url.QueryEscape(path)
}

func decodeContent(t *testing.T, recorder *httptest.ResponseRecorder) contentResponse {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d（body = %q）", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q，期望 JSON", contentType)
	}
	var body contentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, recorder.Body.String())
	}
	return body
}

// --- Text file recognition ---

// TestLooksLikeText 脱离文件系统钉住内容嗅探的判据本身（design D2/D3/D4）：
// binary data byte 区间、空字节奇偶对齐豁免、空输入按文本处理。
// Requirement 的「起始部分」窗口与入口分流由 looksLikeText 之上的
// startsLikeText / isTextFile 与 API 层用例覆盖，这里只钉判据的纯逻辑。
func TestLooksLikeText(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want bool
	}{
		// 纯 ASCII，含制表与换行
		{"纯 ASCII", []byte("all:\n\techo hi\n"), true},
		// 0x09（制表）与 0x1B（ESC）都不在二进制数据字节区间内
		{"含 0x09 与 0x1B", []byte{'\t', 0x1B, 'x'}, true},
		// PNG 魔数里的 0x1A 落在 0x0E–0x1A，命中判据
		{"PNG 魔数", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, false},
		// 无 BOM 的 UTF-16：空字节全部只落一个奇偶侧，按豁免处理
		{"UTF-16LE 无 BOM", []byte{'h', 0x00, 'i', 0x00, '!'}, true},
		{"UTF-16BE 无 BOM", []byte{0x00, 'h', 0x00, 'i', 0x00, '!'}, true},
		// 空字节奇偶混杂：豁免不成立
		{"奇偶混杂 NUL", []byte{'a', 0x00, 0x00, 'b'}, false},
		// 空输入不含任何二进制数据字节
		{"空输入", nil, true},
	}

	for _, c := range cases {
		if got := looksLikeText(c.head); got != c.want {
			t.Errorf("looksLikeText(%q) = %v，期望 %v（%s）", c.head, got, c.want, c.name)
		}
	}
}

// TestTextRecognition 脱离文件系统钉住按名字判定的那一支，对应三条 Scenario：
// A file with a known text extension is requested / A file with a non-text extension is requested /
// A file with a non-text extension contains text content——带扩展名的文件不嗅探内容，
// 名字就一锤定音，因此这里不需要真实的文件内容。
//
// 名称没有扩展名的文件由内容起始部分判定（Change 05 的分流，design D1），
// 判据本身由 TestLooksLikeText 钉住，端到端行为由 API 层的各条 Scenario 用例覆盖。
//
// 白名单被改动时下面那条集合断言会立刻失败，而不是留下一条没人记得为什么变了的 Scenario。
func TestTextRecognition(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// 纯文本
		{"notes.txt", true},
		{"notes.text", true},
		{"README.md", true},
		{"notes.markdown", true},
		{"app.log", true},
		{"rows.csv", true},
		{"rows.tsv", true},
		// 结构化
		{"config.json", true},
		{"config.yaml", true},
		{"config.yml", true},
		{"Cargo.toml", true},
		{"pom.xml", true},
		{"php.ini", true},
		{"nginx.conf", true},
		{"app.cfg", true},
		{"app.properties", true},
		{"config.env", true},
		// 标记
		{"index.html", true},
		{"index.htm", true},
		{"style.css", true},
		{"style.scss", true},
		{"schema.sql", true},
		// 脚本
		{"run.sh", true},
		{"run.bash", true},
		{"run.zsh", true},
		{"main.py", true},
		{"main.rb", true},
		{"main.pl", true},
		{"main.php", true},
		{"main.lua", true},
		// 源码
		{"main.go", true},
		{"main.rs", true},
		{"Main.java", true},
		{"main.c", true},
		{"main.h", true},
		{"main.cc", true},
		{"main.cpp", true},
		{"main.hpp", true},
		{"main.js", true},
		{"main.mjs", true},
		{"main.cjs", true},
		{"main.ts", true},
		{"main.tsx", true},
		{"main.jsx", true},
		{"main.vue", true},
		{"main.svelte", true},
		// Go 工作区：go.mod / go.sum / go.work 是 proposal 点名的动机文件
		{"go.mod", true},
		{"go.sum", true},
		{"go.work", true},
		// 大小写不敏感：README.TXT 与 README.txt 同等对待
		{"README.TXT", true},
		{"Notes.Md", true},
		{"SCRIPT.SH", true},
		// 图片、音视频、压缩包、字体、可执行文件
		{"logo.png", false},
		{"logo.jpg", false},
		{"logo.gif", false},
		{"logo.ico", false},
		{"logo.svg", false},
		{"clip.mp4", false},
		{"clip.mp3", false},
		{"bundle.js.map", false},
		{"archive.zip", false},
		{"archive.tar.gz", false},
		{"font.woff2", false},
		{"app.exe", false},
		// 二进制数据类
		{"a.bin", false},
		{"a.dat", false},
		{"a.db", false},
		{"a.sqlite", false},
		{"rows.parquet", false},
	}

	for _, c := range cases {
		if got := isTextFile(c.name); got != c.want {
			t.Errorf("isTextFile(%q) = %v，期望 %v", c.name, got, c.want)
		}
	}

	// 白名单与正向用例必须一一对应：清单里少了一项、改了一项，都由这条断言暴露，
	// 而不是留下一条没人记得为什么变了的 Scenario。用例里同一个扩展名可以出现多次
	// （大小写折叠那几条），所以先按集合比。
	covered := map[string]bool{}
	for _, c := range cases {
		if c.want {
			covered[strings.ToLower(c.name[strings.LastIndex(c.name, ".")+1:])] = true
		}
	}
	recognized := make([]string, 0, len(covered))
	for ext := range covered {
		recognized = append(recognized, ext)
	}
	slices.Sort(recognized)
	whitelist := make([]string, 0, len(textExtensions))
	for ext := range textExtensions {
		whitelist = append(whitelist, ext)
	}
	slices.Sort(whitelist)
	if !slices.Equal(recognized, whitelist) {
		t.Errorf("被判定为文本的扩展名 = %v，期望恰为白名单 %v（清单与用例必须一一对应）", recognized, whitelist)
	}
}

// --- Text file recognition：内容分流的各条 Scenario（API 层） ---

// TestAFileWithoutAnExtensionIsRequested 对应 Scenario
// A file without an extension is requested：Makefile 式的文件按内容起始部分判定，
// 不含二进制数据字节即提供内容——Change 04 那一刀在此放开。
func TestAFileWithoutAnExtensionIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	const content = "all:\n\t@echo hi\n"
	writeContent(t, filepath.Join(root, "Makefile"), content)

	body := decodeContent(t, get(t, handler, contentURL("Makefile")))

	if body.Path != "Makefile" {
		t.Errorf("path = %q，期望 %q", body.Path, "Makefile")
	}
	if body.Content != content {
		t.Errorf("content = %q，期望该文件的完整内容 %q", body.Content, content)
	}
}

// TestAFileWithANonTextExtensionContainsTextContent 对应 Scenario
// A file with a non-text extension contains text content：扩展名不在白名单就
// 不嗅探内容，装着纯文本也按非文本拒绝（文件名与内容不符的责任在文件系统，
// Change 04 已接受的取舍，本 Change 不推翻）。
func TestAFileWithANonTextExtensionContainsTextContent(t *testing.T) {
	handler, root := newAPI(t)
	writeContent(t, filepath.Join(root, "photo.png"), "其实全是纯文本")

	expectFailure(t, get(t, handler, contentURL("photo.png")), codeNotText, http.StatusBadRequest)
}

// TestAFileWithoutAnExtensionContainsBinaryDataBytes 对应 Scenario
// A file without an extension contains binary data bytes：起始部分命中
// binary data byte 判据（PNG 魔数里的 0x1A）即拒绝。
func TestAFileWithoutAnExtensionContainsBinaryDataBytes(t *testing.T) {
	handler, root := newAPI(t)
	name := filepath.Join(root, "image")
	if err := os.WriteFile(name, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}

	expectFailure(t, get(t, handler, contentURL("image")), codeNotText, http.StatusBadRequest)
}

// TestAFileWithoutAnExtensionAndWithoutAnyContentIsRequested 对应 Scenario
// A file without an extension and without any content is requested：空内容不含
// 任何二进制数据字节，判为文本（.gitkeep 这类占位文件因此可被请求）。
func TestAFileWithoutAnExtensionAndWithoutAnyContentIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	writeContent(t, filepath.Join(root, ".gitkeep"), "")

	body := decodeContent(t, get(t, handler, contentURL(".gitkeep")))

	if body.Path != ".gitkeep" {
		t.Errorf("path = %q，期望 %q", body.Path, ".gitkeep")
	}
	if body.Content != "" {
		t.Errorf("content = %q，期望空字符串而不是错误", body.Content)
	}
}

// TestBinaryDataBytesAppearOnlyAfterTheStartOfTheContent 对应 Scenario
// Binary data bytes appear only after the start of the content：判定只看起始
// 窗口（sniffWindow），窗口之后出现的二进制数据字节不改变认定。
func TestBinaryDataBytesAppearOnlyAfterTheStartOfTheContent(t *testing.T) {
	handler, root := newAPI(t)
	raw := append(bytes.Repeat([]byte("a"), sniffWindow), 0x01) // 二进制字节在窗口之外
	name := filepath.Join(root, "notes")
	if err := os.WriteFile(name, raw, 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}

	body := decodeContent(t, get(t, handler, contentURL("notes")))

	if body.Path != "notes" {
		t.Errorf("path = %q，期望 %q", body.Path, "notes")
	}
	if body.Content != string(raw) {
		t.Errorf("content 长度 = %d，期望窗口外内容原样保留（总长 %d）", len(body.Content), len(raw))
	}
}

// TestAUTF16TextFileWithoutAByteOrderMarkIsRequested 对应 Scenario
// A UTF-16 text file without a byte order mark is requested：空字节全部只落
// 一个奇偶侧时按豁免处理，LE（奇数位）与 BE（偶数位）两个方向都覆盖。
func TestAUTF16TextFileWithoutAByteOrderMarkIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	le := []byte{'h', 0x00, 'i', 0x00, '\n', 0x00}
	be := []byte{0x00, 'h', 0x00, 'i', 0x00, '\n'}
	if err := os.WriteFile(filepath.Join(root, "le"), le, 0o644); err != nil {
		t.Fatalf("准备 le 失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "be"), be, 0o644); err != nil {
		t.Fatalf("准备 be 失败：%v", err)
	}

	for _, c := range []struct {
		path string
		raw  []byte
	}{{"le", le}, {"be", be}} {
		t.Run(c.path, func(t *testing.T) {
			body := decodeContent(t, get(t, handler, contentURL(c.path)))

			// 识别成功即可；内容按 UTF-8 呈现，空字节是合法 UTF-8，原样保留（design D9）。
			if body.Content != string(c.raw) {
				t.Errorf("content = %q，期望原样内容 %q", body.Content, string(c.raw))
			}
		})
	}
}

// TestNULBytesAppearAtBothEvenAndOddPositions 对应 Scenario
// NUL bytes appear at both even and odd positions：奇偶混杂的空字节没有豁免，
// 即使除空字节外不含其他二进制数据字节也判为非文本。
func TestNULBytesAppearAtBothEvenAndOddPositions(t *testing.T) {
	handler, root := newAPI(t)
	name := filepath.Join(root, "dbdump")
	if err := os.WriteFile(name, []byte{'a', 0x00, 0x00, 'b'}, 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}

	expectFailure(t, get(t, handler, contentURL("dbdump")), codeNotText, http.StatusBadRequest)
}

// TestDetectionComesBeforeOversizeAndStaysConsistent 钉住 design D5 的优先级与
// Requirement 的一致性承诺：无扩展名文件的判定先于 too_large（起始部分是文本的
// 稀疏大文件报 too_large 而不是 not_text），带非文本扩展名的超大文件仍报
// not_text，同一文件的认定在重复请求间保持一致。
func TestDetectionComesBeforeOversizeAndStaysConsistent(t *testing.T) {
	handler, root := newAPI(t)

	// 稀疏大文件：起始窗口是文本，其后的洞读回为空字节——若判定先于超限，
	// 响应是 too_large；若实现把嗅探挪到超限之后，这里会拿到 not_text 而失败。
	sparse := filepath.Join(root, "bigfile")
	file, err := os.Create(sparse)
	if err != nil {
		t.Fatalf("准备文件 %s 失败：%v", sparse, err)
	}
	if _, err := file.WriteString(strings.Repeat("a", sniffWindow)); err != nil {
		t.Fatalf("写入起始文本失败：%v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("关闭文件失败：%v", err)
	}
	if err := os.Truncate(sparse, maxContentBytes+1); err != nil {
		t.Fatalf("制造稀疏大文件失败：%v", err)
	}

	expectFailure(t, get(t, handler, contentURL("bigfile")), codeTooLarge, http.StatusBadRequest)
	expectFailure(t, get(t, handler, contentURL("bigfile")), codeTooLarge, http.StatusBadRequest)

	// 带非文本扩展名的超大文件：不打开文件、不嗅探，无论多大都报 not_text。
	huge := filepath.Join(root, "huge.bin")
	if err := os.WriteFile(huge, make([]byte, maxContentBytes+1), 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", huge, err)
	}
	expectFailure(t, get(t, handler, contentURL("huge.bin")), codeNotText, http.StatusBadRequest)
}

// --- Text file content response ---

func TestATextFileIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs")
	const content = "第一行\n第二行\n"
	writeContent(t, filepath.Join(root, "docs", "notes.txt"), content)

	body := decodeContent(t, get(t, handler, contentURL("docs/notes.txt")))

	if body.Path != "docs/notes.txt" {
		t.Errorf("path = %q，期望规范化的相对路径 %q", body.Path, "docs/notes.txt")
	}
	if body.Content != content {
		t.Errorf("content = %q，期望该文件的完整内容 %q", body.Content, content)
	}
}

func TestAnEmptyTextFileIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	writeContent(t, filepath.Join(root, "empty.txt"), "")

	body := decodeContent(t, get(t, handler, contentURL("empty.txt")))

	if body.Path != "empty.txt" {
		t.Errorf("path = %q，期望 %q", body.Path, "empty.txt")
	}
	if body.Content != "" {
		t.Errorf("content = %q，期望空字符串而不是错误", body.Content)
	}
}

// TestContentIsRejectedForAPathTraversingASymbolicLinkOutward 对应 Scenario
// A path inside the root traverses a symbolic link outward。
//
// 用例名与 browse_test.go 里那条同 Scenario 的用例刻意不同：两条 Requirement
// 各自挂了一条措辞完全相同的 Scenario（Change 02 已有的那条讲列表），因此差异放到
// 测试名上区分，不合并、不重名（沿用 Change 02 tasks 4.5 的做法）。
// 保留同一 fixture 钉住「物理出根必拒」：该行为自 improve-root-confinement 起由
// spec 明确承诺，此前宽松承诺的翻案理由见该 Change proposal 的 Why。
func TestContentIsRejectedForAPathTraversingASymbolicLinkOutward(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "linked-payload")
	target := writeContent(t, filepath.Join(outside, "secret.txt"), "根目录之外的内容")

	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	recorder := get(t, handler, contentURL("link.txt"))

	// 物理判定：字面位置在根目录内，但解析符号链接后落在物理根之外，按越界拒绝，
	// 不返回该文件的内容。
	expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
	if strings.Contains(recorder.Body.String(), "根目录之外的内容") {
		t.Errorf("越界请求返回了根目录外文件的内容：%q", recorder.Body.String())
	}
}

func TestTheContentIsNotDecodableAsUTF8(t *testing.T) {
	handler, root := newAPI(t)
	// 0xff 后面紧跟 0xfe：两个字节在 UTF-8 的任何位置都不构成合法序列。
	raw := []byte("前半段\xff\xfe后半段")
	name := filepath.Join(root, "broken.txt")
	if err := os.WriteFile(name, raw, 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}

	recorder := get(t, handler, contentURL("broken.txt"))

	// 请求成功：无法解码的部分以替换字符呈现，而不是整份内容变成错误（design D9）。
	body := decodeContent(t, recorder)
	if body.Path != "broken.txt" {
		t.Errorf("path = %q，期望 %q", body.Path, "broken.txt")
	}
	// strings.ToValidUTF8 的契约是「每段非法字节序列替换成一个替换字符串」，
	// 因此这里是恰好一个 U+FFFD，不是每个非法字节一个。
	if want := "前半段�后半段"; body.Content != want {
		t.Errorf("content = %q，期望 %q（非法字节序列以 U+FFFD 呈现）", body.Content, want)
	}
}

// --- Preview position representation ---

func TestAFilePositionIsReopened(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "docs", "api")
	writeContent(t, filepath.Join(root, "docs", "api", "spec.md"), "内容")

	first := decodeContent(t, get(t, handler, contentURL("docs/api/spec.md")))
	second := decodeContent(t, get(t, handler, contentURL("docs/api/spec.md")))

	if first.Path != "docs/api/spec.md" {
		t.Errorf("首次的 path = %q，期望 %q", first.Path, "docs/api/spec.md")
	}
	if second.Path != first.Path {
		t.Errorf("第二次的 path = %q，期望与首次相同 %q", second.Path, first.Path)
	}
	if second.Content != first.Content {
		t.Errorf("第二次的 content = %q，期望与首次相同 %q", second.Content, first.Content)
	}
}

func TestThePositionContainsRedundantSegments(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "a", "b")
	writeContent(t, filepath.Join(root, "a", "b", "c.txt"), "内容")

	for _, input := range []string{"a/b/c.txt", "./a/b/c.txt", "a//b/c.txt", "a/./b/c.txt", "a/b/c.txt/", "a/b/../b/c.txt"} {
		t.Run(input, func(t *testing.T) {
			body := decodeContent(t, get(t, handler, contentURL(input)))

			if body.Path != "a/b/c.txt" {
				t.Errorf("path = %q，期望规范化后的 %q", body.Path, "a/b/c.txt")
			}
			if body.Content != "内容" {
				t.Errorf("content = %q，期望该文件的内容", body.Content)
			}
		})
	}
}

// --- Content reading failures ---

// TestFailureCausesAreDistinguishableForContent 逐个构造 Content reading failures
// 的七种失败原因，断言各自的机器可读错误标识互不相同——能区分失败原因才叫标识。
func TestFailureCausesAreDistinguishableForContent(t *testing.T) {
	handler, root := newAPI(t)

	mkdir(t, root, "sub")
	mkdir(t, filepath.Dir(root), "outside")
	writeContent(t, filepath.Join(filepath.Dir(root), "outside", "secret.txt"), "根目录之外")
	locked := writeContent(t, filepath.Join(root, "locked.txt"), "读不了")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("设置文件权限失败：%v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o600) })
	makeFifo(t, filepath.Join(root, "pipe.txt"))
	if err := os.WriteFile(filepath.Join(root, "logo.png"), []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatalf("准备 logo.png 失败：%v", err)
	}
	big := filepath.Join(root, "big.txt")
	if err := os.WriteFile(big, make([]byte, maxContentBytes+1), 0o644); err != nil {
		t.Fatalf("准备 big.txt 失败：%v", err)
	}

	causes := []struct {
		name   string
		path   string
		code   string
		status int
	}{
		{"not_found", "missing.txt", codeNotFound, http.StatusNotFound},
		// 目录同样报 not_a_directory，与 /api/list 上一致：同一份判定、同一个标识。
		{"not_a_directory", "sub", codeNotADirectory, http.StatusBadRequest},
		{"not_a_regular_file", "pipe.txt", codeNotARegularFile, http.StatusBadRequest},
		{"permission_denied", "locked.txt", codePermissionDenied, http.StatusForbidden},
		{"outside_root", "../outside/secret.txt", codeOutsideRoot, http.StatusBadRequest},
		{"not_text", "logo.png", codeNotText, http.StatusBadRequest},
		{"too_large", "big.txt", codeTooLarge, http.StatusBadRequest},
	}

	details := make([]errorDetail, 0, len(causes))
	for _, cause := range causes {
		t.Run(cause.name, func(t *testing.T) {
			recorder := get(t, handler, contentURL(cause.path))
			expectFailure(t, recorder, cause.code, cause.status)
			if strings.Contains(recorder.Body.String(), "根目录之外") {
				t.Errorf("越界请求返回了根目录外的内容：%q", recorder.Body.String())
			}
			detail, _ := decodeError(t, recorder)
			details = append(details, detail)
		})
	}

	for i, cause := range causes {
		for j, other := range causes {
			if i == j {
				continue
			}
			if details[i].Code == details[j].Code {
				t.Errorf("%s 与 %s 的 code 相同（%q），调用方无法区分失败原因", cause.name, other.name, details[i].Code)
			}
		}
	}
}

// TestTheFIFORequestReturnsWithoutBlocking 是 design D10「先 Stat 再判定」的守门人：
// 打开命名管道会一直阻塞到有写端出现，因此这条用例必须在一个确定的时间上限内返回。
// 若端点哪天把类型判定挪到 os.Open 之后，症状不是一条断言失败而是一个永不返回的
// 请求——所以这里用超时而不是普通断言。
func TestTheFIFORequestReturnsWithoutBlocking(t *testing.T) {
	handler, root := newAPI(t)
	makeFifo(t, filepath.Join(root, "pipe.txt"))

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, contentURL("pipe.txt"), nil))
		done <- recorder
	}()

	select {
	case recorder := <-done:
		expectFailure(t, recorder, codeNotARegularFile, http.StatusBadRequest)
	case <-time.After(5 * time.Second):
		t.Fatal("请求在 5 秒内没有返回：端点打开命名管道后阻塞，说明类型判定发生在打开文件之后（design D10）")
	}
}

// TestEveryKnownErrorCodeIsMappedToAStatus 钉住「每个机器可读错误标识都映射到状态码」。
// writeError 对未登记的标识退回 500，而那正是前端按 code 分派时最难排查的一种响应。
func TestEveryKnownErrorCodeIsMappedToAStatus(t *testing.T) {
	for _, code := range []string{
		codeNotFound, codeNotADirectory, codePermissionDenied, codeOutsideRoot,
		codeNotText, codeTooLarge, codeNotARegularFile, codeNotAnImage,
	} {
		if _, ok := codeStatus[code]; !ok {
			t.Errorf("codeStatus 里没有 %q，writeError 会退回 500", code)
		}
	}
}
