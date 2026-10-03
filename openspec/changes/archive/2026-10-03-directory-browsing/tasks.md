# Tasks

## 1. 错误信封升级（兑现 Change 01 的 D4 与本 Change 的 service-startup MODIFIED）

- [x] 1.1 把 `errorResponse` 从单字段 `{"error": "..."}` 改为嵌套结构 `{"error": {"code": "...", "message": "..."}}`，未知 API 端点返回 `code` 为 `not_found`、状态码 404、Content-Type 为 JSON；验证：`go build ./...` 通过
- [x] 1.2 更新既有测试 `TestUnknownAPIEndpointReturnsJSONError` 以断言新的嵌套错误体（不得为迁就旧断言而保留旧形状）；验证：`go test ./internal/server/` 通过
- [x] 1.3 确认 `/api/` 区域不产生 HTML 错误页或纯文本错误响应；验证：`TestUnknownAPIEndpointReturnsJSONError` 与 `TestAPIPrefixTakesPrecedenceOverFileService` 同时通过

## 2. 根目录注入 server 层（design D7）

- [x] 2.1 `server.NewAPIHandler` 与 `server.NewHandler` 增加根目录参数，由 `app.run` 把已校验的根目录传入；验证：`go build ./...` 通过
- [x] 2.2 同步 `internal/app` 与 `internal/server` 既有测试的调用点，app 层单测用 `t.TempDir()` 提供根目录；验证：`go test ./...` 通过，即 Change 01 的 14 条 Scenario 全部仍然成立（尤其是根路径返回前端页、`/api/` 前缀优先于文件服务两条）

## 3. 路径解析与根目录边界判定（design D3、D4）

- [x] 3.1 实现相对根目录路径到绝对路径的规范化解析：正斜杠转本地分隔符后做 `Clean`，根目录自身解析为根；验证：3.2 的测试覆盖含 `..` 的输入
- [x] 3.2 用 `filepath.Rel` 而非 `strings.HasPrefix` 判定越界；验证：`TestAPathEscapesTheRootByParentReferences` 与 `TestASiblingDirectorySharesTheRootPathPrefix` 均通过（后者用根目录 `a/b`、请求 `a/bc` 的构造，必须被拒）
- [x] 3.3 越界时返回 `outside_root` 错误而不返回任何目录内容；验证：`TestTheRequestedPathIsOutsideTheRoot` 通过
- [x] 3.4 判定链路中不解析符号链接：根目录内指向外部的符号链接按字面路径正常提供；验证：`TestAPathInsideTheRootTraversesASymbolicLinkOutward` 通过，在无法创建符号链接的平台 `t.Skip` 而非失败

## 4. 目录列表端点与响应内容（design D1、D2、D5）

- [x] 4.1 在 `/api/` 分区新增列表端点，接受 `path` 查询参数，响应包含 `path`、`parent`、`entries`；`path` 缺失或为空时返回根目录列表；验证：`TestADirectoryIsListed` 与 `TestPathParameterIsMissingOrEmpty` 通过
- [x] 4.2 存在但无条目的目录返回空列表而非错误；验证：`TestADirectoryHasNoEntries` 通过
- [x] 4.3 响应中的 `path` 为规范化后的相对路径（多余当前目录标记、重复分隔符、结尾分隔符均被归一）；验证：`TestPathParameterContainsRedundantSegments` 通过
- [x] 4.4 每个条目只带 `name` 与 `type`，不带大小、修改时间或内容；验证：断言反序列化后的条目结构恰为这两个字段
- [x] 4.5 条目标注目录或文件；验证：`TestEntryTypeIsDistinguishedInAMixedDirectory` 通过。注意 spec 中 `Entry type distinction` 与 `List ordering` 各有一条同名 Scenario `A directory contains both directories and files`，两个测试必须用可区分的名字，不得合并或重名
- [x] 4.6 实现排序比较键（目录在前、大小写折叠后的名称、原名 tiebreak）；验证：`TestDirectoriesPrecedeFilesInAMixedDirectory`、`TestEntriesWithinTheSameGroupAreSortedByName`、`TestNamesDifferingOnlyInLetterCaseAreNotInverted` 通过
- [x] 4.7 补 `TestRepeatedListingsOfTheSameDirectoryReturnTheSameOrder`：对同一目录连续两次请求并逐项比对顺序完全相同（该用例要求排序实现自带 tiebreak，`sort.Slice` 不稳定，需显式兜底）
- [x] 4.8 响应中的 `parent` 在子目录时指向上级、在根目录时为空；验证：`TestParentReferenceIsGivenForADirectoryInsideTheRoot` 与 `TestParentReferenceIsAbsentForTheRootDirectory` 通过
- [x] 4.9 同一位置表示重复请求返回同一目录与相同列表；验证：`TestAPositionIsReopened` 与 `TestPositionIsResolvedFromAPathContainingAParentReference` 通过

## 5. 目录访问失败的四种原因

- [x] 5.1 建立 `code` 与 HTTP 状态码的映射：`not_found`→404、`not_a_directory`→400、`permission_denied`→403、`outside_root`→400；验证：`TestTheRequestedPathDoesNotExist` 与 `TestTheRequestedPathIsNotADirectory` 通过
- [x] 5.2 无权限目录返回 `permission_denied`；验证：`TestTheRequestedDirectoryCannotBeRead` 通过。测试用 `0o000` 权限目录构造，**在以 root 身份运行时该目录仍可读会假失败**，须在用例开头检测有效用户 id 并 `t.Skip` 说明原因
- [x] 5.3 补 `TestFailureCausesAreDistinguishableFromTheResponseBody`：同一测试内制造两种不同失败，断言两者 `code` 不等且 `message` 不同——这是 service-startup 被 MODIFIED 后新增的 Scenario，也是本 Change 兑现 Change 01 design D4 的证据

## 6. `/api/health` 暴露根目录

- [x] 6.1 在健康端点响应体追加 `root` 字段供前端显示绝对位置；验证：更新 `TestServiceHealthIsQueried`，同时断言 `status` 仍为 `ok`、`root` 为传入的根目录

## 7. 前端列表视图与地址栏同步（design D8）

- [x] 7.1 `web/index.html` 从单行状态改为列表视图结构：当前位置显示、上级入口、条目列表容器、错误提示区；验证：浏览器打开服务地址能看到结构而非「服务就绪」单行
- [x] 7.2 `web/app.js` 实现拉取与渲染：读 `location.search` 的 `path` 拉列表、渲染条目、上级入口按 `parent` 是否为空决定显示与否；**条目名一律用 `textContent` 写入，不得拼接 `innerHTML`**（文件名可含任意字符），进入目录的链接用 `encodeURIComponent` 处理；验证：手工打开含 `<`、`&`、空格、中文命名的文件与目录，页面正确显示且无脚本执行
- [x] 7.3 用 History API 同步位置：进入目录时 `pushState`，监听 `popstate` 重新拉取渲染，不重载页面；验证：`go build` 后在浏览器中实测前进/后退可用
- [x] 7.4 错误处理：按 `code` 区分展示——`not_found` 提示位置不存在、`permission_denied` 提示无权限、`not_a_directory` 提示目标不是目录、`outside_root` 提示超出浏览范围；不把 `message` 直接当 HTML 插入；验证：手工构造上述四种深链接各看到对应提示
- [x] 7.5 文件条目在本 Change 不可进入，且不呈现可点击外观（无 hover 态、默认光标），避免死胡同手感；验证：鼠标悬停文件行无变化，点击无反应且不报错
- [x] 7.6 `web/style.css` 扩充列表视图样式；验证：列表、当前位置、上级入口在窄窗口下不溢出

## 8. 集成验证

- [x] 8.1 跑 `go vet ./...` 与 `go test ./...`，确认无新增告警、全部测试通过；验证：命令输出无 FAIL
- [x] 8.2 端到端手工验证四项前端行为（零构建约束下无自动化测试，design D9 已声明）：刷新停留在原目录、浏览器前进/后退可用、根目录时上级入口不出现、错误提示可读；验证：逐项记录实际观察结果，未实测的项留空不勾选

### 8.2 手工验证记录

环境：macOS / `go build` 后以 `http-file-browser /tmp/hfb-fixture` 启动，浏览器（Chromium）实测。fixture 含 `<img src=x onerror=alert(1)>.txt`、`a&b <c>.txt`、`带空格.txt`、`中文.txt`、`陷阱 <&> "q" 目录/`、`空目录/`、`locked/`（`0o000`）、`outward`（指向根目录外的符号链接）。

| 前端行为 | 实际观察结果 |
|---|---|
| 刷新停留在原目录 | 在 `/?path=docs%2Fapi` 执行 `location.reload()`，刷新后 URL 仍为 `/?path=docs%2Fapi`，当前位置显示 `/tmp/hfb-fixture/docs/api`，条目仍为 `spec.md` ✓ |
| 浏览器前进/后退可用 | 连续进入 `docs` → `docs/api` 后，历史序列为 `/` → `?path=docs` → `?path=docs/api`；连按 4 次后退依次回到 `?path=docs` → `/` → `?path=docs/api` → `?path=docs`，连按 4 次前进按相反顺序回到 `?path=docs/api`，每一步均重新拉取并渲染对应目录，全程无整页重载 ✓ |
| 根目录时上级入口不出现 | 在 `/` 与 `/?path=` 下 `#parent-link` 的 `hidden` 为 true 且计算样式 `display: none`（首轮实测发现 `.parent-link { display: inline-block }` 盖过 `[hidden]` 的 UA 规则导致误显示，已修 CSS）；进入 `?path=docs` 后该入口 `display: inline-block`、`href="/"`，进入二级目录 `?path=docs/api` 时 `href="/?path=docs"` ✓ |
| 错误提示可读 | 四种深链接各得到对应中文提示且条目区清空：`?path=no-such-dir` → 「该位置不存在」、`?path=locked` → 「没有读取该目录的权限」、`?path=README` → 「该位置不是目录」、`?path=../..` → 「该位置超出浏览范围」；提示区 `display: block` 可见，文字以 `textContent` 写入 ✓ |

同时记录的其他实测项：

- 条目名注入面：`<img src=x onerror=alert(1)>.txt` 以纯文本显示，`#entries` 内 `img`/`script` 元素数为 0，未发生脚本执行
- 目录链接转义：`陷阱 <&> "q" 目录` 的 `href` 为 `/?path=%E9%99%B7%E9%98%B1%20%3C%26%3E%20%22q%22%20%E7%9B%AE%E5%BD%95`，进入后条目 `子<名>&符.txt`、`普通 文件.txt` 正确显示
- 文件条目不可进入：`spec.md` 所在行首元素为 `SPAN`（非 `A`），`cursor: auto`，hover 前后背景色均为 `rgba(0, 0, 0, 0)`，点击后 URL、当前位置、条目数均不变且错误区未出现；目录行首元素为 `A`、`cursor: pointer`、hover 背景变为 `rgb(236, 238, 242)`
- 空目录：`/?path=空目录` 显示「此目录为空」，条目数为 0
- 窄窗口：视口 375px 与 320px 下 `documentElement.scrollWidth` 均等于 `innerWidth`，列表、当前位置、上级入口不溢出
- 符号链接向外：`/?path=outward` 按字面路径正常列出根目录外目标目录的内容，未判为越界
- `/api/` 区域：`/api/nope`、`/api/`、`/api/unknown/deep` 均为 404 + `application/json; charset=utf-8`；`/`、`/app.js`、`/style.css` 仍分别返回 HTML、JavaScript、CSS
- 暗色模式：`prefers-color-scheme: dark` 下配色正常

- [x] 8.3 对照 spec 逐条复核 20 条新 Scenario 与 1 条被修改的 Scenario 是否都有对应测试或明确的手工验证归属；验证：列出 Scenario 到测试名的对照表，确认无遗漏

### 8.3 Scenario 到验证落点对照表

`directory-browsing` 新增 20 条 Scenario（`Entry type distinction` 与 `List ordering` 各有一条同名 Scenario `A directory contains both directories and files`，用不同测试名区分）：

| # | Requirement | Scenario | 验证落点 |
|---|---|---|---|
| 1 | Directory listing response | A directory is listed | `TestADirectoryIsListed` |
| 2 | Directory listing response | Path parameter is missing or empty | `TestPathParameterIsMissingOrEmpty` |
| 3 | Directory listing response | A directory has no entries | `TestADirectoryHasNoEntries` |
| 4 | Directory listing response | Path parameter contains redundant segments | `TestPathParameterContainsRedundantSegments` |
| 5 | Entry type distinction | A directory contains both directories and files | `TestEntryTypeIsDistinguishedInAMixedDirectory` |
| 6 | List ordering | A directory contains both directories and files | `TestDirectoriesPrecedeFilesInAMixedDirectory` |
| 7 | List ordering | Entries within the same group | `TestEntriesWithinTheSameGroupAreSortedByName` |
| 8 | List ordering | Names differ only in letter case | `TestNamesDifferingOnlyInLetterCaseAreNotInverted`（本机 macOS 大小写不敏感 FS 上 skip）+ `TestSortEntriesFallsBackToNameForCaseOnlyDifferences`（脱离 FS 直接钉住 tiebreak） |
| 9 | List ordering | The same directory is listed repeatedly | `TestRepeatedListingsOfTheSameDirectoryReturnTheSameOrder` |
| 10 | Position representation | A position is reopened | `TestAPositionIsReopened` |
| 11 | Position representation | Position is resolved from a path containing a parent reference | `TestPositionIsResolvedFromAPathContainingAParentReference` |
| 12 | Parent directory reference | Listing a directory inside the root | `TestParentReferenceIsGivenForADirectoryInsideTheRoot` |
| 13 | Parent directory reference | Listing the root directory | `TestParentReferenceIsAbsentForTheRootDirectory` |
| 14 | Root directory confinement | A path escapes the root by parent references | `TestAPathEscapesTheRootByParentReferences` |
| 15 | Root directory confinement | A sibling directory shares the root path prefix | `TestASiblingDirectorySharesTheRootPathPrefix` |
| 16 | Root directory confinement | A path inside the root traverses a symbolic link outward | `TestAPathInsideTheRootTraversesASymbolicLinkOutward` |
| 17 | Directory access failures | The requested path does not exist | `TestTheRequestedPathDoesNotExist` |
| 18 | Directory access failures | The requested path is not a directory | `TestTheRequestedPathIsNotADirectory` |
| 19 | Directory access failures | The requested directory cannot be read | `TestTheRequestedDirectoryCannotBeRead` |
| 20 | Directory access failures | The requested path is outside the root | `TestTheRequestedPathIsOutsideTheRoot` |

`service-startup` 被 MODIFIED 的 `JSON error responses`，3 条 Scenario：

| # | Scenario | 验证落点 |
|---|---|---|
| 21 | Unknown API endpoint is requested | `TestUnknownAPIEndpointReturnsJSONError`（已改为断言嵌套错误体）+ `TestAPIPrefixTakesPrecedenceOverFileService`（确认 `/api/` 区域不产生 HTML 或纯文本） |
| 22 | API request fails for a domain reason | `TestTheRequestedPathDoesNotExist` / `TestTheRequestedPathIsNotADirectory` / `TestTheRequestedDirectoryCannotBeRead` / `TestTheRequestedPathIsOutsideTheRoot` 四条共用 `expectFailure`，均断言 JSON 错误体且响应体不含 `<html` |
| 23 | Failure causes are distinguishable from the response body | `TestFailureCausesAreDistinguishableFromTheResponseBody` |

Change 01 其余 4 条 Requirement 的 11 条 Scenario 仍由既有测试覆盖（`TestRootDirArgumentIsProvidedAndUsable`、`TestRootDirArgumentIsMissing`、`TestRootDirArgumentDoesNotExist`、`TestRootDirArgumentIsNotADirectory`、`TestStartupReportContainsSchemeHostAndPort`、`TestDefaultListenerIsLoopbackOnly`、`TestListenAddressCannotBeBound`、`TestServiceHealthIsQueried`、`TestAPIPrefixTakesPrecedenceOverFileService`、`TestRootPathReturnsFrontendPage`、`TestFrontendStaticAssetsAreServedWithMatchingContentType`），全部通过。

前端无自动化断言的四项行为归属 8.2 的手工验证记录。

