# Tasks

## 1. 两栏骨架与布局

- [x] 1.1 将 `web/index.html` 的 `.browser` 单列结构改为两栏骨架：新增 `<aside id="sidebar">`（放搜索与目录树）与 `<main id="main">`（放既有主区元素），保留既有全部元素 id 不变。验证：`go run . testdata` 打开页面，目录列表/预览/图片/搜索四类元素仍存在且可用。
- [x] 1.2 在 `web/style.css` 实现两栏布局（侧栏约 17rem + 主区 `minmax(0,1fr)`）与窄屏收起规则。验证：桌面呈两栏；320px 视口下 agent-browser 断言 `document.documentElement.scrollWidth === window.innerWidth`。
- [x] 1.3 在 `docs/roadmap.md` Change 地图新增 `add-directory-tree` 行并标 🚧。验证：文件内可见该行且状态为 🚧。

## 2. 目录树组件

- [x] 2.1 实现树的懒加载数据层：以路径为键缓存 `entries`（`treeLists`）、用 `Set` 记录展开状态（`treeExpanded`）、缓存进行中请求去重（`treeRequests`），展开目录时请求 `/api/list?path=`（文件不请求）。验证：展开 `testdata` 中一个子目录仅触发一次 `/api/list`，缓存命中后不再重复请求。
- [x] 2.2 渲染嵌套树：`<ul>/<li>` 结构，目录项为 `<button aria-expanded>`、导航项为 `<a>`，目录图标与文件图标沿用既有 `vendor/icons/*.svg` 且外观可区分。验证：走查 spec 场景「Expanding a directory reveals its children」「Collapsing an expanded directory hides its children」「Directory and file nodes are distinguishable」。
- [x] 2.3 实现当前位置标出与祖先自动展开：首屏按 URL `path` 逐级展开到当前节点，`load()` 后重算 active 与祖先展开。验证：走查场景「Opening the browser shows the tree」「Current position is marked with ancestors expanded」「Clicking a directory node navigates」「Clicking a file node opens the file」。
- [x] 2.4 实现整棵目录树面板的收起/展开切换。验证：走查场景「The tree can be collapsed and expanded」。
- [x] 2.5 确保树独立于主区互斥函数：`hideEntries`/`hideMatches`/`hidePreview` 不清空或隐藏目录树。验证：走查场景「Tree remains available in file view」，并额外在错误态与搜索结果视图下确认树仍在且可导航。
- [x] 2.6 统一目录树与主区列表字号（同为 `0.95rem`，design D14），消除两栏视觉分裂。验证：`.tree-label` 与 `.entry` 的 `font-size` 计算值相等。
- [x] 2.7 将目录树行距从初版 `0.2rem` 适度放大到 `padding 0.35rem`（+ `line-height 1.55`，design D14），仍比主区文件表紧凑。验证：`.tree-label` 字号与 `.entry` 相等，行高介于初版与主区列表之间（约 35px）。

## 3. 主区 GitHub 风格化

- [x] 3.1 将 `#location` 纯文本升级为面包屑：首段为根目录绝对路径，其后每段为可点链接指向对应祖先。验证：走查面包屑任一段可点并跳转到该祖先目录，位置显示与其一致。
- [x] 3.2 目录列表套用 GitHub 文件表样式：圆角边框容器、表头底/字色、行分隔线、行 hover 背景、名称文字近黑、目录图标蓝色，并在表内新增首行 `..` 指向上级。验证：agent-browser 截图与 GitHub 目录页比对（列改为 图标/名称/大小/修改时间），点击 `..` 回到上级。
- [x] 3.3 深色外观适配侧栏、目录树、面包屑与文件表容器，沿用 Change 16 的 GitHub Primer 深色值。验证：`prefers-color-scheme: dark` 下截图走查，树与表在深色下可读、对比度足够。
- [x] 3.4 将搜索框从 toolbar 移入侧栏顶部，保持「以当前位置为基准按名称搜索子树」语义不变。验证：在侧栏输入并提交仍进入搜索结果视图，URL 仍为 `/?path=<base>&q=<query>`。
- [x] 3.5 修复图标着色（既有缺陷）并换用 GitHub 实心文件夹：图标改为 CSS mask + `currentColor` 着色，`dir.svg` 换成 Octicons `file-directory-fill`，`<img>` 改 `<span aria-hidden>`（design D10）。验证：截图取样列表与树的目录图标为蓝、文件图标为灰，深色下分别为 `#4493f8` / `#8b949e`。

## 4. 集成与回归

- [x] 4.1 端到端走查四类主区视图（目录列表 / 文本 / Markdown 渲染与源码 / 图片）、搜索结果、错误态，以及前进/后退与深层路径首屏展开。验证：无回归，树在各视图下位置标出正确。
- [x] 4.2 走查认证失效路径（远程模式清 Cookie 后 API 401 跳登录）在引入侧栏后仍成立。验证：清 Cookie 后交互触发 401 并回到登录入口，不卡死。
- [x] 4.3 运行 `go test ./...` 与 `go vet ./...`。验证：服务端既有测试全部通过、无 vet 报告。
