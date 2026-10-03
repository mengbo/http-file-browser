---
created: "2026-10-03"
description: ""
---
# OpenSpec HTTP File Browser 实战手册

> 本文是一份面向 **OpenCode + OpenSpec** 的实战路线。目标不是只把一个文件浏览器做出来，而是通过这个项目完整走通 OpenSpec 的 SDD（Spec-Driven Development）流程，并刻意演练一次需求变更、回溯、重新规划和归档。
>
> OpenSpec 当前以 **OPSX** 为标准工作流。核心动作包括 `explore`、`propose`、`apply`、`update`、`sync`、`archive`；扩展工作流还包括 `new`、`continue`、`ff`、`verify`、`bulk-archive`、`onboard`。OpenCode 已被官方支持，初始化时会安装对应的 skills/commands。  
> 官方文档：OpenSpec Documentation、CLI、OPSX、spec-driven schema。

---

## 1. 这个项目为什么非常适合练 OpenSpec

准备做的项目可以暂时叫：

**`http-file-browser`**

它是一个零配置 HTTP 文件浏览器。

从命令行指定一个目录：

```bash
http-file-browser /Users/mengbo/Documents
```

程序启动一个本地 HTTP 服务，浏览器访问：

```text
http://127.0.0.1:8080
```

看到类似 macOS Finder 的文件浏览界面。

但它不是简单的目录列表，而是逐渐形成一个轻量级的 Web 文件工作台：

```text
第一阶段
目录浏览
文件/目录图标
进入目录
返回上级
文件基本信息
文本文件查看

第二阶段
代码高亮
Markdown 渲染
图片预览
二进制文件提示
搜索/过滤

第三阶段
文件编辑
保存
冲突处理
大文件限制
只读/可写模式

第四阶段
远程访问
监听非 localhost
启动时生成随机 token
Token 认证
安全边界
```

这个项目特别适合 OpenSpec，原因是它天然存在很多**行为需求**，而这些行为又会随着实现逐渐发生变化。

例如：

最初可能规定：

> 用户访问目录时，系统显示该目录下的文件和子目录。

做完以后你发现：

> 隐藏文件也应该显示。

再做以后又发现：

> 隐藏文件不能默认显示，而应该提供开关。

随后又发现：

> Finder 风格排序应该区分目录和文件。

再后来远程访问加入以后：

> 本地访问不需要 token，远程访问必须 token。

这些都是典型的**Spec 发生变化**，而不是简单修改代码。

所以这个项目不是为了练“怎么用几个 OpenSpec 命令”，而是用真实开发过程验证：

> **Spec 是当前系统行为的正式描述；Change 是一次行为变化的生命周期；Archive 是把变化合并回当前系统事实；Archive History 则保留系统为什么变成今天这样的历史。**

---

# 2. 先理解 OpenSpec 的真正模型

不要把 OpenSpec 理解成：

```text
需求文档
    ↓
AI 写代码
```

更准确的模型是：

```text
                ┌──────────────┐
                │  当前系统 Spec │
                │ openspec/specs │
                └──────┬───────┘
                       │
                       │ 当前系统是什么
                       ↓
              ┌─────────────────┐
              │    Explore      │
              │ 理解问题/调查代码 │
              └────────┬────────┘
                       ↓
              ┌─────────────────┐
              │     Propose     │
              │ 定义一次变化      │
              └────────┬────────┘
                       ↓
             proposal / specs / design
                       ↓
                    tasks
                       ↓
              ┌─────────────────┐
              │      Apply      │
              │ 根据任务实现      │
              └────────┬────────┘
                       ↓
                   Verify
                       ↓
              ┌─────────────────┐
              │     Archive     │
              │ Delta → Main Spec│
              └────────┬────────┘
                       ↓
             openspec/specs/
             成为新的系统事实
```

这里有三个层次必须分清。

### 2.1 `openspec/specs/`

这是：

> **当前系统应该具有什么行为。**

它不是开发计划，也不是历史记录。

例如：

```text
openspec/specs/
├── directory-browsing/
│   └── spec.md
├── text-preview/
│   └── spec.md
├── syntax-highlighting/
│   └── spec.md
└── markdown-preview/
    └── spec.md
```

如果一个 Change 已经 Archive，那么它描述的行为应该已经进入这里。

---

### 2.2 `openspec/changes/`

这里是：

> **正在发生的变化。**

例如：

```text
openspec/changes/
└── add-markdown-preview/
    ├── .openspec.yaml
    ├── proposal.md
    ├── specs/
    │   └── markdown-preview/
    │       └── spec.md
    ├── design.md
    └── tasks.md
```

它描述的不是“系统现在是什么”，而是：

> 从现在到目标状态，要发生什么变化。

其中 `specs/.../spec.md` 是 **delta spec**。

---

### 2.3 `openspec/changes/archive/`

这里是：

> **历史上发生过哪些变化。**

例如：

```text
openspec/changes/archive/
├── 2026-10-03-add-directory-browsing/
├── 2026-10-03-add-text-preview/
├── 2026-10-04-add-syntax-highlighting/
└── 2026-10-05-add-markdown-preview/
```

Archive 并不是删除 Change。

它保留：

```text
为什么做
↓
当时定义了什么
↓
当时怎么设计
↓
当时做了哪些任务
```

因此最终：

```text
specs/
```

回答：

> 系统现在是什么？

而：

```text
changes/archive/
```

回答：

> 系统是怎么一步一步变成现在这样的？

这正是本项目要重点练习的地方。

---

# 3. OpenSpec 当前命令：两个世界

这是最容易混淆的地方。

OpenSpec 有两类命令。

## 3.1 Terminal 命令

在终端执行：

```bash
openspec init
openspec list
openspec list --specs
openspec status
openspec show
openspec view
openspec validate
openspec archive
openspec new
openspec update
openspec config
```

这些是 OpenSpec CLI。

---

## 3.2 AI Chat 命令

在 OpenCode 的聊天窗口输入：

```text
/opsx:explore
/opsx:propose
/opsx:apply
/opsx:update
/opsx:sync
/opsx:archive
```

但不同 AI 工具的命令呈现方式可能不同。

OpenSpec 官方当前针对 OpenCode 安装的是：

```text
.opencode/
├── skills/
│   └── openspec-*/
└── commands/
    └── opsx-*.md
```

因此初始化以后，应以 OpenCode 中实际出现的命令为准。

如果 OpenCode 中显示为：

```text
/opsx-propose
/opsx-apply
```

就使用这个形式。

不要自己假设命令格式。

核心思想不变：

```text
openspec ...       → 终端 CLI

/opsx-...          → OpenCode AI 工作流
```

---

# 4. 第一件事：建立项目

建议先建立一个非常干净的 Git 仓库。

例如：

```bash
mkdir http-file-browser
cd http-file-browser

git init
```

然后进入 OpenCode。

第一步不要让 AI 立即写代码。

先安装并初始化 OpenSpec。

```bash
npm install -g @fission-ai/openspec@latest
```

或者 macOS：

```bash
brew install openspec
```

然后：

```bash
openspec init
```

初始化完成以后检查：

```bash
openspec list
openspec list --specs
openspec view
openspec status
```

初始状态应该类似：

```text
openspec/
├── config.yaml
├── specs/
└── changes/
    └── archive/
```

以及 OpenCode 对应的：

```text
.opencode/
```

相关 workflow 文件。

---

# 5. 第一个重要动作：不要直接 Propose，先 Explore

虽然你已经知道要做什么，但这个项目是专门用来学习 OpenSpec 的。

所以第一轮建议刻意使用：

```text
/opsx:explore
```

而不是：

```text
/opsx:propose
```

在 OpenCode 中输入：

```text
/opsx:explore

我准备开发一个名为 http-file-browser 的零配置 HTTP 文件浏览器。

目标是通过命令行指定一个目录，然后启动一个 Web 服务，让用户在浏览器中像使用 macOS Finder 一样浏览目录和文件。

它最终需要支持：
- 目录浏览
- 文件信息
- 文本文件查看
- 程序代码语法高亮
- Markdown 渲染
- 后续文件编辑
- 后续远程访问
- 远程访问通过程序启动时输出的 token 认证

这次探索的目标不是写代码，也不是立即创建完整 Change。

请先：
1. 调查当前项目目录；
2. 分析这个产品的核心用户行为；
3. 把明显属于不同 capability 的功能拆开；
4. 区分 MVP 与后续能力；
5. 找出现在必须决定的架构问题，以及可以推迟的问题；
6. 特别指出哪些需求将来最容易发生变化；
7. 不要写实现代码；
8. 不要为了“完整”而提前设计所有未来功能。

最后给出一个适合使用 OpenSpec 分阶段实现的 Change 序列。
```

这里要观察 AI 的行为。

**Explore 的价值不是让 AI 给你一个漂亮方案。**

真正的价值是：

> 在正式形成 Change 之前，把问题空间搞清楚。

OpenSpec 官方也明确把 Explore 定义成一个“no-stakes thinking”阶段：调查代码、比较方案、澄清问题，不直接进入实现。 

---

# 6. 这个项目建议采用的 capability 划分

第一版不要按代码模块划分。

例如不要：

```text
frontend
backend
filesystem
http
```

这些是实现结构。

OpenSpec 的 capability 应该尽量对应：

> **用户能够感知的稳定系统行为。**

我建议初期使用：

```text
directory-browsing
file-metadata
text-preview
syntax-highlighting
markdown-preview
image-preview
search
file-editing
remote-access
authentication
```

但注意：

**不是一开始全部建立。**

只是先形成产品地图。

---

# 7. 建议的开发 Change 序列

推荐整个项目大致按下面顺序推进。

## Change 01：最小 HTTP 服务

```text
bootstrap-http-server
```

目标：

```bash
http-file-browser /tmp
```

能够启动 HTTP 服务并返回最基本页面。

这一 Change 的意义：

> 建立项目骨架。

这里可以练：

```text
proposal
spec
design
tasks
apply
archive
```

---

## Change 02：目录浏览

```text
directory-browsing
```

实现：

```text
/
├── Documents
├── Downloads
├── README.md
└── test.txt
```

支持：

- 当前目录显示
- 子目录进入
- 上级目录
- 文件/目录区分
- 基本排序
- 路径显示

这是整个项目第一个真正的核心 capability。

---

## Change 03：文件元信息

```text
file-metadata
```

例如：

```text
README.md
12.4 KB
2026-10-03 09:21
Markdown
```

这里练习：

> 一个 capability 是否应该继续拆成多个 requirement？

---

## Change 04：文本文件预览

```text
text-preview
```

支持：

```text
.txt
.log
.json
.yaml
.yml
.toml
.xml
.csv
```

最关键的不是列扩展名，而是定义：

> 系统如何判断一个文件可以作为文本打开。

这里可以产生第一个非常好的需求变更练习。

最初：

> 通过扩展名判断。

后来发现：

> `.conf`、无扩展名文件也可能是文本。

然后改变为：

> 基于 MIME、扩展名和内容检测综合判断。

这就是一个非常典型的：

```text
MODIFIED Requirement
```

---

# 8. 第一个刻意设计的“需求变更实验”

建议你在 `text-preview` 完成并 Archive 以后，故意提出一个需求：

> 无扩展名的 UTF-8 文本文件也应该可以预览。

此时不要直接修改：

```text
openspec/specs/text-preview/spec.md
```

而是创建一个新的 Change：

```text
improve-text-file-detection
```

然后：

```text
/opsx:propose improve-text-file-detection
```

给 AI：

```text
现在项目已经完成 text-preview，并且已经 archive。

新的需求是：
原来的文本文件判断主要依赖扩展名，但实际文件系统中存在大量没有扩展名的 UTF-8 文本文件，例如配置文件、脚本文件和日志文件。

希望修改已有的 text-preview capability：
- 无扩展名但内容明确是 UTF-8 文本的文件也可以预览；
- 二进制文件仍然不能被当作文本显示；
- 不要改变已有的代码文件和 Markdown 文件行为。

请先读取当前 openspec/specs/text-preview/spec.md。
不要直接修改 main spec。
创建一个 Change，使用 MODIFIED Requirements 描述这个需求变化。
```

这一步非常重要。

因为它体现：

> Change 不是“新功能文档”。

它可以是：

```text
已有能力的行为修改
```

---

# 9. MODIFIED Requirement 是 OpenSpec 的关键机制

假设原来的主 Spec：

```markdown
### Requirement: Text file detection

The system SHALL determine whether a file is suitable for text preview based on its known text file types.

#### Scenario: Known text file

- **WHEN** the user opens a known text file
- **THEN** the system displays the file as text
```

新的 delta 不应该只写：

```markdown
## MODIFIED Requirements

### Requirement: Text file detection

Also support extensionless files.
```

这是错误的思路。

OpenSpec 当前要求：

> MODIFIED 必须把原 Requirement 整个复制出来，然后完整修改。

正确形式类似：

```markdown
## MODIFIED Requirements

### Requirement: Text file detection

The system SHALL determine whether a file is suitable for text preview using file type information and content-based text detection.

#### Scenario: Known text file

- **WHEN** the user opens a known text file
- **THEN** the system displays the file as text

#### Scenario: Extensionless UTF-8 text file

- **WHEN** the user opens an extensionless file whose content is valid UTF-8 text
- **THEN** the system allows the file to be previewed as text

#### Scenario: Binary file

- **WHEN** the user opens a binary file
- **THEN** the system does not render the file as text
```

原因很简单：

Archive 最终需要用这个 delta 去替换 main spec 中原来的 Requirement。

所以：

```text
MODIFIED
```

实际上是：

```text
旧 Requirement
      ↓
完整复制
      ↓
修改
      ↓
替换
```

而不是：

```text
补充说明
```

---

# 10. ADDED / MODIFIED / REMOVED / RENAMED

OpenSpec 的 delta spec 主要有四种变化。

## ADDED

新增行为。

例如：

```markdown
## ADDED Requirements

### Requirement: Markdown rendering

The system SHALL render Markdown files as formatted HTML.

#### Scenario: Open Markdown file

- **WHEN** the user opens a Markdown file
- **THEN** the system displays rendered Markdown content
```

---

## MODIFIED

已有 Requirement 的行为发生变化。

```text
旧行为
↓
修改
↓
新行为
```

必须保留完整 Requirement。

---

## REMOVED

删除已有行为。

而且需要：

```text
Reason
Migration
```

例如：

```markdown
## REMOVED Requirements

### Requirement: Plain HTML preview

**Reason**: Replaced by the unified Markdown rendering pipeline.

**Migration**: Markdown files are now rendered through the Markdown preview capability.
```

---

## RENAMED

只是名称改变。

例如：

```markdown
## RENAMED Requirements

- FROM: `Text file detection`
- TO: `Text content detection`
```

它和 MODIFIED 不一样。

如果行为没有改变，只是 Requirement 名称变化，就应该使用 RENAMED。

---

# 11. Spec 的写法：不要写成设计文档

这是你整个实践中最值得注意的地方。

不要：

```markdown
系统采用 React + Go + WebSocket。
```

这不是行为 Spec。

也不要：

```markdown
服务端使用 Gin，前端使用 Monaco Editor。
```

这是 Design。

Spec 应该描述：

> 用户做什么，系统应该怎样响应。

例如：

```markdown
### Requirement: Directory listing

The system SHALL display the entries of the current directory.

#### Scenario: Directory contains files and directories

- **WHEN** the user opens a directory
- **THEN** the system displays both files and subdirectories
```

设计放到：

```text
design.md
```

例如：

```text
后端采用 Go。
HTTP 使用标准库 net/http。
前端采用静态 HTML + CSS + JavaScript。
代码高亮使用 Prism。
Markdown 使用 marked。
```

当然，最终具体技术栈应该通过 Explore/Design 决定，而不是在本路线中预先强制规定。

---

# 12. 一个 Requirement 应该有多大？

不要写成：

```text
### Requirement: File browser

The system SHALL support browsing, previewing, editing, searching,
authentication, remote access, syntax highlighting and Markdown.
```

这会变成一个超级 Requirement。

应该拆成：

```text
Directory listing
Path navigation
File metadata
Text preview
Syntax highlighting
Markdown rendering
File editing
Remote access
Authentication
```

原则：

> 一个 Requirement 尽量表达一个独立的行为约束。

这样未来修改的时候，OpenSpec 才能准确地：

```text
MODIFIED
```

或者：

```text
ADDED
```

而不会每次都修改一大坨东西。

---

# 13. Scenario 是什么

Scenario 不是测试代码。

但：

> 一个好的 Scenario 应该可以直接转化为测试。

例如：

```markdown
#### Scenario: Navigate into a directory

- **WHEN** the user clicks a directory
- **THEN** the browser navigates to that directory
```

它可以对应：

```text
E2E test
```

但 Spec 本身并不要求你现在就写测试代码。

这是一个很重要的区别：

```text
Spec
  ↓
定义系统行为

Test
  ↓
验证系统是否满足行为
```

不要把：

```text
Spec = Test
```

也不要把：

```text
Test = Design
```

混在一起。

---

# 14. 第一阶段真正应该做什么

我建议 MVP 只做到：

```text
http-file-browser
│
├── 命令启动
│
├── 指定根目录
│
├── HTTP 服务
│
├── 目录浏览
│
├── 路径导航
│
├── 文件基本信息
│
├── 文本文件查看
│
├── Markdown 显示
│
└── 程序代码高亮
```

暂时不做：

```text
文件编辑
远程访问
Token
用户系统
上传
删除
复制
移动
压缩
全文搜索
实时文件监听
多用户
```

尤其不要一开始就做：

> “Finder Web 化”。

那会让第一个 Change 巨大到失去 OpenSpec 的意义。

---

# 15. 第二阶段：代码高亮

Change：

```text
add-syntax-highlighting
```

Capability：

```text
syntax-highlighting
```

需要定义：

```text
支持哪些语言？
如何识别语言？
未知语言怎么办？
超大文件怎么办？
高亮失败怎么办？
```

例如：

```markdown
### Requirement: Source code syntax highlighting

The system SHALL render recognized source code files with syntax highlighting.

#### Scenario: JavaScript file

- **WHEN** the user opens a JavaScript source file
- **THEN** the source is displayed with JavaScript syntax highlighting

#### Scenario: Unknown source type

- **WHEN** the system cannot determine a supported language
- **THEN** the source is displayed as plain text
```

注意：

> “支持 JavaScript”是行为。

而：

> “使用 Prism.js”是设计。

---

# 16. 第三阶段：Markdown

Change：

```text
add-markdown-preview
```

Capability：

```text
markdown-preview
```

可以定义：

```text
Markdown 正常渲染
代码块高亮
链接
图片
表格
目录
原始 Markdown 查看
```

但是第一版不要全部做。

建议：

```text
Markdown → HTML
代码块
基本链接
图片
```

先完成。

---

# 17. 一个很有价值的第二次需求变更

Markdown 做完后，故意提出：

> “Markdown 文件默认显示渲染结果，但用户应该能够查看原始 Markdown。”

这个变化非常适合 OpenSpec。

原始：

```text
打开 README.md
→ 渲染 Markdown
```

新行为：

```text
打开 README.md
→ 默认渲染
→ 可以切换 Source
→ 查看原始 Markdown
```

这是一个典型的：

```text
MODIFIED
```

而不是：

```text
ADDED markdown-source-view
```

因为它修改的是：

> Markdown preview 的已有行为。

---

# 18. 文件编辑应该单独做 Change

Change：

```text
add-file-editing
```

不要和 Markdown preview 放一起。

因为编辑会引入大量新的语义：

```text
读取
↓
编辑
↓
保存
↓
覆盖
↓
失败
↓
并发修改
↓
冲突
```

尤其是：

> 用户打开文件后，文件在磁盘上被其他程序修改怎么办？

这是一个真正的需求问题。

不要让 AI 自己决定。

应该 Explore：

```text
/opsx:explore

我们准备给 http-file-browser 增加文件编辑。

请重点分析保存语义：
1. 文件被外部程序修改怎么办？
2. 是否需要 mtime/size/hash 判断？
3. 保存失败怎么办？
4. 是否需要显式 Save？
5. 是否允许覆盖？
6. 是否需要只读模式？
7. 是否需要限制文件大小？
8. 哪些文件可以编辑？

不要写代码。
不要预设最终方案。
先列出需要产品决策的问题和可选方案。
```

然后根据讨论结果再：

```text
/opsx:propose add-file-editing
```

---

# 19. 远程访问必须独立成为 Change

Change：

```text
add-remote-access
```

这一 Change 不只是：

```text
listen 0.0.0.0
```

真正涉及：

```text
本地访问
远程访问
监听地址
Token
认证
HTTP Header / Query 参数
错误响应
Token 生命周期
日志
启动输出
```

因此应该先 Explore。

---

# 20. Token 认证的需求应该怎么写

假设最终确定：

```bash
http-file-browser /data --listen 0.0.0.0:8080
```

启动时：

```text
HTTP File Browser
Listening on 0.0.0.0:8080
Access token: 7f2...
```

浏览器：

```text
http://server:8080/?token=...
```

或者更合理地使用：

```http
Authorization: Bearer <token>
```

这里不要直接在 Spec 中决定 HTTP Header。

Spec 应该先写行为：

```markdown
### Requirement: Remote access authentication

The system SHALL require authentication for HTTP requests received through a non-loopback listening address.

#### Scenario: Remote request without authentication

- **WHEN** a client accesses the server without valid authentication
- **THEN** the server rejects the request

#### Scenario: Remote request with valid token

- **WHEN** a client provides the valid startup token
- **THEN** the server allows access to the requested resource

#### Scenario: Local loopback access

- **WHEN** a client accesses a server bound only to a loopback address
- **THEN** the server does not require the remote authentication mechanism
```

具体：

```text
Authorization: Bearer
```

还是：

```text
?token=
```

应该属于 Design。

---

# 21. 推荐的最终 Change 地图

建议你实际开发时大致形成：

```text
01 bootstrap-http-server
    └── 最小 HTTP 服务

02 directory-browsing
    └── Finder 风格目录浏览

03 file-metadata
    └── 文件信息

04 text-preview
    └── 文本文件查看

05 improve-text-file-detection
    └── 第一次需求变更
    └── MODIFIED

06 add-syntax-highlighting
    └── 代码高亮

07 add-markdown-preview
    └── Markdown 渲染

08 improve-markdown-preview
    └── Source / Preview
    └── 第二次需求变更

09 add-image-preview
    └── 图片预览

10 add-file-search
    └── 文件搜索

11 add-file-editing
    └── 编辑和保存

12 add-edit-conflict-detection
    └── 编辑过程中的外部修改

13 add-remote-access
    └── 监听地址

14 add-token-authentication
    └── Token 认证

15 improve-authentication
    └── 认证行为变化

16 polish-file-browser
    └── 最终体验优化
```

不要求全部做完。

前 5～8 个 Change 就已经足够把 OpenSpec 学明白。

---

# 22. 每一个 Change 的标准流程

以后你可以把这个流程当成固定习惯。

## 情况 A：问题还没想清楚

```text
/opsx:explore
```

然后：

```text
/opsx:propose
```

---

## 情况 B：需求已经非常清楚

直接：

```text
/opsx:propose add-something
```

---

## Propose 后

先不要 Apply。

检查：

```text
proposal.md
specs/
design.md
tasks.md
```

顺序建议：

```text
proposal
    ↓
spec
    ↓
design
    ↓
tasks
```

最重要的是：

> Spec 对不对？

其次：

> Design 对不对？

最后：

> Tasks 是否覆盖 Spec？

---

# 23. Proposal 应该检查什么

看到：

```markdown
## Why
```

问：

> 为什么必须做？

不是：

> AI 为什么觉得这个很酷？

---

看到：

```markdown
## What Changes
```

问：

> 这次 Change 到底改变什么？

---

看到：

```markdown
## Capabilities
```

问：

> 是新增 capability，还是修改已有 capability？

尤其注意：

```text
Modified Capabilities
```

是否错误地创建了一个新 capability。

例如已经有：

```text
text-preview
```

就不要新造：

```text
text-viewer
```

然后把原来的东西复制一份。

---

# 24. Spec Review 是整个流程最重要的一关

我建议你形成一个固定检查法。

看到：

```text
### Requirement:
```

先问：

> 这是用户能观察到的行为吗？

再问：

> 是否明确？

再问：

> 是否可以测试？

再看：

```text
#### Scenario:
```

问：

> 有没有正常情况？

> 有没有边界情况？

> 有没有失败情况？

例如目录浏览：

```text
正常目录
空目录
无权限目录
不存在目录
路径穿越
符号链接
```

不要一次把所有情况都塞进去。

当前 Change 需要什么，就写什么。

---

# 25. Design Review

Design 负责：

> 怎么实现。

例如：

```text
HTTP Server
    ↓
Router
    ↓
Path Resolver
    ↓
Filesystem Service
    ↓
Renderer
```

前端：

```text
Browser
    ↓
App
    ↓
Directory View
File View
Markdown View
Code View
```

这才属于：

```text
design.md
```

---

# 26. Tasks Review

Tasks 应该能够从 Spec 推导出来。

例如：

```markdown
- [ ] 1.1 Implement root directory configuration
- [ ] 1.2 Implement safe path resolution
- [ ] 1.3 Implement directory listing endpoint
- [ ] 1.4 Implement directory browser UI
- [ ] 1.5 Add tests for directory navigation
```

不要出现：

```text
- [ ] 1.1 Make code better
- [ ] 1.2 Improve UI
- [ ] 1.3 Finish backend
```

这种任务无法验证。

---

# 27. Apply

进入新会话。

这是官方推荐的实践：

> Planning 和 implementation 最好分开 context。

OpenCode 中：

```text
/opsx-apply
```

或者对应的 `/opsx:apply` 形式。

让 AI：

```text
按照当前 Change 的 tasks.md 开始实施。

严格以 proposal、specs、design 和 tasks 为依据。

逐项执行任务。
完成一项就更新 tasks.md。
遇到与当前 Spec 不一致的地方不要自行扩大需求。

如果发现计划有问题：
1. 停下来说明问题；
2. 不要偷偷改变需求；
3. 必要时先更新 Change artifacts，再继续实施。
```

这段提示词非常值得长期保留。

---

# 28. 为什么 Apply 不能成为“自由发挥”

AI 很容易这样：

```text
Task:
实现目录浏览

AI:
顺便加了搜索
顺便加了排序
顺便加了暗色模式
顺便加了上传
```

这正是 SDD 想避免的。

正确关系：

```text
Spec
  ↓
Tasks
  ↓
Implementation
```

而不是：

```text
AI觉得应该有什么
  ↓
Implementation
```

如果 Apply 过程中发现：

> “这个功能最好顺便做一下。”

应该：

```text
停止
↓
提出需求
↓
修改 Change
或者
创建新的 Change
```

---

# 29. 什么时候使用 `/opsx:update`

这是你这个实验最值得练习的命令之一。

假设：

```text
proposal
spec
design
tasks
```

已经写完。

Apply 做到一半发现：

> 原来的方案不合理。

不要直接修改代码然后假装没发生。

使用：

```text
/opsx:update
```

提示：

```text
在实施当前 Change 的过程中，我们发现：

原设计假设所有文本文件都可以一次性读入内存。
实际测试发现大文件会造成明显的内存压力。

请重新审查当前 Change：
1. proposal
2. specs
3. design
4. tasks

判断哪些内容需要修改。
不要直接写代码。
先更新 Change artifacts，使它们与新的设计一致。
同时保持已经完成的任务与新的计划一致。
```

这就是：

```text
Plan
  ↓
Apply
  ↓
发现问题
  ↓
Update
  ↓
继续 Apply
```

而不是：

```text
Plan
  ↓
Apply
  ↓
发现问题
  ↓
偷偷改代码
```

---

# 30. `sync` 与 `archive` 的区别

这个地方非常容易误解。

## sync

```text
/opsx:sync
```

作用：

> 把当前 Change 的 delta spec 合并到 `openspec/specs/`，但 Change 仍然保持 active。

适合：

> Change 已经产生了一部分明确的规格更新，但整体工作还没有结束。

---

## archive

```text
/opsx:archive
```

作用：

```text
Change 完成
    ↓
验证
    ↓
Delta Spec 合并到 main specs
    ↓
Change 移动到 archive
```

因此通常：

```text
apply
↓
verify
↓
archive
```

而不是：

```text
apply
↓
手动复制 spec
↓
删除 Change
```

---

# 31. Verify

如果你启用了扩展工作流：

```text
/opsx:verify
```

它用于检查：

> 当前实现是否真的符合 Change artifacts。

我建议你的项目一定启用它。

因为这个项目的学习目标不是：

> “AI 能不能写代码？”

而是：

> “Spec → Implementation 是否形成闭环？”

所以最好形成：

```text
Spec
 ↓
Tasks
 ↓
Code
 ↓
Verify
 ↓
Archive
```

---

# 32. Archive 前的最终检查

终端：

```bash
openspec validate
```

然后：

```bash
openspec status
```

确认：

```text
tasks.md
```

已经全部：

```text
[x]
```

再：

```text
openspec archive <change-name>
```

或者让 AI：

```text
/opsx:archive
```

Archive 后：

```bash
openspec list
openspec list --specs
openspec view
```

检查：

```text
changes/
```

是否已经清空当前 Change。

再看：

```text
specs/
```

是否已经变成新的事实。

---

# 33. 真正的“回溯实验”

这是本项目最重要的实验。

假设第一版已经完成：

```text
directory-browsing
text-preview
markdown-preview
```

全部 Archive。

现在：

```text
openspec/specs/
```

描述当前系统。

例如：

```text
text-preview/spec.md
```

已经规定：

```text
系统支持文本文件预览。
```

然后用户提出：

> Markdown 默认渲染，但我还需要查看原始 Markdown。

此时不要修改 main spec。

创建：

```text
improve-markdown-preview
```

流程：

```text
/opsx:explore
        ↓
/opsx:propose
        ↓
review
        ↓
/opsx:apply
        ↓
/opsx:verify
        ↓
/opsx:archive
```

最终：

```text
openspec/specs/markdown-preview/spec.md
```

变成新的事实。

而：

```text
openspec/changes/archive/
└── 2026-10-xx-improve-markdown-preview/
```

保留了：

```text
旧行为
+
为什么改变
+
新行为
+
如何实现
```

这就是完整回溯。

---

# 34. 更进一步：刻意做一次“错误需求”

为了真正理解 OpenSpec，建议你再做一次实验。

先提出：

> Markdown 页面应该默认显示原始 Markdown。

然后实现。

Archive。

接着又改变：

> 默认应该显示渲染后的 Markdown，原始内容作为 Source 模式。

这时候你就会看到：

```text
旧 Spec
    ↓
MODIFIED
    ↓
新 Spec
```

而 Archive history 会保存：

```text
第一次决定
    ↓
为什么改变
    ↓
第二次决定
```

这就是 OpenSpec 真正有价值的地方。

---

# 35. OpenSpec 的“回溯”应该如何理解

这里需要区分两个概念。

## 回溯一：实现尚未归档

例如：

```text
Change A
↓
Apply
↓
发现需求理解错了
```

此时：

```text
/opsx:update
```

修改：

```text
proposal
spec
design
tasks
```

然后继续：

```text
/opsx:apply
```

这属于：

> **当前 Change 内的计划修正。**

---

## 回溯二：Change 已经归档

例如：

```text
Change A
↓
Archive
↓
一个月后
↓
发现需求需要改变
```

这时不要回头修改 Archive。

而应该：

```text
新 Change B
```

然后：

```text
MODIFIED Requirements
```

修改当前主 Spec。

这属于：

> **系统演进。**

---

# 36. 这两个回溯千万不要混

```text
未 Archive
    ↓
Update 当前 Change
```

而：

```text
已 Archive
    ↓
创建新的 Change
    ↓
MODIFIED / ADDED / REMOVED / RENAMED
```

不要把历史 Archive 当成“可以随便修改的最新文档”。

---

# 37. 建议项目的 Git 提交方式

推荐一个 Change 对应若干 commit，但至少保证 Change 和代码一起进入 Git。

例如：

```text
feat: bootstrap http server
```

同时包含：

```text
src/
openspec/changes/bootstrap-http-server/
```

完成后：

```text
archive: bootstrap http server
```

或者直接在一个 feature commit 中完成。

重点是：

> OpenSpec 文件不是临时文件。

应该提交：

```text
openspec/
.opencode/
```

因为它们构成项目开发知识的一部分。

---

# 38. 推荐的 Git 历史

理想情况下可能看到：

```text
* archive: add markdown source view
* feat: add markdown source view
* archive: add markdown preview
* feat: add markdown preview
* archive: improve text detection
* feat: improve text detection
* archive: add text preview
* feat: add text preview
* archive: directory browsing
* feat: directory browsing
* initial: project bootstrap
```

这样未来：

```text
git log
```

和：

```text
openspec/changes/archive/
```

都可以解释项目演进。

---

# 39. 建议在 `openspec/config.yaml` 中加入项目规则

OpenSpec 当前支持项目级配置：

```text
openspec/config.yaml
```

它用于告诉 AI：

> 这个项目有什么背景、约束和开发规则。

例如可以逐步加入：

```yaml
context: |
  http-file-browser is a zero-configuration HTTP file browser.

  The application is started from the command line with a root directory.
  It is primarily intended for local file browsing and later supports
  authenticated remote access.

rules:
  specs:
    - Specifications describe observable system behavior.
    - Do not put implementation details into requirements.
    - Every requirement must contain concrete scenarios.
  design:
    - Keep implementation decisions in design.md.
  tasks:
    - Tasks must be independently understandable and verifiable.
    - Do not add functionality that is not required by the current change.
```

具体格式以当前 `openspec init` 生成的配置模板为准，不要照抄旧版本教程中的字段。

原则是：

> 项目级长期规则放 config。

而：

> 一次 Change 的特殊要求放 Change artifacts。

---

# 40. 不要过早定技术栈

这个项目技术栈可以有很多组合：

```text
Go
Rust
Node.js
Python
```

前端也可以：

```text
Vanilla JS
React
Vue
Svelte
```

但不要因为：

> “我比较喜欢 X”

就在第一个 Prompt 中强制整个项目。

更好的做法：

```text
/opsx:explore
```

让 AI 根据：

```text
零配置
单二进制
启动快
资源占用低
文件系统访问
静态资源
远程访问
未来编辑
```

比较方案。

然后你做技术决策。

技术决策进入：

```text
design.md
```

如果这个决策对长期架构非常重要，可以进一步记录 ADR。

---

# 41. OpenSpec 与 ADR 的关系

这个项目里会出现很多架构决策：

```text
为什么选择 Go？
为什么前后端不分离？
为什么使用静态资源？
为什么使用某个 Markdown renderer？
为什么代码高亮放浏览器端？
为什么 Token 不持久化？
为什么远程认证使用 Bearer Token？
```

但不要每个决定都写 ADR。

原则可以简单理解为：

```text
Spec
    ↓
系统应该怎样表现

Design
    ↓
这个 Change 准备怎样实现

ADR
    ↓
一个重要架构决策为什么这样选
```

例如：

> “应用采用单二进制部署，不依赖 Node runtime。”

如果这是一个长期架构约束，就很适合 ADR。

---

# 42. 推荐的 ADR 实验

在：

```text
bootstrap-http-server
```

或者：

```text
add-remote-access
```

阶段做一个真正的 ADR。

例如：

```text
ADR-001-single-binary-deployment
```

内容回答：

```text
Context
Decision
Alternatives
Consequences
```

例如：

```text
Context

The application is intended to be zero-configuration.

Decision

The application will be distributed as a single executable
containing the HTTP server and static frontend assets.

Alternatives

- Separate frontend server
- Runtime dependency on Node.js
- Docker-only deployment

Consequences

- Easier deployment
- Smaller operational surface
- Frontend build becomes part of release process
```

这样你会同时看到：

```text
Spec
Design
ADR
Tasks
```

各自负责什么。

---

# 43. OpenSpec 实践中的一个重要原则

不要让 AI 自己决定：

> 需求发生变化以后，到底是修改现有 Requirement 还是新增 Requirement。

你应该先判断：

### 行为真的改变了？

使用：

```text
MODIFIED
```

### 完全新增行为？

使用：

```text
ADDED
```

### 行为被删除？

使用：

```text
REMOVED
```

### 只是名字变化？

使用：

```text
RENAMED
```

这是你学习 OpenSpec 时最应该掌握的判断。

---

# 44. 推荐的“每次 Change Prompt”

你以后可以直接使用这个模板。

```text
/opsx:propose

我们现在要处理一个新的产品变化。

请先读取：
1. 当前 openspec/specs/
2. 当前 active changes
3. 与本次需求相关的现有 capability

本次需求：

<在这里写需求>

请：
1. 判断这是新增 capability 还是修改已有 capability；
2. 不要重复创建已有 capability；
3. 创建 proposal.md；
4. 创建对应 delta specs；
5. 创建 design.md；
6. 创建 tasks.md；
7. Spec 只描述可观察行为；
8. Design 描述实现方式；
9. Tasks 必须可以逐项执行和验证；
10. 如果需求存在未决策的问题，先指出问题，不要擅自决定；
11. 完成 planning 后停止，不要写代码。

尤其检查：
- ADDED / MODIFIED / REMOVED / RENAMED 是否使用正确；
- MODIFIED 是否完整复制原 Requirement；
- 每个 Requirement 是否至少有一个 Scenario；
- Scenario 是否可以成为测试案例。
```

---

# 45. Explore Prompt 模板

```text
/opsx:explore

我们准备对 http-file-browser 做如下变化：

<需求>

暂时不要写代码。

请：
1. 阅读相关现有 specs；
2. 阅读相关代码；
3. 解释当前系统行为；
4. 找出需求中不明确的地方；
5. 区分产品行为和技术实现；
6. 判断这是新增 capability 还是修改已有 capability；
7. 给出可能的方案及其影响；
8. 指出哪些决策现在必须做；
9. 指出哪些决策可以推迟。

不要为了完成任务而替我做未经确认的产品决策。
```

---

# 46. Apply Prompt 模板

```text
/opsx:apply

开始实施当前 Change。

要求：
1. 先读取 proposal.md、specs、design.md 和 tasks.md；
2. 严格按照 tasks.md 执行；
3. 每完成一个任务就更新 tasks.md；
4. 为关键行为增加测试；
5. 不自行增加当前 Change 未定义的产品功能；
6. 如果实现过程中发现 Spec 有问题，暂停并说明；
7. 不要通过修改代码来掩盖需求变化；
8. 如果需要改变计划，先更新 Change artifacts；
9. 最终确保实现与 Spec 一致。

现在开始。
```

---

# 47. Verify Prompt 模板

```text
/opsx:verify

请验证当前 Change 的实现。

逐项检查：
1. proposal 中描述的目标是否实现；
2. 每一个 Requirement 是否都有对应实现；
3. 每一个 Scenario 是否可以通过测试或手工验证；
4. 是否存在 Spec 没有定义但实现额外加入的用户行为；
5. 是否存在 tasks 已完成但实际行为没有实现；
6. 是否存在实现与 design.md 不一致；
7. 是否存在明显安全问题；
8. 是否存在路径穿越、权限、超大文件等边界问题。

不要修改代码。

输出：
- 已满足项
- 不满足项
- 额外行为
- 建议修改
```

---

# 48. Update Prompt 模板

```text
/opsx:update

当前 Change 已经进入实施阶段。

我们发现：

<描述实际发现的问题>

请重新检查：
- proposal.md
- specs/
- design.md
- tasks.md

判断：
1. 这是实现问题还是需求问题；
2. 哪些 artifact 需要修改；
3. 哪些已完成任务仍然有效；
4. 哪些任务需要调整；
5. 是否需要增加/修改/删除 Requirement。

先更新 Change artifacts，使整个计划内部一致。

不要直接修改业务代码。
完成计划调整后停止。
```

---

# 49. 归档 Prompt

```text
/opsx:archive

当前 Change 已经完成。

请先确认：
1. tasks 是否全部完成；
2. delta specs 是否有效；
3. implementation 是否已经验证；
4. 是否存在未解决的问题。

如果没有阻塞：
执行 archive。

归档时请确保：
- delta specs 正确合并到 main specs；
- Change 完整移动到 archive；
- 不修改历史 Change；
- 最后报告新的 main specs 状态。
```

---

# 50. 一个非常重要的实践：不要一次做完所有 artifacts

虽然：

```text
/opsx:propose
```

会快速生成：

```text
proposal
specs
design
tasks
```

但是学习阶段建议你至少有几个 Change 使用更细粒度的工作流：

```text
/opsx:new
```

然后：

```text
/opsx:continue
```

一步一个 artifact。

例如：

```text
new
 ↓
proposal
 ↓
review
 ↓
continue
 ↓
spec
 ↓
review
 ↓
continue
 ↓
design
 ↓
review
 ↓
continue
 ↓
tasks
```

这样你才能真正体会：

> Proposal、Spec、Design、Tasks 为什么要分开。

等理解了以后，再使用：

```text
/opsx:propose
```

快速完成规划。

---

# 51. 建议开启 expanded workflow

因为你这个项目的目的就是学习 OpenSpec。

建议不要只使用 core。

可以考虑：

```bash
openspec config profile
```

选择 expanded workflow，或者按照当前 CLI 支持的配置方式启用：

```text
new
continue
ff
verify
```

然后：

```bash
openspec update
```

OpenSpec 官方当前把 core 作为默认工作流，而扩展工作流提供更细的控制。 

你的实验项目特别适合：

```text
new
continue
verify
```

而不是永远只：

```text
propose
apply
archive
```

---

# 52. 但是生产项目未必需要这么复杂

最终实际工作中，我建议：

### 小 Change

```text
/opsx:propose
↓
review
↓
/opsx:apply
↓
/opsx:archive
```

### 不确定的 Change

```text
/opsx:explore
↓
/opsx:propose
↓
review
↓
apply
↓
archive
```

### 复杂 Change

```text
explore
↓
new
↓
continue
↓
review each artifact
↓
apply
↓
verify
↓
archive
```

### 实施过程中发生需求变化

```text
apply
↓
发现问题
↓
update
↓
继续 apply
```

### 已经发布后需求变化

```text
new Change
↓
MODIFIED
↓
apply
↓
verify
↓
archive
```

---

# 53. 本项目第一轮具体执行顺序

现在真正开始的时候，我建议你不要先写代码。

## Step 0

建立仓库：

```bash
mkdir http-file-browser
cd http-file-browser
git init
```

---

## Step 1

安装 OpenSpec：

```bash
brew install openspec
```

或者：

```bash
npm install -g @fission-ai/openspec@latest
```

---

## Step 2

初始化：

```bash
openspec init
```

---

## Step 3

检查：

```bash
openspec view
openspec list
openspec list --specs
openspec status
```

---

## Step 4

OpenCode：

```text
/opsx:explore
```

使用本文第 5 节 Prompt。

---

## Step 5

和 AI 一起决定：

```text
技术栈
MVP 范围
capability 划分
第一个 Change
```

---

## Step 6

创建第一个 Change：

```text
bootstrap-http-server
```

---

## Step 7

Review：

```text
proposal
spec
design
tasks
```

---

## Step 8

Apply：

```text
/opsx:apply
```

---

## Step 9

Verify：

```text
/opsx:verify
```

---

## Step 10

Validate：

```bash
openspec validate --all
```

---

## Step 11

Archive：

```text
/opsx:archive
```

---

## Step 12

检查：

```bash
openspec list
openspec list --specs
openspec view
```

---

# 54. 第一轮完成以后不要马上继续写代码

这是一个非常重要的学习方法。

完成：

```text
bootstrap-http-server
```

之后，停下来。

看看：

```text
openspec/specs/
openspec/changes/archive/
```

问自己三个问题：

### 第一

> `specs/` 是不是已经能够描述当前系统？

如果不能，说明 Spec 没写好。

### 第二

> archive 能不能解释这个系统为什么有今天的行为？

如果不能，说明 Change 没有形成历史。

### 第三

> 如果现在提出一个需求变化，我能不能准确找到应该修改哪个 Requirement？

如果不能，说明 capability 划分有问题。

---

# 55. 这个项目最终应该形成的 OpenSpec 结构

理想状态：

```text
http-file-browser/
│
├── .opencode/
│   ├── commands/
│   │   └── opsx-*.md
│   └── skills/
│       └── openspec-*/
│
├── openspec/
│   ├── config.yaml
│   │
│   ├── specs/
│   │   ├── directory-browsing/
│   │   │   └── spec.md
│   │   ├── file-metadata/
│   │   │   └── spec.md
│   │   ├── text-preview/
│   │   │   └── spec.md
│   │   ├── syntax-highlighting/
│   │   │   └── spec.md
│   │   ├── markdown-preview/
│   │   │   └── spec.md
│   │   ├── file-editing/
│   │   │   └── spec.md
│   │   └── remote-access/
│   │       └── spec.md
│   │
│   └── changes/
│       ├── archive/
│       │   ├── 2026-xx-bootstrap-http-server/
│       │   ├── 2026-xx-directory-browsing/
│       │   ├── 2026-xx-text-preview/
│       │   ├── 2026-xx-improve-text-file-detection/
│       │   ├── 2026-xx-syntax-highlighting/
│       │   └── ...
│       │
│       └── <active-change>/
│           ├── .openspec.yaml
│           ├── proposal.md
│           ├── specs/
│           │   └── capability/
│           │       └── spec.md
│           ├── design.md
│           └── tasks.md
│
├── src/
├── tests/
├── README.md
└── ...
```

---

# 56. 最后形成一个完整的“需求演进链”

这个项目真正做完以后，最好能够拿出这样一条链：

```text
系统最初
│
├── bootstrap-http-server
│
├── directory-browsing
│
├── file-metadata
│
├── text-preview
│
│   └── improve-text-file-detection
│       └── MODIFIED
│
├── syntax-highlighting
│
├── markdown-preview
│
│   └── improve-markdown-preview
│       └── MODIFIED
│
├── image-preview
│
├── file-editing
│
│   └── edit-conflict-detection
│
└── remote-access
    │
    └── token-authentication
```

然后：

```text
openspec/specs/
```

是：

> **当前系统的规范。**

而：

```text
openspec/changes/archive/
```

是：

> **系统规范的演进史。**

这时候你才真正理解 OpenSpec，而不是只会几个命令。

---

# 57. 最值得观察的三个现象

在整个项目过程中，我建议你特别记录三个东西。

## 现象一：AI 会不会把 Spec 写成实现方案？

如果会：

```text
“使用 React + Monaco + Go”
```

出现在 Requirement 中，

就把它改掉。

---

## 现象二：AI 会不会把一个需求变化错误地做成 ADDED？

例如：

```text
已有：
text-preview

新需求：
改变 text-preview 判断规则
```

AI 却创建：

```text
text-detection-v2
```

这就是一个很好的 OpenSpec 练习。

应该回到：

```text
MODIFIED
```

---

## 现象三：Apply 会不会偷偷扩大范围？

例如：

```text
实现 Markdown preview
```

AI 顺便：

```text
加入搜索
加入上传
加入编辑
加入删除
```

这时候不要接受。

让它回到：

```text
当前 Change
```

然后另开 Change。

---

# 58. 最终的工作哲学

这个项目真正应该建立的习惯不是：

```text
会用 OpenSpec 命令
```

而是：

```text
想法
 ↓
Explore
 ↓
明确问题
 ↓
Proposal
 ↓
定义行为
 ↓
Spec
 ↓
决定实现
 ↓
Design
 ↓
拆分任务
 ↓
Tasks
 ↓
实现
 ↓
Apply
 ↓
验证
 ↓
Verify
 ↓
合并当前事实
 ↓
Archive
 ↓
下一次变化
```

更重要的是：

```text
如果 Change 尚未完成：

需求变化
    ↓
Update 当前 Change

如果 Change 已经完成：

需求变化
    ↓
创建新的 Change
    ↓
MODIFIED / ADDED / REMOVED / RENAMED
```

这样 OpenSpec 才真正成为：

> **AI 时代的软件需求与系统演进记录，而不是一个给 AI 写代码前看的 Markdown 文件夹。**

---

# 59. 本项目的第一阶段验收标准

当你完成前几个 Change 后，应该能够回答下面这些问题：

```text
□ 我知道 openspec/specs 和 changes 的区别

□ 我知道 delta spec 是什么

□ 我知道 ADDED 和 MODIFIED 的区别

□ 我知道 MODIFIED 为什么必须保留完整 Requirement

□ 我知道 Scenario 与测试的关系

□ 我知道 Proposal、Spec、Design、Tasks 各自解决什么问题

□ 我知道什么时候使用 explore

□ 我知道什么时候使用 update

□ 我知道 sync 与 archive 的区别

□ 我知道为什么已经 Archive 的 Change 不应该直接修改

□ 我能够从历史 Change 回溯某个系统行为为什么存在

□ 我能够从当前 Spec 判断系统应该有什么行为

□ 我能够让 AI 在实现过程中停止自行扩大需求

□ 我能够让 AI 根据新的需求创建新的 Change

□ 我能够让 AI 在需求变化后重新同步 Spec、Design 和 Tasks
```

如果这些都能做到，那么 OpenSpec 的核心工作方式基本就掌握了。

---

# 60. 现在就开始：第一条实际 Prompt

不要继续设计整个项目。

进入 OpenCode 后，直接：

```text
/opsx:explore

我准备通过 OpenSpec 实际开发一个项目，项目暂定名为 http-file-browser。

它是一个零配置 HTTP 文件浏览器。用户通过命令行指定一个目录后启动服务，在浏览器中以类似 macOS Finder 的方式浏览文件系统。

长期目标包括：
- 目录浏览
- 文件/目录信息
- 文本文件查看
- 程序代码语法高亮
- Markdown 渲染
- 图片预览
- 文件搜索
- 文件编辑
- 远程访问
- 启动时生成 token，并使用 token 认证远程访问

但是第一阶段只准备实现最小可运行的文件浏览器，不做编辑和远程认证。

这是一个 OpenSpec 学习项目，所以我希望优先验证完整的 SDD 工作流，而不是追求一次把产品做完。

请先不要写代码。

请调查当前代码库，然后：
1. 分析这个项目的核心用户行为；
2. 建议合理的 capability 划分；
3. 给出适合 OpenSpec 的小 Change 序列；
4. 区分 MVP 和后续能力；
5. 指出现在必须做出的架构决策；
6. 指出可以推迟的决策；
7. 特别指出哪些需求适合未来用 MODIFIED Requirements 演练需求变更；
8. 不要创建最终代码；
9. 不要直接开始实现。

我们先通过 Explore 把问题空间搞清楚，然后再进入第一个 Change。
```

**到这里就停。**

不要在第一次对话里再让 OpenCode：

```text
propose
apply
```

先看它的 Explore 结果。

然后下一步才是：

```text
/opsx:propose bootstrap-http-server
```

这样开始，你得到的不是一个“AI 帮你写了一个文件浏览器”的项目，而是一个可以完整观察 **Explore → Spec → Design → Tasks → Apply → Verify → Archive → Change → MODIFIED → 再 Archive** 的 OpenSpec 实验项目。

---

## 参考资料

本文中的 OpenSpec 工作流、CLI、OPSX、spec-driven schema 和 OpenCode 支持以 OpenSpec 当前官方文档为准：

- OpenSpec 官方站点
- OpenSpec Quickstart
- OpenSpec CLI
- OPSX Workflow
- spec-driven Schema
- Supported Tools

由于 OpenSpec 仍在快速迭代，实际使用时如果 OpenCode 中显示的命令形式与本文不同，以 `openspec init` 输出和当前项目生成的 `.opencode/commands/`、`.opencode/skills/` 为准。
