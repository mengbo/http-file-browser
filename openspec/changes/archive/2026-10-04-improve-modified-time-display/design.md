# Design

## Context

- 现状：`web/app.js` 的 `formatModifiedAt(seconds)` 用 `new Date(seconds * 1000).toLocaleString()` 渲染（app.js:236-242），唯一调用点是 `renderEntries` 里的 `entry-time` 单元格（app.js:169）。输出随运行环境 locale 变化。
- 取值侧契约不变：`directory-browsing / Entry metadata` 已承诺 `modified_at` 为 Unix 整秒、跨时区与语言取值相同。本 Change 只动呈现，不动取值。
- 历史决定：归档 `2026-10-03-add-file-metadata` 的 design 把「修改时间渲染为本地时区的可读形式」列为「spec 不约束、design 定方向」。本 Change 推翻其中「随 locale 排版」的部分——保留「本地时区」，只把排版固定下来。
- 呈现层行为按 [ADR-0003](../../../docs/adr/0003-presentation-acceptance.md) 用浏览器走查验收，无 Go 测试。
- 列宽纪律（Change 02 design D8、Change 16）：`.entry` / `.entries-header` 用 `grid-template-columns: auto minmax(0,1fr) 5.5rem 10rem`（style.css:280），窄视口断点为 `auto minmax(0,1fr) 4.25rem 8.25rem`（style.css:711）。

## Goals / Non-Goals

**Goals:**

- 目录列表条目的修改时间呈现为固定 `YYYY-MM-DD HH:mm:ss`，不随 locale 变化。
- 保持既有列宽与窄屏布局纪律不被破坏。
- 在 spec 层把该呈现行为固定为契约。

**Non-Goals:**

- 不改 `/api/list` 响应与 `modified_at` 取值（仍为 Unix 整秒）。
- 不改大小格式 `formatSize` 及其呈现（同属旧 design 的「呈现不约束」，本次只做一件事）。
- 不改时区语义（仍为本地时区）。
- 不给搜索结果视图或文件视图加时间或大小（属独立 Change）。
- 不引入日期库（dayjs、date-fns 等），也不引入新的 `Intl`/locale 依赖。

## Decisions

### D1. 显式补零手写格式化，不用 `toLocaleString('sv-SE')`

候选：

- `new Date(s * 1000).toLocaleString('sv-SE')` 恰好产出 `YYYY-MM-DD HH:mm:ss` 形状，一行即得；但它依赖运行环境对该 locale 的排布实现，是隐式约定，与项目「判断依据是读上游产物，不是猜」的取向（Change 06 D4）不符，且秒位是否给出并非标准保证。
- `Intl.DateTimeFormat('en-CA', {...})` 同理，仍依赖 locale 数据。
- **手写补零**：从 `Date` 取 `getFullYear` / `getMonth()+1` / `getDate` / `getHours` / `getMinutes` / `getSeconds`，各 `padStart(4, "0")` 后拼接。

选**手写补零**：零 locale 依赖、零新依赖、结果完全确定。改动收敛在 `formatModifiedAt` 一个函数内，既有 `typeof !== "number" || !isFinite` 守卫与空串返回保持不变。

### D2. 时区维持本地

`getFullYear()` 等取值函数本就返回本地时区字段，手写补零自然沿用本地时区，无需额外换算。这是「只固定排版、不改时区」的落地方式。明确记录：本次不把呈现改为 UTC。

### D3. 缺失值仍留空、不占位

保留现有「非有限数值返回空串」的行为（呼应 `Entry metadata` 的省略语义与前端 `textCell` 的空串约定）。固定格式只在有值时生效。

### D4. 列宽先验证、再决定是否调整

固定格式 19 字符，比现有本地化输出（如 `2026/9/16 13:20:00`，17–18 字符）略宽。宽屏 10rem 余量充足；窄屏 8.25rem（约 132px、0.7rem 字号）余量较小，`overflow-wrap: anywhere` 可能把时间折成两行。**默认不改 CSS**：在浏览器走查任务中以 19 字符最坏情况在 34rem 断点验证，仅当实测溢出或折行才调整 `style.css:711` 的窄屏 `entry-time` 列宽。把「是否改 CSS」交给验证结果，避免无谓改动。

## Risks / Trade-offs

- [窄屏时间折行] → 走查以固定 19 字符在 34rem 断点验证；溢出则调窄屏 `entry-time` 列宽并同步表头。
- [推翻旧 design 造成历史不一致] → 归档 design 不可改；本 Change 的 proposal/design 已说明来由，archive 后 spec 成为新事实。
- [手写格式引入易错点（月份从 0 起、补零遗漏）] → 走查断言一个已知 `modified_at` 的具体期望字符串，覆盖月份 +1 与补零两处。
- [时区在 spec 中首次承诺] → 仅把既有行为写实，不引入新行为；现有实现天然满足。

## Open Questions

无。
