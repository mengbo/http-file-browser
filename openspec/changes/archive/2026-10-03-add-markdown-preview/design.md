# Design

## Context

`.md`/`.markdown` 已在 `text-preview` 的文本扩展名白名单内（`internal/server/content.go`），今天查看 Markdown 文件的路径是：content API 返回纯文本 → hljs 语言别名表命中 `markdown` → 以语法高亮源码呈现（`web/app.js` 的 `renderContent`）。本 Change 把这条分支换成渲染管线。

会被触及的现状：

- `web/app.js` 的 `renderContent`：`highlightHTMLFor` → `safeHighlightHTML`（Change 06 D2 形状校验）→ `innerHTML`，回退 `textContent`。
- Change 06 D2 已对 D11 不变量做过第一次收窄（高亮路径允许来自 hljs 输出的 HTML）。Markdown 渲染是第二次收窄，且收窄的形状不同（见 D2）。
- `web/web.go` 的 embed 清单已含 `vendor` 目录（Change 06 D4 落地），新增 vendored 文件无需改 embed。
- `http.FileServer` 自动服务 `/vendor/...`，服务端零代码。
- 约束：ADR-0002（JSON API + 内嵌零构建前端）、ADR-0003（呈现层 Scenario 验收分派）。

## Goals / Non-Goals

**Goals:**

- `.md`/`.markdown` 以渲染形式呈现；content API 与三个既有 capability 零 delta，唯一既有 spec 变更是 syntax-highlighting 的一条 MODIFIED。
- 安全靠配置不靠消毒组件：渲染器不为 HTML 开门，不引入 sanitizer。
- 链接在应用内可导航，外部资源按浏览器语义加载；代码块复用 vendored hljs 高亮。
- 呈现层 Scenario 的验收按 ADR-0003 分派并落地。

**Non-Goals:**

- 不做内嵌 HTML 的渲染与消毒（不支持，字面呈现——用户决策，见 D2）。
- 不做渲染/源码视图切换（roadmap Change 08 的 MODIFIED 演练；本 Change 的 MODIFIED 措辞已按「呈现形式」挂条件为其留位）。
- 不做相对路径图片的可加载（需原始文件字节端点，`/api/content` 只返回 JSON，原理上不可达；记想法池）。
- 不做标题锚点 id、目录、任务列表等构造扩展（markdown-it 默认构造集为准，见 Risks）。
- 不做服务端渲染（D1，Change 06 D1 的论证直接继承）。

## Decisions

### D1. 客户端渲染，vendored markdown-it 全托管

Change 06 D1 已论证过客户端呈现变换的完整理由：API 形状与 spec 冻结、不削弱 ADR-0002、复用前端管线。服务端 goldmark 需要新增 HTML 形态字段或新端点，`text-preview` 被迫动 spec——拒绝。markdown-it 有单文件 UMD 构建产物，与 hljs 同形态进 `web/vendor/`。

### D2. 内嵌 HTML 关闭：安全靠配置，不靠消毒

markdown-it `html: false`：文件中形似 HTML 的标记被转义为字面文本呈现。与 syntax-highlighting spec 已有的「呈现的是该标记文本的字面形式」逐字同构，也延续 Change 04 D9 的「不检测比检测错更诚实」。

- **这是 D11 不变量谱系的第二次收窄，但防线形状不同**：高亮路径（Change 06 D2）是「输出形状校验」——信任库输出、校验拦截；Markdown 路径是「配置性封闭」——解析器根本不为 HTML 开门，无需输出校验。前者防库输出被污染，后者防输入被解释，两套论证各自成立，互不替代。
- 安全注脚（均为库默认，零成本）：`javascript:`/`vbscript:`/`file:`/`data:`（除 `data:image`）链接被 markdown-it `validateLink` 默认拦截；`<img onerror=...>` 属内嵌 HTML，已被 `html: false` 关闭。
- 拒绝的替代：marked + DOMPurify（渲染后消毒）。消毒器成为安全关键组件（+一个信任面），且「静默剥除」不如「字面可见」诚实。用户已决策不支持内嵌 HTML，消毒失去必要性。

### D3. 识别：扩展名 `md`/`markdown`，纯名字判定

装饰不是门：识别即渲染，识别错的代价只是「本该素文本的东西被渲染」（`.md` 按惯例就是 Markdown，误报方向几乎不存在）；渲染抛错回退素文本（spec: Rendering does not succeed）。不引入内容嗅探，与 `text-preview` 判定零耦合——到不了文本判定的文件到不了渲染。

### D4. 链接：相对路径重写为应用内导航

```
被查看文件 content.path = "docs/notes.md"
[x](other.md)     →  解析基准 = "docs"   →  urlFor("docs/other.md")
[x](../a/b.md)    →  上级片段归一         →  urlFor("a/b.md")
[x](/root.md)     →  / 开头按根相对       →  urlFor("root.md")
[x](#anchor)      →  原样保留（页内锚点；标题默认无 id，多数落空，无害）
[x](https://…)    →  target="_blank" rel="noopener"，当前呈现不变
```

- 实现：渲染写入 DOM 后对 `a[href]` 做一遍遍历，`setAttribute` 重写——不经 innerHTML 拼接，href 注入面为零。
- 目标不存在时的行为由既有 `load()` 流程兜底：列表 → `not_a_directory` → content → 现有错误态呈现。与目录列表「条目一律是链接、可否预览由服务端判定」的哲学同构，前端不另立一份「可点性」清单。

### D5. 图片：浏览器语义加载，相对路径破图

不做任何重写。绝对地址图片由浏览器加载；相对地址解析到静态资源服务，404 破图——诚实呈现「资源拿不到」，好过伪装。原始文件端点是想法池事项，与「渲染内相对链接直达原始下载」一起留给未来 Change。

### D6. 代码块高亮：markdown-it highlight 钩子接 vendored hljs

fence 信息串经 `hljs.getLanguage` 命中即 `highlight`；无标注/未识别走 `highlightAuto`（沿用 `renderContent` 的 `AUTO_MIN_RELEVANCE` 低置信回退）；抛错回退素文本——与 spec「语言未被识别的代码块以普通文本形式呈现」对齐。**代码块的 hljs 输出仍过 `safeHighlightHTML` 形状校验**（Change 06 D2 的既有防线，对来自 hljs 的输出继续适用）。渲染器对高亮失败的处理：该代码块以素文本呈现，文件其余部分照常渲染。

### D7. 渲染写入与失败回退

`renderContent` 的 Markdown 分支：`markdown-it` 渲染 → `previewEl.className` 切换为 `preview markdown` → `innerHTML`。渲染器抛错（病态输入）→ 整段回退 `textContent` 素文本，不报错（spec: Rendering does not succeed）。`safeHighlightHTML` 只作用于高亮与代码块路径，不作用于 Markdown 主输出——**`app.js` 中 Change 06 D2 的注释（「交给 HTML 解析器的字符串只能来自 highlight.js 的输出」）需同步改写为两条管线各自的防线表述**，避免注释撒谎（tasks 已列）。

### D8. vendored 形态与版本记录

markdown-it 单文件 minified UMD 落位 `web/vendor/markdown-it.min.js`，版本号写进文件旁的版本注释（照抄 Change 06 D4 的做法：vendored 文件没有 lock 文件，注释就是版本记录）。`index.html` 以 `<script>` 引入。

### D9. Scenario → 断言位置分派（ADR-0003 落地）

| Scenario | 断言位置 | 方式 |
|---|---|---|
| A markdown file is viewed | 呈现层 | agent-browser：`#preview` 出现渲染节点（标题/段落结构） |
| Content resembling HTML markup is viewed | 呈现层（机器可断言） | agent-browser：标记文本以字面出现（`textContent` 含原样文本），无可执行标记节点 |
| Rendering does not succeed | 呈现层 | agent-browser：构造触发抛错的输入（如替换渲染函数注入异常），`#preview` 素文本呈现且无错误提示 |
| A relative link is activated | 呈现层 | agent-browser：点击后 URL 与呈现切换到解析出的目标位置 |
| A relative link containing parent segments is activated | 呈现层 | agent-browser：`../` 解析正确，落点为目标位置 |
| An external link is activated | 呈现层 | agent-browser：`target="_blank"` 且当前页呈现不变 |
| An image referenced by an absolute URL is viewed | 呈现层 | agent-browser：`img` 元素存在并加载 |
| An image whose address cannot be loaded is viewed | 呈现层 | agent-browser：无错误提示，其余内容完好 |
| A code block with a recognized language is viewed | 呈现层 | agent-browser：代码块出现 hljs 类名节点 |
| A code block without a recognized language is viewed | 呈现层 | agent-browser：素文本呈现 |
| Text is taken from a highlighted code block | 呈现层（机器可断言） | agent-browser：代码块 `textContent` 与 fence 内源文本逐字符一致 |
| The presentation resources are requested | **API 层** | Go `httptest`：`GET /vendor/markdown-it.min.js` 200、非空、`text/javascript` |

唯一能进 `go test ./...` 全量回归的是最后一行；呈现层十一条在 tasks.md 验收清单里以 agent-browser 走查勾选记录。

## Risks / Trade-offs

- [外图加载的隐私面：预览会向文件引用的外部地址发请求] → 「导航加载放行」的固有属性，用户决策接受；Change 13/14 开放远程访问时如需收紧，用 MODIFIED 调整。
- [markdown-it 默认构造集（表格、删除线等）与用户期望漂移] → 构造集属 design 不属 spec；实测不符预期时调 renderer 规则即可，不动 spec。
- [相对链接重写的解析边界（URL 编码、空格、查询串）] → 按 URL 语义解析 + 路径归一，边界用例进 tasks 走查清单；解析错误的最坏结果被既有错误态兜住。
- [hljs auto 对代码块误判语言] → 接受。同 Change 06 的装饰哲学；代码块的 textContent 一致承诺保证信息无损。
- [渲染大文件耗时] → markdown-it 性能充裕；超大文件已被 `too_large` 挡在 `text-preview` 门外，接受。
- [vendored 文件手动升级漂移] → D8 版本注释 + tasks 验收项「注释版本号与运行时一致」。

## Open Questions

（无——markdown-it 具体版本、highlight 钩子的正则细节属实现层，tasks 内解决，不影响 spec 与结构。）
