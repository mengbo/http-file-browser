(function () {
  "use strict";

  var locationEl = document.getElementById("location");
  var errorEl = document.getElementById("error");
  var parentLinkEl = document.getElementById("parent-link");
  var entriesEl = document.getElementById("entries");
  var entriesHeaderEl = document.getElementById("entries-header");
  var emptyEl = document.getElementById("empty");
  var previewEl = document.getElementById("preview");

  // 错误提示按机器可读错误标识分支，而不是匹配服务端说明文本。
  var ERROR_TEXT = {
    not_found: "该位置不存在",
    permission_denied: "没有读取该位置的权限",
    not_a_directory: "该位置不是目录",
    not_a_regular_file: "该位置不是普通文件，无法预览",
    outside_root: "该位置超出浏览范围",
    not_text: "这是二进制文件，无法以文本预览",
    too_large: "文件过大，无法以文本预览"
  };

  var root = "";

  function currentPath() {
    return new URLSearchParams(window.location.search).get("path") || "";
  }

  // 浏览位置只放在 query 里：带 query 的 / 仍然是 /，不触碰静态资源与 /api/ 的分区边界。
  function urlFor(path) {
    return path === "" ? "/" : "/?path=" + encodeURIComponent(path);
  }

  function joinPath(base, name) {
    return base === "" ? name : base + "/" + name;
  }

  // parentOf 从相对根目录的路径推出上级相对路径，根目录自身与一级条目都得到空字符串。
  // 内容响应不带 parent（spec 只承诺 path 与 content），文件视图的上级入口因此在客户端算。
  function parentOf(path) {
    var cut = path.lastIndexOf("/");
    return cut < 0 ? "" : path.slice(0, cut);
  }

  function showError(text) {
    // 永远用 textContent 写入：文件名与服务端说明都可能含有需要被转义的字符。
    errorEl.textContent = text;
    errorEl.hidden = false;
  }

  function clearError() {
    errorEl.hidden = true;
    errorEl.textContent = "";
  }

  function locationText(path) {
    var relative = path === "" ? "" : "/" + path;
    return (root === "" ? "" : root) + relative;
  }

  function renderLocation(path) {
    locationEl.textContent = locationText(path);
  }

  function renderParent(list) {
    // 上级相对路径为空字符串既表示「上级就是根目录」，也表示「当前就是根目录」，
    // 因此用 path 是否为空来判定当前所在位置，parent 只作为上级入口的目标。
    if (list.path === "") {
      parentLinkEl.hidden = true;
      parentLinkEl.removeAttribute("href");
      return;
    }
    parentLinkEl.setAttribute("href", urlFor(list.parent));
    parentLinkEl.hidden = false;
  }

  function renderEntries(list) {
    entriesEl.replaceChildren();

    list.entries.forEach(function (entry) {
      var item = document.createElement("li");
      item.className = "entry";

      // 目录与文件一律是链接：是否可读文本由服务端判定（design D5、D8）。
      // 前端不给「可否预览」加一条分支，那份清单与服务端的清单必然漂移；
      // 点开非文本文件时服务端回 not_text，界面据此给一句解释。
      var link = document.createElement("a");
      // entry-name 表达「这是名称列」，entry-link 表达「这一行可点」。
      link.className = "entry-name entry-link";
      var target = joinPath(list.path, entry.name);
      link.setAttribute("href", urlFor(target));
      link.textContent = entry.name;
      link.addEventListener("click", function (event) {
        // 前进/后退由 History API 统一处理，不让浏览器整页重载。
        event.preventDefault();
        navigate(target);
      });
      item.appendChild(link);

      item.appendChild(textCell("entry-size", formatSize(entry.size)));
      item.appendChild(textCell("entry-time", formatModifiedAt(entry.modified_at)));

      entriesEl.appendChild(item);
    });

    emptyEl.hidden = list.entries.length !== 0;
    // 列名行跟着条目数走：零行时三个列名没有对应的列。
    // 隐藏靠 style.css 顶部的 [hidden] 兜底规则，不在这里另写 display（Change 02 观察 2）。
    entriesHeaderEl.hidden = list.entries.length === 0;
  }

  // textCell 建一个只装文本的单元格。条目名的 textContent 不变量覆盖全部三个字段：
  // 元信息不可得时传入空字符串，这一列就留空，而不是显示占位符（design D7）。
  function textCell(className, text) {
    var cell = document.createElement("span");
    cell.className = className;
    cell.textContent = text;
    return cell;
  }

  // formatSize 把字节数渲染为人类可读形式。
  function formatSize(bytes) {
    if (typeof bytes !== "number" || !isFinite(bytes)) {
      return "";
    }
    if (bytes < 1024) {
      return bytes + " B";
    }
    var units = ["KB", "MB", "GB", "TB"];
    var value = bytes / 1024;
    var index = 0;
    while (value >= 1024 && index < units.length - 1) {
      value = value / 1024;
      index = index + 1;
    }
    return value.toFixed(1) + " " + units[index];
  }

  // formatModifiedAt 把 Unix 整秒渲染为本时区的可读形式。
  // 服务端保证的是「取值」与时区无关（design D6），显示字符串本就是本地化的——
  // 同一目录在两台时区不同的机器上看到不同的日期字符串是正确的，不是缺陷。
  function formatModifiedAt(seconds) {
    if (typeof seconds !== "number" || !isFinite(seconds)) {
      return "";
    }
    // Date 构造器吃毫秒，秒/毫秒差一千倍是这里最容易犯的错。
    return new Date(seconds * 1000).toLocaleString();
  }

  // hideEntries 把列表视图整体收起来：文件视图与错误态都不显示条目与列名。
  function hideEntries() {
    entriesEl.replaceChildren();
    emptyEl.hidden = true;
    entriesHeaderEl.hidden = true;
  }

  // 两条 HTML 写入管线各有防线（Change 06 D2 的第一次收窄 + markdown-preview 的第二次收窄，形状不同）：
  // 高亮路径（文件体高亮、Markdown 渲染内的代码块高亮）靠输出形状校验——交给 HTML 解析器的字符串只能
  // 来自 highlight.js 的输出，且写入前经此校验：所有标签必须是 <span>（开标签可带且仅可带 class 属性），
  // 闭标签必须是裸 </span>；文本部分的 <、> 只能是已被转义的实体（&lt; 等），校验后残留任何裸 <、> 即整段拒绝。
  // Markdown 主输出靠配置性封闭——markdown-it 以 html:false 运行，解析器根本不为内嵌 HTML 开门，
  // 无需输出校验（markdown-preview design D2）。两套论证各自成立，互不替代。
  // 拒绝的方向都是回退素文本而非报错：呈现永远安全失败（spec: Fallback 的精神）。
  var SPAN_OPEN = /<span(?:\s+class="[^"]*")?>/g;
  var SPAN_CLOSE = /<\/span>/g;

  function safeHighlightHTML(html) {
    var stripped = html.replace(SPAN_OPEN, "").replace(SPAN_CLOSE, "");
    if (/[<>]/.test(stripped)) {
      return null;
    }
    return html;
  }

  function hidePreview() {
    previewEl.textContent = "";
    previewEl.hidden = true;
  }

  function render(list) {
    clearError();
    renderLocation(list.path);
    renderParent(list);
    renderEntries(list);
    hidePreview();
  }

  // D3 语言识别：名字优先、内容兜底的激进策略。这是「装饰」不是「门」——判错只是颜色不对，
  // spec 的 Content preservation 与 Fallback 两个 Requirement 兜底，永不改变内容、不报错。
  // 扩展名映射直接用 hljs 内建别名表（getLanguage），不另立白名单；无映射时跑 highlightAuto。
  // 与 text-preview 的文本判定零耦合：不被判为文本的文件根本到不了这里，两道门各管各的。
  // relevance 低于该阈值的 auto 结果（.txt、自然语言）视为未识别，回退素文本。
  var AUTO_MIN_RELEVANCE = 5;

  function extensionOf(path) {
    var name = path.slice(path.lastIndexOf("/") + 1);
    var dot = name.lastIndexOf(".");
    // 点开头的文件（.gitignore）没有扩展名：最后一个点就是首字符。
    return dot <= 0 ? "" : name.slice(dot + 1).toLowerCase();
  }

  // highlightHTMLFor 返回高亮 HTML，识别失败或低置信时返回 null（素文本回退由调用方处理）。
  function highlightHTMLFor(path, content) {
    var ext = extensionOf(path);
    try {
      var mapped = ext !== "" ? hljs.getLanguage(ext) : null;
      // 「Plain text」命中视同无映射（design D3）：.txt 一律走 auto/回退，不做无谓高亮。
      if (mapped && mapped.name !== "Plain text") {
        // 名字命中即按映射语言，不跑 auto（design D3）。
        return hljs.highlight(content, { language: ext, ignoreIllegals: true }).value;
      }
      var auto = hljs.highlightAuto(content);
      return auto.relevance >= AUTO_MIN_RELEVANCE ? auto.value : null;
    } catch (err) {
      // 识别或高亮抛错同样安全失败：素文本呈现，不向用户报错（spec: Fallback）。
      return null;
    }
  }

  // ============ Markdown 渲染管线（markdown-preview，design D1-D7）============

  // D3 识别：纯名字判定，扩展名 md/markdown 即渲染。装饰不是门：识别错的代价只是
  // 「本该素文本的东西被渲染」，.md 按惯例就是 Markdown。与 text-preview 的文本判定零耦合：
  // 不被判为文本的文件根本到不了这里，两道门各管各的。
  var MARKDOWN_EXTENSIONS = { md: true, markdown: true };

  function isMarkdownPath(path) {
    return MARKDOWN_EXTENSIONS[extensionOf(path)] === true;
  }

  // D6 代码块高亮钩子：fence 语言串经 hljs 别名表命中即按该语言 highlight；无标注（lang 为空）
  // 或未识别走 highlightAuto，relevance 低于 AUTO_MIN_RELEVANCE 视为未识别——与文件体识别
  // 同一套低置信回退哲学。返回空串让 markdown-it 走默认转义，该代码块即素文本。
  // hljs 输出仍过 safeHighlightHTML 形状校验（Change 06 D2 既有防线，见上）；校验不过或
  // 抛错只影响本块——该块回退素文本，文件其余部分照常渲染（spec: 语言未被识别的代码块以普通文本形式呈现）。
  // 返回的 <pre… 前缀串被 markdown-it 原样采用：hljs 类名接住 vendored 主题的着色。
  function fenceHighlightHTML(code, lang) {
    if (typeof hljs === "undefined") {
      return "";
    }
    try {
      var mapped = lang !== "" ? hljs.getLanguage(lang) : null;
      var html = null;
      if (mapped && mapped.name !== "Plain text") {
        html = hljs.highlight(code, { language: lang, ignoreIllegals: true }).value;
      } else {
        var auto = hljs.highlightAuto(code);
        if (auto.relevance >= AUTO_MIN_RELEVANCE) {
          html = auto.value;
        }
      }
      if (html === null || safeHighlightHTML(html) === null) {
        return "";
      }
      return '<pre class="hljs' + (mapped ? " language-" + lang : "") + '"><code>' + html + "</code></pre>";
    } catch (err) {
      return "";
    }
  }

  // D2 安全靠配置不靠消毒：html:false——解析器根本不为内嵌 HTML 开门，文件中形似 HTML 的
  // 标记文本以转义后的字面形式呈现，不引入 sanitizer（用户决策：不支持内嵌 HTML）。
  // javascript:/vbscript: 等危险 href 由 markdown-it 的 validateLink 默认拦截。
  // 实例惰性构建：vendored 脚本未就绪（typeof 兜底）只让 Markdown 分支回退素文本，不牵连 app.js 其余部分。
  var markdownRenderer = null;

  function markdownIt() {
    if (typeof markdownit !== "function") {
      return null;
    }
    if (markdownRenderer === null) {
      markdownRenderer = markdownit({ html: false, highlight: fenceHighlightHTML });
    }
    return markdownRenderer;
  }

  // D7 渲染写入与失败回退：构造或渲染抛错（病态输入、渲染器未就绪）返回 null，
  // 由调用方整段回退素文本、不向用户报错（spec: Rendering does not succeed）。
  function markdownHTMLFor(content) {
    try {
      var md = markdownIt();
      return md === null ? null : md.render(content);
    } catch (err) {
      return null;
    }
  }

  // ============ 渲染视图中的链接导航（design D4）============

  // SCHEME_RE 识别带 scheme 的 href（https:、mailto:、…）。markdown-it 的 validateLink 已
  // 拦截 javascript:/vbscript:/file: 等危险 scheme，剩余带 scheme 者皆为外部目标，交给浏览器。
  var SCHEME_RE = /^[a-z][a-z0-9+.-]*:/i;

  // resolveLinkTarget 把相对 href 以被查看文件所在目录 base 为基准，解析成应用内浏览路径
  // （urlFor 参数的相对根形式）。按 URL 语义处理：片段 #… 剥离；%XX 转义逐段解码（畸形转义
  // 保留原段，最坏结果 not_found 由既有错误态兜住）；"." 与空段（./、//、尾 /）剔除；
  // ".." 弹出一级（弹出根即止）。返回 "" 表示根目录。
  function resolveLinkTarget(href, base) {
    var rest = href;
    var hash = rest.indexOf("#");
    if (hash > -1) {
      rest = rest.slice(0, hash);
    }
    if (rest === "") {
      return null;
    }
    var parts = rest.charAt(0) === "/"
      ? rest.slice(1).split("/")
      : base === "" ? rest.split("/") : base.split("/").concat(rest.split("/"));
    var resolved = [];
    for (var i = 0; i < parts.length; i++) {
      var segment = parts[i];
      if (segment === "") {
        continue;
      }
      try {
        segment = decodeURIComponent(segment);
      } catch (err) {
        // 畸形转义保留原段。
      }
      if (segment === ".") {
        continue;
      }
      if (segment === "..") {
        resolved.pop();
        continue;
      }
      resolved.push(segment);
    }
    return resolved.join("/");
  }

  // rewriteRenderedLinks 在渲染结果写入 DOM 后遍历 a[href] 重写（design D4）：
  // 相对路径（含上级目录片段与 / 开头的根相对）→ 应用内导航；外部链接 → target="_blank"
  // rel="noopener" 新标签打开、当前呈现不变；# 页内锚点原样保留。重写一律经 setAttribute、
  // 不经 innerHTML 拼接，href 注入面为零。
  function rewriteRenderedLinks(container, filePath) {
    var base = parentOf(filePath);
    var links = container.querySelectorAll("a[href]");
    Array.prototype.forEach.call(links, function (link) {
      var href = link.getAttribute("href");
      if (href === "" || href.charAt(0) === "#") {
        return;
      }
      if (SCHEME_RE.test(href)) {
        link.setAttribute("target", "_blank");
        link.setAttribute("rel", "noopener");
        return;
      }
      var target = resolveLinkTarget(href, base);
      if (target === null) {
        return;
      }
      link.setAttribute("href", urlFor(target));
      // 与目录条目同一套导航：不让浏览器整页重载，前进/后退由 History API 统一处理。
      link.addEventListener("click", function (event) {
        event.preventDefault();
        navigate(target);
      });
    });
  }



  // renderContent 渲染文件视图。三条呈现路径、两种防线（见 safeHighlightHTML 上方注释）：
  // 素文本路径维持 Change 05 起的 textContent 不变承诺；高亮路径的字符串只能来自 hljs 输出且
  // 过形状校验；Markdown 路径的字符串来自 markdown-it（html:false 配置性封闭，形状校验不适用——
  // 那是高亮路径的防线）。素文本与高亮殊途同归：#preview 的 textContent 永远等于文件内容——
  // 高亮只是着色，不增删改任何字符（spec: Content preservation under highlighting）。
  // Markdown 路径的行为由 markdown-preview 定义：渲染未成功整段回退素文本、不报错。
  function renderContent(content) {
    clearError();
    hideEntries();
    renderLocation(content.path);
    // 文件的上级就是它所在的目录；文件位于根目录时该入口指向根目录本身。
    parentLinkEl.setAttribute("href", urlFor(parentOf(content.path)));
    parentLinkEl.hidden = false;
    if (isMarkdownPath(content.path)) {
      var markdownHTML = markdownHTMLFor(content.content);
      if (markdownHTML === null) {
        previewEl.className = "preview";
        previewEl.textContent = content.content;
      } else {
        // preview 类切换为 markdown：渲染排版样式挂在它上面，素文本与高亮样式不受牵连。
        previewEl.className = "preview markdown";
        previewEl.innerHTML = markdownHTML;
        rewriteRenderedLinks(previewEl, content.path);
      }
      previewEl.hidden = false;
      return;
    }
    // highlightHTMLFor 的 null（未识别/低置信/抛错）直接走素文本；
    // 只有高亮输出本身才需要过 D2 形状校验。
    var html = highlightHTMLFor(content.path, content.content);
    if (html !== null && safeHighlightHTML(html) !== null) {
      // hljs 类名接住 vendored 主题的着色。
      previewEl.className = "preview hljs";
      previewEl.innerHTML = html;
    } else {
      previewEl.className = "preview";
      previewEl.textContent = content.content;
    }
    previewEl.hidden = false;
  }

  function renderFailure(code, message) {
    hideEntries();
    hidePreview();
    parentLinkEl.hidden = true;
    parentLinkEl.removeAttribute("href");
    locationEl.textContent = locationText(currentPath());
    showError(ERROR_TEXT[code] || message || "无法打开该位置");
  }

  function fetchJSON(url) {
    return fetch(url, { headers: { Accept: "application/json" } }).then(function (response) {
      return response.json().then(function (body) {
        return { ok: response.ok, body: body };
      });
    });
  }

  function errorCode(result) {
    return result.body && result.body.error ? result.body.error.code : "";
  }

  function errorMessage(result) {
    return result.body && result.body.error ? result.body.error.message : "";
  }

  // load 按响应的成败分派这次渲染列表还是预览，不预判 ?path= 指向什么（design D2）。
  // 同一个位置参数在两个端点上指向不同类型的对象：列表成功就是目录，
  // not_a_directory 就是文件，转问内容端点；其余失败原因两边一致，直接报错。
  function load(path) {
    var listed = "/api/list?path=" + encodeURIComponent(path);
    var content = "/api/content?path=" + encodeURIComponent(path);

    fetchJSON(listed)
      .then(function (result) {
        if (result.ok) {
          render(result.body);
          return null;
        }
        if (errorCode(result) === "not_a_directory") {
          return fetchJSON(content);
        }
        renderFailure(errorCode(result), errorMessage(result));
        return null;
      })
      .then(function (result) {
        if (result === null) {
          return;
        }
        if (result.ok) {
          renderContent(result.body);
          return;
        }
        renderFailure(errorCode(result), errorMessage(result));
      })
      .catch(function (err) {
        renderFailure("", "无法连接服务：" + err.message);
      });
  }

  function navigate(path) {
    window.history.pushState({}, "", urlFor(path));
    load(path);
  }

  // 根目录来自 /api/health：浏览位置在 URL 与列表响应中都是相对路径，
  // 绝对位置由健康端点提供的根目录拼出。
  fetch("/api/health", { headers: { Accept: "application/json" } })
    .then(function (response) {
      return response.json();
    })
    .then(function (body) {
      root = body.root || "";
    })
    .catch(function () {
      root = "";
    })
    .then(function () {
      load(currentPath());
    });

  window.addEventListener("popstate", function () {
    load(currentPath());
  });
})();
