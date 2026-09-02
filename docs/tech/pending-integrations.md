---
doc_type: flow
covers: ["apps/api/cmd/server/routes.go", "apps/api/internal/modules/ride/transport/ride_handler.go", "apps/api/internal/modules/reporting/transport/dashboard_handler.go", "apps/web/src/api/users.ts", "apps/web/src/api/roles.ts", "apps/web/src/api/notifications.ts", "apps/web/src/api/cases.ts", "apps/web/src/api/masters.ts"]
---

# 待串接功能串接紀錄

本文件原記錄「前端 UI 已實作、後端未串接」的 9 個功能缺口。全部已於本輪完成串接，本文件改為紀錄串接後的實際契約與已知限制；路由與角色詳情以 [backend-api-reference.md](backend-api-reference.md) 為準，前後端契約落差以 [integration-contract.md](integration-contract.md) 為準。

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

- **Supabase Service Role Key 留空**：項目 7、8、9 所在的 `identity` 模組已完整實作，但 `SUPABASE_SERVICE_ROLE_KEY` 於本輪部署中留空。金鑰未設定時，`/users`、`/roles`、`/auth/change-password` 等端點一律回 `503`（`CodeServiceUnavailable`），不會退化成假資料。待金鑰就位後即可直接運作，無需再改程式。
- **自訂角色的 API 存取層級**：已改為 `auth.RequirePermission` 直接查角色的模組權限矩陣（`view`/`edit`/`delete` 三軸），不再是 `viewer`/`staff`/`admin` 三字串白名單，詳見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md)。個人層級的 `customPermissions` 覆蓋仍只影響前端 UX，尚未接上 API 層，是這次刻意保留、未擴大的既有落差。
- **匯入異常分頁的 `caseName` 語意**：`import_error` 分頁回傳的 `caseName` 實際內容是 `driver_name_raw`（回報文字/欄位），非個案姓名，詳見 [integration-contract.md](integration-contract.md)。
- **Supabase Admin API 回應形狀未經真實環境驗證**：金鑰留空狀態下無法對真實環境跑一次，欄位假設（`app_metadata` 命名、`ban_duration` 語義等）僅依官方文件推測，見 [integration-contract.md](integration-contract.md) 的 Unverified 段落。

---

## 三、已清理死碼紀錄

| 檔案路徑 | 清理說明 | 清理日期 |
| :--- | :--- | :--- |
| [`apps/web/src/composables/useGooglePicker.ts`](file:///c:/Users/jacky/Documents/ltc-system/apps/web/src/composables/useGooglePicker.ts) | 早期規劃之 Google Drive / Picker 試算表在線選擇模組（共 241 行）。因系統已全面改由後端直接解析使用者上傳之本機 `.xlsx` 檔案，全專案無任何元件引用，已完整移除。連同 `apps/web/.env.example` 內已無引用的 `VITE_GOOGLE_*` 三個變數一併清除。 | 2026-09-02 |
