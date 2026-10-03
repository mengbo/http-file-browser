# Tasks

## 1. vendored 资源引入与静态服务

- [x] 1.1 取 markdown-it 单文件 minified UMD 构建产物，落位 `web/vendor/markdown-it.min.js`，并写版本说明注释（design D8）。验证：文件存在、注释含版本号、不引入任何 npm 清单。
- [x] 1.2 `web/index.html` 以 `<script>` 引入 vendored 文件（`web/web.go` 的 embed 已含 `vendor` 目录，预期零变化）。验证：`go build ./...` 成功；启动服务打开页面，浏览器控制台无 404 与报错。
- [x] 1.3 在 `internal/server` 补 Go 测试覆盖 spec Scenario「The presentation resources are requested」：`GET /vendor/markdown-it.min.js` 返回 200、非空、`Content-Type` 为 `text/javascript`。验证：`go test ./internal/server/` 通过。

## 2. 渲染管线与安全防线

- [x] 2.1 `renderContent` 接入 Markdown 分支：扩展名 `md`/`markdown` 走 markdown-it（`html: false`）渲染，`innerHTML` 写入并切换 `preview markdown` 类；渲染抛错整段回退 `textContent` 素文本、不报错（design D2、D3、D7）。验证：`go run .` 后 agent-browser 打开样例 `.md` 看到渲染结构；以注入异常方式验证回退呈现且无错误提示。
- [x] 2.2 改写 `web/app.js` 中 Change 06 D2 的注释（「交给 HTML 解析器的字符串只能来自 highlight.js 的输出」），表述为两条管线各自的防线：高亮与代码块路径靠形状校验，Markdown 主输出靠 `html: false` 配置性封闭（design D2、D7）。验证：注释与新行为一致，无过时表述。
- [x] 2.3 代码块 highlight 钩子接 vendored hljs：fence 语言经 `hljs.getLanguage` 命中即 `highlight`，无标注/未识别走 `highlightAuto` 低置信回退素文本，hljs 输出过 `safeHighlightHTML` 形状校验，单块失败不影响文件其余部分（design D6）。验证：agent-browser 走查标注 `python` 的代码块高亮、无标注代码块素文本。
- [x] 2.4 `web/style.css` 增加 Markdown 渲染排版样式（标题、列表、代码块、引用、表格等），协调 vendored hljs 主题与既有 preview 样式。验证：走查渲染文件与素文本、高亮文件三者互不样式冲突。

## 3. 链接与图片

- [x] 3.1 相对链接重写为应用内导航（design D4）：解析基准为被查看文件所在目录，处理上级目录片段归一与 `/` 开头的根相对，`#` 锚点原样保留，重写经 `setAttribute` 实现。验证：agent-browser 点击同级、上级、根相对三类链接，URL 与呈现均落到解析出的目标位置。
- [x] 3.2 外部链接加 `target="_blank"` 与 `rel="noopener"`（design D4）。验证：agent-browser 断言属性存在且激活后当前页呈现不变。
- [x] 3.3 图片不做重写，按浏览器语义加载（design D5）。验证：agent-browser 走查绝对地址图片加载呈现、相对地址图片呈现加载失败且无错误提示、其余内容完好。

## 4. 集成验收（design D9 分派表）

- [x] 4.1 agent-browser 走查「A markdown file is viewed」「Content resembling HTML markup is viewed」「Rendering does not succeed」：渲染节点出现；标记文本字面呈现且无可执行标记节点；回退素文本且无错误提示。验证：走查记录（截图或 DOM 断言输出）。
- [x] 4.2 agent-browser 断言 Link navigation 三条与 Image 两条 Scenario：相对链接（含上级片段）落点正确、外部链接新标签且当前页不变、图片按 D5 行为呈现。验证：断言脚本输出通过。
- [x] 4.3 agent-browser 断言 Code block 三条 Scenario：识别语言高亮、未识别素文本、代码块 `textContent` 与 fence 内源文本逐字符一致。验证：断言脚本输出通过。
- [x] 4.4 全量回归：`go test ./...` 与 `openspec validate` 通过；`web/vendor/` 注释中的版本号与运行时一致。验证：两条命令退出码为 0。
