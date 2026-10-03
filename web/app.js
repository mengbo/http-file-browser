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

  // D2 形状校验：高亮路径上交给 HTML 解析器的字符串，只能来自 highlight.js 的输出，
  // 且写入前经此校验——所有标签必须是 <span>（开标签可带且仅可带 class 属性），
  // 闭标签必须是裸 </span>。文本部分的 <、> 只能是已被转义的实体（&lt; 等），
  // 校验后残留任何裸 <、> 即整段拒绝。拒绝的方向是回退素文本而非报错：
  // 呈现永远安全失败（spec: Fallback 的精神）。
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

  // renderContent 渲染文件视图。素文本路径维持 Change 05 起的 textContent 不变承诺；
  // 高亮路径是 D2 对 D11 的显式收窄：交给 HTML 解析器的字符串只能来自 highlight.js 的
  // 输出，且写入前经 safeHighlightHTML 形状校验，校验不过整段回退素文本。
  // 两条路径殊途同归：#preview 的 textContent 永远等于文件内容——高亮只是着色，
  // 不增删改任何字符（spec: Content preservation under highlighting）。
  function renderContent(content) {
    clearError();
    hideEntries();
    renderLocation(content.path);
    // 文件的上级就是它所在的目录；文件位于根目录时该入口指向根目录本身。
    parentLinkEl.setAttribute("href", urlFor(parentOf(content.path)));
    parentLinkEl.hidden = false;
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
