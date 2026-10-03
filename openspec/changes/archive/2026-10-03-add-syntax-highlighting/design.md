# Design

## Context

文件视图（`#preview`）目前以 `textContent` 素文本呈现内容。Change 02 的 design D11 立有安全不变量：「内容经 textContent 写入，绝不拼接 innerHTML——这个元素永远不交给 HTML 解析器」。语法高亮的核心机制（把文本变成带 `<span>` 的 HTML 并交给解析器）恰恰与该不变量冲突，本 Change 必须自觉地收窄它，而不是无意识绕开。

会被触及的现状：

- `web/app.js` 的 `renderContent`：`previewEl.textContent = content.content`。
- `web/index.html`：`<pre id="preview">` 上方注释声明 D11 不变量。
- `web/web.go`：`//go:embed index.html app.js style.css` 逐文件枚举。
- `internal/server/server.go:103`：`http.FileServer(http.FS(assets))` 服务全部内嵌资源，新增文件无需服务端代码。
- content API 返回的 `content` 字段是「UTF-8 呈现、不可解码字节为替换字符」后的文本——高亮的输入就是它。

## Goals / Non-Goals

**Goals:**

- 查看被支持语言的文件时内容高亮呈现；判定失败时素文本回退，永不因高亮而拒绝或改变内容。
- 高亮是纯客户端呈现变换：`/api/` 端点零变化，三个既有 capability 零 delta。
- 全部资源内嵌自包含，`go build` 即跑，不引入 npm 与打包器（ADR-0002）。
- 呈现层 Scenario 的验收按 ADR-0003 分派并落地。

**Non-Goals:**

- 不做行号、主题切换、语言手动指定（留给 Change 16 polish）。
- 不做服务端高亮（见 D1）。
- 不改 `text-preview` 的文本判定（高亮判定的宽窄独立于文本判定，见 D3）。
- 不追求语言识别的完备性：识别错了只是颜色不对（spec 的 Fallback + Content preservation 兜底）。

## Decisions

### D1. 客户端 vendored highlight.js 全托管

ADR-0002 已预见「渲染库以 vendored 静态文件进仓库」并点名 highlight.js，本次落实：

```
内容 API 返回 content 字段 (纯文本, 已转 UTF-8)
        |
        v
renderContent: hljs.highlight(content, {language})   <- 库内部先 escapeHtml 再插 span
        |                                             <- 库返回 HTML 字符串
        v
(防线, D2) 校验输出形状后写 innerHTML
```

**为什么不是服务端高亮（Go chroma）**：content API 需要新增 HTML 形态字段或改响应形状，`text-preview` spec 要动；「Go 只提供 JSON API」的 ADR-0002 决策被削弱。且服务端高亮无法复用将来 Markdown 渲染（Change 07）的前端管线。客户端高亮让 API 形状与 spec 完全冻结。

**为什么不是自写 tokenizer**：安全论证从「信任业界充分审计的库」退化为「信任自己临时写的解析器」，见 D2。

### D2. D11 不变量的收窄：转义交给库，形状校验自持

本 Change 显式推翻 D11 的字面表述，收窄为：**高亮路径上交给 HTML 解析器的字符串，只能来自 highlight.js 的输出，且写入前经形状校验——校验输出中所有标签均为 `<span>`（开/闭），属性仅允许 `class`**。校验约二十行，不通过则整段回退素文本。

- 信任模型：XSS 防线的第一层是 hljs 的 escapeHtml（其核心特性、业界反复审计）；第二层是自持的形状校验，把「信任库完全正确」收窄为「信任库输出形状」——即使库将来被替换或输出被污染，非 `<span>` 结构也会被拦下。
- 校验失败的回退方向是素文本而非报错：与 spec 的 Fallback 精神一致（呈现永远安全失败）。
- **实现时同步改写 `index.html` 与 `app.js` 中 D11 的注释**，指向本决策，避免注释撒谎（tasks 已列）。

### D3. 语言识别：名字优先、内容兜底的激进策略

与 `text-preview` 判定结构同构、策略相反：

| | text-preview (Change 05) | 本 Change (D3) |
|---|---|---|
| 判定性质 | **门**：判错把二进制吐成文本 | **装饰**：判错只是颜色不对 |
| 名字（扩展名→语言映射） | 白名单内按名字，**不嗅探** | 命中即按映射语言，不跑 auto |
| 内容 | 无扩展名才嗅探，窗口 4096 | 无映射即跑 `highlightAuto` |
| 失败模式 | 拒绝内容（功能受损） | 回退素文本（无害） |

扩展名映射用 hljs 内建的语言别名表（`getLanguage`），不另立白名单；`.txt` 与自然语言文本会得到 `plaintext` 或低置信结果，一律走回退。**不引入与 text-preview 判定的耦合**：一个不被判定为文本的文件根本到不了高亮；而一个无扩展名文本文件（如 `Makefile`）在 text-preview 下靠嗅探进入预览，在本 Change 下靠 auto 得到高亮——两条判定各管各的门。

### D4. vendored 形态：common 语言集 + 固定浅色主题，版本入注释

- 取 highlight.js 的 **common bundle**（约 40 种常用语言，minified 约百 KB 量级）。文件浏览器面对的是任意目录，无法预知用户的语言组合，自选小语言集的省体积换来「浏览到陌生语言全是素文本」的体验断层；全量语言包约 MB 级，不值。
- 文件落位 `web/vendor/highlight.min.js`、`web/vendor/highlight.min.css`（一个固定浅色主题，选接近 Finder 素雅的风格）；`//go:embed` 增加 `vendor` 目录，`http.FileServer` 自动服务 `/vendor/...`。
- 不引入 npm：从 hljs release 直接取构建产物。**版本号写进 `web/vendor/` 内的说明注释**——vendored 文件没有 lock 文件，注释就是版本记录，升级 = 换文件 + 改注释。

### D5. Scenario → 断言位置分派（ADR-0003 首次落地）

| Scenario | 断言位置 | 方式 |
|---|---|---|
| A file in a highlighted language is viewed | 呈现层 | agent-browser 走查：`#preview` 内出现 hljs 类名节点 |
| The same file is viewed repeatedly | 呈现层 | agent-browser 走查：两次渲染类名一致 |
| Text is taken from a highlighted presentation | 呈现层（机器可断言） | agent-browser 断言 `preview.textContent === content API 的 content 字段` |
| A file whose content contains markup-like text is viewed | 呈现层（机器可断言） | 同上等价断言 + 无可执行标记节点 |
| A file whose language is not recognized is viewed | 呈现层 | agent-browser 走查：`#preview` 无 hljs 节点且文本一致 |
| The presentation resources are requested | **API 层** | Go `httptest`：`GET /vendor/highlight.min.js` 200、非空、`text/javascript` |

唯一能进 `go test ./...` 全量回归的是最后一行；呈现层五条在 tasks.md 验收清单里以 agent-browser 走查勾选记录。

## Risks / Trade-offs

- [信任 hljs 转义的正确性] → D2 的形状校验兜底；校验失败即回退素文本。
- [auto 检测对大文件耗时] → 检测输入截断采样（hljs auto 对超长输入自行截断；如仍慢，在调用侧截前 32KB），高亮结果对截断不敏感——只影响开头无 token 的文件。
- [hljs auto 误判语言（如把 Python 认成 Ruby）] → 接受。内容不变承诺保证信息无损；Change 16 可加语言指示器后再议手动纠正。
- [二进制体积 + 百 KB 量级] → 接受。ADR-0001 的单二进制价值在免运行时依赖，不在体积极限。
- [vendored 文件手动升级漂移] → D4 的版本注释 + tasks 验收项「注释中版本号与文件内 `hljs.version` 一致」。

## Open Questions

（无——D4 的主题具体选型、D2 校验的正则实现粒度属实现层，tasks 内解决，不影响 spec 与结构。）
