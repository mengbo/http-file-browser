# Spec Delta — image-preview

## Purpose

定义查看图片文件时用户可观察到的行为契约：哪些文件被当作图片文件、图片内容端点返回什么、无法提供图片内容时如何标识原因，以及文件视图中图片的呈现与加载失败回退。不包含文本文件的判定与呈现（属 `text-preview`），不包含文本型图像格式（如 SVG）的呈现，不包含图片内容的修改与写回。

## ADDED Requirements

### Requirement: Image content response

系统 SHALL 提供读取图片文件内容的端点，接受一个表示位置的相对根目录路径参数，返回该文件内容的字节序列，响应内容类型为按该文件扩展名对应的图片内容类型。系统 SHALL NOT 返回根目录之外的文件的图片内容。该端点的成功响应 SHALL NOT 以 JSON 响应体返回。

#### Scenario: An image file is requested

- **WHEN** 客户端请求一个被认定为图片文件的路径
- **THEN** 系统返回该文件内容的字节序列，响应内容类型为按该文件扩展名对应的图片内容类型，而非 JSON 响应体

#### Scenario: A path inside the root traverses a symbolic link outward

- **WHEN** 客户端请求的路径字面位置位于根目录之内，但经由符号链接指向根目录之外
- **THEN** 系统按该路径提供图片内容，不将其视为越界

#### Scenario: The position contains redundant segments

- **WHEN** 客户端提供的路径参数包含冗余片段，例如多余的当前目录标记、重复的分隔符或结尾分隔符
- **THEN** 系统按规范化后的路径提供图片内容

### Requirement: Image file recognition

系统 SHALL 将名称的扩展名属于已知图片扩展名的文件视为图片文件。系统 SHALL 仅依据文件的名称进行判定，SHALL NOT 依据文件的内容进行判定。系统 SHALL NOT 将其余文件视为图片文件：名称的扩展名不属于已知图片扩展名的文件，即使其内容为图片内容，也 SHALL NOT 被视为图片文件；名称没有任何扩展名的文件，即使其内容为图片内容，也 SHALL NOT 被视为图片文件。

#### Scenario: A file with a known image extension is requested

- **WHEN** 客户端请求一个名称的扩展名属于已知图片扩展名的文件的路径
- **THEN** 系统将该文件视为图片文件并提供其内容

#### Scenario: A file with a non-image extension is requested

- **WHEN** 客户端请求一个名称的扩展名不属于已知图片扩展名的文件的路径
- **THEN** 系统不将该文件视为图片文件，返回机器可读错误标识为 `not_an_image` 的 JSON 错误响应体

#### Scenario: A file with a non-image extension contains image content

- **WHEN** 客户端请求一个名称的扩展名不属于已知图片扩展名的文件的路径，且该文件的内容为图片内容
- **THEN** 系统仍不将该文件视为图片文件，返回机器可读错误标识为 `not_an_image` 的 JSON 错误响应体

#### Scenario: A file without an extension contains image content

- **WHEN** 客户端请求一个名称没有任何扩展名的文件的路径，且该文件的内容为图片内容
- **THEN** 系统不将该文件视为图片文件，返回机器可读错误标识为 `not_an_image` 的 JSON 错误响应体

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

- **WHEN** 客户端请求的路径经字面解析后落在根目录之外
- **THEN** 系统返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体

#### Scenario: The requested file is not recognized as an image

- **WHEN** 客户端请求的路径不被视为图片文件
- **THEN** 系统返回机器可读错误标识为 `not_an_image` 的 JSON 错误响应体

### Requirement: Image presentation in file view

对于扩展名属于已知图片扩展名的被查看文件，系统 SHALL 在文件视图中以图片形式呈现该文件，由浏览器从图片内容端点加载。图片无法加载时，系统 SHALL 在文件视图中呈现回退说明，指示该文件无法以图片查看。

#### Scenario: An image file is viewed

- **WHEN** 用户查看一个扩展名属于已知图片扩展名的文件
- **THEN** 该文件以图片形式呈现在文件视图中

#### Scenario: An image that cannot be loaded is viewed

- **WHEN** 用户查看一个扩展名属于已知图片扩展名的文件，且其内容无法被浏览器作为图片加载
- **THEN** 文件视图呈现回退说明，指示该文件无法以图片查看
