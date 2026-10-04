# Spec Delta: text-preview

## MODIFIED Requirements

### Requirement: Text file content response

系统 SHALL 提供读取文件内容的端点，接受一个表示位置的相对根目录路径参数，返回该文件的规范化相对路径与该文件的内容。系统 SHALL NOT 返回根目录之外的文件的内容。系统 SHALL 以 UTF-8 呈现内容，无法解码为 UTF-8 的字节序列以替换字符呈现。

#### Scenario: A text file is requested

- **WHEN** 客户端请求一个被认定为可读文本的文件的路径
- **THEN** 系统返回 JSON 响应体，其中包含该文件的规范化相对路径与该文件的内容

#### Scenario: A path inside the root traverses a symbolic link outward

- **WHEN** 客户端请求的路径字面位置位于根目录之内，但解析后指向根目录之外
- **THEN** 系统拒绝该请求，返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体，不返回该文件的内容

#### Scenario: The content is not decodable as UTF-8

- **WHEN** 被认定为可读文本的文件包含无法解码为 UTF-8 的字节序列
- **THEN** 系统仍返回成功响应，其中该部分字节以替换字符呈现

### Requirement: Content reading failures

系统 SHALL 在内容读取请求无法完成时返回 JSON 错误响应体，其机器可读错误标识区分失败原因：路径不存在为 `not_found`，路径指向目录为 `not_a_directory`，路径存在但不是普通文件为 `not_a_regular_file`，无访问权限为 `permission_denied`，超出根目录范围为 `outside_root`，不被视为可读文本的文件为 `not_text`，超过可提供内容的最大字节数为 `too_large`。

#### Scenario: The requested path does not exist

- **WHEN** 客户端请求的路径在文件系统中不存在
- **THEN** 系统返回机器可读错误标识为 `not_found` 的 JSON 错误响应体

#### Scenario: The requested path points to a directory

- **WHEN** 客户端请求的路径指向一个目录
- **THEN** 系统返回机器可读错误标识为 `not_a_directory` 的 JSON 错误响应体

#### Scenario: The requested path is not a regular file

- **WHEN** 客户端请求的路径存在但不是普通文件
- **THEN** 系统返回机器可读错误标识为 `not_a_regular_file` 的 JSON 错误响应体

#### Scenario: The requested file cannot be read

- **WHEN** 客户端请求的文件存在但当前进程无读取权限
- **THEN** 系统返回机器可读错误标识为 `permission_denied` 的 JSON 错误响应体

#### Scenario: The requested path is outside the root

- **WHEN** 客户端请求的路径解析后落在根目录之外
- **THEN** 系统返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体

#### Scenario: The requested file is not recognized as text

- **WHEN** 客户端请求的路径不被视为可读文本的文件
- **THEN** 系统返回机器可读错误标识为 `not_text` 的 JSON 错误响应体

#### Scenario: The requested file exceeds the maximum size

- **WHEN** 客户端请求的文件超过系统可提供内容的最大字节数
- **THEN** 系统返回机器可读错误标识为 `too_large` 的 JSON 错误响应体
