# Tasks

## 1. 后端搜索核心（遍历与匹配）

- [x] 1.1 新建 `internal/server/search.go`：手写深度优先遍历（每层 `os.ReadDir` → `sortEntries` → 输出命中 → 对目录递归）、子串匹配（`strings.Contains` + `strings.ToLower` 简单折叠）、瘦命中条目（name/type/path）与响应形状 `{path, query, matches}`。验证：单测覆盖 Result ordering 三条 Scenario——树序（基准命中先于子树命中、子树命中连续）、目录内列表顺序（目录先于文件、大小写折叠排序）、重复请求顺序完全相同
- [x] 1.2 匹配语义测试：子串命中、仅大小写不同命中、查询词只在所在路径出现时不命中、查询缺失/为空时全量命中、无命中返回空列表成功响应。验证：Name matching semantics 全部 Scenario 各有确定性测试
- [x] 1.3 遍历边界测试：软链目录自身可命中但其子树零命中、悬空软链可命中且不失败、无权限子目录跳过后其余命中照常（无权限 fixture 沿用 `browse_test.go` 处理 `permission_denied` 的既有做法）。验证：Traversal boundaries 全部 Scenario 各有确定性测试

## 2. 搜索端点与路由

- [x] 2.1 `handleSearch` + `apiHandler` 注册 `/api/search`：参数解析（`path` 缺失/为空取根、`q` 透传）、复用 `resolve` 做基准位置规范化与字面+物理越界判定、基准失败经 `classify` 映射现有错误码。验证：端点测试覆盖 Search endpoint 三条 Scenario（基准子树搜索、基准缺省为根、冗余片段规范化）与 Base path access failures 三条（not_found / not_a_directory / permission_denied），Search scope confinement 三条（`..` 逃逸、软链向外、软链向内）
- [x] 2.2 响应形状测试：响应回显规范化 `path` 与 `query`，命中条目只含 name/type/path 且不含任何内容字段；与 `/api/list` 响应字段无串扰。验证：Match entry shape 两条 Scenario 有确定性测试；`go vet ./...` 通过

## 3. 前端搜索视图

- [x] 3.1 `web/index.html` header 增加搜索框（空输入提交不动作）+ `web/style.css` 结果视图样式。验证：`go run . testdata` 启动后浏览器可见搜索框，页面无样式串扰
- [x] 3.2 `web/app.js`：`load()` 增加 `q` 分支（带 `q` 时请求 `/api/search` 渲染结果视图，否则走既有分派）；结果行显示名称、所在目录（`path` 剥末段，根内显示根绝对位置）与类型；空命中呈现无命中说明、错误走 `ERROR_TEXT` 机器可读分支；点击目录命中进列表、点击文件命中进既有预览分派。验证：testdata 浏览器走查 Search view and URL reproducibility 的提交、无命中、两类命中点击四条 Scenario
- [x] 3.3 URL 态：搜索提交 `pushState` 到 `/?path=<base>&q=<query>`，前进/后退在列表视图与结果视图间正确切换，重开同一 URL 重现结果。验证：浏览器走查重开与前进/后退两条 Scenario

## 4. 收尾集成检查

- [x] 4.1 `go test ./...` 全绿，既有测试（list/content/image）无回归。验证：命令输出零失败
- [x] 4.2 `/api/` 分区回归走查：未知端点仍返回 JSON 错误、列表/文本/图片预览行为不变。验证：testdata 浏览器走查 + `curl` 抽查
- [x] 4.3 `openspec validate add-file-search --strict` 通过。验证：命令零错误退出
