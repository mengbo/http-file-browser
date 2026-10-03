# Proposal

## Why

目录浏览能力已经完备，但文件条目不可进入——系统只能列出文件名、大小与修改时间，用户无法看到任何文件的内容。这是「像 Finder 一样浏览」这一产品目标最大的一块落差。

Change 03 在 `Directory listing response` 里留下了一条有意识的排除：「系统 SHALL NOT 在列表中提供条目的内容」，并配了一条 Scenario `A listed directory contains readable files` 让它从一句措辞变成可断言的行为。本 Change 正是要跨过这道门——但走独立端点，而不是把内容塞进列表响应。

## What Changes

- **新增 capability `text-preview`**：提供按相对根目录路径读取文本文件内容的端点；规定内容响应包含什么；规定哪些文件被当作可读文本的文件；规定读取失败时的机器可读错误标识。
- **文件条目在前端变为可点击**，点开后展示文件内容；这一变化**刻意不写成 Requirement**（理由见 design D1）。
- **`directory-browsing` 零 delta。** `Directory listing response` 那条排除与它的 Scenario 保持原样：列表响应仍然不携带任何条目内容，本 Change 用一个独立端点跨过这道门，而不是推翻它。
- **浏览位置复用 `?path=`**，不引入第二个位置参数；前端按响应的成败分派「这次渲染列表还是预览」（理由见 design D2）。
- **判定为「非文本」时返回 `not_text` 错误**，而不是静默拒绝，也不是把内容原样返回让前端自己猜（理由见 design D5）。

本 Change 明确**不包含**以下内容，均已在探索阶段单独决定并记录：

- **不做内容嗅探。** 判定只用扩展名白名单，内容嗅探（WHATWG binary data byte）留给 Change 05 `improve-text-file-detection` 做 MODIFIED。**这是一次明知故犯的不完整判定**，理由与完整方案见 design D6 与 D?。
- **不做文件类型分类字段。** 不在列表条目上加「可否预览」的字段，所有文件条目一律可点，非文本点进去报错。`directory-browsing` 因此保持零 delta（理由见 design D5）。
- **不做编码检测。** GBK / Shift_JIS 等遗留中文编码按 UTF-8 处理，无法解码的字节以替换字符呈现（理由见 design D7）。
- **不做语法高亮、Markdown 渲染、图片预览。** 分别属于 Change 06 / 07 / 09。
- **不做截断预览。** 超过大小上限的文件返回错误而不是给出一段截断内容；「要不要截断」留到想法池。
- **不引入第三方依赖。** 判定表是十几行字节区间，不需要 libmagic 或任何库。

## Capabilities

### New Capabilities

- `text-preview`：在命令行指定的根目录范围内向用户提供**文本文件内容查看**时可观察到的行为契约——内容响应包含什么、哪些文件被当作可读文本的文件、浏览位置如何表示、以及系统在什么情况下拒绝提供内容。不包含目录浏览本身的行为，不包含文件类型识别能力（Change 05 之后才有），不包含内容的编辑或写回。

### Modified Capabilities

无。`directory-browsing` 的全部八条 Requirement 一字不改——新增端点落在独立的 capability 里，既有列表契约、排序语义、上级语义、越界判定与错误信封都不受影响。

## Impact

- **服务端** `internal/server/`：新增内容端点与文本判定；新增两个错误标识 `not_text`、`too_large` 并接入既有 `codeStatus` 映射与错误信封。`browse.go` 的列表路径、`resolve()` 的字面越界判定、`sortEntries` 全部不动。
- **服务端** `internal/server/browse_test.go`：新增内容端点的测试。既有测试**无需修改**——这是 `directory-browsing` 零 delta 换来的直接结果（对比 Change 03 的 MODIFIED 连带改了既有断言）。
- **前端** `web/app.js`、`web/index.html`、`web/style.css`：文件条目由 `<span>` 改为可点击，预览内容以 `<pre>` 呈现；新增「当前 `?path=` 指向文件」的渲染分支与错误分支。Change 02 的 D8（条目名一律 `textContent`，绝不 `innerHTML`）在本 Change 扩展到**文件内容本身**。
- **API 契约**：新增 `GET /api/content?path=`。既有 `GET /api/list` 与 `GET /api/health` 的响应形状不变，属非破坏性扩展。
- **Spec**：归档后 `openspec/specs/` 新增 `text-preview`，`directory-browsing` 不变。
- **依赖**：无新增。仍为零第三方依赖、零构建、单二进制。
