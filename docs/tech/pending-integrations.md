---
doc_type: flow
covers: ["apps/api/cmd/server/routes.go", "apps/api/internal/modules/ride/transport/ride_handler.go", "apps/api/internal/modules/reporting/transport/dashboard_handler.go", "apps/web/src/api/users.ts", "apps/web/src/api/roles.ts", "apps/web/src/api/notifications.ts", "apps/web/src/api/cases.ts", "apps/web/src/api/masters.ts"]
---

# 待串接功能串接紀錄

本文件原記錄「前端 UI 已實作、後端未串接」的 9 個功能缺口。現況是對應的 route／service code 已存在，但「完成串接」只代表靜態 wiring，不代表真實 Supabase、DB、外部 provider、pagination、error contract 或 production delivery 已驗證。路由與 permission 詳情以 [backend-api-reference.md](backend-api-reference.md) 為準，前後端契約落差以 [integration-contract.md](integration-contract.md) 與 [2026-09-04 Full-stack Review](../reviews/2026-09-04-full-stack-review.md) 為準。

---

## 已完成項目總覽

| 編號 | 功能分類 | 端點 / 模組 | 完成方式 |
| :--- | :--- | :--- | :--- |
| **1** | 儀表板最近匯出 | `GET /api/v1/dashboard/stats` | 讀取 `export_jobs` 最近 5 筆，回應形狀重用 `exportH` 的 `ExportJobDTO` |
| **2** | 異常搭乘集中清單 | `GET /api/v1/rides/issues` | 三分頁各自資料路徑：`conflict` 聚合 `ride_sources`、`unreported` 重用整月未回報查詢、`import_error` 讀 `form_submissions.anomaly_flags` |
| **3** | 混車衝突人工裁決 | `POST /api/v1/rides/:id/resolve-conflict` | 新增 `conflict_resolution_note` 欄位（migration 000014），裁決寫稽核 |
| **4** | 單筆搭乘紀錄查詢 | `GET /api/v1/rides/:id` | 補上實作 |
| **5** | 主檔刪除端點 | `DELETE /cases/:id`、`/vehicles/:id`、`/drivers/:id` | 軟刪除（`deleted_at`/`deleted_by`，migration 000015），車輛/司機刪除前檢查生效中綁定 |
| **6** | 通知收件人批次操作 | `POST /settings/notification-recipients/batch`、`/batch-delete` | 單一事務批次寫入，`topic`+`email` 去重 |
| **7** | 使用者帳號管理 | `GET/POST/PATCH/DELETE /users` | 新增 `identity` 模組，串接 Supabase Auth Admin API |
| **8** | 角色身分與權限管理 | `GET/POST/PATCH/DELETE /roles` | 新建 `roles` 資料表（migration 000016）並 seed 5 個系統角色 |
| **9** | 個人密碼修改 | `POST /auth/change-password` | 後端接管並真正驗證舊密碼，見 [password-change-server-side-verification.md](../decisions/password-change-server-side-verification.md) |

---

## 已知限制

- **Supabase Service Role Key／runtime 未驗證**：項目 7、8、9 所在的 `identity` module route code 已存在，但 production config 要求 `SUPABASE_SERVICE_ROLE_KEY`；缺少或 provider failure 時相關 endpoint 會回 service-unavailable，而不應退化成假資料。金鑰就位後仍需驗證 Admin API response shape、pagination、password flow、role metadata 與 self-admin safety，不能只以設定存在宣稱完成。
- **自訂角色的 API 存取層級**：已改為 `auth.RequirePermission` 直接查角色的模組權限矩陣（`view`/`edit`/`delete` 三軸），不再是 `viewer`/`staff`/`admin` 三字串白名單，詳見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md)。個人層級的 `customPermissions` 覆蓋也已接上 API 層，與角色矩陣同一套「查詢＋約 30 秒 process-local TTL 快取」機制（資料來源是 Supabase Admin API），詳見 [custom-permission-admin-api-enforcement.md](../decisions/custom-permission-admin-api-enforcement.md)。production 缺少 `SUPABASE_SERVICE_ROLE_KEY` 會被 config 拒絕啟動；local fallback 與 cache delay 仍需以 runtime 驗證。
- **匯入異常分頁的 `caseName` 語意**：`import_error` 分頁回傳的 `caseName` 實際內容是 `driver_name_raw`（回報文字/欄位），非個案姓名，詳見 [integration-contract.md](integration-contract.md)。
- **Supabase Admin API 回應形狀未經真實環境驗證**：金鑰留空狀態下無法對真實環境跑一次，欄位假設（`app_metadata` 命名、`ban_duration` 語義等）僅依官方文件推測，見 [integration-contract.md](integration-contract.md) 的 Unverified 段落。

---

## 三、已清理死碼紀錄

| 檔案路徑 | 清理說明 | 清理日期 |
| :--- | :--- | :--- |
| `apps/web/src/composables/useGooglePicker.ts`（已移除，historical） | 早期規劃之 Google Drive / Picker 試算表在線選擇模組（共 241 行）。因系統改由後端解析使用者上傳之本機 `.xlsx` 檔案，全專案無任何元件引用；此列只保留清理紀錄，不是現行檔案連結。 | 2026-09-02 |
