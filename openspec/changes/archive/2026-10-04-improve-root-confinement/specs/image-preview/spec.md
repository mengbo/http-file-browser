# Spec Delta: image-preview

## MODIFIED Requirements

### Requirement: Image content response

系统 SHALL 提供读取图片文件内容的端点，接受一个表示位置的相对根目录路径参数，返回该文件内容的字节序列，响应内容类型为按该文件扩展名对应的图片内容类型。系统 SHALL NOT 返回根目录之外的文件的图片内容。该端点的成功响应 SHALL NOT 以 JSON 响应体返回。

#### Scenario: An image file is requested

- **WHEN** 客户端请求一个被认定为图片文件的路径
- **THEN** 系统返回该文件内容的字节序列，响应内容类型为按该文件扩展名对应的图片内容类型，而非 JSON 响应体

#### Scenario: A path inside the root traverses a symbolic link outward

- **WHEN** 客户端请求的路径字面位置位于根目录之内，但解析后指向根目录之外
- **THEN** 系统拒绝该请求，返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体，不返回该文件的图片内容

#### Scenario: The position contains redundant segments

- **WHEN** 客户端提供的路径参数包含冗余片段，例如多余的当前目录标记、重复的分隔符或结尾分隔符
- **THEN** 系统按规范化后的路径提供图片内容

### Requirement: Image content failures

系统 SHALL 在图片内容请求无法完成时返回 JSON 错误响应体，其机器可读错误标识区分失败原因：路径不存在为 `not_found`，路径指向目录为 `not_a_directory`，路径存在但不是普通文件为 `not_a_regular_file`，无访问权限为 `permission_denied`，超出根目录范围为 `outside_root`，不被视为图片文件为 `not_an_image`。

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

#### Scenario: The requested file is not recognized as an image

- **WHEN** 客户端请求的路径不被视为图片文件
- **THEN** 系统返回机器可读错误标识为 `not_an_image` 的 JSON 错误响应体
