(function () {
  "use strict";

  var locationEl = document.getElementById("location");
  var errorEl = document.getElementById("error");
  var parentLinkEl = document.getElementById("parent-link");
  var entriesEl = document.getElementById("entries");
  var emptyEl = document.getElementById("empty");

  // 错误提示按机器可读错误标识分支，而不是匹配服务端说明文本。
  var ERROR_TEXT = {
    not_found: "该位置不存在",
    permission_denied: "没有读取该目录的权限",
    not_a_directory: "该位置不是目录",
    outside_root: "该位置超出浏览范围"
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

  function showError(text) {
    // 永远用 textContent 写入：文件名与服务端说明都可能含有需要被转义的字符。
    errorEl.textContent = text;
    errorEl.hidden = false;
  }

  function clearError() {
    errorEl.hidden = true;
    errorEl.textContent = "";
  }

  function renderLocation(list) {
    var relative = list.path === "" ? "" : "/" + list.path;
    locationEl.textContent = (root === "" ? "" : root) + relative;
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

      if (entry.type === "directory") {
        var link = document.createElement("a");
        link.className = "entry-link";
        link.setAttribute("href", urlFor(joinPath(list.path, entry.name)));
        link.textContent = entry.name;
        link.addEventListener("click", function (event) {
          // 前进/后退由 History API 统一处理，不让浏览器整页重载。
          event.preventDefault();
          navigate(joinPath(list.path, entry.name));
        });
        item.appendChild(link);
      } else {
        // 文件在本 Change 不可进入，因此不是链接，也不呈现可点击外观。
        var label = document.createElement("span");
        label.className = "entry-name";
        label.textContent = entry.name;
        item.appendChild(label);
      }

      entriesEl.appendChild(item);
    });

    emptyEl.hidden = list.entries.length !== 0;
  }

  function render(list) {
    clearError();
    renderLocation(list);
    renderParent(list);
    renderEntries(list);
  }

  function renderFailure(code, message) {
    entriesEl.replaceChildren();
    emptyEl.hidden = true;
    parentLinkEl.hidden = true;
    parentLinkEl.removeAttribute("href");
    locationEl.textContent = (root === "" ? "" : root) + "/" + currentPath();
    showError(ERROR_TEXT[code] || message || "无法打开该位置");
  }

  function load(path) {
    fetch("/api/list?path=" + encodeURIComponent(path), { headers: { Accept: "application/json" } })
      .then(function (response) {
        return response.json().then(function (body) {
          return { ok: response.ok, body: body };
        });
      })
      .then(function (result) {
        if (!result.ok) {
          renderFailure(result.body && result.body.error ? result.body.error.code : "", result.body && result.body.error ? result.body.error.message : "");
          return;
        }
        render(result.body);
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
