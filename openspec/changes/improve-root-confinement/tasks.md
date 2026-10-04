# Tasks

## 1. 判定核心：`resolve()` 物理化（design D2/D3）

- [x] 1.1 `browser` 增加物理根字段：构造时对根目录执行一次 `filepath.EvalSymlinks` 并保存结果；解析失败时服务启动报错退出（根不可解析则无边界可言），验证：以不存在的路径作根启动服务得到启动错误
- [x] 1.2 `resolve()` 在既有字面检查之后追加物理判定：`EvalSymlinks` 请求路径 → 与物理根做 `Rel` → `..` 开头返回 `outside_root`；`EvalSymlinks` 失败时不做越界判定、回落到既有 Stat 分类（design D3），代码注释如实记录 TOCTOU 窗口（design D5），验证：`go build ./...` 通过
- [x] 1.3 `browse_test.go`：既有「经软链出根仍提供」断言反转为期待 `outside_root`；新增用例——根内软链（指向根内另一位置）可列出、根路径自身经软链（包住 `t.TempDir` 的软链作根）全部正常、路径中间段经软链出根被拒、悬空软链返回 `not_found`，验证：`go test ./internal/server/` 全绿

## 2. 内容端点：测试反转（端点代码零变化，复用同一 `resolve()`）

- [x] 2.1 `content_test.go`：既有「经软链出根仍提供内容」断言反转为期待 `outside_root`，保留同一 fixture 钉住「物理出根必拒」，验证：`go test ./internal/server/ -run TestContent` 全绿
- [x] 2.2 `image_test.go`：既有「经软链出根仍提供图片」断言反转为期待 `outside_root`，保留同一 fixture，验证：`go test ./internal/server/ -run TestImage` 全绿

## 3. 全量验证与走查

- [x] 3.1 全量回归：`go test ./...` 全绿、`go vet ./...` 无告警、`GOOS=windows go build ./...` 交叉编译通过（ADR-0001 承诺不回退），验证：三条命令退出码均为 0
- [x] 3.2 `openspec validate --change improve-root-confinement` 通过，验证：命令退出码为 0
- [x] 3.3 浏览器走查一条（design D8）：启动服务后请求一个经软链指向根外的路径，确认前端错误说明正常出现且不显示内容，验证：走查观察到错误说明
- [x] 3.4 Windows 行为实证（design D6 / Open Questions）：在有 Windows 环境时验证 junction 与 8.3 短名下越界判定的实际行为，结论记入代码注释或 `docs/journal.md`；无环境时如实记录未验证，验证：`docs/journal.md` 有相应记录
