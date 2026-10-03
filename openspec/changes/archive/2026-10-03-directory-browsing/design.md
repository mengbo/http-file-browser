# Design

## Context

动机见 proposal.md — Why。此处只记录约束本方案的现状：

- `internal/server.NewAPIHandler()` 当前不接受任何参数，也不持有根目录（`server.go:11`）；根目录只在 `internal/app.run` 里算出来（`app.go:28`），从未传给 HTTP 层。新增列表端点必须先把根目录送进 server 层。
- 现有测试名与 Scenario 标题一一对应（`TestServiceHealthIsQueried` ↔ `Service health is queried`），ADR-0002 要求每个 Scenario 落一个 `net/httptest` API 层测试。Change 01 的 design D7 还确立了「前端自身不做自动化测试」，因为零构建意味着没有测试运行器。
- Change 01 的 design D4 记录了一个被刻意推迟的决策，见下方 D6。
- 前端目前是 17 行 `index.html` + 21 行 `app.js`，只有一行状态文本，没有任何路由或视图层。

## Goals / Non-Goals

**Goals:**

- 让服务端持有根目录，并以此为唯一的越界判定基准
- 确立一套后续 Change 可直接沿用的「位置表示 + 列表响应」契约，避免 `file-metadata`、`text-preview` 各自发明一套路径表达
- 把可观察行为尽量表述在 API 层，使 20 个新 Scenario 中尽可能多的部分有自动化断言
- 让 Change 01 留下的两个显式技术债（错误信封、Scenario→测试映射细则）在本 Change 内结清

**Non-Goals:**

- 不引入任何 JS 测试设施、不引入 npm 或打包器（ADR-0002 已接受零构建的代价）
- 不做目录分页或虚拟滚动。大目录会一次性返回全部条目，这是已知限制而非本 Change 的设计选择
- 不引入新的第三方 Go 依赖

## Decisions

### D1. 浏览位置由 query 参数 `?path=` 承载，不用 path-based 路由

```
http://127.0.0.1:8080/?path=Developer/http-file-browser
```

**为什么不用 `/browse/a/b` 这种把位置放进 URL 路径的方案**：它要求服务端对任意非静态资源路径回落到 `index.html`，也就是要在 `http.FileServer` 兜底之外加一层 SPA fallback（`server.go:33`）。那会改动已归档的 `service-startup` / `HTTP surface partitioning` Requirement——该 Requirement 只承诺根路径与前端引用的静态资源返回前端页。`?path=` 完全不碰它：`http.FileServer` 本来就在 `/` 返回 `index.html`（`TestRootPathReturnsFrontendPage` 已覆盖），带 query 的 `/` 仍然是 `/`。Change 02 的服务端 diff 因此是纯增量的。

**为什么不用「位置只存在前端内存里」**：那样刷新回到根目录、浏览器后退键失效、当前目录无法收藏。Finder 类工具里「刷新还在原地」是用户可感知的基本预期，而它的实现成本只是十几行 History API。

**为后续 Change 预留的余量**：`?path=a/b&view=preview`（Change 08 的渲染/源码切换）、`?path=a/b&q=foo`（Change 10 的搜索）都能直接加参数。path-based 方案要为 view 状态另发明一套机制。

### D2. `path` 用相对根目录的路径，不用绝对路径

绝对路径方案的问题集中在转义与长度：整条 `/Users/mengbo/Developer/...` 作为单个 query 值要整体转义，而文件名可以包含 `&`、`#`、空格、中文。相对路径按段转义，且天然可读。

额外收益是**深链接可跨机器复用**：同一个 `?path=docs` 在别人机器上以另一个根目录启动后照样能打开，对一个「零配置、拷走即用」的工具是实打实的价值。

绝对路径的显示需求由 `/api/health` 响应体追加一个 `root` 字段满足，前端用它拼出「当前位置」的完整路径。追加字段不违反该 Requirement 既有的 Scenario（`Service health is queried` 只要求「返回 JSON 响应体，指示服务已就绪」，未约束字段集合），所以同样不需要 MODIFIED。

### D3. 越界判定纯字面，不解析符号链接

服务端把请求路径与根目录都当作**字面路径**处理，全程不调用 `EvalSymlinks`。判定规则因此是：规范化请求路径后，检查它是否等于根目录或以「根目录 + 路径分隔符」开头。

**为什么不解析符号链接**：

1. 纯字面判定**内部自洽**。若只对请求路径解析符号链接、而不对根目录解析，会在 macOS 上产生假越界——`/tmp` 本身是指向 `/private/tmp` 的符号链接，两侧解析深度不一致时 `Rel` 会给出 `../../..`。两边都不解析就没有这个不对称。
2. 当前形态是回环单用户，符号链接指向根目录之外的实际风险接近零：能发起请求的人本来就能浏览本机文件系统。
3. 解析符号链接会改变用户可感知行为——根目录内指向外盘的常用软链（`~/Downloads/外盘`）会点不开，这个代价现在不值得付。

**代价必须写进 spec 而不是藏起来**：所以 `Root directory confinement` 里有一条 Scenario 明确承诺「字面位置在根目录内、但经符号链接指向外部的路径，按其字面路径提供内容，不视为越界」。spec 承诺的保证强度必须等于实现提供的保证强度，否则就是欠实现。

**何时必须翻案**：Change 13 `add-remote-access` 一旦允许非本机主机访问，这条「不保证」就从可接受变成漏洞。届时用 MODIFIED 正式化（见 Risks）。

### D4. 越界检查用 `filepath.Rel`，不用裸 `strings.HasPrefix`

```go
// 有坑：根目录 /a/b 时，/a/bc 的字符串前缀也匹配
if !strings.HasPrefix(resolved, root) { ... }
```

`/a/b` 与 `/a/bc` 是两个不同的目录，前缀判断会把兄弟目录放进根目录内。改用 `filepath.Rel(root, resolved)`，结果为 `..` 开头或等于 `..` 即判定越界；结果为 `.` 即是根目录本身。

这条值得在 spec 里单独立一个 Scenario（`A sibling directory shares the root path prefix`），因为它是「看起来正确」的写法最容易漏掉的一类 bug，测试也最容易漏测。

### D5. 排序：目录优先 + 大小写折叠 + 字节序 tiebreak

比较键为 `(非目录在前, strings.ToLower(name), name)` 三元组。

- 前两段满足「目录在前」与「按名称排序、不因大小写颠倒顺序」
- 第三段保证**仅大小写不同的同名条目**（`README` 与 `readme`）在多次请求间顺序完全一致，对应 `List ordering` 的第四条 Scenario

用 `strings.ToLower` 的 Unicode 简单折叠，**不做 locale 排序**。`sort.Slice` 不稳定，若不补 tiebreak，`README`/`readme` 这类条目的相对顺序会在每次请求间抖动——那正是上面那条 Scenario 要防的。中文文件名按码位排序（不按拼音），这是一个明确取舍：locale 排序会依赖运行环境语言，`fr` 与 `en` 下顺序不同，与「顺序可复现」的 Scenario 冲突。

### D6. 错误信封升级为 `{"error": {"code", "message"}}`，兑现 Change 01 的 D4

```json
{ "error": { "code": "outside_root", "message": "路径超出根目录范围" } }
```

Change 01 的 design D4 当时有意选了单字段 `error`，并**预先写下了这个字段的触发条件**：

> 结构化 code 的价值在于多客户端或前端需要分支处理时，那属于后续 capability 出现后的演进，届时用 MODIFIED 表达，而不是现在预留一个无人使用的字段。

目录浏览就是那个消费者：不存在（可以提示并留在原处）、无权限（提示无权限）、不是目录（用户点错了）、越界（不该从界面可达，但深链接可能触发）——四种情况前端的反应各不相同，单一字符串只能靠匹配文本分支。

HTTP 状态码继续保留，与 code 并存：

| code | 状态码 | 理由 |
|---|---|---|
| `not_found` | 404 | 路径不存在；同时也是未知 API 端点的 code |
| `not_a_directory` | 400 | 请求指向了一个非目录资源 |
| `permission_denied` | 403 | 身份合法但资源不可读 |
| `outside_root` | 400 | **请求本身不合法**（路径越界），不是身份受限 |

`outside_root` 用 400 而非 403 是有意的区分：403 意味着「换个身份可以」，而越界是路径本身超出了服务承诺的范围。`not_found` 被两个场景共用（未知端点、路径不存在），与 Change 01 现有的 404 行为一致，message 区分二者。

**为什么嵌套而非扁平 `{"error": "...", "code": "..."}`**：嵌套让「code 与 message 是同一次失败的两种表述」这层从属关系在类型上就成立，前端不会写出同时读 `error` 字符串和 `code` 又可能读到其中一个缺失的代码。

### D7. 根目录注入 server 层，延续既有的注入式测试 seam

`server.NewAPIHandler()` → `server.NewAPIHandler(root string)`，`server.NewHandler(assets fs.FS)` → `server.NewHandler(root string, assets fs.FS)`。`app.run` 已经算出 `root`，直接传下去。

延续 Change 01 design D7 第 1 条确立的形状：生产入口 `Run` 用真实根目录，内部 `run` 保持可注入，使 `server` 包单测能用 `t.TempDir()` 造出「根目录」而不依赖调用方的机器状态。

不在本 Change 引入配置层（配置文件、环境变量、命令行 flag）——根目录来源仍只有命令行参数，与 `service-startup` 的参数校验 Requirement 保持一致。

### D8. 前端用 History API 同步位置，且禁止 `innerHTML`

位置同步：初始加载读 `location.search` 取 `path`；点击子目录时 `history.pushState` 后重新拉取渲染；监听 `popstate` 使浏览器前进/后退可用。全程不重载页面，因此 `/api/` 分区与静态资源的边界始终不变。

**条目名一律用 `textContent` 写入，绝不拼接 `innerHTML`**。这是文件名可以包含任意字节所直接导致的注入面：一个名为 `<img src=x onerror=...>` 的文件用 `innerHTML` 渲染就会执行脚本。零依赖的前端没有转义库可用，`textContent` 是唯一正确的默认选择。目录链接同理——条目名进入 `href` 时需要 `encodeURIComponent`。

### D9. Scenario 到测试的映射（兑现 ADR-0002）

沿用 Change 01 D7 的约定：测试名 = Scenario 名的 snake_case 变体，断言全部落在 `net/httptest` API 层。

```
spec 中的 Scenario                      测试落点
──────────────────────────────────────────────────────────────
Directory listing response / 4 个        server 包 httptest，
                                        以 t.TempDir() 造目录树，
                                        断言 path / parent / entries

Entry type distinction                   server 包 httptest，
                                        断言混合目录下条目的 type 取值

List ordering / 4 个                     server 包 httptest，
                                        断言顺序；对仅大小写不同的
                                        同名条目重复请求两次比对

Position representation / 2 个           server 包 httptest，
                                        断言同一路径两次请求结果一致、
                                        含上级片段的路径解析正确

Parent directory reference / 2 个        server 包 httptest，
                                        断言子目录 parent 正确、
                                        根目录 parent 为空

Root directory confinement / 3 个        server 包 httptest；
                                        前两条用字面路径构造；
                                        第三条（符号链接）单列并
                                        在不支持符号链接的平台跳过

Directory access failures / 4 个         server 包 httptest，
                                        断言各自的 code 与状态码

Failure causes are distinguishable      同一测试内制造两种不同失败，
(Change 01 MODIFIED 新增 Scenario)        断言两个 code 不等、message 不同

未知 API 端点（Change 01 既有 Scenario）  现有测试需同步更新：
                                        错误体形状从单字段变嵌套
```

**两个需要额外照料的 Scenario：**

1. `A path inside the root traverses a symbolic link outward` —— 造一个指向根目录外部的符号链接。Windows 上创建符号链接需要特权，测试在该平台 `t.Skip` 而非失败。
2. `The requested directory cannot be read` —— 造一个 `0o000` 权限的目录。**以 root 身份运行测试时该目录仍可读，用例会假失败**。缓解：测试开头检查有效用户 id，为 root 时 `t.Skip` 并说明原因，而不是让它在某些 CI 环境里变红。

**前端行为的验证方式**（沿用 Change 01 D7 第 3 条，零构建下不做前端自动化测试）：刷新停留在原目录、后退/前进可用、根目录时上级入口不出现、错误提示可读——这四项列为 tasks.md 中的手工验证项，不伪装成有自动化断言的 Scenario。API 层只对「响应中的 `path` 是规范化后的相对路径」负责，而前端照抄这个值——契约在有测试的一侧被钉住。

### D10. `parent` 为空字符串有两种含义，客户端一律用 `path` 判断是否在根目录

`Parent directory reference` 的 Requirement 同时规定了两件事：子目录要给出上级相对路径、根目录不给出上级路径。这两条落到取值上并不互斥——根目录的相对路径本身就是空字符串，于是

| 当前目录 | `path` | `parent` |
|---|---|---|
| 根目录 | `""` | `""`（不给出上级） |
| 根目录的一级子目录 `docs` | `"docs"` | `""`（上级就是根） |

两者在 `parent` 上无法区分。**决策：保持这一形状，不改响应结构**，客户端判定「是否在根目录」一律看 `path`，`parent` 只当作上级入口的目标值使用。

**为什么不把根目录的 `parent` 改成 `null` 或直接省略**：那要 MODIFIED `Parent directory reference`，而整个响应的信息量并没有增加——`path` 已经唯一确定「是否在根目录」，任何要渲染面包屑的消费者都必须先读 `path`。为一个客户端能自行推导的字段引入第二种「无值」表示（`null` 与缺省），只会让 JSON 形状在两种情况下分叉，反而更难消费。

**代价**：只看 `parent` 的消费者会踩坑。缓解：写进本 design、`TestParentReferenceIsGivenForADirectoryInsideTheRoot` 的注释与表格中，并在 `docs/roadmap.md` 想法池登记——若后续 Change（如 `file-metadata` 的面包屑）发现按 `parent` 推断根目录更方便，再 MODIFIED 正式化。

### D11. 绝对路径形式的浏览位置按 `outside_root` 拒绝

spec 约定 `path` 是相对根目录的路径，但没写收到 `?path=/etc` 该怎么办。`filepath.Join(root, "/etc")` 会把它重解释成 `root/etc`——不构成越界，却是个让人困惑的静默改写。**决策：先判 `filepath.IsAbs`，是则直接返回 `outside_root`。**

理由有三：拒绝比静默改写更符合「浏览位置是相对路径」这一已声明的契约；深链接里出现的绝对路径几乎都是人手拼错的，用「超出浏览范围」提示比「打开了一个同名子目录」更接近用户的真实处境；代价只是一行判断。

### D12. 条目类型按字面判断，指向目录的符号链接显示为文件

列表用 `os.ReadDir` 的 `DirEntry.IsDir()`，它对符号链接返回 false（不跟随）。因此根目录内指向某个目录的符号链接会以 `type: "file"` 呈现、不可点击，但直接以它的字面路径请求仍能列出目标内容——这正是 D3 承诺的强度。

**为什么保持字面**：`IsDir()` 免去每个条目一次 `stat`（大目录一次性返回全部条目，D5 已接受该取舍）；更重要的是跟随会让同一套代码里出现「越界判定不解析符号链接、条目分类却解析符号链接」的不对称，而 D3 明确选择了「纯字面、内部自洽」。既然系统承诺按字面路径提供内容，按字面报告类型就是这个承诺的一致延伸。

**代价**：用户看到一个能进去的目录却点不动。缓解：登记到想法池——`file-metadata` 要展示「种类」时，自然会面临「要不要把符号链接作为第三类条目」的抉择，那时用 MODIFIED `Entry type distinction` 正式化比现在猜要准。


## Risks / Trade-offs

- **符号链接可绕过根目录边界** → 这是本 Change 已知且**被 spec 明确承认**的限制，不是遗漏（见 D3）。缓解：spec 措辞与实现强度一致，不做虚假承诺；`add-remote-access`（Change 13）必须先用 MODIFIED 正式化符号链接策略再开放非回环监听。该决策已记入 `docs/roadmap.md` 想法池。
- **大目录一次性返回全部条目** → 十万级文件的目录会产生很大的响应体。缓解：本 Change 不分页，限制记录在案；若日后需要分页，那会改变 `Directory listing response` 的响应形状，属于新增 Requirement 而非本 Change 的实现调整。
- **前端四项行为无自动化断言** → 刷新/前进后退可能回归而不被发现。缓解：把可观察行为尽量表述为 API 层行为（Change 01 D7 第 3 条已确立此策略），并把四项手工验证写进 tasks.md 而非含糊带过。
- **`permission_denied` 测试依赖运行身份** → 以 root 运行时假失败。缓解：见 D9，检测到 root 即 skip 并说明。
- **排序用简单大小写折叠而非 locale 排序** → 中文文件名按码位而非拼音排列，与 macOS Finder 的观感不同。缓解：这是为「顺序可复现」Scenario 做的自觉取舍（见 D5）；若日后要改，必须 MODIFIED `List ordering`。
- **Change 01 的既有测试需同步更新**（`TestUnknownAPIEndpointReturnsJSONError` 断言错误体含 `error` 字段，形状变了）→ 缓解：这是 MODIFIED 的应有代价，任务里显式列出，不静默改测试。

## Migration Plan

不适用。无数据迁移，无既有部署。错误信封的结构变化对前端与后端同处一个内嵌二进制，不存在版本错配窗口；若需回滚，撤销本 Change 的提交即可，前端回到单行状态视图。

## Open Questions

以下问题可推迟到后续 Change 回答，届时不改变本 Change 的 spec、方案与任务划分：

- 列表加载过程中与加载失败时的视觉呈现（骨架、空态、错误提示的具体措辞与位置）——纯呈现层，spec 未约束。
- 错误 `message` 是否附带底层系统错误串（如 `permission denied` 的原始英文文本）——附带的可调试性收益与信息暴露之间的取舍，属于错误文案约定，可与 Change 13 的安全边界一并决定。
