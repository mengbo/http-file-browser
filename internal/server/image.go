package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// imageExtensions 是已知图片扩展名白名单，同时是扩展名 → 图片内容类型的映射
// （image-preview design D2）。
//
// 与 textExtensions 同一分工：清单进 design 不进 spec——spec 承诺的是「扩展名属于
// 已知图片扩展名」这条规则，具体清单是可变的实现细节。命中即按名字认定、不读内容、
// 不验魔数：文件名与内容不符的责任在文件系统（Change 04 已接受取舍的同构重演），
// 解码失败由浏览器 onerror 兜底、前端给回退说明——「不检测比检测错更诚实」的又一实例。
//
// web/app.js 的 IMAGE_EXTENSIONS 是本表的前端镜像（isImagePath，design D4）：
// 两份清单各自独立声明、注释互引提醒同步；漂移两个方向都无害（见 design D4 的论证），
// 但同步仍是本意。
//
// 排除项（design D2）：tiff/heic——目标浏览器渲染支持面割裂，白名单只承诺
// 「进来就能渲染」的格式；svg——文本型图像，记入想法池独立立 Change。
var imageExtensions = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"gif":  "image/gif",
	"webp": "image/webp",
	"bmp":  "image/bmp",
	"ico":  "image/x-icon",
	"avif": "image/avif",
}

// handleImage 服务图片内容端点 GET /api/image?path=...（image-preview design D1）。
// 端点留在 /api/ 分区内、路径走 query 参数：全项目的路径约定就是 ?path=，且
// ServeMux 对 URL 路径段的清理重定向会与 normalize-then-check 打架，query 绕开全部这些。
func (b *browser) handleImage(w http.ResponseWriter, r *http.Request) {
	if apiErr := b.image(w, r, r.URL.Query().Get("path")); apiErr != nil {
		writeError(w, apiErr)
	}
}

// image 把相对路径 rel 所指文件的图片内容按流式发送给客户端。
//
// 判定顺序继承 Change 04 design D10 的纪律（image-preview design D2）：
// resolve → Stat（存在性/类型，命名管道在打开之前被拦下）→ 扩展名识别 → 打开 → 发送。
// 识别先于打开：非图片文件即使恰好不可读也报 not_an_image——先打开再识别的话，
// permission_denied 会暗示「有权限就能看」，那是句假话，与 not_text 先于 too_large
// 同一诚实逻辑。
func (b *browser) image(w http.ResponseWriter, r *http.Request, rel string) *apiError {
	// 越界判定复用 resolve 的同一份实现（image-preview design D2 的复用清单）：
	// 端点语义解耦，但字面路径的越界判定只有一份代码，少一处将来会漂移的地方。
	abs, _, apiErr := b.resolve(rel)
	if apiErr != nil {
		return apiErr
	}

	// 存在性与类型在打开文件之前用一次 Stat 拿到：打开命名管道会一直阻塞到有写端
	// 出现，not_a_regular_file 必须在 os.Open 之前判定（Change 04 design D10）。
	info, err := os.Stat(abs)
	if err != nil {
		return classify(err)
	}
	if info.IsDir() {
		return fail(codeNotADirectory, "目标不是目录")
	}
	if !info.Mode().IsRegular() {
		return fail(codeNotARegularFile, "目标不是普通文件")
	}

	// 识别只看名字（design D2）：扩展名折叠大小写，与 isTextFile 同一取法——
	// dot > 0 使点开头的文件（.gitignore）按无扩展名对待。命中即给，不读内容。
	name := filepath.Base(abs)
	dot := strings.LastIndex(name, ".")
	contentType := ""
	if dot > 0 {
		contentType = imageExtensions[strings.ToLower(name[dot+1:])]
	}
	if contentType == "" {
		return fail(codeNotAnImage, "该文件不是可识别的图片文件")
	}

	file, err := os.Open(abs)
	if err != nil {
		// 权限等读取错误在打开这一步才浮现：识别已通过，这里的问题是「读不了」
		// 而不是「不是图片」，两类失败因此各得其所。
		return classify(err)
	}
	defer file.Close()

	// ServeContent 对 *os.File（天然 ReadSeeker）做 32 KB 缓冲流式拷贝（design D3）：
	// 服务端内存与文件大小无关，不设大小上限——文本的 1 MiB 是整份进内存那条传输
	// 路径的保险丝，流式路径没有它要保护的对象。顺带获得 Range/Last-Modified，
	// 不承诺也不禁止（design D6 的行为面说明），测试只覆盖 GET。
	// name 传空串：内容类型由本端点按扩展名决定，不交给 ServeContent 按名字再嗅探。
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "", info.ModTime(), file)
	return nil
}
