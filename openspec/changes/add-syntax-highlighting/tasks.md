# Tasks

## 1. vendored 资源引入与静态服务

- [x] 1.1 从 highlight.js 官方构建产物取 common bundle 的 `highlight.min.js` 与浅色主题 `github.min.css`，落位 `web/vendor/`，并写版本说明注释（design D4）。验证：两文件存在、注释含版本号、不引入任何 npm 清单。
- [x] 1.2 `web/web.go` 的 `//go:embed` 清单加入 `vendor` 目录。验证：`go build ./...` 成功。
- [x] 1.3 `web/index.html` 以 `<script>` 与 `<link>` 引入 vendored 文件。验证：启动服务后打开页面，浏览器控制台无 404 与报错。
- [x] 1.4 在 `internal/server` 补 Go 测试覆盖 spec Scenario「The presentation resources are requested」：`GET /vendor/highlight.min.js` 返回 200、非空、`Content-Type` 为 `text/javascript`。验证：`go test ./internal/server/` 通过。

## 2. 高亮呈现与安全防线

- [x] 2.1 实现 design D2 的输出形状校验：仅允许 `<span>` 开/闭标签、属性仅 `class`，校验失败回退素文本。验证：以构造的畸形输入（含 `<script>`、非 span 标签的字符串）调用该函数确认拦截，走查确认正常 hljs 输出可通过。
- [x] 2.2 `renderContent` 接入 highlight.js：扩展名经 hljs 语言别名映射优先，无映射时 `highlightAuto`，未识别/低置信回退素文本（design D3）；高亮与回退两路都写入后不改变 `#preview` 的 `textContent`。验证：`go run .` 后 agent-browser 打开样例 `.go` 与无扩展名二进制文件，分别看到高亮与素文本。
  - 走查注记：验证后半句的「无扩展名二进制文件」按 spec 的落点是「无扩展名且语言无法识别的文本文件」（`notes`）——真二进制文件在 `text-preview` 判定即被拒（`not_text`），到不了高亮层，本 Change 不改该行为。
- [x] 2.3 改写 `web/index.html` 与 `web/app.js` 中 D11 不变量的注释，指向本 Change design D2 的新表述。验证：注释与新行为一致，无「永不交给 HTML 解析器」的过时表述。
- [x] 2.4 协调 vendored 主题样式与 `web/style.css` 既有 preview 样式（背景、字号、换行）。验证：走查高亮文件视觉与素文本文件无样式冲突。

## 3. 集成验收（design D5 分派表）

- [x] 3.1 agent-browser 走查「A file in a highlighted language is viewed」与「The same file is viewed repeatedly」：样例代码文件两次渲染的 hljs 类名一致。验证：走查记录（截图或 DOM 断言输出）。
- [x] 3.2 agent-browser 断言「Text is taken from a highlighted presentation」与「A file whose content contains markup-like text is viewed」：`#preview` 的 `textContent` 与 content API 返回的 `content` 逐字符一致，markup-like 文件无可执行标记节点。验证：断言脚本输出通过。
- [x] 3.3 agent-browser 走查「A file whose language is not recognized is viewed」：素文本呈现、`textContent` 一致、无错误提示。验证：走查记录。
- [x] 3.4 全量回归：`go test ./...` 与 `openspec validate` 通过；`web/vendor/` 注释中的版本号与运行时 `hljs.version` 一致。验证：两条命令退出码为 0。
  - 走查注记：11.12.0 构建产物的运行时版本属性名为 `hljs.versionString`（值 11.12.0，与注释一致）；`hljs.version` 在该版本不存在。
