package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func imageURL(path string) string {
	return "/api/image?path=" + url.QueryEscape(path)
}

func writeImageContent(t *testing.T, name string, raw []byte) string {
	t.Helper()
	if err := os.WriteFile(name, raw, 0o644); err != nil {
		t.Fatalf("准备文件 %s 失败：%v", name, err)
	}
	return name
}

func decodeImage(t *testing.T, recorder *httptest.ResponseRecorder, wantContentType string) []byte {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d（body = %q）", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	contentType := recorder.Header().Get("Content-Type")
	if contentType != wantContentType {
		t.Fatalf("Content-Type = %q，期望 %q", contentType, wantContentType)
	}
	// service-startup 的 Image content endpoint succeeds：成功响应 SHALL NOT 以 JSON
	// 响应体返回。上一条相等断言已把它排除；这里显式写出，让 design D6 末两行的
	// 要求一眼可见，也防将来有人把成功路径误改成先写 JSON 头。
	if strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("图片内容端点的成功响应以 JSON 响应体返回（spec: Image content endpoint succeeds）")
	}
	return recorder.Body.Bytes()
}

// pngMagic 是真实 PNG 的魔数。识别只看名字不看内容（design D2），用例不需要完整
// 可解码的图片；魔数只负责把「内容为图片内容」这件事写实，供不命中与无扩展名用例装载。
var pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

// --- Image content response / Image file recognition：命中的一侧 ---

// TestAnImageFileIsRequested 对应 Scenario An image file is requested（design D6 首行），
// 以及 service-startup 的 Image content endpoint succeeds：成功响应是该文件内容的
// 字节序列，内容类型按扩展名对应的图片内容类型，而非 JSON 响应体。
func TestAnImageFileIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	raw := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0x00, 0xFF}, 32)...)
	writeImageContent(t, filepath.Join(root, "logo.png"), raw)

	body := decodeImage(t, get(t, handler, imageURL("logo.png")), "image/png")

	if !bytes.Equal(body, raw) {
		t.Errorf("响应体与文件内容不一致：got %d 字节，期望 %d 字节逐字节相同", len(body), len(raw))
	}
}

// TestAKnownImageExtensionIsServedWithItsContentType 对应 Scenario
// A file with a known image extension is requested：白名单每一项都按名字命中、
// 不读内容（用例内容统一装 PNG 魔数），内容类型逐项等于映射表的取值。
func TestAKnownImageExtensionIsServedWithItsContentType(t *testing.T) {
	handler, root := newAPI(t)

	for ext, wantType := range imageExtensions {
		t.Run(ext, func(t *testing.T) {
			writeImageContent(t, filepath.Join(root, "image."+ext), pngMagic)

			decodeImage(t, get(t, handler, imageURL("image."+ext)), wantType)
		})
	}

	// 大小写折叠：README.TXT 与 README.txt 同等对待（design D2 与 isTextFile 同一取法）。
	writeImageContent(t, filepath.Join(root, "LOGO.PNG"), pngMagic)
	decodeImage(t, get(t, handler, imageURL("LOGO.PNG")), "image/png")
}

// --- Image file recognition：拒绝的一侧 ---

// TestAFileWithANonImageExtensionIsRequested 对应 Scenario
// A file with a non-image extension is requested：扩展名不在图片白名单即拒绝，
// 与内容无关（识别只看名字，design D2）。
func TestAFileWithANonImageExtensionIsRequested(t *testing.T) {
	handler, root := newAPI(t)
	writeImageContent(t, filepath.Join(root, "clip.mp4"), []byte("其实不是图片"))

	expectFailure(t, get(t, handler, imageURL("clip.mp4")), codeNotAnImage, http.StatusBadRequest)
}

// TestAFileWithANonImageExtensionContainsImageContent 对应 Scenario
// A file with a non-image extension contains image content：.bin 装着图片字节
// 仍按名字拒绝——文件名与内容不符的责任在文件系统，Change 04 取舍的同构重演。
func TestAFileWithANonImageExtensionContainsImageContent(t *testing.T) {
	handler, root := newAPI(t)
	writeImageContent(t, filepath.Join(root, "payload.bin"), pngMagic)

	expectFailure(t, get(t, handler, imageURL("payload.bin")), codeNotAnImage, http.StatusBadRequest)
}

// TestAFileWithoutAnExtensionContainsImageContent 对应 Scenario
// A file without an extension contains image content：无扩展名一律拒绝、不嗅探——
// 与 text-preview 对无扩展名文件的内容判定刻意不同（design D2）。
func TestAFileWithoutAnExtensionContainsImageContent(t *testing.T) {
	handler, root := newAPI(t)
	writeImageContent(t, filepath.Join(root, "image"), pngMagic)

	expectFailure(t, get(t, handler, imageURL("image")), codeNotAnImage, http.StatusBadRequest)
}

// --- Image content failures 与字面路径的两条边界 Scenario ---

// TestFailureCausesAreDistinguishableForImage 逐个构造 Image content failures
// 的六种失败原因（design D6：六个错误标识各一条），断言各自的机器可读错误标识。
func TestFailureCausesAreDistinguishableForImage(t *testing.T) {
	handler, root := newAPI(t)

	mkdir(t, root, "sub")
	mkdir(t, filepath.Dir(root), "outside")
	writeImageContent(t, filepath.Join(filepath.Dir(root), "outside", "secret.png"), pngMagic)
	locked := writeImageContent(t, filepath.Join(root, "locked.png"), pngMagic)
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("设置文件权限失败：%v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o600) })
	makeFifo(t, filepath.Join(root, "pipe.png"))
	writeImageContent(t, filepath.Join(root, "notes.txt"), []byte("纯文本"))

	causes := []struct {
		name   string
		path   string
		code   string
		status int
	}{
		{"not_found", "missing.png", codeNotFound, http.StatusNotFound},
		// 目录同样报 not_a_directory，与 /api/list、/api/content 上一致：同一份判定、同一个标识。
		{"not_a_directory", "sub", codeNotADirectory, http.StatusBadRequest},
		{"not_a_regular_file", "pipe.png", codeNotARegularFile, http.StatusBadRequest},
		{"permission_denied", "locked.png", codePermissionDenied, http.StatusForbidden},
		{"outside_root", "../outside/secret.png", codeOutsideRoot, http.StatusBadRequest},
		{"not_an_image", "notes.txt", codeNotAnImage, http.StatusBadRequest},
	}

	for _, cause := range causes {
		t.Run(cause.name, func(t *testing.T) {
			recorder := get(t, handler, imageURL(cause.path))
			expectFailure(t, recorder, cause.code, cause.status)
			if strings.Contains(recorder.Body.String(), "根目录之外") {
				t.Errorf("越界请求返回了根目录外的内容：%q", recorder.Body.String())
			}
		})
	}
}

// TestImageIsRejectedForAPathTraversingASymbolicLinkOutward 对应 Scenario
// A path inside the root traverses a symbolic link outward。
//
// 用例名与 content_test.go / browse_test.go 里同 Scenario 的既有用例刻意不同：
// 多条 Requirement 各自挂了措辞完全相同的 Scenario，差异放到测试名上区分
// （沿用 Change 02 tasks 4.5 的做法）。
// 保留同一 fixture 钉住「物理出根必拒」：该行为自 improve-root-confinement 起由
// spec 明确承诺，此前宽松承诺的翻案理由见该 Change proposal 的 Why。
func TestImageIsRejectedForAPathTraversingASymbolicLinkOutward(t *testing.T) {
	handler, root := newAPI(t)
	outside := mkdir(t, filepath.Dir(root), "linked-payload")
	target := writeImageContent(t, filepath.Join(outside, "secret.png"), pngMagic)

	link := filepath.Join(root, "link.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("无法创建符号链接（Windows 上可能需要特权），跳过：%v", err)
	}

	recorder := get(t, handler, imageURL("link.png"))

	// 物理判定：字面位置在根目录内，但解析符号链接后落在物理根之外，按越界拒绝，
	// 不返回该文件的图片内容。
	expectFailure(t, recorder, codeOutsideRoot, http.StatusBadRequest)
	if bytes.Contains(recorder.Body.Bytes(), pngMagic) {
		t.Errorf("越界请求返回了根目录外图片的内容：%q", recorder.Body.String())
	}
}

// TestTheImagePositionContainsRedundantSegments 对应 Scenario
// The position contains redundant segments：冗余片段按规范化后的路径提供图片内容。
func TestTheImagePositionContainsRedundantSegments(t *testing.T) {
	handler, root := newAPI(t)
	mkdir(t, root, "a", "b")
	writeImageContent(t, filepath.Join(root, "a", "b", "c.png"), pngMagic)

	for _, input := range []string{"a/b/c.png", "./a/b/c.png", "a//b/c.png", "a/./b/c.png", "a/b/c.png/", "a/b/../b/c.png"} {
		t.Run(input, func(t *testing.T) {
			body := decodeImage(t, get(t, handler, imageURL(input)), "image/png")

			if !bytes.Equal(body, pngMagic) {
				t.Errorf("响应体与规范化路径的内容不一致：got %d 字节", len(body))
			}
		})
	}
}

// --- service-startup：HTTP surface partitioning（收窄后） ---

// TestAPIPathReturnsJSONForImageEndpointErrors 对应 service-startup MODIFIED 后的
// Scenario API path is requested（design D6 末行）：分区承诺只在成功响应上开口，
// 图片端点的错误响应仍是 JSON 信封——同一端点、错误路径的直接证据。
// 既有各 /api/ 端点的 JSON 行为由 server_test.go 的用例继续钉住，全量回归兜底。
func TestAPIPathReturnsJSONForImageEndpointErrors(t *testing.T) {
	handler, root := newAPI(t)
	writeImageContent(t, filepath.Join(root, "file.txt"), []byte("纯文本"))

	recorder := get(t, handler, imageURL("file.txt"))

	expectFailure(t, recorder, codeNotAnImage, http.StatusBadRequest)
	// expectFailure 经 decodeError 已断言 Content-Type 恰为 JSON；这里再钉一次
	// 「JSON 错误信封优先于图片字节」，防将来有人把错误路径也改成裸字节。
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q，期望错误信封仍是 JSON", contentType)
	}
}
