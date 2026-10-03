# markdown-preview Specification

## Purpose
定义查看 Markdown 文件时渲染呈现的行为契约：哪些文件以渲染形式呈现、渲染如何处理文件内形似 HTML 的标记与链接图片等引用、代码块的呈现方式，以及渲染资源的自包含。不包含哪些文件可被读取为文本的判定（属 `text-preview`），不包含渲染呈现之外的语法高亮呈现（属 `syntax-highlighting`），不包含文件内容的修改与写回。

## Requirements

### Requirement: Markdown rendered presentation

对于扩展名为 `md` 或 `markdown` 的被查看文本文件，系统 SHALL 默认以 Markdown 渲染形式呈现。当该文件的呈现形式已被用户切换为源码形式（见 Presentation form switching）时，系统 SHALL NOT 对其适用 Markdown 渲染呈现。渲染呈现 SHALL NOT 将文件中形似 HTML 的标记解释为可执行标记，此类标记文本 SHALL 以字面形式呈现。以渲染形式呈现且渲染未成功时，系统 SHALL 以普通文本形式呈现该文件内容，SHALL NOT 因渲染未成功而拒绝提供内容或向用户报告错误。

#### Scenario: A markdown file is viewed

- **WHEN** 用户查看一个扩展名为 `md` 或 `markdown` 的文本文件，且其呈现形式未被切换为源码形式
- **THEN** 内容以 Markdown 渲染形式呈现

#### Scenario: Content resembling HTML markup is viewed

- **WHEN** 以渲染形式呈现的一个 Markdown 文件，其内容包含形似 HTML 标记的文本
- **THEN** 该标记文本以字面形式呈现，不作为可执行标记呈现

#### Scenario: Rendering does not succeed

- **WHEN** 系统以渲染形式呈现一个 Markdown 文件，且其内容未被成功渲染
- **THEN** 内容以普通文本形式呈现，且系统不向用户报告错误

### Requirement: Presentation form switching

对于被查看的 Markdown 文件，系统 SHALL 使用户能将其呈现形式在渲染形式与源码形式之间切换。切换只影响当前查看：离开被查看文件后再次查看同一文件时，系统 SHALL 以渲染形式呈现该文件，SHALL NOT 保留此前的源码形式选择。本 Requirement 只规定呈现形式的切换与默认，不规定源码形式的呈现形态（见 `syntax-highlighting`）。

#### Scenario: The presentation form is switched to source

- **WHEN** 用户将一个以渲染形式呈现的 Markdown 文件切换为源码形式
- **THEN** 该文件以源码形式呈现，不再以 Markdown 渲染形式呈现

#### Scenario: The presentation form is switched back to rendered

- **WHEN** 用户将一个以源码形式呈现的 Markdown 文件切换为渲染形式
- **THEN** 该文件以 Markdown 渲染形式呈现

#### Scenario: The presentation form does not persist across views

- **WHEN** 用户将一个 Markdown 文件切换为源码形式，随后离开该文件，并在新的查看中再次打开同一文件
- **THEN** 该文件以 Markdown 渲染形式呈现

### Requirement: Link navigation in rendered presentation

渲染视图中指向相对路径的链接被激活时，系统 SHALL 导航到以被查看文件所在目录为基准解析出的目标位置，并按该位置的浏览行为呈现。指向外部的链接被激活时，系统 SHALL 在新的浏览器标签页中打开该链接，当前呈现保持不变。

#### Scenario: A relative link is activated

- **WHEN** 用户激活渲染视图中一个指向相对路径的链接
- **THEN** 系统导航到以被查看文件所在目录为基准解析出的目标位置，并呈现该位置

#### Scenario: A relative link containing parent segments is activated

- **WHEN** 用户激活渲染视图中一个路径含有上级目录片段的相对链接
- **THEN** 系统导航到以被查看文件所在目录为基准解析上级目录片段后的目标位置，并呈现该位置

#### Scenario: An external link is activated

- **WHEN** 用户激活渲染视图中一个指向外部的链接
- **THEN** 该链接在新的浏览器标签页中打开，当前呈现保持不变

### Requirement: Image presentation in rendered presentation

系统 SHALL 在渲染视图中呈现文件引用的图片，由浏览器按图片的地址加载。图片地址无法加载时，系统 SHALL NOT 因此向用户报告错误，SHALL NOT 改变其余内容的呈现。

#### Scenario: An image referenced by an absolute URL is viewed

- **WHEN** 用户查看一个引用了以绝对地址表示的图片的 Markdown 文件
- **THEN** 该图片在渲染视图中加载并呈现

#### Scenario: An image whose address cannot be loaded is viewed

- **WHEN** 用户查看一个引用了无法加载图片的 Markdown 文件
- **THEN** 系统不向用户报告错误，该文件其余内容的渲染呈现保持不变

### Requirement: Code block highlighting in rendered presentation

渲染视图中，语言被识别的代码块 SHALL 以该语言的语法高亮形式呈现；语言未被识别的代码块 SHALL 以普通文本形式呈现。从高亮呈现的代码块中取得的文本 SHALL 与该代码块在文件中的内容一致。

#### Scenario: A code block with a recognized language is viewed

- **WHEN** 用户查看一个包含标注了被识别语言的代码块的 Markdown 文件
- **THEN** 该代码块以该语言的语法高亮形式呈现

#### Scenario: A code block without a recognized language is viewed

- **WHEN** 用户查看一个包含未标注语言或语言未被识别的代码块的 Markdown 文件
- **THEN** 该代码块以普通文本形式呈现

#### Scenario: Text is taken from a highlighted code block

- **WHEN** 用户从渲染视图中一个高亮呈现的代码块取得其全部文本
- **THEN** 得到的字符序列与该代码块在文件中的内容一致

### Requirement: Self-contained presentation resources

系统 SHALL 使 Markdown 渲染呈现所需的全部前端资源由服务自身以内嵌静态资源的形式提供。系统 SHALL NOT 要求浏览器从服务外部获取任何 Markdown 渲染呈现所需资源。

#### Scenario: The presentation resources are requested

- **WHEN** 客户端请求 Markdown 渲染呈现所需前端资源的路径
- **THEN** 系统返回成功响应，响应体为该资源内容
