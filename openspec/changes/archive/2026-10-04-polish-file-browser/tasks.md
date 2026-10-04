# Tasks

## 1. 图标资源 vendoring

- [x] 1.1 下载 Lucide v0.x.x（design D2，固定版本号）的 `folder.svg` 与 `file.svg`，分别落盘为 `web/vendor/icons/dir.svg` 与 `web/vendor/icons/file.svg`。文件首行注释标注版本号、获取 URL、获取时间（沿用 Change 06 task 1.4 vendored 注释格式）。验证：`go build ./...` 通过（embed 编译期会校验文件存在）。
- [x] 1.2 `web/web.go` 的 `//go:embed` 声明无须手动改——Go embed 自动包含 `vendor` 下所有子目录（与现有 `highlight.min.css` / `markdown-it.min.js` 同机制）。验证：手工 `curl http://127.0.0.1:8080/vendor/icons/dir.svg` 返回 SVG 字节，`Content-Type` 为 `image/svg+xml`。
- [x] 1.3 验证 vendored SVG 自带 `stroke="currentColor"`：读取 `web/vendor/icons/dir.svg` 文件内容，确认根 `<svg>` 元素的 `stroke` 属性为 `currentColor`（Lucide 默认即此，验收点是文件原样入库）。验证：`grep 'stroke="currentColor"' web/vendor/icons/*.svg` 各命中一次。

## 3. 前端：图标渲染与视觉调优

- [x] 3.1 `web/app.js` 的 `renderEntry(entry)` 在 `.entry` 内、`.entry-link` 之前插入 `<img class="entry-icon" src="/vendor/icons/<dir|file>.svg" alt="" aria-hidden="true">`：根据 `entry.type === "directory"` 选 `dir.svg`、否则 `file.svg`；不引入新 helper、不动 `.entry-link` 的内容（spec「条目中只提供名称」仍成立）。验证：手工 `go run . testdata` 后在浏览器开发者工具看到每个 `#entry` 内 `.entry-icon` 在 `.entry-link` 之前；点击文件/目录仍走既有分派（图片扩展名进图片视图、Markdown 进渲染等），零回归。
- [x] 3.2 `web/style.css` 新增 `.entry-icon` 规则：`width: 1rem`、`height: 1rem`、`flex-shrink: 0`、`vertical-align: middle`、`margin-right: 0.4rem`（design D3）；`color` 浅色 `#6e6e73`、深色 `#98989d`（design D4）。验证：DevTools 计算样式断言与 design D3/D4 一致。
- [x] 3.3 `web/style.css` 视觉调优（design D5，逐项替换）：主色 `#0b62d0` → `#007AFF`；深色模式主色 `#64a8ff` → `#0a84ff`；背景 `#f5f5f7` → `#f2f2f7`；条目圆角 `6px` → `8px`；卡片圆角 `8px` → `10px`；条目行 padding `0.4rem 0.5rem` → `0.55rem 0.5rem`；行高 `1.5` → `1.55`；hover 背景叠加 `box-shadow: 0 1px 3px rgba(0,0,0,0.08)`（浅色）与 `rgba(0,0,0,0.4)`（深色）。**不改**：`.browser` max-width、grid 列宽、34rem 媒体查询断点、`#preview` 纸面语言、字体栈。验证：`grep` 各旧值在仓库中零残留（注释同步改写，例如 vendored highlight 注释里的 `#0b62d0` 引用如存在则一并改）；DevTools 计算样式逐项断言（design D8 表）。
- [x] 3.4 `web/index.html` 的 `<style>` / `<script>` 引用注释复核：vendor 注释中如有「`#0b62d0` 之类的具体色值引用」（之前 Change 06/07/09 的注释），按 D5 的新色值改写。**不**改 .browser 容器结构、不动 `#preview` 不变量注释。验证：`grep '0b62d0\|64a8ff' web/` 零命中（vendored 库二进制文件除外——见 1.1 的版本号注释位）。

## 4. 呈现层验收（ADR-0003 与 design D7 分派表）

- [x] 4.1 agent-browser 走查「目录条目显示目录图标」与「文件条目显示文件图标」（design D7）：列出 `testdata` 根（含 `docs/`、`code/`、`photos/` 等目录与 `README.md` 等文件），断言 `#entries > .entry` 数与 `/api/list` 返回的条目数相等；每个条目的 `.entry-icon` 的 `src` 在文件条目为 `/vendor/icons/file.svg`、目录条目为 `/vendor/icons/dir.svg`；图标的 `naturalWidth === 24`（Lucide 视图框）。验证：走查记录（断言输出粘贴）。
- [x] 4.2 agent-browser 走查「目录与文件图标在外观上可区分」：同 4.1 的目录，断言所有目录条目的 `.entry-icon` `src` 全部相同、所有文件条目的 `.entry-icon` `src` 全部相同、且两组 `src` 不同（可由目录组与文件组各取一条对比）。验证：走查记录。
- [x] 4.3 agent-browser 走查「图标在浅色与深色外观下均可识别」（design D4 对比度）：浅色模式下 `getComputedStyle(icon).color === 'rgb(110, 110, 115)'`（即 `#6e6e73`）；深色模式下断言 `color === 'rgb(152, 152, 157)'`（即 `#98989d`）。**不**计算对比度（依赖工具链会越界——视觉合格的关节点通过代查工具离线算，附在本任务验收里：浅色 4.6:1、深色 7.4:1）。验证：DevTools 断言输出 + 对比度数值的离线算式贴出。
- [x] 4.4 agent-browser 走查「文件视图与搜索结果视图不显示视觉图标」：依次进入 `testdata/photos/spectrum.png`（图片视图）、`testdata/docs/notes.md`（Markdown 渲染视图）、`testdata/docs/hello.txt`（素文本视图），断言 `#preview` 与 `#image-view` 内**均无** `.entry-icon` 元素；执行 `?q=hello`（搜索结果视图），断言 `.match` 内**均无** `.entry-icon`。验证：走查记录。
- [x] 4.5 视觉调优非 spec 断言（design D8）：DevTools 计算样式逐项验证主色 `#007AFF` / `#0a84ff`、背景 `#f2f2f7` / `#1c1c1e`、条目圆角 `8px`、卡片圆角 `10px`、hover 状态 `box-shadow` 非 `none`。验证：DevTools 断言输出。
- [x] 4.6 窄视口回归（Change 02 D8 / Change 03 观察 4 纪律）：视口宽度 320px 下断言 `document.documentElement.scrollWidth === window.innerWidth === 320`；`.entries-header` / `.entry` 的 grid 列宽切换为 `minmax(0, 1fr) 4.25rem 8.25rem`；三列全部保留（修改时间不被隐藏）、图标随列宽收窄与文字列共用首列且不溢出。验证：DevTools 断言 + 截图。
- [x] 4.7 回归走查（design D7 + Change 09 task 3.3 同形）：点击目录条目进子目录、点 `.txt` 进文本预览、点 `.md` 进 Markdown 渲染（默认渲染形式）、点 `.png` 进图片视图、点 `.go` 进语法高亮源码、点不存在路径得 `not_found`、点空目录得「此目录为空」、搜索空查询得全部条目。所有既有行为零变化（视觉精修与图标均不触动交互逻辑）。验证：走查记录 + `go test ./...` 全量回归通过。
- [x] 4.8 全量回归：`go build ./...`、`go test ./...`、`openspec validate --strict` 三条退出码为 0。验证：三条命令各自输出尾部粘贴。

## 5. 记账

- [x] 5.1 `docs/roadmap.md` Change 地图 #16 💡→🚧，备注补「视觉精修 + 类型化视觉图标；spec 进 `directory-browsing` ADDED，CSS 调优属 design 不入 spec」。验证：roadmap 已更新。
- [x] 5.2 `docs/journal.md` 补 Change 16 节，按教材第 57 节三个现象记（现象一 spec 是否写成实现、现象二 是否有被误做成 ADDED 的需求变化、现象三 apply 是否偷偷扩大范围）。归档时一并完成。验证：journal 节已落。
- [x] 5.3 vendored 资源同步纪律登记：想法池新增一条「升级 Lucide 图标库时复查 dir/file 图标的 currentColor 兼容性、视图框尺寸、视觉对比度」，与 Change 06「vendored 资源升级时这一步要重做」纪律同源。验证：roadmap 想法池已写入。

## 6. 风格修订重验（apply 阶段用户实机评审后，极简中性取代 macOS 蓝）

- [x] 6.1 CSS 风格修订落地：链接色 `#007AFF`/`#0a84ff` → 正文近黑 `#1d1d1f`/近白 `#f5f5f7`（条目、上级、Markdown 链接、切换按钮、focus 边框深色 `#98989d`）；hover 移除 `box-shadow`（浅深两套），纯背景 `#eceef2`/`#2c2c2e`；Markdown 链接常驻细下划线 `#c7c7cc`（深色 `#48484a`，hover 满色）；`.entry-link:hover` 补下划线。图标灰阶刻意不动。验证：`grep '007AFF\|0a84ff\|box-shadow' web/style.css` 零命中。
- [x] 6.2 artifacts 同步：proposal What Changes/Impact 改写为极简中性描述；design D5 表格重写 + 修订记录、D4 链接色句、D8 断言值更新（链接色近黑/近白、hover 阴影断言反转为 `none`）。验证：`grep '007AFF\|0a84ff' openspec/changes/polish-file-browser/` 仅命中 D5/D8 的「废弃值」记录处。
- [x] 6.3 走查重验（浅色）：`bodyBg` rgb(242,242,247)、`linkColor` rgb(29,29,31)、`iconColor` rgb(110,110,115)（不变）、圆角 8px/10px、hover 背景 rgb(236,238,242) 且 `boxShadow === "none"`。（深色）：`linkColor` rgb(245,245,247)、`iconColor` rgb(152,152,157)（不变）、focus 边框 rgb(152,152,157)。Markdown 链接 `#1d1d1f` + 下划线 rgb(199,199,204)。验证：agent-browser 断言输出（本节记录即粘贴）。
- [x] 6.4 快速回归 + 全量三连：目录→`notes.md` 渲染视图、`spectrum.png` 图片加载、不存在路径 `该位置不存在`；`go build ./...`、`go test ./...`、`openspec validate --changes --strict` 全部退出码 0。验证：输出尾部 ALL_GREEN。
## 7. GitHub 风格二次修订（update 阶段经用户确认后由 apply 落地）

- [x] 7.1 `web/style.css` 按 design D5（GitHub 定稿）整表替换：画布 `#ffffff`/`#0d1117`、链接 `#0969da`/`#4493f8`、辅助信息 `#656d76`/`#8b949e`、边框 `#d0d7de`/`#30363d`、行分隔 `#d8dee4`/`#21262d`、hover `#f6f8fa`/`#161b22`、圆角统一 6px、输入框 `#f6f8fa`/`#161b22` + focus `#0969da` + 光圈、按钮 `#f6f8fa`/`#21262d`、字体栈补 `"Segoe UI", "Noto Sans"`、Markdown 链接改为 GitHub 蓝 hover 下划线。**不改**：`#preview` 纸面（白底不随外观）、grid 列宽与 34rem 断点、图标尺寸/DOM。验证：`grep` 旧值零残留。
- [x] 7.2 agent-browser 重验（design D8 新值）：浅色 `bodyBg` rgb(255,255,255)、`linkColor` rgb(9,105,218)、`iconColor` rgb(101,109,118)、圆角 6px、hover rgb(246,248,250) 且无阴影；深色 `bodyBg` rgb(13,17,23)、`linkColor` rgb(68,147,248)、`iconColor` rgb(139,148,158)；Markdown 链接 rgb(9,105,218)；320px 窄视口仍无溢出。验证：走查记录 + 截图。
- [x] 7.3 全量回归：`go build ./...`、`go test ./...`、`openspec validate --changes --strict` 退出码 0。验证：输出尾部 ALL_GREEN。
- [x] 7.4 记账：journal Change 16 节补第三轮风格定稿观察；确认 spec 零 delta（再次印证 D5）。验证：journal 已落。
