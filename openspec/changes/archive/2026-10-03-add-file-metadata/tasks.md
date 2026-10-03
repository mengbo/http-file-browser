# Tasks

## 1. 条目元信息的服务端取数（design D4、D7、D9）

- [x] 1.1 `listEntry` 增加 `size` 与 `modified_at` 两个字段，`size` 为指针类型或带 `omitempty`，使目录条目与元信息不可得的条目不出现该字段（design D5：缺省而非 `null`）；验证：`go build ./...` 通过
- [x] 1.2 元信息取数**必须**走 `DirEntry.Info()`（lstat 语义），不得改用 `os.Stat`；验证：2.3 的符号链接用例通过即为证据（`os.Stat` 会让该用例失败）
- [x] 1.3 `modified_at` 取自条目 `Info().ModTime()` 并以 Unix 整秒输出，不得经过任何时区或本地化格式化（design D6）；验证：`go test ./internal/server/` 中 1.6、1.7 两条用例通过
- [x] 1.4 在 `browser` 上保留一个可注入的条目元信息取数函数，默认实现为 `DirEntry.Info`（design D9 的 seam，**本任务即为其显式声明**）；验证：包内单测能替换该函数且不影响生产路径
- [x] 1.5 条目按 Change 02 的 D5 排序键排序，`size` 与 `modified_at` **不得**参与排序；验证：`TestDirectoriesPrecedeFilesInAMixedDirectory`、`TestEntriesWithinTheSameGroupAreSortedByName`、`TestNamesDifferingOnlyInLetterCaseAreNotInverted`、`TestSortEntriesFallsBackToNameForCaseOnlyDifferences` 四条既有用例不改代码即通过

## 2. `Entry metadata` 的服务端测试（design D9，兑现 Change 02 的 D9 惯例）

- [x] 2.1 补 `TestAFileEntryIsListed`：断言普通文件条目给出该文件自身的大小与最后修改时间；验证：用例通过
- [x] 2.2 补 `TestADirectoryEntryIsListed`：断言子目录条目给出最后修改时间且**不出现** `size` 字段；验证：用例通过，且用 `map[string]any` 断言字段集合而非仅断言取值
- [x] 2.3 补 `TestAnEntryIsASymbolicLink`（Scenario `An entry is a symbolic link`）：在根目录内造一个指向根目录外某个较大文件的符号链接，断言其 `size` 等于**目标路径字符串的字节数**而非目标文件大小；验证：用例通过。在无法创建符号链接的平台 `t.Skip` 并说明原因（沿用 Change 02 的 skip 惯例）。**这条是 design D4 唯一的守门人，实现若误用 `os.Stat` 必然在此失败**
- [x] 2.4 补 `TestModificationTimeFallsWithinTheSameSecond`（Scenario `Modification time falls within the same second`）：用 `os.Chtimes` 把两个条目的修改时间设为同一秒内的不同亚秒值，断言两者 `modified_at` 取值相同；验证：用例通过
- [x] 2.5 补 `TestModificationTimeDoesNotDependOnTheHostEnvironment`（Scenario `Modification time is read under a different environment`）：同一目录分别在 `TZ=Asia/Tokyo` 与 `TZ=America/New_York` 下请求（用 `t.Setenv`），断言两次响应中每个条目的 `modified_at` 取值完全相同；验证：用例通过。该用例锁住 design D6——若实现改成格式化字符串，本用例会失败
- [x] 2.6 补 `TestAnEntrysMetadataCannotBeObtained`（Scenario `An entry's metadata cannot be obtained`）：用 1.4 的 seam 让某个条目的取数返回错误，断言请求成功、该条目仍出现在列表中、其 `size` 与 `modified_at` 均被省略；验证：用例通过
- [x] 2.7 补 `TestRepeatedListingsReturnTheSameMetadata`（Scenario `Repeated listings return the same metadata`）：对同一目录连续两次请求，逐条目比对 `size` 与 `modified_at` 取值相同；验证：用例通过

## 3. `Directory listing response` 的 MODIFIED 与既有测试连带更新

- [x] 3.1 更新 `TestADirectoryIsListed` 中「条目字段恰为 `[name type]`」的断言（`browse_test.go` 中该用例尾部按字段名排序比对的那段）：改为断言文件条目的字段恰为 `[modified_at name size type]`，并同时断言响应中不含该文件的任何内容——这是新增 Scenario `A listed directory contains readable files` 的落点；验证：用例通过，且**不得**为迁就旧断言而保留 `name`/`type` 之外的旧形状
- [x] 3.2 新增断言：列表响应中不存在任何承载文件内容的字段（对照 `Entry metadata` 只承诺元信息）；验证：3.1 的用例同时覆盖
- [x] 3.3 确认 Change 02 的其余 Scenario 用例不受影响：条目排序、上级路径、越界拒绝、四种错误标识、越界符号链接按字面提供；验证：`go test ./...` 全绿，尤其 `TestAPathInsideTheRootTraversesASymbolicLinkOutward` 未被改动

## 4. 前端呈现（design D8）

- [x] 4.1 `web/index.html` 的条目列表结构从「名称单列」调整为名称、大小、修改时间三列；验证：浏览器打开服务地址能看到三列
- [x] 4.2 `web/app.js` 渲染每个条目的三个字段，**全部经 `textContent` 写入**，禁止用模板字符串拼接 `innerHTML`（Change 02 的 D8 不变量在节点数翻倍后必须重新守住，design D8）；验证：手工构造名为 `<img src=x onerror=alert(1)>` 的文件与目录，确认三个字段均正确显示且无脚本执行
- [x] 4.3 大小渲染为人类可读形式（如 `1.2 MB`），修改时间渲染为本地时区可读形式；目录条目的大小一栏留空。**注意**：design D6 保证的是**取值**跨时区可复现，不是**显示字符串**可复现；验证：同一目录在不同 `TZ` 下取值相同（服务端已测），本地显示为本时区可读形式
- [x] 4.4 条目元信息缺失时（`size` 与 `modified_at` 都不存在，对应 design D7）该行正常显示名称且不显示大小与时间，页面不报错；验证：手工构造该场景或用 2.6 的 seam 造出响应后确认
- [x] 4.5 `web/style.css` 用 grid 对齐三列；验证：320px 视口下 `document.documentElement.scrollWidth === window.innerWidth`，长文件名不撑破布局

## 5. 文档与集成收尾

- [x] 5.1 更新 `docs/roadmap.md`：Change 地图第 03 行改为 🚧；Capability 地图中 `file-metadata` 改写为「不成立——文件元信息是 `directory-browsing` 的 ADDED Requirement」（design D2）；想法池补记「符号链接大小为链接自身长度」这一 D12 延伸；验证：roadmap 中不再有把 `file-metadata` 列为待建 capability 的表述
- [x] 5.2 跑 `go vet ./...` 与 `go test ./...`；验证：输出无 FAIL、无新增告警
- [x] 5.3 端到端手工验证并记录实际观察结果（Change 02 的 design D9 声明本项目零构建约束下不做前端自动化测试）：三列显示正确、`textContent` 不变量守住、缺元信息的行可降级显示、目录排序与 Change 02 一致；验证：逐项记录实际观察结果，**未实测的项留空不勾选**
- [x] 5.4 记录一项 archive 阶段的待办（**不在本 Change 执行**）：`openspec/specs/directory-browsing/spec.md` 的 Purpose 写着「不包含文件元信息与文件内容读取」，本 Change 使前半句失效，而 archive 只合并 Requirement、不重写 Purpose。`AGENTS.md` 禁止直接修改 `openspec/specs/`，因此这一句留到 archive 时处理。验证：本任务不改动 `openspec/specs/` 下任何文件，仅在 `docs/journal.md` 或 `docs/roadmap.md` 想法池中留下该待办记录

## 5.3 实际观察结果

验证方式：真实浏览器（agent-browser / Chromium）打开真实运行的服务，根目录指向一个临时夹具目录（内含普通文件、1.2MB 文件、子目录、指向根目录外 1MB 文件的符号链接、空目录、空格与超长文件名，以及名为 `<img src=x onerror=alert(1)>` 的文件与 `<img src=x onerror=alert(2)>` 的目录）。「缺元信息的行」用 `network route` 拦截 `/api/list` 注入一条无 `size`/`modified_at` 的响应构造，未改动任何产品代码。

| 观察项 | 实际结果 |
|---|---|
| 三列显示（4.1） | 1024px 视口下表头「名称 / 大小 / 修改时间」与条目行按 grid 对齐；`medium.bin`(1 300 000 B) 显示 `1.2 MB`，`small.txt`(11 B) 显示 `11 B` |
| 目录大小一栏留空（4.3） | `docs`、`empty-dir`、两个 `<img …>` 目录的大小单元格为空字符串 |
| 修改时间为本地时区可读形式（4.3） | `modified_at = 1758000000` 显示为 `2025/9/16 13:20:00`；本机为 UTC+8，同一 Unix 值 UTC 为 05:20——**显示是本地化的而取值不是**，与 design D6 的区分一致。若实现改成格式化字符串，此处会随 TZ 变，且 2.5 的服务端用例会红 |
| `textContent` 不变量（4.2） | `#entries` 内 `img` 元素数 = 0、`script` 元素数 = 0、`#entries.innerHTML` 中不含 `<img` 字面量；名为 `<img src=x onerror=alert(1)>` 的文件与 `…(2)>` 的目录都按字面文本显示，无 alert 弹出；点击 `…(2)>` 目录链接正常进入（`?path=%3Cimg%20src%3Dx%20onerror%3Dalert(2)%3E`） |
| 符号链接大小（4.3 / D4） | `linked-big.bin` 指向根目录外 1 048 576 B 的文件，列表显示 `85 B`（= 目标路径字符串的字节数），符合 Scenario `An entry is a symbolic link` |
| 缺元信息降级（4.4） | 注入响应中无 `size`/`modified_at` 的 `no-meta.txt` 与无任何元信息的 `dir-no-meta` 两行：名称照常显示，大小与时间单元格为空字符串；`agent-browser errors` 与 `console` 均无输出 |
| 目录排序与 Change 02 一致（5.3） | 目录条目全部排在文件条目之前；同组内按折叠名称（`<img …>`、`docs`、`empty-dir` / `full-meta.txt`、`no-meta.txt`）；`?path=mock` 的上级入口文案为「↑ 上级目录」 |
| 320px 不撑破布局（4.5） | `document.documentElement.scrollWidth === window.innerWidth === 320`；超长文件名换行为多行而非撑宽；时间字符串仍单行显示 |
| 导航未受影响 | 从根目录点目录链接进入子目录、后退回到上级，条目与元信息均正常渲染 |

未能实测、无勾选依据的项：无——上表每一行都对应一次实际观察。唯一的环境限制是 `TestNamesDifferingOnlyInLetterCaseAreNotInverted` 在 macOS 大小写不敏感文件系统上按 Change 02 既有惯例 skip（见 1.5）。
EOF

## verify 后的修订

verify 结论：0 CRITICAL / 0 WARNING / 3 SUGGESTION，全部可归档。三条 SUGGESTION 的处置如下。

- **Scenario 改名**（已改 delta spec）：`Entry metadata` 的 `The same directory is listed repeatedly` 与基线 `List ordering` 下的同名 Scenario 撞名。OpenSpec 按 Requirement 限定作用域、不违规，但归档后同一 spec 文件内会出现两个同名 Scenario，追溯时容易认错。改为 `Repeated listings return the same metadata`，tasks 2.7 的引用同步更新。**取舍**：Scenario 标题是给人读的追溯锚点，不是行为本身，因此改名不构成行为变化，不需要新的 Requirement。
- **零条目时隐藏列名行**（改前端）：`renderEntries` / `renderFailure` 按条目数隐藏 `.entries-header`。spec 不约束呈现（design D8 已声明），属实现侧决定；依据是「一个描述零行的表头没有意义」。沿用 Change 02 journal 观察 2 的做法，靠全局 `[hidden] { display: none !important }` 兜底，而不是新写一条 `display` 规则与 `hidden` 打架。
- **CSS 恢复显式类词汇**（改前端）：把链接的样式从标签选择器 `.entry a` 改回 Change 02 的 `.entry-link`，链接挂 `entry-name entry-link` 两个类。理由是 `entry-name` 现在同时挂在 `<a>` 与 `<span>` 上，靠 `span.` 前缀区分可点击性比原来的「一类一语义」隐晦。列宽对齐仍由 grid 模板承担，类只表达语义。
