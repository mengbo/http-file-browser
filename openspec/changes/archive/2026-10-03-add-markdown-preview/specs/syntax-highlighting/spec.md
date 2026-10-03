# Spec Delta — syntax-highlighting

## MODIFIED Requirements

### Requirement: Syntax highlighted presentation

系统 SHALL 在被查看文本文件的语言被识别为语法高亮支持的语言时，以该语言的语法高亮形式呈现内容；但被 Markdown 渲染呈现的文件除外（见 `markdown-preview`），此类文件以 Markdown 渲染形式呈现，系统 SHALL NOT 对其适用语法高亮呈现。

#### Scenario: A file in a highlighted language is viewed

- **WHEN** 用户查看一个文本文件，其语言被识别为语法高亮支持的语言
- **THEN** 内容以该语言的语法高亮形式呈现

#### Scenario: The same file is viewed repeatedly

- **WHEN** 用户多次查看同一文本文件且其语言识别结果未发生变化
- **THEN** 每次呈现的高亮形式一致

#### Scenario: A file presented as rendered markdown is viewed

- **WHEN** 用户查看一个被 Markdown 渲染呈现的文件
- **THEN** 内容以 Markdown 渲染形式呈现，不以语法高亮形式呈现
