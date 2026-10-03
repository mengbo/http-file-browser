# Design

## Context

Change 04 的 design D7 已把内容嗅探方案定到可照抄的粒度：判据用 WHATWG MIME Sniffing 的 binary data byte，窗口 4096 字节，不引入 `net/http.DetectContentType` / libmagic / filetype / charset（理由见 D7 原文，本次不重新调研）。已知缺口（无 BOM 的 UTF-16 会被 NUL 误杀）也在 D7 预告，本 Change 一并修复。

会被触及的现状（`internal/server/content.go`）：

- 判定函数 `isTextFile` 只看名字；带非白名单扩展名的文件不打开文件即拒绝。
- 判定位于 `too_large` 之前，注释写明理由：「名字已经说明这个文件不会被预览，此时报文件过大会暗示它小一点就能看，那是句假话」。
- `Stat` 先行，FIFO 不会阻塞（`not_a_regular_file` 在 `os.Open` 之前判定）。
- 前端只有 `not_text` 的文案映射（`web/app.js`），无任何扩展名逻辑。

## Goals / Non-Goals

**Goals:**

- 无扩展名的文本文件（`Makefile`、`LICENSE`、`.gitignore`、`.gitkeep`）可被预览；`go.mod`、`go.sum` 这类扩展名未被收录的文本文件经白名单补充可被预览（见 D6）。
- 无 BOM 的 UTF-16 文本文件可被识别（不必可正确解码——见 Non-Goals）。
- 判定仍然快、可复现、零依赖；白名单文件路径零额外开销。
- `directory-browsing`、`service-startup` 零 delta，前端零改动。

**Non-Goals:**

- 不做 BOM/编码解码：UTF-16 文件可被打开，内容仍按 UTF-8 呈现（替换字符交错）。解码属 D9 领地，想法池已登记，等真实需求另立 Change。
- 不放宽带扩展名文件的判定：`logo.png` 装纯文本仍然拒绝（见 D1）。
- 不追求嗅探的完备性：启发式有已知误报（见 Risks），spec 如实承诺规则本身。

## Decisions

### D1. 嗅探只适用于名称没有任何扩展名的文件

```
扩展名 ∈ 白名单   ->  可读文本（不动，不打开文件）
名称没有扩展名    ->  嗅探内容起始部分（本 Change 新增）
其余（有扩展名
但不在白名单）    ->  拒绝（不动，不嗅探）
```

**为什么不是「白名单拒绝的全部嗅探」**：那会让 `logo.png` 装着纯 ASCII 变成可预览。Change 04 的 Risks 明确接受过「a.png 里装文本会被拒绝」（文件名与内容不符的责任在文件系统），备选方案会无声推翻一条已记录的决策。本次动机（Makefile / LICENSE / .gitignore / go.mod）全部是无扩展名文件，窄口径恰好覆盖。

**「名称没有任何扩展名」的实现口径**：最后一个 `.` 不存在、位于名字首位（`.gitignore` 这类点开头文件）、或其后为空（`notes.` 这类尾点名字）——三者都视为没有扩展名，进入嗅探。与 content.go 现有注释「两者都没有可用的扩展名」的口径一致；spec 维持「名称没有任何扩展名」的措辞，此口径留在 design。

### D2. 窗口 4096 进 design，判据区间进 spec

沿 1 MiB 不进 spec 的先例做精度切分：

| 内容 | 去处 | 理由 |
|---|---|---|
| binary data byte 区间（`0x00–0x08`、`0x0B`、`0x0E–0x1A`、`0x1C–0x1F`） | spec | 是判据的定义本身，来自 WHATWG 标准，非可调参数 |
| 窗口 4096 字节 | design | 可调旋钮，同 `maxContentBytes` 的地位；调它不动任何 Scenario |
| 「判定仅依据起始部分」 | spec 一句话 + 边界 Scenario | 二进制字节在窗口之后 → 仍判文本，这是用户可观察行为 |

4096 的依据已在 Change 04 D7 论证（Go 的 512 太窄：NUL 落在偏移 512 之后的 PNG 会误判为文本），照抄即可。

### D3. UTF-16 靠空字节奇偶对齐救回，不做任何 BOM 特判

对起始窗口内每个空字节（NUL）记录其偏移奇偶：**全部只落偶数位（BE 特征）、或全部只落奇数位（LE 特征）→ 不计为二进制数据字节**；奇偶混杂 → 仍判二进制。

**为什么一条规则覆盖三种情况**：

- 无 BOM UTF-16（roadmap 点名的那刀）：NUL 奇偶对齐 → 救回。
- 有 BOM UTF-16：BOM 本身（`FF FE` / `FE FF`）不是 binary data byte（`0xFF`、`0xFE` 不在区间内），误杀来自内容 NUL，同一条对齐规则救回——**识别不需要 BOM 特判**。
- 纯 CJK 内容的 UTF-16：BMP 内汉字编码不产生 NUL，本来就能过纯判据——无需处理。

**为什么不解码**：见 Non-Goals。识别宽、解码守，分界线就是 D9。

### D4. 不设「最少 NUL 个数」阈值

探索时曾计划「窗口 4096 与 NUL 阈值都进 design」；写 spec 时发现阈值是**没有保护价值的旋钮**，砍掉：

- 二进制格式若 NUL 天然对齐，多少都会超过任何小阈值，防不住；
- 阈值唯一能拦的是「只有一两个对齐 NUL」的边角，而这种文件判为文本的代价是显示一个替换字符，无害；
- spec 措辞因此可以不出现任何数字（「全部只出现于偶数位置或全部只出现于奇数位置」），design 零新增常量，未来也少一个要解释的参数。

### D5. 嗅探占据 `isTextFile` 的既有槽位

调用点不动（`too_large` 之前），函数内部分流：扩展名命中白名单 → 直接 true；无扩展名 → 打开文件读头 4096 字节判定；其余 → false。这样「判定先于超限」的原则与注释原样成立：2 GB 的 Makefile 嗅出文本后报 `too_large`（小一点就能看是真话），2 GB 的 `data.bin` 仍报 `not_text`（小一点也看不了）。开销只在无扩展名文件上发生：一次 `os.Open` + 一次 ≤4096 字节的 `Read`；白名单与非白名单路径不打开文件，FIFO 防护不变。

### D6. 白名单补充 Go 工作区扩展名（`mod`、`sum`、`work`）

`go.mod` / `go.sum` / `go.work` 有扩展名，不在 D1 的嗅探覆盖内——但它们是 proposal Why 点名的动机文件。解法不是放宽 D1，而是用 Change 04 D6 已授权的清单可变性：往白名单补 `mod`、`sum`、`work` 三个条目。清单是 design 细节，spec 只承诺「扩展名属于已知文本扩展名」这条规则本身，因此零 spec 变化、零新 Scenario。

## Risks / Trade-offs

- **对齐 NUL 的误报**：NUL 恰好全对齐的二进制格式（某些数据库文件）会被判为文本，打开显示乱码 → 启发式的固有误报，spec 如实承诺规则而非「正确性」；与 GBK 乱码同一取舍谱系（不猜测更诚实）。
- **`logo.png` 装文本仍被拒**：D1 的自觉取舍，Change 04 已接受，本 Change 不推翻。
- **UTF-16 打开后乱码**：识别成功但渲染为替换字符交错 → 与 GBK 同一处理哲学；若实测碍事，BOM 解码是自然的下一个 Change（stdlib `unicode/utf16`，零依赖）。
- **MODIFIED 的固有测试债**：`content_test.go` 的「无扩展名」用例反转、白名单交叉检查跟进、新增 7 个 Scenario 对应测试 → Change 03 journal 观察 5 已记录同类代价，计划内。
- **归档时 Purpose 冲突**：`text-preview` 的 Purpose「不包含文件类型识别能力」将变假 → 不预先代改，归档时摆出冲突、由用户授权修改并记 journal（roadmap 想法池已登记）。

## Migration Plan

不适用。无数据迁移，无既有部署。判定放宽对既有消费者是纯扩展：白名单文件行为不变，`not_text` 的适用面收窄（无扩展名文本文件从拒绝变成功），错误信封无新增标识。回滚方式为撤销本 Change 的提交。
