# syntax-highlighting Specification

## Purpose
定义查看文本文件时内容呈现层的语法高亮行为契约：什么情况下内容以高亮形式呈现、高亮呈现永远不改变内容的字符序列、语言无法识别时的回退呈现，以及高亮呈现依赖的前端资源必须自包含地随服务提供。不包含哪些文件可被读取为文本的判定（属 `text-preview`），不包含语言识别的具体方法，不包含 Markdown、图片等其他类型的渲染呈现。

## Requirements

### Requirement: Syntax highlighted presentation

系统 SHALL 在被查看文本文件的语言被识别为语法高亮支持的语言时，以该语言的语法高亮形式呈现内容。

#### Scenario: A file in a highlighted language is viewed

- **WHEN** 用户查看一个文本文件，其语言被识别为语法高亮支持的语言
- **THEN** 内容以该语言的语法高亮形式呈现

#### Scenario: The same file is viewed repeatedly

- **WHEN** 用户多次查看同一文本文件且其语言识别结果未发生变化
- **THEN** 每次呈现的高亮形式一致

### Requirement: Content preservation under highlighting

系统 SHALL NOT 因语法高亮改变呈现内容的字符序列。用户从高亮呈现中取得的文本 SHALL 与文件内容一致。

#### Scenario: Text is taken from a highlighted presentation

- **WHEN** 用户从高亮呈现中取得全部呈现内容
- **THEN** 得到的字符序列与该文件的内容一致

#### Scenario: A file whose content contains markup-like text is viewed

- **WHEN** 用户查看一个内容形似 HTML 标记的文本文件，且其语言被识别
- **THEN** 呈现的是该标记文本的字面形式，不将其解释为可执行标记

### Requirement: Fallback to plain text presentation

语言未被识别时，系统 SHALL 以普通文本形式呈现内容。系统 SHALL NOT 因语言未被识别而拒绝提供内容、改变内容或向用户报告错误。

#### Scenario: A file whose language is not recognized is viewed

- **WHEN** 用户查看一个文本文件，其语言未被识别为语法高亮支持的语言
- **THEN** 内容以普通文本形式呈现，且呈现的字符序列与该文件的内容一致

### Requirement: Self-contained presentation resources

系统 SHALL 使语法高亮呈现所需的全部前端资源由服务自身以内嵌静态资源的形式提供。系统 SHALL NOT 要求浏览器从服务外部获取任何语法高亮呈现所需资源。

#### Scenario: The presentation resources are requested

- **WHEN** 客户端请求语法高亮呈现所需前端资源的路径
- **THEN** 系统返回成功响应，响应体为该资源内容
