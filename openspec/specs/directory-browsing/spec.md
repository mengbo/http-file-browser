# directory-browsing Specification

## Purpose

定义服务在命令行指定的根目录范围内向用户提供目录浏览时用户可观察到的行为契约：目录列表响应包含什么、条目如何区分类型、列表以什么顺序呈现、浏览位置如何表示与重现，以及服务在什么情况下拒绝提供内容。不包含文件内容读取。

## Requirements

### Requirement: Directory listing response

系统 SHALL 提供目录列表端点，接受一个表示位置的相对根目录路径参数，返回该目录的规范化相对路径、上级相对路径与条目列表。系统 SHALL 在路径参数缺失或为空时返回根目录的列表。系统 SHALL 在每个条目中提供名称与类型，以及该条目的大小与最后修改时间，其取值规则见 `Entry metadata`。系统 SHALL NOT 在列表中提供条目的内容。

#### Scenario: A directory is listed

- **WHEN** 客户端请求一个存在且可读的目录的列表
- **THEN** 系统返回 JSON 响应体，其中包含该目录的规范化相对路径、上级相对路径与该目录下的条目列表

#### Scenario: Path parameter is missing or empty

- **WHEN** 客户端在未提供路径参数或提供空路径参数的情况下请求列表
- **THEN** 系统返回根目录的列表，且响应中的路径为空字符串

#### Scenario: A directory has no entries

- **WHEN** 客户端请求一个存在、可读但不包含任何条目的目录
- **THEN** 系统返回条目列表为空的成功响应，不返回错误

#### Scenario: Path parameter contains redundant segments

- **WHEN** 客户端提供的路径参数包含冗余片段，例如多余的当前目录标记、重复的分隔符或结尾分隔符
- **THEN** 系统按规范化后的路径返回列表，且响应中的路径为该规范化结果

#### Scenario: A listed directory contains readable files

- **WHEN** 客户端请求的目录包含一个存在且可读的文件
- **THEN** 该文件在响应中只以元信息的形式出现，响应不包含该文件的任何内容

### Requirement: Entry metadata

系统 SHALL 在目录列表的每个条目中给出该条目的最后修改时间。系统 SHALL 为非目录条目给出该条目自身的大小，并 SHALL NOT 为目录条目给出大小。系统 SHALL 给出的大小是该条目自身占用的字节数；对于符号链接条目，该值是链接自身的长度，而不是其目标内容的大小。系统 SHALL 以自 Unix 纪元起的整数秒表示最后修改时间，该表示 SHALL NOT 随运行环境的时区或语言变化。系统 SHALL 在某个条目的大小或最后修改时间无法获得时省略该值，而不是给出占位值。系统 SHALL NOT 因单个条目的元信息无法获得而使整个列表请求失败。

#### Scenario: A file entry is listed

- **WHEN** 客户端请求的目录包含一个普通文件
- **THEN** 该条目在列表中给出该文件自身的大小与最后修改时间

#### Scenario: A directory entry is listed

- **WHEN** 客户端请求的目录包含一个子目录
- **THEN** 该条目在列表中给出该子目录的最后修改时间，且不给出该子目录的大小

#### Scenario: An entry is a symbolic link

- **WHEN** 客户端请求的目录包含一个指向其他位置的符号链接
- **THEN** 该条目给出的大小是链接自身的长度，而不是其目标内容的大小

#### Scenario: Modification time falls within the same second

- **WHEN** 某个条目的最后修改时间含有不足一秒的精度
- **THEN** 系统给出的是向下取整到整秒的取值

#### Scenario: Modification time is read under a different environment

- **WHEN** 同一个目录在不同时区或不同语言环境的进程中被列出
- **THEN** 两次响应中每个条目的最后修改时间取值相同

#### Scenario: An entry's metadata cannot be obtained

- **WHEN** 客户端请求的目录在系统读取期间发生变化，某个已列出的条目的大小或最后修改时间无法获得
- **THEN** 系统返回成功响应，该条目仍出现在列表中且其无法获得的元信息被省略

#### Scenario: Repeated listings return the same metadata

- **WHEN** 客户端对同一目录多次请求列表，且期间该目录内容未发生变化
- **THEN** 每次响应中每个条目的大小与最后修改时间取值都相同

### Requirement: Entry type distinction

系统 SHALL 在列表中区分目录条目与文件条目。

#### Scenario: A directory contains both directories and files

- **WHEN** 客户端请求一个同时包含子目录与文件的目录
- **THEN** 系统在每个条目中标明该条目是目录还是文件

### Requirement: List ordering

系统 SHALL 以确定的顺序返回条目：所有目录条目排在所有文件条目之前，两组各自按名称排序；名称比较不区分大小写；仅大小写不同的同名条目之间的相对顺序 SHALL 在多次请求之间保持一致。

#### Scenario: A directory contains both directories and files

- **WHEN** 客户端请求一个同时包含子目录与文件的目录
- **THEN** 目录条目全部出现在文件条目之前

#### Scenario: Entries within the same group

- **WHEN** 客户端请求一个包含多个同类型条目的目录
- **THEN** 这些条目按名称排序后返回

#### Scenario: Names differ only in letter case

- **WHEN** 同一组内存在名称仅大小写不同的条目
- **THEN** 排序不因大小写而颠倒这些条目的相对顺序

#### Scenario: The same directory is listed repeatedly

- **WHEN** 客户端对同一目录多次请求列表
- **THEN** 每次返回的条目顺序完全相同

### Requirement: Position representation and reproducibility

系统 SHALL 用相对根目录的路径表示浏览位置。系统 SHALL 使同一个位置表示在重复请求时解析到同一目录并返回相同的列表。

#### Scenario: A position is reopened

- **WHEN** 客户端此前请求过某个位置的列表，随后以同一位置表示再次请求
- **THEN** 系统返回同一目录的列表，且响应中的路径与首次相同

#### Scenario: Position is resolved from a path containing a parent reference

- **WHEN** 客户端提供的相对路径中包含指向上级目录的片段，且规范化后仍位于根目录之内
- **THEN** 系统返回规范化后所指目录的列表

### Requirement: Parent directory reference

系统 SHALL 在列表响应中给出当前目录的上级相对路径。系统 SHALL 在当前目录为根目录时不给出上级路径。

#### Scenario: Listing a directory inside the root

- **WHEN** 客户端请求根目录内某个子目录的列表
- **THEN** 响应中给出该子目录的上级相对路径

#### Scenario: Listing the root directory

- **WHEN** 客户端请求根目录的列表
- **THEN** 响应中不给出上级路径

### Requirement: Root directory confinement

系统 SHALL NOT 提供根目录之外的内容。系统 SHALL 按物理位置判断越界：请求路径解析其全部符号链接后得到的物理位置位于根目录的物理位置之内时，系统 SHALL 按该路径提供内容；位于根目录的物理位置之外时，系统 SHALL 拒绝该请求，并返回 JSON 错误响应。根目录的物理位置指根目录路径解析其全部符号链接后得到的位置。

#### Scenario: A path escapes the root by parent references

- **WHEN** 客户端提供的相对路径经规范化后指向根目录之外的目录
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应，不返回该目录的列表

#### Scenario: A sibling directory shares the root path prefix

- **WHEN** 客户端请求的路径与根目录同名但并非根目录本身，例如根目录为 `a/b` 而请求路径解析到 `a/bc`
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应

#### Scenario: A path inside the root traverses a symbolic link outward

- **WHEN** 客户端请求的路径字面位置位于根目录之内，但解析后指向根目录之外
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应，不返回该路径的内容

#### Scenario: A path traverses a symbolic link to another location inside the root

- **WHEN** 客户端请求的路径经由符号链接指向根目录内的另一个位置
- **THEN** 系统按该路径提供内容，不将其视为越界

#### Scenario: The root path itself contains a symbolic link

- **WHEN** 指定的根目录路径经由符号链接指向其物理位置，且客户端请求的路径解析后位于该物理位置之内
- **THEN** 系统按该路径提供内容，不将其视为越界

### Requirement: Directory access failures

系统 SHALL 在目录列表请求无法完成时返回 JSON 错误响应体，其机器可读错误标识区分失败原因：路径不存在为 `not_found`，路径存在但不是目录为 `not_a_directory`，无访问权限为 `permission_denied`，超出根目录范围为 `outside_root`。

#### Scenario: The requested path does not exist

- **WHEN** 客户端请求的路径在文件系统中不存在
- **THEN** 系统返回机器可读错误标识为 `not_found` 的 JSON 错误响应体

#### Scenario: The requested path is not a directory

- **WHEN** 客户端请求的路径存在但不是目录
- **THEN** 系统返回机器可读错误标识为 `not_a_directory` 的 JSON 错误响应体

#### Scenario: The requested directory cannot be read

- **WHEN** 客户端请求的目录存在但当前进程无读取权限
- **THEN** 系统返回机器可读错误标识为 `permission_denied` 的 JSON 错误响应体

#### Scenario: The requested path is outside the root

- **WHEN** 客户端请求的路径解析后落在根目录之外
- **THEN** 系统返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体

### Requirement: 类型化视觉图标

对于目录浏览视图中的每个条目，系统 SHALL 在该条目名称的可视位置之前显示一个视觉图标。系统 SHALL 使目录条目的视觉图标与文件条目的视觉图标在外观上可区分——目录条目显示目录图标，文件条目显示文件图标。系统 SHALL 使该图标在浅色外观与深色外观下均与所在背景保持足以识别类型的对比度。系统 SHALL NOT 在文件视图与搜索结果视图的条目名称前显示该视觉图标——这两处的呈现形态不属于本 Requirement 的范围。

#### Scenario: 目录条目显示目录图标

- **WHEN** 目录浏览视图列出包含目录与文件混合的目录
- **THEN** 每个目录条目的名称前显示目录图标

#### Scenario: 文件条目显示文件图标

- **WHEN** 目录浏览视图列出包含目录与文件混合的目录
- **THEN** 每个文件条目的名称前显示文件图标

#### Scenario: 目录与文件图标在外观上可区分

- **WHEN** 目录浏览视图列出包含目录与文件混合的目录
- **THEN** 目录条目与文件条目的视觉图标外观不同，可凭视觉将其区分

#### Scenario: 图标在浅色与深色外观下均可识别

- **WHEN** 系统以浅色外观呈现目录浏览视图
- **THEN** 每个条目的视觉图标与背景之间的对比度足以识别该条目是目录还是文件
- **WHEN** 系统以深色外观呈现目录浏览视图
- **THEN** 每个条目的视觉图标与背景之间的对比度足以识别该条目是目录还是文件

#### Scenario: 文件视图与搜索结果视图不显示视觉图标

- **WHEN** 系统呈现文件视图（文本预览、Markdown 渲染、图片查看等）中的内容
- **THEN** 该视图不显示本 Requirement 所规定的视觉图标
- **WHEN** 系统呈现搜索结果视图中的条目
- **THEN** 每个搜索结果条目的名称前不显示本 Requirement 所规定的视觉图标
