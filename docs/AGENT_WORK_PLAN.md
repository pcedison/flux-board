# Flux Board — Agent 工作接手規劃

最後更新：2026-04-17

## 閱讀順序（接手 Agent 必讀）

在開始任何工作前，請依序閱讀以下文件：

1. **本文件（`docs/AGENT_WORK_PLAN.md`）** — 完整工作規劃與禁止事項
2. `docs/STATUS_HANDOFF.md` — 目前各 Wave 狀態與已驗證證據快照
3. `docs/MASTER_PLAN.md` — 10-Wave 主計畫與完整執行日誌
4. `docs/ARCHITECTURE.md` — 目前架構與目標架構
5. `docs/DEPLOYMENT.md` — 部署假設、dry-run、回滾基線

---

## 專案現況摘要

### 簡化驗收規則
- `W0-W1` 的最終完成標準是 `artifact-complete`。
- `W2-W9` 的最終完成標準是 `remote-closed`。
- `locally-verified` 只表示目前 head 的本地驗證已通過，還不能視為最終關門。

### 目前狀態快照
- `W0-W1`：`artifact-complete`
- `W2-W8`：`locally-verified`
- `W9`：`in_progress`

### W6–W9 現況

**W6 — Go 模組化**
- 已完成：`internal/domain`、`internal/store/postgres`、`internal/service`、`internal/transport/http`、`cmd/flux-board`，以及 root `main` 的薄 shim 化
- 目前狀態：`locally-verified`
- 最終關門還差：記錄 exact current head 的 fresh GitHub Actions 綠燈

**W7 — 前端基礎**
- 已完成：React + TypeScript + Vite scaffold（`web/`）、React Router、TanStack Query、auth-aware 路由、typed mutation（建立/移動/封存/還原）、三欄看板元件化、`/` React runtime takeover、`/legacy/` 回滾路徑、`/next/*` 相容轉址
- 目前狀態：`locally-verified`
- 最終關門還差：push 後觀察新的 GitHub Actions 綠燈，並把 exact-head 證據記錄到 `MASTER_PLAN`

**W8 — Trello 級 UX**
- 已完成：非拖曳建立/移動/封存/還原、Lane 內上移/下移、焦點連續性、44px 觸控目標、a11y 語意
- 已完成補充：`3-A` same-lane pointer-first `dnd-kit` 漸進增強、`3-B` mobile-first layout、`3-C` keyboard/focus/a11y、viewport-aware smoke、drag smoke、keyboard smoke
- 目前狀態：`locally-verified`
- 最終關門還差：記錄 exact current head 與新 smoke lanes 的 fresh remote CI

**W9 — 品質閘門**
- 已完成：Go/web/race 驗證、Chromium + Firefox 煙霧測試、release dry-run 基線、request-id/access-log、健康探針
- 目前狀態：`in_progress`
- 最終關門還差：WebKit 閘門、結構化日誌（slog）、Prometheus 指標、版本策略、GitHub Release 工作流程，以及 exact-head remote CI

---

## 依賴關係圖

```
W6 深層 internal/ 套件化
    └→ W7 Runtime 所有權切換（後端分層完成後才安全）
        └→ W8 3-A / 3-B / 3-C（drag + mobile-first + keyboard/focus complete）
W9 可觀測性（slog、Prometheus）— 與 W6/W7 獨立，可並行
W9 版本策略/Release 治理 — 封閉所有 Wave 的最終前提
```

---

## 完整工作規劃

### 第一階段 — W6：建立真正的 Go 套件邊界（最優先執行）

此階段已完成。
後續 agent 應把這一節當作已完成的實作記錄，而不是待辦；真正的下一步是第二階段 `W7`。

---

#### 切片 1-A — 領域型別（最低風險，純搬移）

**目標**
- 建立 `internal/domain/task.go`

**具體工作**
- 將 `Task`、`ArchivedTask`、`taskReorderInput` 結構體從 `main.go` 移入
- 將哨兵錯誤（`errTaskConflict`、`errTaskNotFound`、`errTaskInvalidAnchor`、`errStoredTaskInvalid`）從 `task_repository.go` 移入
- 將 `task_validation.go` 的純驗證函式移入
- 其他所有 Go 檔案改為 import `flux-board/internal/domain`

**閘門**
- `go build ./...` 通過
- `go test ./...` 通過
- 行為零變更，無 API 異動

---

#### 切片 1-B — Store / Postgres 層

**目標**
- 建立 `internal/store/postgres/task_repository.go`
- 建立 `internal/store/postgres/auth_repository.go`

**具體工作**
- 將 `postgresTaskRepository` struct 與全部 SQL 查詢從 `task_repository.go` 移入 `internal/store/postgres/task_repository.go`
- 將 `auth_service.go` 中的 session 持久化、audit 記錄等 DB 操作移入 `internal/store/postgres/auth_repository.go`
- `TaskRepository` 介面保留在 `internal/domain/`（不依賴 postgres 具體實作）
- 根目錄的 `task_repository.go` 與 `auth_service.go` 中的 DB 操作部分對應刪除或轉為 delegation

**閘門**
- 現有 repository 測試在新套件路徑下通過
- `go vet ./...` 通過

---

#### 切片 1-C — 服務層

**目標**
- 建立 `internal/service/task/service.go`
- 建立 `internal/service/auth/service.go`

**具體工作**
- 將 `task_service.go` 的驗證與業務邏輯移入 `internal/service/task/service.go`
- 將 `auth_service.go` 的 auth 協調邏輯（登入流程、session 建立/撤銷）移入 `internal/service/auth/service.go`
- 服務層依賴 `internal/domain` 介面，不直接 import postgres 套件
- 根目錄對應檔案縮減為純委派或刪除

**閘門**
- 現有服務測試（`task_service_test.go`、`auth_service_test.go`）在新路徑下通過
- `main` 套件不再包含任何業務邏輯，僅剩組裝程式碼

---

#### 切片 1-D — HTTP Transport 層

**目標**
- 建立 `internal/transport/http/` 套件

**具體工作**
- `internal/transport/http/task.go` — 移入 `tasks_http.go` + `tasks_reorder.go`
- `internal/transport/http/auth.go` — 移入 `auth_http.go`、`auth_cookie.go`、`auth_context.go`、`auth_runtime.go`
- `internal/transport/http/health.go` — 移入 `health_http.go`
- `internal/transport/http/observability.go` — 移入 `server_observability.go`
- `internal/transport/http/server.go` — 移入 `server.go`（mux 組裝、HTTP server 建立）

**閘門**
- `server_test.go`、`task_http_test.go`、`server_observability_test.go` 在新路徑下通過
- `go test -race ./...`（需要 CGO + GCC）通過

---

#### 切片 1-E — `cmd/flux-board` 進入點（W6 最終閘門）

**目標**
- 建立 `cmd/flux-board/main.go`

**具體工作**
- `cmd/flux-board/main.go` 內容只有：載入設定 → 連接 DB → 組裝 App → 啟動伺服器
- 根目錄 `main.go` 成為單行轉接 shim（`func main() { cmdfb.Run() }`）或直接移除
- 確認 `go build ./cmd/flux-board` 產出可執行的二進位檔
- 更新 README、ARCHITECTURE.md、DEPLOYMENT.md 中所有對 `main.go` 或 `go run .` 的參照，改為 `go run ./cmd/flux-board`

**閘門**
- `go run ./cmd/flux-board` 正常啟動
- 根目錄 `main` 套件無任何 SQL、業務邏輯、驗證規則
- `go build ./...` 與 `go test ./...` 全部通過
- `./scripts/verify-go.ps1`（或 `.sh`）通過

---

### 第二階段 — W7：Runtime 所有權切換

**前提**：W6 第一階段全部切片通過，本地驗證與 CI 均為綠色。

切片 `2-A` 已於 2026-04-17 完成：`BoardSnapshotPage` 已拆為 `web/src/components/board/*` 的 lane/card/composer/archive/status 元件，`web/` 已補上元件級單元測試，且 preview runtime 已改為相對資產路徑加上 path-aware router basename，方便後續安全執行 `2-B`。

---

#### 切片 2-A — 看板頁面元件完善

**目標**
- 讓 `web/` 的看板頁面具備完整三欄佈局與完整 CRUD UI

**具體工作**
- 在 `BoardSnapshotPage.tsx` 中實作三欄 Lane 佈局（待辦 / 進行中 / 完成）
- 建立 `web/src/components/board/BoardLane.tsx` — 單一 lane 的容器元件
- 建立 `web/src/components/board/BoardTaskCard.tsx` — 單一卡片的呈現與操作元件
- 將 `useBoardMutations` 完整接入 Lane / Card 層級（建立、移動、封存、還原均可操作）
- 補充 `BoardTaskCard.test.tsx` 與 `BoardLane.test.tsx` 的前端單元測試
- 將 preview runtime 調整為相對資產路徑與 path-aware router basename，避免後續從 `/next/` 提升到 `/` 時重做前端打包設定

**閘門**
- `web/` 建置、型別檢查通過
- 前端測試涵蓋 Lane 渲染、Card 渲染、mutation 觸發
- `./scripts/verify-next-preview.ps1` 在 Docker-backed PostgreSQL 下通過，確認 `/next/` 仍可真實登入並完成 CRUD smoke

---

#### 切片 2-B — Runtime 所有權切換（W7 最終閘門）

**前提**：切片 2-A 完成，`/next/` 在 Chromium + Firefox 煙霧測試均通過。

切片 `2-B` 已於 2026-04-17 在本地完成：`/` 現在由 Go 提供 React `web/dist` runtime，`/legacy/` 提供舊版嵌入前端作回滾，`/next/*` 轉址到 canonical root routes，root smoke 改為 `/login -> /board`，而 `verify-next-preview` 改為驗證 `/next/*` 相容轉址與 `/legacy/` 回滾路徑。剩餘的唯一封板條件是 push 後觀察新的 GitHub Actions 全綠。

**具體工作**
- 變更預設路由：`/` 改為提供 React（透過 `web/dist`），不再是 `static/index.html`
- 保留 `static/index.html` 在 `/legacy/` 路由作為緊急回滾路徑
- 將 `web_preview.go` 重新命名為 `web_serve.go`，掛載於 `/`，加入 SPA fallback（所有非 API 路由回傳 `index.html`）
- 更新煙霧測試腳本目標為 `/login`（而非 `/next/login`）
- 更新 `verify-next-preview.ps1/.sh` 以反映新的路由結構

**閘門**
- `./scripts/verify-smoke.ps1` 目標 `/login` 通過（Chromium + Firefox）
- `/legacy/` 可正常存取舊版嵌入前端
- Auth / CRUD 流程無任何退化
- CI 全部綠色

---

### 第三階段 — W8：Trello 級互動體驗

**前提**：第二階段 2-B 完成，React 已取得 `/` 生產環境 runtime 所有權。

---

#### 切片 3-A — `dnd-kit` 漸進增強

**狀態**
- 已於 2026-04-17 完成目前規劃中的最小安全切片。

**實際完成內容**
- 已安裝 `@dnd-kit/core`、`@dnd-kit/sortable`、`@dnd-kit/utilities`
- 看板已用 `<DndContext>` + `PointerSensor` 包裹，Lane 以 `<SortableContext>` 表達 same-lane reorder
- 新增純 helper `getSameLaneDragMove(tasks, activeId, overId)`，將 drag end 轉成既有 `MoveTaskRequest`
- 卡片新增可見 `Drag` handle，但保留既有 Move up / Move down / lane move / Archive 按鈕作為穩定 fallback
- 這一輪刻意只做 same-lane、pointer-first；沒有加入 `KeyboardSensor`、沒有把整張卡片變成 drag activator、沒有加入 cross-lane drag
- 已補上單元測試與獨立 browser smoke，並在 `chromium` / `firefox` 本地驗證通過

**後續銜接**
- `3-C` 會承接鍵盤模型、焦點行為與顯式 a11y 驗證；不要在此切片回頭擴大 3-A 範圍

---

#### 切片 3-B — 行動優先佈局

**狀態**
- 已於 2026-04-17 完成目前規劃中的 mobile-first slice。

**實際完成內容**
- `web/src/styles.css` 已補上較早啟動的 stack/single-column 行為，board side panel 現在會在窄視窗與平板寬度更早收斂
- lane/card header 已能在窄寬度下換行，不再容易把 badge 和長標題擠壞
- side panel、field grid、archive item 在手機與窄平板上已更早簡化
- viewport-aware smoke 已補上 desktop / phone layout 斷言，並在 `390x844` 手機寬度下驗證通過

**後續銜接**
- `3-C` 應在目前的 mobile-first baseline 上補 keyboard/focus/a11y，不需要重做版面主結構

---

#### 切片 3-C — 鍵盤與焦點細化（已完成）

**狀態**
- 已於 2026-04-17 完成目前規劃中的 keyboard/focus/a11y slice。

**實際完成內容**
- Lane 之間完成 `roving tabindex` 導航（左右方向鍵切換 Lane）
- Lane 內可用上下方向鍵切換卡片焦點
- `m` 快捷鍵與工具列「移動到...」對話框皆保留鍵盤可操作性
- 移動/建立/封存/還原成功後，焦點會回到操作後的正確位置
- 已補上 axe 覆蓋與 keyboard smoke，並在 `chromium` / `firefox` 本地驗證通過

**閘門**
- W8 現在已完成；接下來的主線是 W9

---

### 第四階段 — W9：可觀測性、版本策略與 Release 治理

這些工作大部分與第二/三階段獨立，可與 W6/W7/W8 並行推進。

---

#### 切片 4-A — 結構化日誌（Go `log/slog`）

**具體工作**
- 在 `internal/transport/http/server.go`（或 `app_state.go`）中初始化 `slog.Logger`
  - `APP_ENV=production`：JSON handler（`slog.NewJSONHandler`）
  - 其他：文字 handler（`slog.NewTextHandler`）
- 將 `log.Printf` / `log.Fatalf` 呼叫全面替換為 `slog.InfoContext` / `slog.ErrorContext`
- `server_observability.go` 中的 access-log middleware 改為寫入結構化記錄（含 request-id、method、path、status、duration）
- 透過 `App` 結構體或 context 傳遞 `*slog.Logger`，不使用全域 logger

**閘門**
- `APP_ENV=production go run ./cmd/flux-board` 輸出 JSON 格式日誌
- 整合測試（`main_integration_test.go`）仍通過
- `go test ./...` 通過

---

#### 切片 4-B — Prometheus 指標端點

**具體工作**
- 引入 `github.com/prometheus/client_golang`（執行 `go get`）
- 在 `server_observability.go` 的 middleware 中，每個請求完成後更新以下指標：
  - `http_requests_total{method, path, status_code}`（Counter）
  - `http_request_duration_seconds{method, path}`（Histogram）
- 在 `health_http.go` 旁邊新增 `GET /metrics` handler，使用 `promhttp.Handler()`
- `/metrics` 路由無需認證（與 `/healthz` 相同）
- 在 `DEPLOYMENT.md` 新增 Prometheus scrape target 設定範例

**閘門**
- `curl http://localhost:8080/metrics` 回傳 Prometheus 文字格式
- 執行幾個 API 請求後，計數器數值正確遞增
- 新增對應的單元測試或整合測試

---

#### 切片 4-C — WebKit 瀏覽器閘門（W9 瀏覽器矩陣）

**具體工作**
- 在 CI 的 smoke job matrix 中新增 `webkit`：
  ```yaml
  strategy:
    matrix:
      browser: [chromium, firefox, webkit]
  ```
- 更新 `verify-smoke.ps1/.sh` 以接受 `PLAYWRIGHT_BROWSER=webkit`
- 確認 Playwright 已安裝 WebKit（`npx playwright install webkit`）
- 在 CI 工作流程中確保 WebKit 的系統依賴已安裝（`npx playwright install-deps webkit`）

**閘門**
- Chromium + Firefox + WebKit 在 CI 的登入/建立/封存/還原流程均為綠色
- GitHub Actions 顯示三個 browser 的 smoke job 全部通過

---

#### 切片 4-D — 版本策略與 CHANGELOG

**具體工作**
- 建立 `VERSION` 檔案，內容為 `0.1.0`
- 建立 `CHANGELOG.md`，採用 [Keep a Changelog](https://keepachangelog.com/) 格式，補齊 W0–W9 的主要里程碑
- 更新 `scripts/release-dry-run.ps1/.sh`：
  - 讀取 `VERSION` 檔案
  - 將版本號嵌入二進位檔名（例如 `flux-board-v0.1.0-linux-amd64`）
- 在 `DEPLOYMENT.md` 文件化 Release 流程：
  1. 更新 `VERSION` 檔案
  2. 更新 `CHANGELOG.md`
  3. 執行 `./scripts/release-dry-run.sh` 驗證
  4. 推送標籤 `vX.Y.Z`
  5. CI Release 工作流程自動執行（見切片 4-E）

**閘門**
- `./scripts/release-dry-run.sh` 產出命名正確的二進位檔，例如 `flux-board-v0.1.0-linux-amd64`
- `CHANGELOG.md` 存在且格式正確

---

#### 切片 4-E — GitHub Release 工作流程（W9 接近最終閘門）

**具體工作**
- 建立 `.github/workflows/release.yml`，由 `on: push: tags: ['v*']` 觸發
- 工作步驟：
  1. `go build` for Linux（`GOOS=linux GOARCH=amd64`）
  2. `go build` for macOS（`GOOS=darwin GOARCH=amd64`）
  3. `go build` for Windows（`GOOS=windows GOARCH=amd64`）
  4. 計算每個二進位檔的 SHA-256 校驗碼
  5. 使用 `softprops/action-gh-release` 建立 GitHub Release
  6. 上傳三個平台二進位檔與校驗碼檔案
- 在測試環境推送一個真實的 `v0.1.0` 標籤，觀察工作流程實際執行

**閘門**
- 推送 `v0.1.0` 標籤後，GitHub Release 頁面出現三個平台的二進位檔與 SHA-256 校驗碼
- Release 名稱與 body 自動帶入 `CHANGELOG.md` 對應章節

---

### 第五階段 — W9 最終閘門與企業擴充縫合點

---

#### 切片 5-A — OpenTelemetry Trace 縫合點（可選）

**具體工作**
- 引入 `go.opentelemetry.io/otel`
- 在 DB 查詢（`store/postgres`）與 auth 事件（`service/auth`）周圍加入 trace span
- Exporter 可插拔：
  - 預設：no-op exporter
  - 若設定 `OTEL_EXPORTER_OTLP_ENDPOINT` 環境變數，則使用 OTLP exporter
- 這是縫合點設計，不需要完整的 tracing pipeline 或外部 collector

**閘門**
- `go build ./...` 通過
- 不設定 `OTEL_EXPORTER_OTLP_ENDPOINT` 時，應用行為無變化
- ARCHITECTURE.md 更新以描述 trace 擴充點

---

#### 切片 5-B — 企業擴充縫合點（W9-P4，W9 最終閘門）

**具體工作**
- 在 `migrations/` 中新增文件化的 schema 擴充路徑（不實作，以 SQL 註解描述）：
  - `users` 資料表的 RBAC 擴充點：`roles` 欄位、`workspace_id` FK
  - `sessions` 資料表的 SSO 擴充點：`external_id`、`provider` 欄位
- 在 `ARCHITECTURE.md` 新增「企業擴充縫合點」章節：
  - 描述 RBAC、SSO、多工作區的介面設計
  - 說明哪些服務介面已可插拔、哪些需要後續擴充
- 確認所有服務介面（`TaskRepository`、auth service）使用 interface 而非具體實作，以便後續替換

**閘門**
- `ARCHITECTURE.md` 有「企業擴充縫合點」章節
- 現有測試全部通過
- 無任何未完成的實作被意外引入

---

## 優先執行順序

| 優先度 | 切片 | Wave | 原因 |
|---|---|---|---|
| 1 | 1-A 領域型別 | W6 | 解除後續所有阻礙；純安全搬移，風險最低 |
| 2 | 1-B Store/Postgres | W6 | 最大的套件隔離缺口 |
| 3 | 1-C 服務層 | W6 | 讓服務測試可獨立於 HTTP 層執行 |
| 4 | 4-A 結構化日誌 | W9 | 與主線完全獨立；診斷價值高，可立即執行 |
| 5 | 1-D HTTP Transport 層 | W6 | 完成 W6 分層拆分 |
| 6 | 2-A 看板元件完善 | W7 | Runtime 切換的前提 |
| 7 | 4-C WebKit 閘門 | W9 | 獨立；完成三瀏覽器矩陣 |
| 8 | 1-E `cmd/flux-board` | W6 | W6 最終閘門 |
| 9 | 2-B Runtime 所有權切換 | W7 | W7 最終閘門；解除 W8 全部阻礙 |
| 10 | 3-A dnd-kit 拖放 | W8 | 依賴穩定的生產環境 runtime |
| 11 | 4-B Prometheus 指標 | W9 | 與主線獨立；完成可觀測性基線 |
| 12 | 3-B 行動優先佈局 | W8 | 可與 3-A 並行執行 |
| 13 | 3-C 鍵盤/焦點細化 | W8 | 已完成，W8 最終閘門已通過 |
| 14 | 4-D 版本策略 + CHANGELOG | W9 | Release 工作流程的前提 |
| 15 | 4-E GitHub Release 工作流程 | W9 | W9 接近最終閘門 |
| 16 | 5-A OTel Trace 縫合點 | W9 | 企業擴充（可選） |
| 17 | 5-B 企業 Schema 縫合點 | W9 | W9 + 專案最終閘門 |

---

## Wave 完成標準

| Wave | 目前狀態 | 最終完成標準 |
|---|---|---|
| W6 | `locally-verified` | exact current head 的 GitHub Actions 證據已記錄在 `MASTER_PLAN` |
| W7 | `locally-verified` | exact current head 的 root runtime takeover CI run 已觀察並記錄 |
| W8 | `locally-verified` | exact current head 的 `dnd_smoke`、`keyboard_smoke`、root smoke、preview smoke CI 均為綠色且證據已記錄 |
| W9 | `in_progress` | 完成 observability / release / browser-matrix scope，並記錄 exact current head 的完整 workflow 綠燈 |

---

## 每個切片的驗證流程（必須遵守）

每完成一個切片，必須依序執行以下驗證，通過後才能進行下一個切片：

```
1. go build ./...
2. go vet ./...
3. go test ./...
4. ./scripts/verify-go.ps1（或 .sh）
5. （影響 runtime 的切片）./scripts/verify-smoke.ps1（需 Docker + PostgreSQL）
6. （W7/W8 切片）./scripts/verify-web.ps1（或 .sh）
7. 推送後等待 GitHub Actions 確認綠色
8. 更新 docs/MASTER_PLAN.md 的 Execution Log
```

---

## 禁止事項（絕對不能做）

### 架構禁止

- **禁止將 `internal/` 套件拆分作為一個巨型 commit 合併**
  每個切片（1-A、1-B、1-C、1-D、1-E）必須獨立提交並有獨立的 CI 驗證證據，不可一次性合併所有變更。

- **禁止在 W6 第一階段（切片 1-A 到 1-E）完成並通過 CI 驗證前，嘗試 Runtime 所有權切換**
  在後端還是 `package main` 的情況下切換 runtime，會讓後續的 `internal/` 搬移更加複雜且風險更高。

- **禁止在 Runtime 所有權切換（切片 2-B）前實作 dnd-kit**
  在預覽路由 `/next/` 上開發拖放功能，切換後必須重建，徒增工作量。

### 前端禁止

- **禁止將 drag-and-drop 設計為唯一的移動路徑**
  拖放必須是漸進增強。按鈕式的上移/下移（切片 3-B）必須始終可用作備用路徑。

- **禁止靜默宣稱 `/next/` 已是生產環境 runtime 擁有者**
  在切片 2-B 明確執行並驗證前，文件和 README 中不能更改此陳述。

### 日誌與可觀測性禁止

- **禁止在結構化日誌（切片 4-A）完成前加入 Prometheus 指標**
  指標需要與日誌關聯（透過 request-id）。沒有結構化日誌的情況下，指標的診斷價值大幅降低。

- **禁止使用全域 logger 變數**
  `slog.Logger` 必須透過 `App` 結構體或 context 傳遞，確保可測試性。

### Release 禁止

- **禁止在 GitHub Release 工作流程（切片 4-E）實際以真實標籤觸發並觀察到執行通過前，宣稱 W9「完成」**

- **禁止在 `CHANGELOG.md` 存在前推送版本標籤**

### 範圍禁止

- **禁止重新開啟 W0–W5**
  除非出現真正的退化（regression），否則不得修改已完成的 W0–W5 工作。

- **禁止未更新 `docs/MASTER_PLAN.md` 的 Execution Log 就合併主要新範圍**
  每個切片完成後必須在 Execution Log 追加記錄。

- **禁止宣稱企業級 auth（多用戶、RBAC、OIDC）已完成**
  目前的 auth 模型是安全的單一管理員基線，不是多用戶或 OIDC 模型。

- **禁止在沒有 DB-backed 整合測試證據的情況下關閉任何影響 auth/session/reorder 的切片**

### 其他禁止

- **禁止跳過 `--no-verify`**
  不得繞過 git hooks 或 CI 閘門。

- **禁止硬編碼 `APP_PASSWORD` 或任何憑證到任何追蹤的檔案中**

- **禁止使用 `go test` 的 `-count=0` 或快取結果作為通過證據**
  必須使用 `-count=1` 確保測試實際執行。

---

## 目前版本庫檔案佈局（接手時的狀態）

```
cmd/
  flux-board/
    main.go                     ← canonical command entrypoint
internal/
  config/
    config.go
    config_test.go
  domain/
  service/
    auth/
    task/
  store/
    postgres/
  transport/
    http/
main.go                         ← root compatibility shim / thin wiring
static/index.html               ← legacy rollback frontend served on `/legacy/`
web/                            ← React + TypeScript + Vite scaffold
  src/
    app/App.tsx
    components/
      AppShell.tsx
      QueryState.tsx
      board/
    lib/
      api.ts
      queryClient.ts
      useAuthSession.ts
      useBoardMutations.ts
      useBoardSnapshot.ts
    routes/
      BoardSnapshotPage.tsx
      LoginPage.tsx
      OverviewPage.tsx
scripts/
  verify-go.ps1 / .sh
  verify-go-race.ps1 / .sh
  verify-web.ps1 / .sh
  verify-smoke.ps1 / .sh
  verify-next-preview.ps1 / .sh
  verify-dnd-smoke.ps1 / .sh
  release-dry-run.ps1 / .sh
.github/workflows/ci.yml        ← verify + smoke + dnd_smoke + keyboard_smoke + preview_smoke
docs/
  MASTER_PLAN.md
  STATUS_HANDOFF.md
  ARCHITECTURE.md
  DEPLOYMENT.md
  AGENT_WORK_PLAN.md            ← 本文件
```

---

## 中斷後恢復規則

若工作再次中斷，下一個 Agent 必須：

1. 閱讀本文件（`docs/AGENT_WORK_PLAN.md`）
2. 閱讀 `docs/STATUS_HANDOFF.md` 確認目前 Wave 狀態
3. 閱讀 `docs/MASTER_PLAN.md` 的最新 Execution Log 條目
4. 執行 `git log --oneline -10` 確認目前分支與最新提交
5. 確認 GitHub Actions 最新一次 CI run 的狀態
6. 只挑選下一個最小的已驗證切片開始工作，不跳過步驟

## Current Verification Note
- `W0-W1` 已達各自的最終完成標準：`artifact-complete`。
- `W2-W8` 已有本地驗證證據，但最終仍要以 `remote-closed` 為準。
- `W6-W8` 近期已重新審核並再次完成本地驗證。
- Windows 上不要把 `go test ./...` 和 `npm ci` 並行執行；掃描 `web/node_modules` 可能短暫失敗。
