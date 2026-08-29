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
      - 從 stores/auth.ts 取得 Supabase JWT，帶 Authorization: Bearer <token>
      - 若 VITE_ENABLE_MSW=true 且已登入，額外帶 X-Mock-Role / X-Mock-User-ID
  > [本機 APP_ENV=local 且帶 X-Mock-Role] middleware.AuthMiddleware 直接採信 header 角色，略過 JWT 驗證
    [其餘情況] middleware.AuthMiddleware 用 Supabase JWKS 驗簽 JWT
      > 驗簽失敗 > 回 401 UNAUTHENTICATED，流程中止
      > 驗簽成功 > 從 claim 的 user_metadata.role（或 app_metadata.role）取角色，寫入 Gin context（actor_id、actor_role）
  > 路由層 middleware.RequireRoles(...) 比對角色白名單
      > 角色不在白名單 > 回 403 FORBIDDEN，流程中止
  > 對應能力模組的 transport handler 解 request、呼叫 app service、組 response（internal/modules/<capability>/transport/*.go）
  > internal/platform/httpx/response.go 統一封裝成功／失敗 envelope
  > axios response 攔截器（src/api/client.ts）
      - 成功：解出 response.data，view 拿到的就是 { data, meta }，不用再 .data.data
      - 401：自動登出並導回登入頁
      - 403：跳 ElNotification 警告
      - 其他錯誤：ElMessage 顯示 error.message；若有 error.details（欄位級錯誤陣列）逐欄列出
```

Google 表單回填不走這條路徑，是獨立的 `POST /api/v1/ingest/google-form`，用 `X-Ingest-Token` 驗證，見 [backend-flows.md](backend-flows.md) 的表單 ingestion 流程。

## Failure modes

- **前後端角色定義不同步**：前端 `meta.roles` 只是畫面標註，不是實際放行依據（見 [frontend-permission-logic.md](frontend-permission-logic.md)）；後端 `RequireRoles` 才是真正的關卡。新增頁面或改角色時若只改其中一邊，會出現「畫面看得到但每支 API 都 403」，兩邊要對照 [frontend-pages.md](frontend-pages.md) 與 [backend-api-reference.md](backend-api-reference.md) 一起改。
- **契約型別脫鉤**：後端 DTO 改了欄位，前端 `src/types/api.d.ts` 沒重新產生（`npm run gen:types`）時不會在編譯期報錯，只會在 runtime 欄位對不上；`error.details` 的欄位陣列格式也只有前端攔截器單方面假設，沒有型別檢查保證後端一定回這個形狀。
- **Mock 角色 header 外洩到正式環境**：`X-Mock-Role` 免登入捷徑只在 `APP_ENV=local` 生效，寫死在 `internal/platform/auth/auth.go`；改動這段驗證邏輯時，判斷分支寫錯會讓正式環境也吃這個 header，等同繞過登入。
- **展示模式（`demo/demo`）攔截殘留**：非展示帳密登入前若未正確呼叫 `exitDemoModeIfActive()`，前一次展示模式的 MSW 攔截會繼續吃掉真實帳號的請求，讓正式資料被假資料蓋過；細節見 `frontend-framework.md` 的展示帳號段落。

## Unverified

- `error.details` 的欄位名稱／巢狀規則沒有共用的型別定義或 schema 驗證，目前只能靠前後端各自的實作互相對齊，未找到自動化契約測試。
