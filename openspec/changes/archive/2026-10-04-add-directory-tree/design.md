# Design

## Context

- 现状：`web/index.html` 是单列 `.browser`（`max-width: 46rem` 居中）。主区承载四种互斥视图（目录列表 / 文件预览 / 图片 / 搜索结果），由 `app.js` 的 `hideEntries` / `hideMatches` / `hidePreview` 管理。浏览位置来自 URL `?path=`，由 `load()` 单一入口分派，前进/后退天然在视图间切换。
- 服务端已有 `/api/list`（返回一级目录的名称/类型/大小/时间/顺序）、`/api/content`、`/api/image`、`/api/search`、`/api/health`。`/api/list` 正是懒加载目录树所需的数据源。
- 配色已是 GitHub Primer（Change 16），本 Change 复用不重新配色。
- 呈现层验收走 ADR-0003（agent-browser 走查）。
- 需求见本 Change 的 `specs/directory-browsing/spec.md` 之 ADDED `目录树导航`；动机见 `proposal.md`。

## Goals / Non-Goals

**Goals**

- 两栏 GitHub 代码视图布局：左侧持久目录树，右侧主区。
- 零后端改动：目录树复用既有 `/api/list`。
- 主区四种视图与全部既有行为零回归。

**Non-Goals**

- 不引入任何 git/repo 概念（提交列、分支、repo 头、标签页、写操作）。
- 不新增后端 API、不改 `/api/list` 响应形状。
- 不做全树一次性拉取、大目录分页或虚拟滚动（后者记想法池）。
- 不把树的展开状态持久化到跨刷新（见 Open Questions）。

## Decisions

### D1 布局：单列 → 两栏

`.browser` 由居中单列改为全宽两栏：`display: grid; grid-template-columns: 17rem minmax(0, 1fr)`。侧栏约 17rem（≈ GitHub 的 272px），主区吃剩余宽度。

- Alternative：保持 46rem 居中、树挤在内部——两栏会被 46rem 压死；否决。

### D2 树的数据模型与懒加载

不建节点对象；用三个以相对路径为键的模块级集合描述树的状态：`treeLists: Map<path, entries[]>` 缓存已取回的目录条目（`/api/list` 的 `entries`）、`treeExpanded: Set<path>` 记录哪些目录展开、`treeRequests: Map<path, Promise>` 缓存进行中的请求以去重并发。展开一个目录时才请求 `/api/list?path=<dir>`，文件是叶子、不请求；渲染时由「条目数组 + 展开集合」直接推导 DOM，不保留节点对象。

- 理由：`/api/list` 已提供树所需的字段与排序，零后端改动；状态用 `Map`/`Set` 表达比逐节点对象更简单，重绘完全由缓存推导、无局部状态。
- Alternative：节点模型 `{ name, path, type, children: null | [], expanded: boolean }`——初版设计如此；实现后改为上面的集合模型（更简单、行为等价），本决策据实现回写。
- Alternative：新增 `/api/tree` 返回全树——无界、需新 spec、超大目录会炸；否决。
- Alternative：首屏递归拉全树——同上无界；否决。

### D3 初始渲染与祖先自动展开

- 首屏取根目录 `/api/list?path=`（第 0 层），渲染根的直接子条目。
- 就位后按 URL `path` 逐级取各祖先的 `/api/list`，展开到当前位置的父节点，并标出当前位置节点。
- 深度 d 的位置最多 d+1 次 `/api/list`，本地文件系统可接受。
- 与主区共享缓存：`load()` 已为当前目录取的列表可复用给树，避免重复请求。

- Alternative：只渲染根、当前位置不在视野内也不管——违背 spec「标出当前位置」；否决。

### D4 树显示目录与文件

目录节点带 disclosure（可展开/折叠），文件节点为叶子。与截图、与主区列表一致。

- Alternative：仅目录——与截图/主列表不一致，且文件是浏览的主要目标；否决。

### D5 展开状态与当前位置

- 模块级 `expanded: Set<path>`。每次 `load()` 后，把当前位置的祖先链强制加入 `expanded`。
- 用户可折叠当前路径的某个祖先，立即生效；下一次 `load()`（点击/后退/前进）重新强制展开。无持久记忆。
- 理由：简单、可预期，与 GitHub「导航会展开目标」的行为一致。

### D6 树独立于主区视图互斥

`hideEntries` / `hideMatches` / `hidePreview` 只管理主区。目录树是常驻侧栏，不被这些函数清空；`load()` 末尾只更新树的 active/展开，不整棵重绘。错误态、搜索结果、文件视图下树都保持可用（spec 最后一条场景）。

### D7 面包屑与父目录行

- `#location` 由纯文本升级为面包屑：第一段是根目录绝对路径（来自 `/api/health` 的 `root`），其后每段路径一个可点链接，指向对应祖先。
- 目录列表主区在表内新增首行 `..` 指向上级（GitHub 形态）；文件视图不设 `..`，用面包屑导航。
- 归为呈现层：`Parent directory reference` 定义的是 API 响应中的 `parent` 字段，未改动；`Position representation and reproducibility` 已承诺位置表示，面包屑只是其一种呈现。

### D8 GitHub 文件表样式

圆角边框容器、表头底 `#f6f8fa` / 表头文字 `#59636e` / 12px 粗体、行分隔线 `#d1d9e0`、行 hover `#f6f8fa`、名称文字近黑 `#1f2328`（不再用链接蓝）、目录图标蓝色 / 文件图标灰色。列仍为 图标 / 名称 / 大小 / 修改时间（我们没有 commit 列）。纯 CSS，不进 spec。

### D9 搜索移入侧栏

搜索框从 toolbar 移入侧栏顶部（GitHub `Go to file` 的位置），**语义不变**：仍是「以当前浏览位置为基准按名称搜索子树」，提交后进入结果视图。只改位置与样式，不动 `search` 的任何 Requirement。

### D10 展开指示与图标

- 目录 disclosure 用 `<button aria-expanded>` + CSS 画的 chevron/三角，不新增 vendored 资源。
- 图标以 CSS mask 着色：mask 取形状、`background-color: currentColor` 取颜色。**原因**：外链 SVG 经 `<img>` 载入时是独立文档，`currentColor` 不继承页面 `color`，实测所有图标都渲染为黑——Change 16 声称的「深浅外观切换靠 currentColor」从未生效。mask 方案让颜色真正随 `color` 与深浅外观切换，且不引入内联 SVG 的 JS 拼接。
- 目录图标换成 GitHub 同款实心文件夹 `web/vendor/icons/dir.svg`（Octicons `file-directory-fill`，MIT）；文件图标沿用 Lucide `file.svg` 描边。目录蓝取 GitHub 文件树实测值 `#54aeff`（浅色）/ 文件灰 `#656d76`；深色 `#4493f8` / `#8b949e`（GitHub 深色把目录也转成灰 `#9198a1`，与文件灰几乎同色、牺牲可区分性，故保留浅蓝），达成 spec 的「目录/文件节点外观可区分」与任务 3.2 的「目录图标蓝色」。
- 图标元素由 `<img>` 改为装饰性 `<span aria-hidden="true">`（mask 目标），列表、目录树、`..` 行三处同源。

- Alternative：内联 SVG 让 `currentColor` 生效——需在 JS 里内嵌/抓取 SVG 文本，重复 vendored 内容或引入异步；mask 纯 CSS，否决。
- Alternative：再 vendored 一个 chevron SVG——多一份升级复查负担，能省则省。

### D11 可访问性：嵌套列表 + disclosure 按钮

用 `<ul>/<li>` 嵌套；目录项是 `<button aria-expanded>`，导航项是 `<a>`。不采用完整 ARIA `role="tree"` 复合模式。

- 理由：原生列表语义 + 一个按钮比完整 tree 模式（roving tabindex、方向键、aria-level/posinset）更稳，范围可控。
- Alternative：完整 ARIA tree——更贴 GitHub，但键盘模型复杂易错；记为后续可选增强。

### D12 响应式：窄屏收起侧栏

低于断点（沿用既有 34rem 附近）时侧栏默认收起，由工具栏按钮切换，主区占满宽度，维持既有「320px 下 `documentElement.scrollWidth == innerWidth`」纪律。这同时是 spec「目录树可收起与展开」场景的落点。

### D13 前进/后退

`load()` 由 URL `?path=` 驱动，树的 active/展开据此重算；展开状态不入 URL（与 GitHub 一致）。

- Alternative：把展开状态编码进 URL——URL 膨胀且与现有 `?path=`/`?q=` 约定纠缠；否决。

### D14 树与主区排版一致

侧栏目录树的字号与主区目录列表一致（同为 `0.95rem`），图标尺寸 `1rem` 沿用主区既有值；但行距用独立的 `padding 0.35rem + line-height 1.55`，比主区文件表更紧凑——GitHub 的目录树本就把更多条目塞进同宽侧栏，行高不必与文件表完全相等。关键是字号一致（两栏不分裂），同时行距既不回到初版的拥挤、也不放大到文件表的疏朗。

- 评审发现：初版树用 `0.85rem` + `padding 0.2rem`，字号偏小、行距偏挤；一度放大到 `0.5rem` 与列表等行高又偏大，最终定在中间的 `0.35rem`（apply 评审阶段，Change 未归档）。

## Risks / Trade-offs

- 深层路径 → d+1 次 `/api/list` → 本地可接受；缓存 + 仅取祖先链收敛开销。
- 超大目录一次性渲染大量树节点与列表行 → 属分页/虚拟滚动范畴，本 Change 不做，记想法池。
- 符号链接目录在列表与树中显示为 file（lstat 语义，Change 02 / 17）→ 树中不可展开；与既有行为一致，不引入新问题。
- 树被 `hideEntries` 等误清 → D6 明确边界 + 走查覆盖错误态/搜索/文件视图。
- 320px 横向溢出回归 → D12 + 复用既有断言。
- 名称由蓝色改近黑，可能削弱「这是链接」的暗示 → 行 hover 背景 + 名称 hover 下划线补足（GitHub 亦如此）。
- 自定义 disclosure 的键盘/读屏可达性 → D11 的按钮 + `aria-expanded`。

## Migration Plan

纯前端改动，无数据迁移、无服务端部署步骤。单提交 `feat: add-directory-tree`；回滚即 revert。apply 阶段顺手在 `docs/roadmap.md` 新增 `add-directory-tree` 并标 🚧。

## Open Questions

- 是否把树的展开状态持久化到 `localStorage`（跨刷新记忆）——本轮不做，可在不改 spec 与结构的前提下后续追加。
- 是否提供「只看目录」的树过滤开关——GitHub 有类似能力；本轮不做，后续独立决定。
- 窄屏断点的确切取值——落实时按 320px 断言实测选定，不影响 spec 与任务分解。
