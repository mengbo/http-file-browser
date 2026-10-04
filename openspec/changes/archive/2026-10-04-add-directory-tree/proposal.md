# Proposal

## Why

目录浏览当前是**单列平铺**：要到达深层目录只能一级一级地进（点目录条目或「上级目录」），看不到同级兄弟，也看不到当前位于整棵树的哪一支。用户实机评审明确了目标形态——**GitHub 代码视图**：左侧一棵持久的目录树，右侧主区显示当前目录列表或文件内容，一眼看到祖先与同级、任意跳转。roadmap 的长期约束本就锚定「UI 风格借鉴 macOS Finder」，做这件事同时兑现该约束，并把界面结构从 MVP 的单列假设升级为两栏。

浏览、预览、搜索、远程访问、视觉精修均已归档，界面结构是最后一块仍停留在单列形态的部分。现在做，改动面清晰（纯前端呈现 + 一条新行为），且不触碰已有的列表/预览/搜索契约。

## What Changes

- `directory-browsing` 新增一条 ADDED Requirement：**目录树导航**——浏览界面持续显示一棵以根目录为起点的目录树，目录可展开/折叠、当前位置被标出且祖先自动展开、点击节点导航。
- 界面由单列改为 GitHub 代码视图的两栏结构：**左侧目录树**，右侧主区（目录列表 / 文件预览 / 图片 / 搜索结果）。
- 目录树**按需加载**：展开一个目录时才请求既有 `/api/list`，不新增后端端点、不改服务端。
- 以下为呈现层调整（属 design，不进 spec）：路径面包屑、圆角边框文件表容器、表头与行分隔样式、父目录 `..` 行、目录图标蓝色 / 文件图标灰色、名称文字由蓝色改为近黑、搜索框移入侧栏。
- 显式 Non-Goal（见 design）：
  - 不引入任何 git/repo 概念（提交信息列、分支选择器、repo 头、标签页、`Add file`）——那是 GitHub 的产品件，不是文件系统浏览器需要的东西。
  - 不新增后端 API、不改 `/api/list` 响应形状、不新增写操作。
  - 不做大目录分页、不做全树一次性拉取。

## Capabilities

### New Capabilities

（无——目录树是 `directory-browsing` 这一既有能力域内的新导航结构，不构成独立 capability 的边界；与 Change 16 把「类型化视觉图标」并入同一 capability 的取舍一致。）

### Modified Capabilities

- `directory-browsing`: 新增 `Added Requirement: 目录树导航`。既有 Requirement（`Directory listing response`、`Entry metadata`、`Entry type distinction`、`List ordering`、`Position representation and reproducibility`、`Parent directory reference`、`Root directory confinement`、`Directory access failures`、`类型化视觉图标`）一字未改。

## Impact

- `web/index.html`：新增侧栏（目录树）结构、面包屑结构；主区列表套入 GitHub 风格容器；表头/父目录行结构随样式调整。
- `web/style.css`：两栏布局、目录树样式、GitHub 文件表样式、深色外观适配。
- `web/app.js`：目录树组件（懒加载 `/api/list`、展开/折叠、当前位置高亮、祖先自动展开、点击导航）、面包屑渲染；既有列表/预览/搜索渲染复用，不新增分派门。
- `web/vendor/icons/`：展开箭头等若需图标，沿用既有 vendored Lucide 纪律（Change 16）。
- 服务端**零变化**。
- 现有行为零回归的回归面：列表条目渲染、搜索结果视图、文件/Markdown/图片预览、错误态、URL 与前进后退、认证 401 跳转。
- 约束：ADR-0001（单二进制）、ADR-0002（JSON API + 内嵌零构建前端）、ADR-0003（呈现层 Scenario 验收）、ADR-0004（默认只读）。
- 记账（apply 阶段）：roadmap Change 地图新增 `add-directory-tree` 并标 🚧，归档时转 ✅。
