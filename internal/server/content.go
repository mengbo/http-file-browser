package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// maxContentBytes 是可提供内容的最大字节数：1 MiB（design D10）。
//
// 超过它返回 too_large 而不是给一段截断内容：截断之后用户以为看完了整个文件，
// 而这件事在界面上看不出来。返回错误让它完全可见。
//
// 数字可调：源码文件几乎都在这个量级以内，而数据类大文件（CSV / JSON dump）本来
// 也不是「预览」的用途；GitHub 的行内展示上限同为 1 MB。spec 只承诺「超过系统可提供
// 内容的最大字节数」，没有写死这个数字，因此调整它不动任何 Scenario。
const maxContentBytes = 1048576

// textExtensions 是已知文本扩展名白名单（design D6）。
//
// 清单进 design 不进 spec：spec 承诺的是「扩展名属于已知文本扩展名」这条规则，
// 具体清单是可变的实现细节，与 Change 03 的 Entry metadata 同一分工。
//
// 白名单而非黑名单：黑名单要枚举所有不该当文本的后缀，而 .txt 里装着二进制、
// .bin 里装着 JSON 这类反例会把枚举撑爆（design D6）。白名单是有限可枚举的，
// Change 05 的放宽因此是「再纳入一个条件」，而不是「穷举剩下的」。
var textExtensions = map[string]bool{
	// 纯文本
	"txt": true, "text": true, "md": true, "markdown": true, "log": true, "csv": true, "tsv": true,
	// 结构化
	"json": true, "yaml": true, "yml": true, "toml": true, "xml": true,
	"ini": true, "conf": true, "cfg": true, "properties": true, "env": true,
	// 标记
	"html": true, "htm": true, "css": true, "scss": true, "sql": true,
	// 脚本
	"sh": true, "bash": true, "zsh": true, "py": true, "rb": true, "pl": true, "php": true, "lua": true,
	// 源码
	"go": true, "rs": true, "java": true, "c": true, "h": true, "cc": true, "cpp": true, "hpp": true,
	"js": true, "mjs": true, "cjs": true, "ts": true, "tsx": true, "jsx": true, "vue": true, "svelte": true,
}

// isTextFile 判定名称所指的文件是否被当作可读文本的文件（design D6）。
//
// 扩展名取最后一个 `.` 之后的部分并做大小写折叠：README.TXT 与 README.txt 同等对待，
// 与 List ordering 的大小写折叠是同一价值观（顺序可复现，取值也不该随大小写抖动）。
//
// 以 `.` 开头的名字（.gitignore）整名就是名字，没有「最后一个点之后」的部分；
// `notes.` 同样没有可用的扩展名。两者都判为非文本。
func isTextFile(name string) bool {
	dot := strings.LastIndex(name, ".")
	if dot <= 0 {
		return false
	}
	return textExtensions[strings.ToLower(name[dot+1:])]
}

type contentResponse struct {
	// Path 是规范化后的相对根目录路径，与列表响应的 path 同一套规范化。
	Path string `json:"path"`
	// Content 是文件内容，以 UTF-8 呈现。
	Content string `json:"content"`
}

func (b *browser) handleContent(w http.ResponseWriter, r *http.Request) {
	result, apiErr := b.content(r.URL.Query().Get("path"))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// content 返回相对路径 rel 所指文件的内容，rel 为空表示根目录本身。
func (b *browser) content(rel string) (*contentResponse, *apiError) {
	// 越界判定复用列表端点的那一份实现（design D2）：端点语义解耦，但字面路径的
	// 越界判定只有一份代码。少一份实现就少一处将来会漂移的地方。
	abs, normalized, apiErr := b.resolve(rel)
	if apiErr != nil {
		return nil, apiErr
	}

	// 存在性、类型与大小全部在打开文件之前用一次 Stat 拿到（design D10）。
	// 这不只是效率：打开命名管道会一直阻塞到有写端出现，一个永不返回的请求
	// 比任何错误响应都糟，因此 not_a_regular_file 必须在 os.Open 之前判定。
	info, err := os.Stat(abs)
	if err != nil {
		return nil, classify(err)
	}
	if info.IsDir() {
		return nil, fail(codeNotADirectory, "目标不是目录")
	}
	if !info.Mode().IsRegular() {
		return nil, fail(codeNotARegularFile, "目标不是普通文件")
	}

	// 判定为非文本排在超限之前：名字已经说明这个文件不会被预览，
	// 此时报「文件过大」会暗示它小一点就能看，那是句假话。
	if !isTextFile(filepath.Base(abs)) {
		return nil, fail(codeNotText, "该文件不是可读的文本文件")
	}
	if info.Size() > maxContentBytes {
		return nil, fail(codeTooLarge, "文件超过可提供内容的最大字节数")
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, classify(err)
	}

	// 内容一律按 UTF-8 处理，不做编码检测（design D9）：WHATWG 的编码机制结构性地
	// 只会给出 utf-8 / utf-16be / utf-16le / windows-1252，永远不返回 GBK / Shift_JIS，
	// 对本项目最需要的 CJK 场景无用，而真正的 CJK 检测是统计性的（uchardet、ICU），
	// 没有零依赖的 Go 等价物。无法解码的字节以 U+FFFD 呈现而不是报错：GBK 文件
	// 因此显示为乱码，而「不检测比检测错更诚实」——一个自信宣称「这是 GBK」却猜错的
	// 实现比明摆着的乱码更难排查。
	return &contentResponse{
		Path:    normalized,
		Content: strings.ToValidUTF8(string(raw), "�"),
	}, nil
}
