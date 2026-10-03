# Design

## Context

`.md`/`.markdown` 目前在 `web/app.js` 的 `renderContent` 中走单一路径：`isMarkdownPath` 命中 → markdown-it 渲染管线（`markdownHTMLFor` → `innerHTML`，失败回退 `textContent`）。非 Markdown 文本走另一条既有路径：`highlightHTMLFor` → `safeHighlightHTML` → `innerHTML`，回退素文本——Change 07 之前 `.md` 正是走这条路径以高亮源码呈现的。

导航模型是 SPA：条目点击 `preventDefault` + `pushState` + `load(path)`，`popstate` 也归 `load()`；只有 parent 链接与浏览器刷新是整页导航。`load()` 是一切视图切换的单一入口。

约束：ADR-0002（零构建前端）、ADR-0003（呈现层 Scenario 分派浏览器走查）、探索已定的记忆语义 (a)（无记忆，切换只对当前查看有效）与源码形式归属（spec 沉默，由 `syntax-highlighting` 接管，零 delta）。

## Goals / Non-Goals

**Goals:**

- Markdown 文件可切换 渲染 ↔ 源码 两种呈现形式，默认渲染；每次查看从渲染开始。
- 源码形式完全复用既有高亮管线，不新写呈现机制。
- `markdown-preview` delta 之外全部 capability 零变更；服务端零代码变化；无新增 vendored 依赖。

**Non-Goals:**

- 不做记忆语义（会话内全局、localStorage 均为探索时已拒绝的 (b)/(c)）。
- 不用 URL 表达呈现形式（无 query 参数，切换不可分享、不影响 History）。
- 不保留切换时的滚动位置。
- 不为非 Markdown 文件提供任何切换。
- 不改渲染管线本身（html:false、链接重写、fence 高亮、失败回退原样）。

## Decisions

### D1. 状态：模块级单布尔，`load()` 入口重置

`var sourceForm = false`（命名可议）。重置点放在 `load()` 入口——它是条目点击、parent 导航、popstate、首次加载的唯一汇聚点，一处重置即覆盖「离开后再进入」的全部路径（含浏览器刷新，本来就跨文档）。切换动作不经过 `load()`：翻转布尔后直接对当前文件内容重跑呈现分派，避免重置逻辑与切换互相打架。

- 拒绝按路径记忆的 map：那是探索时已否决的 (b)/(c)。
- 拒绝在 `renderContent` 内按「path 变化」重置：切换本身也重渲染同一 path，按键控判别会和切换打架。

### D2. 源码形式：路由回既有高亮管线，零新机制

`renderContent` 的 Markdown 分支变为按 `sourceForm` 分岔：

```
isMarkdownPath(path)
  ├─ sourceForm == false → markdown-it 渲染管线（原样）
  └─ sourceForm == true  → highlightHTMLFor(path, content)
                           → safeHighlightHTML → innerHTML（"preview hljs"）
                           → null 回退 textContent
```

高亮管线是现成的：`extensionOf(path)` 得 `md`，hljs 别名表精确命中 `markdown`——正是 Change 07 之前 `.md` 的呈现方式。spec 对源码形式沉默，实现自然落在 `syntax-highlighting` 的 Requirement 之下（语言被识别 → 高亮呈现；Content preservation 承诺 textContent 不变）。

- `#preview` 的 className 三态：渲染 `preview markdown` / 源码 `preview hljs` / 回退 `preview`，全部复用既有样式。
- 防线归属不变：源码形式的 HTML 写入走形状校验（Change 06 D2 防线），渲染形式走配置性封闭（markdown-preview D2 防线）——两条管线两套论证，本次只是让 `.md` 能在两套之间切换。

### D3. 切换控件：独立于 `#preview` 的按钮

`index.html` 在 preview 区附近加一个切换按钮，`app.js` 控制 `hidden`：仅当当前查看的是 Markdown 文件（渲染、源码、回退三态）时可见，其余视图（目录、错误、非 Markdown 文件）隐藏。按钮文案表达目标形式（渲染形式下显示「查看源码」，源码形式下显示「查看渲染」）。

- 必须独立于 `#preview`：它的内容被 `innerHTML`/`textContent` 整体覆盖，控件放里面活不过第一次写入。
- 渲染失败回退态（回退属渲染形式的呈现）按钮保持可见：此时切到源码形式走高亮管线，行为与 spec 一致，无语义冲突。
- 按钮不写 History、不改 URL：`pushState` 只属于 `navigate()`，切换是纯呈现层动作（Non-Goals 的 URL 决策）。

### D4. 注释同步改写（「注释不撒谎」纪律第三次执行）

`app.js` 两处防线综述注释（`safeHighlightHTML` 上方、`renderContent` 上方）描述的是「`.md` 一律渲染」的现状；分岔引入后需改写为「Markdown 文件两种形式、三条写入路径」的表述。`index.html` preview 元素注释同理（Change 07 现象三的先例：注释复述过时结论时一并改，零行为变化）。

### D5. Scenario → 断言位置分派（ADR-0003 落地）

delta 的全部 Scenario 均为呈现层，无一可进 `go test`。分派：

| Scenario | 断言方式（agent-browser） |
|---|---|
| A markdown file is viewed（未切换） | `#preview` 出现渲染结构节点（标题/段落），无 `hljs` 类 |
| Content resembling HTML markup is viewed | 复跑既有走查：字面文本出现，无可执行标记节点 |
| Rendering does not succeed | 复用 journal 07.4 的 `defineProperty` 故障注入，素文本呈现且无错误提示 |
| The presentation form is switched to source | 点击切换按钮后 `#preview` 有 `hljs` 类名/高亮节点，渲染结构消失 |
| The presentation form is switched back to rendered | 再点一次，渲染结构回来，`hljs` 类消失 |
| The presentation form does not persist across views | 切到源码 → 条目点击进目录 → 再点回该文件（pushState 路径，同文档内）→ 渲染形式 |

最后一条特意走 SPA 路径而不是 parent 整页导航：整页导航重置布尔是平凡的，同文档内经 `load()` 的重置才是 D1 重置点选择的真正验证。另补一条隐含行为走查：popstate 回退到文件视图同样回到渲染形式。

## Risks / Trade-offs

- [`renderContent` 分岔后三条路径纠缠、可读性下降] → 呈现分派保持为 `(path, content, sourceForm)` 的纯函数；注释按 D4 改写。必要时抽 `renderFileView` 小函数，属 tasks 层自由。
- [切换按钮时序 bug（覆盖/残留）] → 按钮状态随每次 `renderContent`/`hidePreview` 重算，不依赖上次状态；走查清单含「从文件切目录后按钮不可见」。
- [高亮对 `.md` 的呈现与用户预期漂移（希望纯素文本）] → 接受：spec 沉默 + `syntax-highlighting` 接管是探索时的明确决策，高亮是「源码形式」的自然形态；内容字符序列不变有 spec 承诺兜底。
- [回退态下按钮语义引起困惑（回退≈素文本，源码形式却高亮）] → D3 已声明回退属渲染形式；若实测困惑明显，属呈现层润色，不动 spec。

## Open Questions

（无——按钮具体文案与样式属 tasks 层，不影响 spec 与结构。）
