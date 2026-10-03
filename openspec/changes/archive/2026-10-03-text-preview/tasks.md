# Tasks

## 1. 文本判定与错误标识（design D5、D6、D8、D10）

- [x] 1.1 实现文本判定：扩展名 ∈ design D6 的初始清单则判定为可读文本；扩展名取最后一个 `.` 之后的部分，比较大小写不敏感；以 `.` 开头的文件名视为**无扩展名**（design D6）；验证：`go build ./...` 通过，且 1.2 的用例覆盖大小写不敏感与点开头文件名两条边界
- [x] 1.2 补 `TestTextRecognition` 表驱动单测，逐条覆盖 design D6 清单里的每一类扩展名、`README.TXT`（大写扩展名）、`.gitignore`（点开头 → 无扩展名）、`Makefile`（无扩展名）、`logo.png`（图片）、`a.bin`（二进制数据）；验证：用例通过
- [x] 1.3 新增错误标识 `not_text`、`too_large`、`not_a_regular_file`，接入 `codeStatus`（`not_text`/`not_a_regular_file` 取 400，`too_large` 取 400）并确认既有四个标识的映射一字未改；验证：既有四种错误标识的测试不改代码即通过
- [x] 1.4 定义大小上限常量，值 1048576 字节（1 MiB，design D10）；验证：常量有注释说明出处与可调性
- [x] 1.5 确认内容读取的**全部存在性与类型判定都在 `Stat` 之后、打开文件之前完成**（design D10：避免打开 FIFO 永久阻塞）；验证：2.5 的 `not_a_regular_file` 用例用一个命名管道构造，且该用例在有限时间内返回而不是挂起

## 2. 内容端点及其测试（design D1、D2）

- [x] 2.1 注册 `GET /api/content`，接受 `path` 查询参数，返回 JSON `{path, content}` 两个字段；`path` 为该文件的规范化相对路径（复用 `parentOf` 同款规范化）；验证：`go build ./...` 通过
- [x] 2.2 **必须**复用 `browse.go` 的 `resolve()` 做越界判定，不得另写一份路径解析（design D2：端点语义解耦，但越界判定只有一份实现）；验证：2.4 与 2.6 的用例在不改动 `resolve()` 的前提下通过
- [x] 2.3 补 `TestATextFileIsRequested`：断言响应含规范化路径与该文件的完整内容；验证：用例通过
- [x] 2.4 补 `TestAPathInsideTheRootTraversesASymbolicLinkOutward`（Scenario `A path inside the root traverses a symbolic link outward`）：在根目录内造一个指向根目录外文件的符号链接，断言按该路径提供内容、不判越界；验证：用例通过。在无法创建符号链接的平台 `t.Skip` 并说明原因（沿用 Change 02 / 03 的 skip 惯例）
- [x] 2.5 补 `TestTheContentIsNotDecodableAsUTF8`（Scenario `The content is not decodable as UTF-8`）：写入含非法 UTF-8 字节的 `.txt`，断言返回**成功**响应且内容中该部分以替换字符（U+FFFD）呈现（design D8）；验证：用例通过
- [x] 2.6 补 `TestAFilePositionIsReopened` 与 `TestThePositionContainsRedundantSegments`（Scenario `A file position is reopened` / `The position contains redundant segments`）：断言重复请求同一位置返回相同内容与路径，且冗余片段按规范化结果处理；验证：两条用例通过
- [x] 2.7 补 `TestFailureCausesAreDistinguishableForContent`（Scenario 全覆盖）：逐个构造 `not_found`（路径不存在）、`not_a_directory`（路径指向目录）、`not_a_regular_file`（路径是命名管道）、`permission_denied`（chmod 000）、`outside_root`（`../` 越界）、`not_text`（`.png`）、`too_large`（超过 1 MiB 的 `.txt`），断言各自返回对应的机器可读错误标识；验证：七个子用例全部通过。`permission_denied` 若因以 root 运行而无法构造则 `t.Skip` 并说明——**并回答「这条 Scenario 要防的东西现在由谁负责」**（沿用 Change 02 的 journal 观察 4）
- [x] 2.8 补 `TestTheRequestedPathIsANotAFileOrDirectory` 的对照项：断言 `/api/content?path=<某个目录>` 返回 `not_a_directory`；验证：2.7 的子用例已覆盖，本任务只需确认
- [x] 2.9 确认 `directory-browsing` 零 delta：`openspec/specs/directory-browsing/spec.md` 的八条 Requirement **一字未改**，既有测试文件无一处因本 Change 修改；验证：`go test ./...` 全绿且 `git diff` 中 `internal/server/browse_test.go` 的既有测试函数无改动行

## 3. 前端呈现（design D2、D8、D11）

- [x] 3.1 `web/app.js` 把**所有**文件条目渲染为可点击链接（当前是 `<span>`），指向同一套 `urlFor()` 生成的地址（design D2：位置复用 `?path=`，不引入第二个参数）；验证：手工点击任一文件条目，地址栏变为该文件的 `?path=`
- [x] 3.2 新增「当前 `?path=` 指向文件」的渲染分支：先请求 `/api/list?path=<当前值>`，收到 `not_a_directory` 后转请求 `/api/content?path=<同一值>`，成功则渲染内容、失败则按错误标识分派提示（design D2：前端不预判 `?path=` 指向什么，按响应成败分派）；验证：直接访问 `/?path=a/b.txt` 的深链接能出内容
- [x] 3.3 预览内容用 `<pre>` 承载，**经 `textContent` 写入，绝不拼接 `innerHTML`**（design D11：Change 02 的 D8 不变量扩展到文件内容本身）；验证：手工构造内容为 `<img src=x onerror=alert(1)>` 与 `<script>alert(2)</script>` 的 `.txt` 文件，确认按字面文本显示、无脚本执行、`#preview` 内 `img`/`script` 元素数均为 0
- [x] 3.4 错误提示按机器可读错误标识分支，`not_text` 显示「这是二进制文件，无法以文本预览」，`too_large` 显示文件过大（design D8）；验证：逐个触发 `not_text` 与 `too_large` 确认文案
- [x] 3.5 预览视图提供返回上级的入口（指向该文件所在目录），且浏览器前进/后退在目录视图与文件视图之间来回可用（design D2：位置镜像到地址栏不变量）；验证：从文件预览点返回上级回到目录；后退回文件视图且内容重新加载
- [x] 3.6 `web/index.html`、`web/style.css` 增加预览区结构与样式；验证：320px 视口下 `document.documentElement.scrollWidth === window.innerWidth`，长行内容横向滚动而不撑破页面布局

## 4. 文档与集成收尾

- [x] 4.1 更新 `docs/roadmap.md`：Change 地图第 04 行改为 🚧；Capability 地图补记 `text-preview`（Change 04）；想法池补记两条——「Change 05 的内容嗅探方案已在 Change 04 的 design D7 定下，实现前先读它，勿重开调研」与「GBK 等遗留编码按 UTF-8 处理，中文乱码已登记为已知取舍」；验证：roadmap 中不再有把 `text-preview` 列为待建的表述，且想法池含上述两条
- [x] 4.2 更新 `README.md` 的「当前状态」：说明已可点击文件条目查看文本内容、判定规则为扩展名白名单、非文本与超限文件的提示文案、以及已知不做编码检测；验证：README 中「尚未实现」一段不再包含「文件内容预览」
- [x] 4.3 跑 `go vet ./...` 与 `go test ./...`；验证：输出无 FAIL、无新增告警
- [x] 4.4 端到端手工验证并记录**实际观察结果**（Change 02 的 design D9 声明本项目零构建约束下不做前端自动化测试）：深链接可用、返回上级可用、前进后退可用、`textContent` 不变量守住、`not_text` 与 `too_large` 提示正确、**以及本 Change 最确定会发生的那次失败——`Makefile` 点进去报 `not_text`**（design D6，这正是 Change 05 的立项理由）；验证：逐项记录实际观察结果，**未实测的项留空不勾选**

### 4.4 手工验证记录

环境：macOS / Chromium（agent-browser 驱动）。fixture 目录含 `notes.txt`、`docs/api/spec.md`、`long.txt`（400 字符长行）、`xss.txt`（内容为 `<img src=x onerror=alert(1)>` 与 `<script>alert(2)</script>`）、`<img src=x onerror=alert(1)>.txt` 与 `<script>alert(2).txt`（恶意**文件名**）、`logo.png`、`Makefile`、`big.txt`（1048577 字节）、`utf8broken.txt`（含 `0xFF 0xFE`）、`pipe.txt`（命名管道）、`locked.txt`（`0o000`）、`sub/`。服务端以 `server.NewHandler` + 内嵌前端在 `127.0.0.1:18099` 起（8080 已被占用，用临时宿主程序起同一份 handler，验证后已删除）。

| 验证项 | 实际观察结果 |
|---|---|
| 深链接可用 | 直接导航到 `/?path=docs/api/spec.md`：`#preview` 的 `textContent` 为 `# 标题\n\n正文一段。\n`，位置显示 `<fixture>/docs/api/spec.md`，条目区与列名行均隐藏，错误区隐藏 ✓ |
| 返回上级可用 | 文件视图下 `#parent-link` 可见（计算样式 `display: inline-block`）、`href="/?path=docs%2Fapi"`；点击后 URL 变为 `?path=docs%2Fapi`、`#preview.hidden === true`、条目数回到 1、位置显示 `<fixture>/docs/api` ✓。根目录下的 `notes.txt` 的上级入口为 `href="/"`，仍然可见（文件一定有所在目录）✓ |
| 前进/后退可用 | 历史序列 `/` → `?path=docs` → `?path=docs%2Fapi` → `?path=docs%2Fapi%2Fspec.md`；后退两次依次回到两个目录视图（`#preview.hidden === true`、条目重新拉取），前进两次回到文件视图且内容重新加载为 `# 标题`，全程无整页重载 ✓ |
| `textContent` 不变量 | `xss.txt` 的两行 payload 按字面文本显示（截图确认为等宽字体下的原文）；`#preview.children.length === 0`、`#preview` 内 `img`/`script` 元素数均为 0，全文档 `img` 数为 0、`script` 数为 1（即页面自身的 `/app.js`）；**关闭 agent-browser 的自动 dismiss 后 `dialog status` 为「无对话框」**，因此没有 alert 执行 ✓。文件名 `<img src=x onerror=alert(1)>.txt` 与 `<script>alert(2).txt` 在列表中同样按纯文本显示且可点 ✓ |
| `not_text` 提示 | `?path=Makefile` → 「这是二进制文件，无法以文本预览」；`?path=logo.png` 同一句。**这就是 design D6 预期的那次失败**：仓库自己的 `Makefile` 点不进去，Change 05 的立项理由在真实使用中兑现 ✓ |
| `too_large` 提示 | `?path=big.txt`（1048577 字节）→「文件过大，无法以文本预览」，未给截断内容 ✓ |
| 其余失败原因 | `?path=pipe.txt`（命名管道）→「该位置不是普通文件，无法预览」且**立即返回、无挂起**；`?path=locked.txt`（`0o000`）→「没有读取该位置的权限」；`?path=../host/main.go` → 「该位置超出浏览范围」且条目数为 0；`?path=sub`（目录）→ 正常列出而不是走内容端点 ✓ |
| 非法 UTF-8 | `?path=utf8broken.txt` 返回 200，内容为「前半段�后半段」——`0xFF 0xFE` 恰好渲染成一个 U+FFFD（design D9）✓ |
| 冗余片段 | `?path=./docs//api/../api/spec.md` 正常出内容（前端把原样参数发给服务端，规范化在服务端完成）✓ |
| 窄视口 | 视口 320px 下 `document.documentElement.scrollWidth === window.innerWidth === 320`（文件视图与目录视图各测一次）；`#preview` 的 `clientWidth` 为 294 而 `scrollWidth` 为 3387，`scrollLeft` 可被设置 → 横向滚动发生在预览区内部，页面布局不被撑破；目录视图 320px 下三列全部保留、长文件名换行 ✓ |
| 文件条目可点 | 根目录 13 个条目的名称元素全部为 `A`（含 `Makefile`、`logo.png`、`pipe.txt`），颜色 `rgb(11, 98, 208)`，与目录条目同一套链接规则 ✓ |

一处环境备注：本机无法创建文件名含 `</` 的文件（`open()` 报 ENOENT，疑似端点安全过滤），因此 `<script>alert(2)</script>.txt` 退化为 `<script>alert(2).txt`。断言不依赖这个差异——`#preview` 内元素数与 `dialog status` 才是判据。

- [x] 4.5 记录一项 archive 阶段的待办（**不在本 Change 执行**）：`openspec/changes/text-preview/specs/text-preview/spec.md` 的 Purpose 写着「不包含文件类型识别能力」，本 Change 使这半句成立，但 **Change 05 `improve-text-file-detection` 归档后它就会变假**——而 archive 只合并 Requirement、不重写 Purpose。按 Change 03 的 journal 观察 6，届时把冲突摆出来由用户授权改那一句，不预先代改。验证：本任务不改动任何 `openspec/specs/` 下的文件，仅在 `docs/roadmap.md` 想法池中留下该待办记录
