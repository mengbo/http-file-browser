# search Specification

## Purpose
定义在基准位置的子树内按文件名递归定位条目的行为契约：搜索端点与参数缺省约定、名称匹配语义、命中条目的形状与顺序、遍历边界（符号链接、无权限子目录）、基准位置失败的机器可读错误，以及前端搜索视图的呈现与 URL 重现。不包含依据文件内容的搜索。

## Requirements

### Requirement: Search endpoint

系统 SHALL 提供搜索端点，接受表示基准位置的相对根目录路径参数与查询词参数，返回基准位置子树内按名称匹配的条目列表、基准位置的规范化相对路径与回显的查询词。系统 SHALL 在基准位置参数缺失或为空时以根目录为基准。

#### Scenario: A subtree is searched

- **WHEN** 客户端以一个存在且可读的目录为基准位置、附带查询词请求搜索
- **THEN** 系统返回 JSON 响应体，其中包含基准位置的规范化相对路径、回显的查询词与命中条目列表

#### Scenario: Base parameter is missing or empty

- **WHEN** 客户端在未提供基准位置参数或提供空基准位置参数的情况下请求搜索
- **THEN** 系统以根目录为基准执行搜索，且响应中的基准路径为空字符串

#### Scenario: Base parameter contains redundant segments

- **WHEN** 客户端提供的基准位置参数包含冗余片段，例如多余的当前目录标记、重复的分隔符或结尾分隔符
- **THEN** 系统按规范化后的基准位置执行搜索，且响应中的基准路径为该规范化结果

### Requirement: Name matching semantics

系统 SHALL 仅以条目自身的名称进行匹配，SHALL NOT 依据条目的所在路径、内容或元信息决定匹配。匹配规则 SHALL 是查询词作为名称的子串包含，名称比较不区分大小写。系统 SHALL 在查询词缺失或为空时使基准位置子树内的全部条目成为命中。

#### Scenario: A query is a substring of a name

- **WHEN** 查询词是某个条目名称的连续子串
- **THEN** 该条目成为命中

#### Scenario: A query differs from a name only in letter case

- **WHEN** 查询词与某个条目名称的某个子串仅大小写不同
- **THEN** 该条目成为命中

#### Scenario: A query appears only in the location path

- **WHEN** 查询词只出现在某个条目的所在目录路径中，而未出现在该条目自身名称中
- **THEN** 该条目不是命中

#### Scenario: Query is missing or empty

- **WHEN** 客户端在未提供查询词或提供空查询词的情况下请求搜索
- **THEN** 基准位置子树内的全部条目都是命中

#### Scenario: Nothing matches

- **WHEN** 基准位置子树内没有名称包含查询词的条目
- **THEN** 系统返回命中条目列表为空的成功响应，不返回错误

### Requirement: Match entry shape

系统 SHALL 在每个命中条目中提供名称、类型（目录或文件）与相对根目录的完整路径。系统 SHALL NOT 在命中条目中提供条目内容。

#### Scenario: A match entry carries name, type, and full path

- **WHEN** 基准位置子树内的某个条目成为命中
- **THEN** 该命中条目给出自身名称、目录或文件的类型标识，以及相对根目录的完整路径

#### Scenario: Match entries contain no content

- **WHEN** 搜索返回某个文件的命中条目
- **THEN** 该命中条目不包含该文件的任何内容

### Requirement: Result ordering

系统 SHALL 以确定的顺序返回命中条目：先给出基准位置自身的全部命中，其相对顺序与浏览该目录时列表的顺序一致（目录条目在文件条目之前，各组内按名称排序，名称比较不区分大小写）；随后按同一目录顺序依次给出每个子目录子树中的命中，递归执行。同一基准位置与查询词的重复请求，在子树未发生变化时 SHALL 返回相同顺序。

#### Scenario: Matches are arranged in tree order

- **WHEN** 基准位置子树内多个目录各自存在命中
- **THEN** 基准位置自身的命中全部出现在任何子树命中之前，每个子目录子树中的命中连续排列

#### Scenario: Matches within the same directory follow listing order

- **WHEN** 同一目录内同时存在命中的目录条目与命中的文件条目
- **THEN** 命中的目录条目出现在命中的文件条目之前，各组内按名称排序，且名称仅大小写不同的命中之间相对顺序保持一致

#### Scenario: Repeated searches return the same order

- **WHEN** 客户端以同一基准位置与查询词多次请求搜索，且期间子树内容未发生变化
- **THEN** 每次响应中命中条目的顺序完全相同

### Requirement: Search scope confinement

系统 SHALL 仅返回基准位置子树内条目的命中。系统 SHALL NOT 提供根目录之外的内容：基准位置按物理位置判断越界，即基准路径解析其全部符号链接后得到的物理位置位于根目录的物理位置之内时执行搜索，位于之外时系统 SHALL 拒绝该请求，并返回指示越界的 JSON 错误响应。

#### Scenario: A base escapes the root by parent references

- **WHEN** 客户端提供的基准位置经规范化后指向根目录之外
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应，不返回任何命中

#### Scenario: A base traverses a symbolic link outward

- **WHEN** 客户端提供的基准位置字面上位于根目录之内，但解析后指向根目录之外
- **THEN** 系统拒绝该请求，返回指示越界的 JSON 错误响应

#### Scenario: A base resolves via a symbolic link to inside the root

- **WHEN** 客户端提供的基准位置经由符号链接指向根目录内的另一个位置
- **THEN** 系统以该位置为基准执行搜索，不将其视为越界

### Requirement: Traversal boundaries

遍历 SHALL NOT 进入符号链接指出的目录；符号链接条目自身按名称参与匹配。系统 SHALL NOT 因子树中某个子目录无法读取而使整个搜索失败；无法读取的子目录所属子树不产生命中，其余命中照常返回。

#### Scenario: A symbolic link directory is matched but not descended

- **WHEN** 基准位置子树内存在一个指向其他目录的符号链接目录
- **THEN** 该符号链接条目自身按名称参与匹配，且它指向的目录中的条目不产生命中

#### Scenario: A dangling symbolic link can match

- **WHEN** 基准位置子树内存在一个指向不存在目标的符号链接
- **THEN** 该条目按名称参与匹配，且搜索不因此失败

#### Scenario: An unreadable subdirectory is skipped

- **WHEN** 基准位置子树内某个子目录存在但当前进程无读取权限
- **THEN** 系统返回成功响应，该子树不产生命中，其余命中照常返回

### Requirement: Base path access failures

系统 SHALL 在以基准位置发起的搜索无法完成时返回 JSON 错误响应体，其机器可读错误标识与目录列表一致：路径不存在为 `not_found`，路径存在但不是目录为 `not_a_directory`，无访问权限为 `permission_denied`，超出根目录范围为 `outside_root`。

#### Scenario: The base path does not exist

- **WHEN** 客户端以一个不存在的路径作为基准位置请求搜索
- **THEN** 系统返回机器可读错误标识为 `not_found` 的 JSON 错误响应体

#### Scenario: The base path is not a directory

- **WHEN** 客户端以一个存在但不是目录的路径作为基准位置请求搜索
- **THEN** 系统返回机器可读错误标识为 `not_a_directory` 的 JSON 错误响应体

#### Scenario: The base path cannot be read

- **WHEN** 客户端以一个存在但当前进程无读取权限的目录作为基准位置请求搜索
- **THEN** 系统返回机器可读错误标识为 `permission_denied` 的 JSON 错误响应体

### Requirement: Search view and URL reproducibility

前端 SHALL 提供搜索入口，提交后在结果视图中列出每个命中条目的名称、所在目录与类型。点击目录类型的命中 SHALL 进入该目录的列表视图，点击文件类型的命中 SHALL 进入该文件的预览视图。搜索视图 SHALL 以带基准位置与查询词参数的页面 URL 表示；同一 URL 重新打开时，若子树未发生变化 SHALL 呈现相同结果；浏览器前进与后退 SHALL 在列表视图与结果视图之间正确切换。

#### Scenario: A search is submitted

- **WHEN** 用户通过搜索入口提交一个查询词
- **THEN** 界面呈现结果视图，每个命中条目显示名称、所在目录与类型，页面 URL 携带基准位置与查询词

#### Scenario: No matches are found

- **WHEN** 用户提交的查询词在基准位置子树内没有任何命中
- **THEN** 界面呈现无命中的说明，而不是错误提示

#### Scenario: A directory match is clicked

- **WHEN** 用户点击一个目录类型的命中条目
- **THEN** 界面进入该目录的列表视图

#### Scenario: A file match is clicked

- **WHEN** 用户点击一个文件类型的命中条目
- **THEN** 界面进入该文件的预览视图

#### Scenario: A search view is reopened

- **WHEN** 用户以携带基准位置与查询词的 URL 重新打开页面，或通过浏览器前进、后退在列表视图与结果视图之间切换
- **THEN** 界面呈现该 URL 所指的视图：结果视图重现相同基准与查询词的搜索结果，列表视图照常呈现目录列表
