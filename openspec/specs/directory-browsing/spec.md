# directory-browsing Specification

## Purpose

定义服务在命令行指定的根目录范围内向用户提供目录浏览时用户可观察到的行为契约：目录列表响应包含什么、条目如何区分类型、列表以什么顺序呈现、浏览位置如何表示与重现，以及服务在什么情况下拒绝提供内容。不包含文件元信息与文件内容读取。

## Requirements

### Requirement: Directory listing response

系统 SHALL 提供目录列表端点，接受一个表示位置的相对根目录路径参数，返回该目录的规范化相对路径、上级相对路径与条目列表。系统 SHALL 在路径参数缺失或为空时返回根目录的列表。系统 SHALL 在每个条目中只提供名称与类型，不提供大小、修改时间或内容。

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

系统 SHALL NOT 提供根目录之外的内容。系统 SHALL 在请求路径按字面解析后落在根目录之外时拒绝该请求，并返回 JSON 错误响应。系统 SHALL 在判断越界时只依据字面路径：请求路径的字面位置位于根目录之内时，系统 SHALL 按该路径提供内容，即使该路径经由符号链接指向根目录之外。

#### Scenario: A path escapes the root by parent references

- **WHEN** 客户端提供的相对路径经规范化后指向根目录之外的目录
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应，不返回该目录的列表

#### Scenario: A sibling directory shares the root path prefix

- **WHEN** 客户端请求的路径与根目录同名但并非根目录本身，例如根目录为 `a/b` 而请求路径解析到 `a/bc`
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应

#### Scenario: A path inside the root traverses a symbolic link outward

- **WHEN** 客户端请求的路径字面位置位于根目录之内，但经由符号链接指向根目录之外
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

- **WHEN** 客户端请求的路径经字面解析后落在根目录之外
- **THEN** 系统返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体
