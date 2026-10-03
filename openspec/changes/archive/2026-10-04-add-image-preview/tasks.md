# Tasks

## 1. 服务端：图片内容端点

- [x] 1.1 `internal/server` 新增图片识别表 `imageExtensions`（扩展名 → Content-Type，design D2 的八项），注释与 `textExtensions` 同分工（清单进 design 不进 spec），并与 `web/app.js` 的前端映射注释互引提醒同步。验证：`go build ./...` 通过。
- [x] 1.2 实现图片内容端点 `GET /api/image?path=...` 并注册路由：`resolve` → Stat（存在性/类型，FIFO 不阻塞）→ 扩展名识别（识别先于打开，design D2）→ `os.Open` → `http.ServeContent` 流式发送（design D1/D3）；错误经既有 `fail`/`writeError` 走 JSON 信封，新错误标识 `not_an_image`。验证：`go build ./...` 通过，手工 `curl` 一个 png 得到字节与 `image/png`。
- [x] 1.3 Go 测试覆盖 spec「An image file is requested」「A file with a known image extension is requested」「Image content endpoint succeeds」（design D6 前两行 + service-startup 新 Scenario）：200、`Content-Type` 按扩展名、响应体逐字节等于文件内容、成功响应内容类型非 `application/json`。验证：`go test ./internal/server/` 通过。
- [x] 1.4 Go 测试覆盖「Image file recognition」四条 Scenario：命中提供；非图片扩展名 → `not_an_image`；`.bin` 装图片字节 → 仍 `not_an_image`；无扩展名装图片字节 → 仍 `not_an_image`。验证：`go test ./internal/server/` 通过。
- [x] 1.5 Go 测试覆盖「Image content failures」六条与「A path inside the root traverses a symbolic link outward」「The position contains redundant segments」：六个错误标识各一条，符号链接向外提供、冗余片段按规范化路径提供。验证：`go test ./internal/server/` 通过。
- [x] 1.6 Go 测试覆盖 service-startup 收窄后的「API path is requested」：`GET /api/image?path=file.txt` 返回 JSON 错误响应（错误信封仍 JSON 的直接证据，design D6 末行）。验证：`go test ./internal/server/` 通过。

## 2. 前端：分派与图片视图

- [x] 2.1 `web/app.js` 新增 `isImagePath` 与图片扩展名映射（`isMarkdownPath` 同形，design D4）：`load()` 的 `not_a_directory` 分支按扩展名分派——图片扩展名进图片视图，不请求 `/api/content`；其余走既有 content 流程，行为零变化。验证：`go run .` 后点击 `.go` 文件仍走文本预览，`curl` 之外的请求路径无回归。
- [x] 2.2 `web/index.html` 新增 `<img id="image-view" hidden>` 挂载点（与 `#preview` 平级，design D5）；`web/style.css` 补图片视图样式（居中、尺寸约束、加载态）。验证：页面无控制台报错，文本/Markdown 视图样式不受牵连。
- [x] 2.3 实现图片视图渲染与 `onerror` 回退（design D5）：`src` 指向 `/api/image?path=<被查看路径>`；加载失败显示回退说明（新文案）；`hidePreview` 一并清理图片视图与载荷；呈现形式切换按钮对图片文件隐藏；`ERROR_TEXT` 增补 `not_an_image`。验证：手工走查正常 png 显示、切换目录后图片视图与载荷被清空。
- [x] 2.4 复核 `web/app.js` 中两条防线的注释表述（Change 06/07 的注释链）是否需要为图片分支补一句「图片不进 `#preview` 的原因」（design D5 的不变量论证）。验证：注释与实际行为一致，无过时表述。

## 3. 集成验收（design D6 分派表）与记账

- [x] 3.1 agent-browser 走查「An image file is viewed」：点击 png/jpg/webp 各一，图片在文件视图中加载呈现，位置与上级入口正确。验证：走查记录（截图或 DOM 断言输出）。
- [x] 3.2 agent-browser 走查「An image that cannot be loaded is viewed」：构造内容损坏的 `.png`，回退说明出现、无静默破图占位、不误报为服务错误。验证：走查记录。
- [x] 3.3 回归走查：点击文本、Markdown、目录条目与点开不存在文件，确认既有行为零变化（分派只加分支不改旧路）。验证：走查记录。
- [x] 3.4 全量回归：`go test ./...` 与 `openspec validate --strict` 通过。验证：两条命令退出码为 0。
- [x] 3.5 记账：`docs/roadmap.md` Change 地图 #09 💡→🚧（备注补「携带 service-startup MODIFIED：/api/ 分区为图片字节开口；SVG 记入想法池」）；想法池新增「SVG 图片预览」条目（拖 text 白名单、呈现形式推广、直接打开 URL 的脚本执行面三件事）。验证：roadmap 两处更新与探索结论一致。
