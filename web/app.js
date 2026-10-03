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

  // renderContent 渲染文件视图。内容经 textContent 写入，绝不拼接 innerHTML：
  // Change 02 的条目名不变量在这里扩展到文件内容本身（design D11）。
  // 顺带说明为什么内容走 JSON 通道就是安全的：内容从不交给 HTML 解析器，
  // 因此一个以 HTML 注释或标签开头的 .html 文件不会变成存储型 XSS。
  function renderContent(content) {
    clearError();
    hideEntries();
    renderLocation(content.path);
    // 文件的上级就是它所在的目录；文件位于根目录时该入口指向根目录本身。
    parentLinkEl.setAttribute("href", urlFor(parentOf(content.path)));
    parentLinkEl.hidden = false;
    previewEl.textContent = content.content;
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
