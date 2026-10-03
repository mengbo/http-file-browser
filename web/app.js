(function () {
  "use strict";

  var statusEl = document.getElementById("status");

  fetch("/api/health")
    .then(function (response) {
      if (!response.ok) {
        throw new Error("HTTP " + response.status);
      }
      return response.json();
    })
    .then(function (body) {
      statusEl.textContent = "服务就绪（status: " + body.status + "）";
      statusEl.classList.add("status--ok");
    })
    .catch(function (err) {
      statusEl.textContent = "服务不可用：" + err.message;
      statusEl.classList.add("status--error");
    });
})();
