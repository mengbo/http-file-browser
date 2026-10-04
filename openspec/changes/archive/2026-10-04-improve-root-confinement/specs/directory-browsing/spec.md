# Spec Delta: directory-browsing

## MODIFIED Requirements

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
