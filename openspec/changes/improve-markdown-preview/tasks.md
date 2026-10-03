# Tasks

## 1. 切换状态与控件

- [x] 1.1 `web/index.html` 在 preview 区附近添加切换按钮（独立于 `#preview`，默认 `hidden`），`web/app.js` 顶部获取该元素引用。验证：启动服务，浏览器中按钮存在于 DOM 且初始不可见。
- [x] 1.2 `web/app.js` 引入模块级源码形式状态（design D1），在 `load()` 入口重置为渲染形式。验证：此步后一切行为与现状一致（渲染照旧），`go build ./...` 通过。

## 2. 呈现分岔与切换行为

- [x] 2.1 `renderContent` 的 Markdown 分支按状态分岔（design D2）：源码形式复用既有高亮管线（`highlightHTMLFor` → `safeHighlightHTML` → `innerHTML`，`preview hljs` 类；null 回退 `textContent`），渲染形式分支原样不动。验证：手动将一个 `.md` 文件切入源码形式，呈现为高亮源码；字符序列与文件内容一致（选中复制比对）。
- [x] 2.2 切换按钮行为（design D3）：点击翻转状态并对当前文件内容重跑呈现分派（不经过 `load()`、不写 History/URL）；按钮 `hidden` 逻辑——仅 Markdown 文件的渲染/源码/回退三态可见，目录、错误态、非 Markdown 文件隐藏；文案表达目标形式（渲染形式下「查看源码」，源码形式下「查看渲染」）。验证：agent-browser 或手动——`.md` 切换往返形态正确；`.txt`、目录视图、错误态下按钮不可见。
- [x] 2.3 注释同步改写（design D4）：`app.js` `safeHighlightHTML` 上方与 `renderContent` 上方两处防线综述改为「Markdown 文件两种呈现形式、两条防线」的表述；`index.html` preview 元素注释同步。验证：逐句对照实际行为，无一句与代码不符。

## 3. 呈现层验收（ADR-0003，agent-browser 走查）

- [x] 3.1 未切换查看（Scenario: A markdown file is viewed）：`#preview` 出现渲染结构节点（标题/段落），无 `hljs` 类；复跑 Scenario: Content resembling HTML markup is viewed——形似 HTML 的文本以字面出现，无可执行标记节点。
- [x] 3.2 渲染失败回退（Scenario: Rendering does not succeed）：按 journal 07.4 的 `defineProperty` 故障注入使 `markdownit` 构造即抛错，`#preview` 素文本呈现且无错误提示。
- [x] 3.3 形式切换（Scenario: The presentation form is switched to source / switched back to rendered）：点击切换后 `#preview` 有 `hljs` 类名且渲染结构消失；再点渲染结构恢复、`hljs` 类消失。
- [x] 3.4 不跨查看保留（Scenario: The presentation form does not persist across views）：切到源码 → 条目点击进目录 → 再点回该文件（pushState 同文档路径，design D5）→ 以渲染形式呈现；补充走查 popstate 回退到文件视图同样回到渲染形式。
- [x] 3.5 渲染形式行为回归：源码形式切回后，相对链接重写导航、字面 HTML、代码块高亮均照常（对照 `markdown-preview` 既有 Scenario 抽查）。

## 4. 回归与收尾

- [x] 4.1 `go test ./...` 全量回归通过（服务端零变化的确认）；`go vet ./...` 无新告警。
- [x] 4.2 `openspec validate "improve-markdown-preview"` 通过；tasks 全部勾选后按仓库 Git 约定准备 `feat: improve-markdown-preview` 提交内容（不自行提交）。
