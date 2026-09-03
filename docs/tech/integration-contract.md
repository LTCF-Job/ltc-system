---
doc_type: flow
covers: ["apps/web/src/api/client.ts", "apps/api/internal/platform/auth/", "apps/api/internal/platform/httpx/response.go"]
---

# 前後端整合契約

`frontend-framework.md` 的攔截器說明與 `backend-framework.md` 的 Auth／Response 說明是這條流程的兩半，這份文件把兩邊接起來，回答「一個畫面的請求從送出到顯示錯誤，中間完整經過哪些關卡」。單看某一支 API 的路由與角色，見 [backend-api-reference.md](backend-api-reference.md)；單看某個畫面打哪支 API，見 [frontend-pages.md](frontend-pages.md)。

## Trigger

使用者在畫面上觸發任一次 API 呼叫（頁面載入、送出表單、按下操作按鈕）。

## Steps

```
view 元件呼叫 src/api/*.ts
  > axios request 攔截器（src/api/client.ts）
      - 從 stores/auth.ts 取得 JWT，帶 Authorization: Bearer <token>
  > [本機 APP_ENV=local 且 token 為 mock_jwt_ 開頭] middleware.AuthMiddleware 依 token 字尾取角色，不驗簽，仍檢查 data plane
    [其餘情況] middleware.AuthMiddleware 用 Supabase JWKS 驗簽 JWT，並驗 iss／aud
      > 驗簽失敗或 iss／aud 不符 > 回 401 UNAUTHENTICATED，流程中止
      > 驗簽成功 > 只從 claim 的 app_metadata.role 取角色（user_metadata.role 與頂層 role claim 一律不採信，缺值時預設 viewer），寫入 Gin context（actor_id、actor_role）
  > 路由層 middleware.RequirePermission(module, action) 查角色目前的模組權限矩陣（含個人 customPermissions 覆蓋）
      > 對應 action 未授權 > 回 403 FORBIDDEN，流程中止
  > 對應能力模組的 transport handler 解 request、呼叫 app service、組 response（internal/modules/<capability>/transport/*.go）
  > internal/platform/httpx/response.go 統一封裝成功／失敗 envelope
  > axios response 攔截器（src/api/client.ts）
      - 成功：解出 response.data，view 拿到的就是 { data, meta }，不用再 .data.data
      - 401：自動登出並導回登入頁
      - 403：跳 ElNotification 警告
      - 其他錯誤：ElMessage 顯示 error.message；若有 error.details（欄位級錯誤陣列）逐欄列出
```

司機接送匯報是檔案上傳（`multipart/form-data`）而非 JSON，見 [backend-flows.md](backend-flows.md) 的匯報匯入流程；回應仍走同一套成功／錯誤封裝。

## Failure modes

- **前後端模組權限不同步**：前端 `hasPermission(module, action)` 與後端 `RequirePermission(module, action)` 現在共用同一份權限矩陣（透過 `GET /api/v1/auth/me` 取得，見 [frontend-permission-logic.md](frontend-permission-logic.md)），但新增頁面時若忘了在路由掛正確的 `meta.module`，或後端忘了在對應路由掛 `RequirePermission`，兩邊還是可能對不上，出現「畫面看得到但 API 403」或反過來。新增頁面或模組時，兩邊要對照 [frontend-pages.md](frontend-pages.md) 與 [backend-api-reference.md](backend-api-reference.md) 一起改。
- **契約型別脫鉤**：後端 DTO 改了欄位，前端 `src/types/api.d.ts` 沒重新產生（`npm run gen:types`）時不會在編譯期報錯，只會在 runtime 欄位對不上；`error.details` 的欄位陣列格式也只有前端攔截器單方面假設，沒有型別檢查保證後端一定回這個形狀。
- **本機免驗證憑證外洩到正式環境**：`mock_jwt_` 前綴憑證只在 `APP_ENV=local` 生效，寫死在 `internal/platform/auth/auth.go`；改動這段驗證邏輯時，判斷分支寫錯會讓正式環境也吃這種 token，等同繞過登入。
- **展示（demo）帳號被誤認為前端 mock**：`demo` 只是登入頁的帳號代稱，會被換成 `demo@ltc.example.com` 送進 Supabase 做真實驗證，資料則來自後端獨立的 demo 資料平面（依 JWT 的 `app_metadata.data_plane` 切換 `VITE_DEMO_API_BASE_URL`），前端已無任何 mock 攔截層；把它當成純前端假資料來排查問題會找錯方向。細節見 `frontend-framework.md` 的登入帳號代稱段落與 [`../decisions/demo-data-plane-architecture.md`](../decisions/demo-data-plane-architecture.md)。

## 刻意保留的契約落差

- **`DashboardStatsDTO` 的未回欄位**：前端型別宣告 7 個欄位，後端 `GET /dashboard/stats` 只真的回 `recentExports`，其餘欄位（來自不同資料源）目前由 `GET /dashboard/metrics` 分開提供，未合併進 `stats`。既有漂移，本輪只補上 `recentExports` 這一項，不擴大範圍統一兩支端點。
- **`import_error` 分頁的 `caseName` 語意**：`GET /rides/issues?issueType=import_error` 回傳的 `caseName` 欄位實際內容是 `driver_name_raw`（原始回報文字/欄位），不是個案姓名——該分頁前端標題本來就是「回報文字/欄位」，欄位名稱沿用共用 DTO 只是為了三種 `issueType` 共用同一個回應形狀。同理 `caseId` 回空字串、`legSeq` 回 `0`，前端這兩欄在該分頁本來就不顯示。

## Unverified

- `error.details` 的欄位名稱／巢狀規則沒有共用的型別定義或 schema 驗證，目前只能靠前後端各自的實作互相對齊，未找到自動化契約測試。
- **Supabase Admin API 的實際回應形狀**：本輪開發時 `SUPABASE_SERVICE_ROLE_KEY` 留空，`identity` 模組（`/users`、密碼變更）的請求／回應欄位（`app_metadata` vs `raw_app_meta_data`、`ban_duration` 停用語義、`grant_type=password` 的錯誤 payload）皆依官方文件推測，未經真實環境驗證。金鑰就位後第一件事應是打真實環境對照 `identity/infra/supabase_admin_client.go` 的假設。
