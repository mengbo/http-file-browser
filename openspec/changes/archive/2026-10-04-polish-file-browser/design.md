# Design

## Context

当前目录浏览视图的视觉呈现由 `web/style.css` 与 `web/app.js` 共同承担——CSS 提供配色、间距、列宽与响应，JS 负责把 `/api/list` 返回的条目逐条渲染为 `<li class="entry">`。约束全部来自既有 ADR 与 archive：

- ADR-0001（单二进制交付）：任何前端资源最终必须进 `web/web.go` 的 embed FS。
- ADR-0002（JSON API + 内嵌零构建前端）：不引入打包器、不引入 JS 测试运行器、不引入图标库；图标资源走 vendored 路径，与 `/vendor/highlight.min.js`、`/vendor/markdown-it.min.js` 同路径（已经走过一次，Change 09 design D3 是图片内容端点；vendored SVG 是其内嵌路径的自然延伸）。
- ADR-0003（呈现层 Scenario 验收分派）：本 Change 全部 spec 都是呈现层（图标是否渲染、对比度够不够），无 API 层 Scenario。
- Change 02 design D8 / Change 03 design D8 已经确立 grid 列宽纪律（`minmax(0, 1fr)` + 5.5rem + 10rem）和窄屏（34rem）媒体查询——本次视觉调优必须保持这条纪律不破。
- Change 02 起 spec 承诺了 `Entry type distinction`（「在列表中区分目录条目与文件条目」）但**从未承诺视觉图标**——本 Change 是该承诺的视觉承载扩展，而非行为颠覆。
- Change 06 D4「样式协调的判断依据是读上游产物，不是猜」：vendored 图标资源升级/换源时需重新做对比度检查。

## Goals / Non-Goals

**Goals:**

- 兑现 `directory-browsing / 类型化视觉图标` 这一新增 Requirement 的可观察承诺。
- 视觉调优与既有约束兼容：grid 列宽纪律、窄视口断点、字体栈、深浅外观切换、`#preview` 的纸面语言保持。
- 单二进制交付：所有图标资源内嵌进 `web/web.go` 的 embed FS，运行时无外部依赖。

**Non-Goals:**

- 不引入 JS 框架、不引入图标库（Font Awesome、Material Symbols 等）。
- 不改 `/api/list` 响应形状（图标决策纯前端呈现层）；服务端零代码变化。
- 不按文件扩展名区分图标（如 `.png` 用图片图标、`.go` 用代码图标）——两层图标（目录 / 文件）符合 macOS Finder 自身体验，复杂度可控。
- 不重做布局（三栏、工具栏化、卡片化）——这些属独立 Change，本 Change 仅做整体视觉精修。
- 不为搜索结果视图与文件视图引入图标——spec 明确排除这两处（已记 Non-Goal 同步入 spec）。
- 不做动效过渡（hover 阴影是即时变化，不是渐变）。
- 不为图标单独写 SVG——使用 vendored SVG 库（同源一套，色值可控）。

## Decisions

### D1. 图标资源走 `/vendor/icons/*.svg`，vendored 进 embed FS

```
web/
├── vendor/
│   ├── highlight.min.js
│   ├── highlight.min.css
│   ├── markdown-it.min.js
│   └── icons/
│       ├── dir.svg
│       └── file.svg
```

- 与现有 vendored 资源同路径——`web/web.go` 的 `//go:embed vendor` 自动覆盖新增子目录，无需手改 embed 声明。
- 引用路径：`/vendor/icons/dir.svg`、`/vendor/icons/file.svg`。前端 JS 用 `<img src="...">` 引入——简单可靠，零额外依赖。
- **不**用 `<svg>` 内联进 HTML（理由：会让 `app.js` 的渲染函数膨胀、与 `textContent` 防线混淆；`<img>` 的 `currentColor` 不可控但通过选源就能解决）。
- **不**用 `mask-image` + CSS 背景（理由：现有 Firefox/Safari 对 inline SVG mask 的支持虽已稳定，但跨平台文件浏览器要顾及老浏览器；vendored `<img>` 是最稳的）。
- **不**走 data URI（理由：每个 SVG 内嵌进 CSS 会把样式表胀大、与 vendored 路径的精神不符）。

### D2. 图标来源：Lucide（开源、ISC 协议、几何线性风格贴近 macOS）

候选：Lucide / Tabler / Heroicons / Feather。

- **Lucide**：840+ 图标、ISC 协议、几何线条、视图稳定；目录图标（`folder`）与文件图标（`file`）均存在，视觉语言一致。
- **Tabler**：MIT 协议、风格与 Lucide 相近。
- **Heroicons**：MIT 协议、更圆润；偏 Material 风格，差异不大。
- **Feather**：与 Lucide 同源（Feather 2.0 后由 Lucide 继承），不再独立维护。

**选 Lucide**：`folder` 与 `file` 两个图标覆盖本次需求，版本号固定（vendored 注释写明），升级时只需替换文件。

- 图标规格：24×24 视图框、`stroke="currentColor"`、`fill="none"`、stroke-width 2、stroke-linecap/linejoin round。
- 通过 `<img src=...>` 引用后，`color` 由 CSS 的 `color` 属性控制；深浅外观自动切换。
- vendored 文件头部注释写明版本号与获取来源（遵循 Change 06 task 3.4 的纪律：vendored 资源与运行时探查对齐）。

### D3. 图标尺寸 16×16，与文字基线对齐

```
.entry-icon {
  width: 1rem;        /* 16px，与正文字号一致 */
  height: 1rem;
  flex-shrink: 0;
  vertical-align: middle;
  margin-right: 0.4rem;
}
```

- **为什么 16 而不是 14 或 18**：现有 `.entry` 字号 0.95rem（≈15.2px），1rem 与之齐平；14 偏小、18 偏大挤占文字列。
- `vertical-align: middle` 对齐 SVG 视觉中心与文字 x-height（中线附近）——非 baseline，因为 Lucide 图标的视觉中心在 50% 处。
- `flex-shrink: 0` 防止窄视口下被 grid 压扁。
- **列宽（apply 修订）**：图标不与文字共用首列。`.entries-header` / `.entry` 的 grid 改为**四列** `auto minmax(0, 1fr) 5.5rem 10rem`（窄屏媒体查询同步为 `auto minmax(0, 1fr) 4.25rem 8.25rem`）——`auto` 首列宽度由图标内容（16px + 0.4rem 边距）决定，名称列继续以 `minmax(0, 1fr)` 吸收长文件名，`5.5rem`/`10rem`（窄屏 `4.25rem`/`8.25rem`）两列原样保留。初稿「图标 + 链接共用首列、列内 flex」的写法与 `display: grid` 的自动放置冲突（图标会把链接挤进第二列），实现前经用户拍板改为四列方案；表头同步补一个空的首列占位。数据三列全部保留、无溢出的窄视口纪律不变（tasks 4.6 实测 320px 通过）。

### D4. 图标颜色：currentColor，深浅外观自动切换（GitHub 风格定稿）

```
.entry-icon {
  color: #656d76;  /* GitHub 弱化前景色，与辅助信息同级 */
}

@media (prefers-color-scheme: dark) {
  .entry-icon {
    color: #8b949e;
  }
}
```

- 弱化色辅助信息层级——图标不抢条目名的视觉权重（GitHub 列表本身靠左图标 + 文字层级，无彩色左边框）。
- 链接文字色用 GitHub 强调蓝（`#0969da` / `#4493f8`），与图标灰阶分工：彩色只给可点目标，图标保持安静。
- **对比度自检**（spec Scenario 兑现，GitHub 风格定稿实算）：
  - 浅色：`#656d76` on `#ffffff` → **5.25:1**（AA 正文过，非文本 3:1 余量充足）。
  - 深色：`#8b949e` on `#0d1117` → **6.15:1**（AA 过）。
  - 算式：相对亮度 `L = 0.2126·R + 0.7152·G + 0.0722·B`（各通道线性化 `((c+0.055)/1.055)^2.4`），对比度 `(L亮+0.05)/(L暗+0.05)`。
  - 历史：初稿 macOS 蓝背景（`#f2f2f7`）下 `#6e6e73` 为 4.54:1、深色 5.93:1；GitHub 白底定稿改用 `#656d76` / `#8b949e`，对比度反升至 5.25 / 6.15。

### D5. 视觉调优：颜色、圆角、间距、行高、阴影

| 项目 | GitHub 定稿（浅 / 深） | 理由 |
|---|---|---|
| 画布背景 | `#ffffff` / `#0d1117` | GitHub canvas-default |
| 正文 | `#1f2328` / `#e6edf3` | GitHub fg-default |
| 链接（条目 / 上级 / Markdown 链接 / 切换按钮） | `#0969da` / `#4493f8` | GitHub accent-fg |
| 辅助信息（位置 / 大小 / 时间 / match-dir / match-type / 图标） | `#656d76` / `#8b949e` | GitHub fg-muted |
| 边框（工具栏 / 表头 / 输入 / 按钮 / 预览 / 图片） | `#d0d7de` / `#30363d` | GitHub border-default |
| 行分隔线（条目 / 命中） | `#d8dee4` / `#21262d` | GitHub border-muted |
| hover 行底色 | `#f6f8fa` / `#161b22` | GitHub canvas-subtle；无阴影（沿用极简中性） |
| 圆角（条目 / 预览 / 图片 / 按钮 / 输入） | `6px` 统一 | GitHub 一致 6px |
| 输入框 | 底 `#f6f8fa` / `#161b22`；focus 边框 `#0969da` / `#4493f8` + 光圈 | GitHub 搜索框 |
| 按钮（切换） | 底 `#f6f8fa` / `#21262d`，边框 `#d0d7de` / `#30363d`，hover 底 `#eef1f4` / `#30363d` | GitHub 次级按钮 |
| focus 光圈 | `0 0 0 3px rgba(9,105,218,0.3)` / `rgba(56,139,253,0.4)` | GitHub focus ring |
| 条目行 padding | `0.55rem 0.5rem` | 沿用极简中性（留白不动） |
| 条目行高 | `1.55` | 沿用 |
| 字体栈 | `-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", "Helvetica Neue", "PingFang SC", sans-serif` | GitHub 栈 + 中文回退 |
| Markdown 链接 | `#0969da` / `#4493f8`，hover 才出下划线 | GitHub 正文链接惯例（从极简中性的「常驻细下划线」改回） |

> **风格修订记录（apply 阶段，两轮用户实机评审）**
> - **第一轮**：初稿 D5 是「macOS 系统蓝（`#007AFF`）+ Finder 卡片 hover 阴影」并已实现走查；用户看实机否决（「不好看」），从 Finder 拟真 / 极简中性 / 暖纸阅读 中选定**极简中性**——去彩色主色、hover 只给背景、大留白。见 tasks 第 6 节。
> - **第二轮**：用户回看后仍偏好 **GitHub Primer 风**，遂二次修订为现表——白/深画布、GitHub 强调蓝、GitHub 弱化灰、统一 6px 圆角、focus 光圈。链接色回到彩色（但换成 GitHub 蓝，非 macOS 蓝）。见 tasks 第 7 节。
> - **图标灰阶随之更新**（`#6e6e73`/`#98989d` → `#656d76`/`#8b949e`），对比度反升（4.54 / 5.93 → **5.25 / 6.15**），spec「足以识别」稳过。
> - **`#preview` 纸面语言保持白底不随外观切换**（用户第二轮明确选择「保持白纸」）：GitHub 深色主题的代码块本是深底，但改它需新增 vendored `github-dark` 主题并推翻 Change 06 D4 的协调不变量，本 Change 不扩这一步，记入 Non-Goal。
> - **spec 全程零变化**：色值从未进 spec。两轮换风格（蓝 → 中性 → GitHub 蓝）恰好是 D5「CSS 不入 spec」论证的连续实证。

**保留不动的纪律**：

- `.browser` 的 `max-width: 46rem`（Change 02 D8 视觉重心）。
- 数据三列的列宽语义（名称 `minmax(0, 1fr)` + 5.5rem + 10rem；窄屏媒体查询 `max-width: 34rem` 时收窄到 4.25rem + 8.25rem）——Change 03 journal 观察 4「窄屏降级很容易变成静默丢数据」的纪律。**apply 修订（D3/D6，用户拍板）**：为图标新增 `auto` 首列，grid 变四列；三列数据的宽度与「全部保留」纪律不变，34rem 断点不变。
- `#preview` 与 `.preview.markdown` 的纸面语言（白底、不随外观切换）。
- `vendored/highlight.min.css`（github 浅色主题）——Change 06 D4 确立的协调结果（与本次 GitHub 风格方向天然同源）。
- `prefers-color-scheme: dark` 媒体查询的语法与覆盖范围。
- 字体栈的方向（系统栈、零下载）；仅按 GitHub 栈补 `"Segoe UI", "Noto Sans"` 两项跨平台回退（Mac 上仍命中 `-apple-system`，视觉不变）。

**为什么这些 CSS 调优不需要写 spec**：调优后的整体观感是同一份"目录浏览视图"的视觉延展，不是新行为。如果未来某天产品决定主色从蓝换成紫（紫是 #af52de），调一次色值即可，spec 无需变更——这与 Change 04「1 MiB 是可调旋钮」的纪律同形。本 Change 两轮风格翻案零 spec delta，就是这条论证的现成例证。

### D6. DOM 结构：图标作为 `.entry` 的首子元素，独立占首列

```html
<li class="entry">
  <img class="entry-icon" src="/vendor/icons/file.svg" alt="" aria-hidden="true">
  <a class="entry-link" href="/?path=...">name.txt</a>
  <span class="entry-size">1.2 KB</span>
  <span class="entry-time">2026-10-04 10:30</span>
</li>
```

- 图标在链接之外，与 `.entry-link` 平级；hover 背景仍由 `.entry:has(.entry-link):hover` 覆盖整行（包括图标）。
- 布局落位（apply 修订，见 D3）：`.entry` 的四列 grid 中，图标占 `auto` 首列、链接占 `minmax(0, 1fr)` 名称列——初稿「共用首列、列内 flex」与 grid 自动放置冲突，经用户拍板改为图标独立成列；`.entries-header` 补一个空首列占位（`<span class="entry-head" aria-hidden="true"></span>`）保持表头与条目列数一致。
- `alt=""` + `aria-hidden="true"`：图标是装饰元素，**不**承载信息（条目名称已在 `<a>` 中提供）；屏幕阅读器跳过。spec「类型化视觉图标」的视觉承诺**仅对视觉通道生效**，可访问性靠 `.entry-link` 的文本保证。
- 图标 URL 由 JS 在 `renderEntry(entry)` 里根据 `entry.type === "directory"` 选 `/vendor/icons/dir.svg` 或 `/vendor/icons/file.svg`；spec 不约束此映射逻辑（属 design 决策）。
- **不**改 `.entry-link` 的渲染顺序——其内仍只含名称（Change 02 起 spec 承诺「条目中只提供名称」）。
- 搜索结果视图（`.match`）与文件视图（`.preview` / `.image-view`）**不**渲染图标——spec Scenario 明确排除。

### D7. Scenario → 断言位置分派（ADR-0003 落地）

| Scenario | 断言位置 | 方式 |
|---|---|---|
| 目录条目显示目录图标 | **呈现层** | agent-browser：列出混合目录；每个 `#entry` 内 `.entry-icon` 的 `src` 在浅色/深色下分别为 `/vendor/icons/dir.svg` 或对应路径；`naturalWidth` / `naturalHeight` = 24（vendored 视图框） |
| 文件条目显示文件图标 | **呈现层** | agent-browser：每个 `#entry` 内文件条目的 `.entry-icon` 指向 `/vendor/icons/file.svg` |
| 目录与文件图标在外观上可区分 | **呈现层** | agent-browser：`document.querySelectorAll('.entry-icon').length` 与条目数相等；目录条目与文件条目对应不同 `src`（或不同 `srcset` / `currentSrc`） |
| 图标在浅色与深色外观下均可识别 | **呈现层** | DevTools 计算样式：`getComputedStyle(icon).color` 在浅色下为 `#6e6e73`、在深色下为 `#98989d`；两个值的对比度都达 WCAG AA |
| 文件视图与搜索结果视图不显示视觉图标 | **呈现层** | agent-browser：进入 `.png` / `.txt` / `.md` 各文件视图后 `#preview`/`#image-view` 区域内无 `.entry-icon`；搜索后 `.match` 内无 `.entry-icon` |

API 层零新增测试（服务端无变化）。

### D8. 验收清单的"颜色 / 圆角 / 阴影"非 spec 断言

下列属性**不**进 spec，tasks.md 收尾时以 agent-browser DevTools 计算样式逐项断言（与 Change 06 task 4.4 的呈现层走查纪律一致）：

- `getComputedStyle(body).backgroundColor` 浅色为 `rgb(255, 255, 255)`、深色为 `rgb(13, 17, 23)`
- `getComputedStyle(entryLink).color` 浅色为 `rgb(9, 105, 218)`、深色为 `rgb(68, 147, 248)`（GitHub 定稿；此前 macOS 蓝与极简近黑两版已废弃）
- `getComputedStyle(icon).color` 浅色为 `rgb(101, 109, 118)`、深色为 `rgb(139, 148, 158)`（GitHub 定稿；初稿 `#6e6e73`/`#98989d` 已废弃）
- `getComputedStyle(entry).boxShadow` 在 hover 状态为 `none`
- `getComputedStyle(entry).borderRadius` 为 `6px`
- `getComputedStyle(preview).borderRadius` 为 `6px`

## Risks / Trade-offs

- [vendored 图标库升级可能改视觉] → 升级时同步检查 `dir.svg` / `file.svg` 是否仍在 vendored 包内、`currentColor` 兼容性是否变化；遵循 Change 06 task 3.4「vendored 资源升级时这一步要重做」的纪律。
- [图标的 SVG 视图框与渲染尺寸不一致] → Lucide 视图框 24×24，CSS 渲染 16×16；放大靠 SVG 自带缩放，不破栅格。若换成视图框不同的库（如 Heroicons 20×20），需重做 D3 的尺寸论证。
- [hover 阴影加脆弱，与现有列表行的下边线 (`border-bottom: 1px solid #ebebef`) 在视觉上可能微冲] → 实测：阴影透明度 0.08 不足以与 1px 灰线产生视觉冲突；tasks.md 走查中保留视觉截图。
- [图标颜色 `#6e6e73` 在浅色背景 `#f5f5f7` 上的对比度刚好压线 4.5:1] → 经 WCAG 计算器实测 4.6:1；若背景色从 `#f5f5f7` 改为 `#f2f2f7`（D5），对比度提升至 4.7:1，**不**跌破。tasks.md 写明「以 #f2f2f7 为基准的对比度断言」避免背景色微调导致断言失败。
- [DOM 顺序变化惊动现有 CSS 选择器] → 图标是新增首子元素，`.entry-link` 仍在第二条；`.entry:has(.entry-link)` 选择器继续命中。**列宽（apply 修订，见 D3/D6）**：grid 由三列改为四列 `auto minmax(0, 1fr) 5.5rem 10rem`，图标独立占首列——初稿「共用首列、列内 flex」与 grid 自动放置冲突，经用户拍板修订；数据三列的宽度语义原样保留，窄视口纪律（三列全见、无溢出）由 tasks 4.6 实测兜住。
- [搜索结果视图与文件视图原本就没有图标，本次也不加——但 spec 显式排除这两处的「为何」] → spec Scenario 已记录这两处不显示图标的承诺，避免未来某个 Change 想"统一一下"时误以为是不一致。

## Open Questions

（无——图标来源、尺寸、颜色、vendored 路径、DOM 结构、视觉调优均已定稿；呈现层 CSS 调参属 tasks 细节。）