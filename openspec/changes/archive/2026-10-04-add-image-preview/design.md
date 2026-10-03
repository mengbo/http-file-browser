# Design

## Context

今天点击任何文件都走同一条管线：`load()` → `/api/list` → `not_a_directory` → `/api/content` → JSON（path + content UTF-8 字符串）。扩展名不在文本白名单的文件被 `isTextFile` 以 `not_text` 拒绝（`internal/server/content.go`），前端据 `ERROR_TEXT.not_text` 显示「这是二进制文件」。

会被触及的现状：

- 服务端基础设施齐备：`b.resolve`（规范化 + 字面越界判定）、`classify`/`fail`/`writeError`（JSON 错误信封）、`writeJSON`。图片端点全部复用，不新增一份越界判定。
- 判定先于打开的顺序纪律（Change 04 design D10）：Stat 先行，命名管道在 `os.Open` 之前被 `not_a_regular_file` 拦下。图片端点继承同一顺序。
- 前端「装饰非门」先例：`isMarkdownPath` 纯名字判定只做分派，门在服务端（Change 07 D3）；更早的 Change 02 D5/D8 决策是前端**不持**「可否预览」清单——点开非文本文件由服务端 `not_text` 说话。
- `#preview` 是 `<pre>` 元素，`innerHTML`/`textContent` 整体覆盖写入；呈现形式切换按钮必须独立于它（Change 08 D3 的教训在 index.html 注释里）。
- 约束：ADR-0001（单二进制）、ADR-0002（JSON API + 内嵌零构建前端）、ADR-0003（呈现层 Scenario 验收分派）。

## Goals / Non-Goals

**Goals:**

- 图片扩展名的文件可查看：端点返回字节序列，浏览器解码呈现；识别纯按名，错误诚实。
- 零新依赖、零新 vendored 库——图片解码完全交给浏览器。
- 唯一的既有 spec 变更是 `service-startup` 的分区承诺开口；其余四个 capability 零 delta。

**Non-Goals:**

- 不做 SVG：文本型图像格式拖着 text 白名单变更、呈现形式推广、直接打开 URL 时的脚本执行面三件事，记想法池独立立 Change（探索已定）。
- 不做图片嗅探（无扩展名文件按内容判图）——与 `text-preview` 的无扩展名嗅探**刻意不同**，见 D2。
- 不设大小上限，也不写任何限制逻辑——见 D3。
- 不动 `text-preview` 的 1 MiB 上限——它的传输路径没变。
- 不做缩略图、EXIF 方向处理、旋转缩放——浏览器语义已覆盖（`<img>` 默认按 EXIF 方向渲染），不写 spec 不做实现。
- 不做图片编辑、写回、下载 attachment 语义。

## Decisions

### D1. 端点 `/api/image?path=...`：/api/ 区域内、query 取路径、成功响应为字节

- **命名取 `/api/image` 而非 `/api/raw`**：门只开图片扩展名，「raw」暗示任意字节可取，名字会撒谎。`/api/image` 与实际承诺一致。
- **留在 `/api/` 区域内、路径走 query 参数**（不设 `/files/<path>` 路径段端点）：全项目路径约定就是 `?path=`；`http.ServeMux` 会清理并重定向含 `..`/多余斜杠的 URL 路径，与我们自己的 normalize-then-check 打架，`%2F` 编码斜杠是另一整类问题。query 参数绕开全部这些。
- **MODIFIED 的形状**（specs/service-startup delta）：分区承诺只在**成功响应**上开口——图片字节 + 按扩展名的图片内容类型。错误响应仍一律 JSON 信封：`JSON error responses` 的既有 Scenario 是端点无关的（「因目标不存在、无权限等原因」），对图片端点继续适用，零 delta。
- 拒绝的替代：端点挪出 `/api/`（同样要 MODIFIED 分区，还要解释第三个区域；无收益）。

### D2. 识别：扩展名白名单，命中即给，不验魔数

```
var imageExtensions = map[string]string{
  "png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg",
  "gif": "image/gif", "webp": "image/webp", "bmp": "image/bmp",
  "ico": "image/x-icon", "avif": "image/avif",
}
```

- 白名单与 Content-Type 映射同一张表（扩展名 → 内容类型），进 design 不进 spec（与 `textExtensions` 同一分工：spec 承诺规则，清单是可变细节）。大小写折叠同 `isTextFile`。
- **命中即给，不读内容**：`logo.png` 装垃圾字节照样按名字提供。Change 04「文件名与内容不符的责任在文件系统」的同构重演——服务端不校验魔数，解码失败由浏览器 `onerror` 兜底、前端给回退说明。「不检测比检测错更诚实」的又一实例。
- **无扩展名文件一律拒绝，不嗅探**：`text-preview` 对无扩展名文件嗅探内容，是因为「文本」无法从名字确认；图片恰恰相反——内容是图片而名字不带图片扩展名的文件（数据文件、无主二进制）不值得预览。且图片嗅探（魔数表）是另一套机制，Change 05 的 binary data byte 判据在这里无用。拒绝，返回 `not_an_image`。
- **排除项**：`tiff`/`heic`——目标浏览器渲染不可靠（Safari 与 Chromium 支持面割裂），白名单只承诺「进来就能渲染」的格式；`svg`——Non-Goal。
- **判定顺序**：`resolve` → Stat（存在性/类型，FIFO 不阻塞）→ 扩展名识别 → Open（权限错误在此浮现）→ 发送。识别先于打开：非图片文件报 `permission_denied` 会暗示「有权限就能看」，是句假话——与 `not_text` 先于 `too_large` 同一诚实逻辑。

### D3. 无大小上限：流式发送，内存与文件大小无关

- 实现 `http.ServeContent`（`*os.File` 天然是 `io.ReadSeeker`）：内部 32 KB 缓冲流式拷贝，服务端内存恒定，顺带免费获得 Range/Last-Modified（不承诺、不写 spec，测试只覆盖 GET）。
- **文本的 1 MiB 是传输路径的保险丝**：整份内容进内存 → JSON 字符串 → DOM，三步各吃一份「文件多大就多大」。流式路径没有这个风险，上限在这里**没有保护对象**。
- **`<img>` 看不见 JSON 错误体**：`too_large` 的诚实性（「明说看不了，好过截断让你以为看完了」）依赖错误对用户可见；经 `<img>` 这条路，任何 JSON 错误都塌缩成同一个 `onerror`。若设上限，用户会把「太大」误读成「图坏了」——上限在这里不是诚实装置，是新误会的制造机。
- 真实照片 3–8 MB，任何「防病态大文件」的数字（50 MiB 级）都远高于日常所需且防不住真正的病态（GB 级 png）；病态文件的代价由浏览器解码承担，服务端无恙。探索已拍板接受。
- `text-preview` 的 1 MiB **不翻案**：它的路径理由完整保留。

### D4. 前端分派：扩展名装饰映射（装饰非门）

- `isImagePath(path)` 与 `isMarkdownPath` 同形：前端持一份图片扩展名集合（与后端白名单**各自独立声明**，注释互相引用提醒同步），仅决定走哪条呈现管线。
- 点击流不变：`load()` → list → `not_a_directory` → **按扩展名分派**。图片文件直接进图片视图（不先请求 `/api/content`，零浪费请求）；扩展名不像图片的走既有 content 流程，全部行为零变化。
- **两个方向的漂移都无害**：前端认了后端不认 → `<img>` 收到 JSON 错误响应体 → `onerror` → 回退说明（诚实）；后端认了前端没认 → 落回 content 端点 → `not_text`「这是二进制文件」消息（少预览了，但不是谎言）。服务端的门始终是唯一判定者。
- 拒绝的替代：列表响应加 `kind` 字段——同时惊动 `directory-browsing`（响应形状）、Change 03「不拆 metadata capability」的论证、想法池 D12 符号链接条目类型问题；为图片分派背这三样不成比例。等真有「列表就区分种类」的需求（如缩略图）再整体决策。
- `ERROR_TEXT` 增补 `not_an_image`（文案「该文件不是可识别的图片文件」）：`<img>` 路径看不见它，但它保持错误映射表完整，供潜在的非 `<img>` 消费路径与未来调试。

### D5. 文件视图：独立挂载点 + onerror 回退说明

- **`<img>` 不进 `<pre id="preview">`**：`#preview` 的不变量是「textContent 等于文件内容」（素文本/高亮路径）或「渲染 HTML」（Markdown 路径），图片不属于任何一种。新增独立元素 `<img id="image-view" hidden>`，与 `#preview` 平级、同受 `hidePreview` 清理。`src` 指向 `/api/image?path=<被查看路径>`。
- **加载失败 → 回退说明，不留静默破图**：`onerror` 显示一条说明（新文案，如「无法以图片查看该文件」）。与 markdown-preview 的内嵌图「SHALL NOT 报错」**刻意不同**：内嵌图失败不该拖垮其余内容，而这里图片就是整个视图，静默破图等于装作看见了。路径级错误（`not_found` 等）经 `<img>` 同样塌缩为 `onerror`，同一回退说明——spec 只承诺「回退说明出现」，与塌缩现实一致。
- **无呈现形式切换**：光栅图没有第二种呈现形式（「查看源码」是 SVG 的事）；切换按钮对图片文件隐藏。
- 位置显示用点击目标的路径（列表条目 + `joinPath` 构造，本就规范化）；上级入口沿用 `parentOf` 客户端推导，零变化。

### D6. Scenario → 断言位置分派（ADR-0003 落地）

| Scenario | 断言位置 | 方式 |
|---|---|---|
| An image file is requested | **API 层** | Go `httptest`：`GET /api/image?path=x.png` → 200、`Content-Type: image/png`、响应体逐字节等于文件内容 |
| A path inside the root traverses a symbolic link outward | API 层 | 指向根外的符号链接 → 200 提供目标内容 |
| The position contains redundant segments | API 层 | `./a/./b.png` 等冗余形式 → 提供规范化路径的内容 |
| A file with a known image extension is requested | API 层 | 同第一行（白名单命中） |
| A file with a non-image extension is requested | API 层 | → JSON 错误 `not_an_image` |
| A file with a non-image extension contains image content | API 层 | png 字节装进 `.bin` → 仍 `not_an_image` |
| A file without an extension contains image content | API 层 | png 字节、无扩展名 → 仍 `not_an_image` |
| The requested path does not exist / points to a directory / is not a regular file / cannot be read / is outside the root | API 层 | 五个路径级错误各一条 Go 测试，断言复用的错误标识 |
| The requested file is not recognized as an image | API 层 | 与 `not_an_image` 各 Scenario 同源覆盖 |
| Image content endpoint succeeds（service-startup） | API 层 | 成功响应的内容类型非 `application/json`、响应体为图片字节 |
| API path is requested（收窄后，service-startup） | API 层 | `GET /api/image?path=file.txt` → JSON 错误响应（错误信封仍 JSON 的直接证据）；既有各 `/api/` 端点测试继续通过 |
| An image file is viewed | 呈现层 | agent-browser：点击 `.png` 条目 → 图片元素可见且加载完成 |
| An image that cannot be loaded is viewed | 呈现层 | agent-browser：构造内容损坏的 `.png` → 回退说明出现、无破图占位 |

API 层全部进 `go test ./...` 全量回归；呈现层两条在 tasks.md 验收清单以 agent-browser 走查勾选记录。

## Risks / Trade-offs

- [Content-Type 与实际内容不符（`.png` 装 JPEG 字节）] → 浏览器图片解码自带内容嗅探，错标类型通常仍能渲染；解码失败有 `onerror` 兜底。接受。
- [无上限：GB 级病态文件拖垮浏览器标签页] → 服务端流式无恙；单用户本地工具，打开什么由用户选择。探索已接受。
- [前后端白名单漂移] → 两个方向都无害（D4 论证）；两张表在各自文件里注释互引，提醒同步。
- [ico 的内容类型拼写（`image/x-icon` vs `image/vnd.microsoft.icon`）] → 浏览器两者皆认，取常见惯例 `image/x-icon`；spec 只承诺「对应的图片内容类型」，不影响任何 Scenario。
- [`http.ServeContent` 附带 HEAD/Range 行为面] → 免费获得、不承诺也不禁止；测试只覆盖 GET，行为面扩大不改变 spec 承诺。
- [SVG 缺席让文档目录体感缺一块] → 有意的范围纪律；想法池条目承接，届时独立 Change 连同 text 白名单与安全分析一起做。

## Open Questions

（无——白名单成员、端点命名、判定顺序、回退语义均已定稿；样式类名等纯实现细节属 tasks 层。）
