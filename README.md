# http-file-browser

零配置的 HTTP 文件浏览器：指定一个目录启动本地服务，在浏览器中像 macOS Finder 一样浏览文件系统。

本项目同时是 OpenSpec SDD（Spec-Driven Development）工作流的学习实验项目：所有行为变化都经由 Explore → Propose → Apply → Verify → Archive 演进，`openspec/` 目录记录系统当前行为与完整演进史。

## 当前状态

✅ Change 02 `directory-browsing` 已归档：浏览器中可按 Finder 风格浏览命令行指定的根目录——进入子目录、返回上级、看到当前位置，浏览位置镜像到地址栏（刷新停留在原目录、前进/后退可用、当前目录可作深链接）。列表顺序为「目录在前 + 名称不区分大小写 + 原名 tiebreak」。服务不提供根目录之外的内容。行为规范见 [openspec/specs/directory-browsing/](openspec/specs/directory-browsing/) 与 [openspec/specs/service-startup/](openspec/specs/service-startup/)。

✅ Change 03 已归档：列表的每个条目除名称与类型外还给出大小与最后修改时间，三列对齐显示，大小为人类可读形式、时间为本地时区形式。符号链接的大小是链接自身的长度而非目标大小（元数据检测不解析链接，与按物理位置越界判定的策略一致）；目录不给出大小；单个条目元信息取不到时该行保留名称、其余留空。列表不提供文件内容。

✅ Change 04 `text-preview` 已归档：文件条目可以点开，以纯文本查看其内容，位置仍然镜像到地址栏（`/?path=docs/notes.txt` 可作深链接、刷新停在原位、前进后退在目录与文件之间可用）。行为规范见 [openspec/specs/text-preview/](openspec/specs/text-preview/)。

✅ Change 05 `improve-text-file-detection` 已归档：文本判定放宽为两级——扩展名在白名单内的照旧按名字判定；**没有扩展名的文件改为按内容起始 4096 字节判定**，不含二进制数据字节即可预览（`Makefile`、`LICENSE`、`.gitignore`、`go.mod` 由此点得进去）。白名单同时补充 `mod`/`sum`/`work`。带扩展名但不在白名单的文件不嗅探内容，`logo.png` 装纯文本仍拒绝。行为规范见 [openspec/specs/text-preview/](openspec/specs/text-preview/)。

✅ Change 06 `add-syntax-highlighting` 已归档：代码文件以语法高亮呈现（vendored highlight.js v11.12.0，随二进制内嵌自包含提供）——扩展名命中语言别名表按名高亮，无映射时内容自动检测，识别不出或低置信一律回退普通文本。高亮只是着色：无论哪条呈现路径，预览取出的文本与文件内容逐字符一致，内容里形似 HTML 的文字以字面形式呈现、绝不被执行。行为规范见 [openspec/specs/syntax-highlighting/](openspec/specs/syntax-highlighting/)。

✅ Change 07 `add-markdown-preview` 已归档：Markdown 文件（`md`/`markdown`）以渲染形式呈现，不再以高亮源码呈现（vendored markdown-it v14.3.2，内嵌自包含）——形似 HTML 的标记文本以字面呈现、绝不被解释执行（`html: false` 配置封闭，无消毒组件）；渲染视图内相对链接在应用内导航（上级目录片段与 `/` 开头的根相对路径以被查看文件所在目录为基准解析），外部链接新标签打开、当前呈现不变，图片按浏览器语义加载；代码块复用 vendored highlight.js 按语言高亮，识别不出回退素文本且文本与源文件逐字符一致；渲染未成功整段回退普通文本且不报错。行为规范见 [openspec/specs/markdown-preview/](openspec/specs/markdown-preview/)。

✅ Change 08 `improve-markdown-preview` 已归档：Markdown 文件的呈现形式可切换（「查看源码 / 查看渲染」按钮，默认渲染）——源码形式复用既有语法高亮管线呈现源码，切换只影响当前查看（离开后再次打开同一文件回到渲染形式，不写 URL/History、不可分享）；源码形式下内容字符序列与文件逐字符一致。渲染形式的既有行为（字面 HTML、相对链接导航、图片、代码块高亮、渲染失败回退素文本）全部挂上「以渲染形式呈现时」条件、行为不变。行为规范见 [openspec/specs/markdown-preview/](openspec/specs/markdown-preview/)。

✅ Change 09 `add-image-preview` 已归档：图片文件（png/jpg/jpeg/gif/webp/bmp/ico/avif）点开即以图片呈现——服务端按扩展名识别图片、经 `/api/image` 以流式字节响应提供内容（无大小上限），由浏览器解码呈现；识别只看名字、不读内容也不验魔数，装着图片字节的无扩展名或非图片扩展名文件照样拒绝（`not_an_image`）。图片无法加载时文件视图给一句中性回退说明，不留静默破图、不误报为服务错误。`/api/` 分区为图片字节开唯一的口：该端点的错误响应仍一律 JSON 信封。SVG 刻意不做，记入想法池。行为规范见 [openspec/specs/image-preview/](openspec/specs/image-preview/) 与 [openspec/specs/service-startup/](openspec/specs/service-startup/)。

✅ Change 17 `improve-root-confinement` 已归档：越界判定由**字面路径**改为**物理路径**——请求路径解析其全部符号链接后，物理位置落在根目录物理位置之内才提供内容；根内经软链指向根外的路径一律拒绝（`outside_root`），无任何开关（此前「按字面路径放行出根软链」的承诺被正式翻案，为远程访问 Change 扫清安全前置）。根内软链仍可用；根目录路径自身经软链（如 macOS `/tmp` → `/private/tmp`）零回归；悬空软链维持 `not_found`。涉及 directory-browsing / text-preview / image-preview 三个 capability，行为规范见 [openspec/specs/directory-browsing/](openspec/specs/directory-browsing/)。

✅ Change 10 `add-file-search` 已归档：在当前目录子树内按文件名递归搜索（子串 + 大小写折叠 + 空查询匹配一切），结果行显示名称、所在目录与类型，URL 镜像为 `/?path=<base>&q=<query>`（前进后退在列表与结果视图间切换、重开重现）。服务端瘦命中条目仅 `name`/`type`/`path`，不触碰目录列表「不携带内容」的门；越界判定复用物理路径基线，零新错误码。符号链接条目自身参与匹配但其子树不展开（lstat 语义），无权限子目录跳过其余命中照常。行为规范见 [openspec/specs/search/](openspec/specs/search/)。

- **判定规则是「扩展名白名单 + 无扩展名嗅探内容」**（Change 05 起）：带已知文本扩展名的文件按名字直接判定；名称没有扩展名的文件按内容起始窗口判定，窗口内的空字节若全部只落偶数位或只落奇数位（UTF-16 特征）则豁免。
- 判定为非文本时给出「这是二进制文件，无法以文本预览」；超过 1 MiB 的文件给出「文件过大，无法以文本预览」（不静默截断）。路径是目录、越界、无权限等各有各的提示。
- 内容一律按 UTF-8 呈现，无法解码的字节显示为替换字符。**不做编码检测**：GBK 等遗留中文编码会显示为乱码，这是自觉的取舍（不检测比检测错更诚实）。

尚未实现：文件编辑、远程访问。整体进度见 [docs/roadmap.md](docs/roadmap.md)。

## 快速开始

需要 Go 1.27 或更高版本，除此之外无任何依赖（无 npm、无打包器）。

```bash
go build -o http-file-browser .
./http-file-browser <要浏览的目录>
```

启动成功时标准输出打印访问地址：

```
服务已就绪，访问 http://127.0.0.1:8080 浏览 /path/to/dir
```

在浏览器打开该地址即可看到该目录的条目列表。

几点说明：

- 目录参数是必需的。缺失、路径不存在或路径不是目录时，程序向标准错误输出中文原因并以非零状态退出，不会启动服务。
- 服务默认只监听 `127.0.0.1:8080`，本机之外的主机无法连接。端口被占用时直接报错退出，不会自动顺延。
- 前端页面与静态资源内嵌在二进制中，产物可以单独拷走运行，不需要随附资源目录。
- 文件条目可点开查看文本内容；不是文本的文件点进去会得到一句明确的提示，而不是一个点不动的死条目。内容通过 `?path=` 定位，因此「当前打开的文件」同样可以分享成深链接。
- 浏览位置用相对根目录的路径表示（形如 `/?path=docs/api`），因此深链接可以跨机器复用：同一个 `?path=docs` 在以另一个根目录启动的服务上照样能打开。


## 文档

| 位置 | 内容 |
|---|---|
| [docs/roadmap.md](docs/roadmap.md) | 长期规划与 Change 地图（活文档） |
| [docs/journal.md](docs/journal.md) | SDD 学习实验观察记录 |
| [docs/adr/](docs/adr/) | 架构决策记录（ADR） |
| [docs/OpenSpec_HTTP_File_Browser.md](docs/OpenSpec_HTTP_File_Browser.md) | OpenSpec 实战教材（静态参考） |
| [openspec/specs/](openspec/specs/) | 系统当前行为规范 |
| [openspec/changes/archive/](openspec/changes/archive/) | 系统演进史（Change 归档） |
