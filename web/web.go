package web

import "embed"

// FS 是内嵌的前端静态资源，随二进制一同分发，运行时不依赖外部资源目录。
// web/ 自身是 Go 包：embed 指令只能内嵌本包目录及其子目录，internal/ 下的包无法直接引用仓库根目录的 web/。
//
//go:embed index.html login.html app.js style.css vendor
var FS embed.FS
