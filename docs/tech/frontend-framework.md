# 前端框架與專案結構

給要動 `apps/web` 程式碼的人看。技術棧、目錄結構、資料流、路由與權限、Mock 邊界、狀態管理原則。頁面完整清單另見 [frontend-pages.md](frontend-pages.md)，功能流程另見 [frontend-flows.md](frontend-flows.md)。

## 技術棧

| 項目 | 選擇 |
|---|---|
| 框架 | Vue 3（Composition API）+ TypeScript |
| 建置工具 | Vite |
| UI 元件庫 | Element Plus（自動 import，見 `vite.config.ts` 的 `unplugin-vue-components`，畫面裡不需要手動 `import` Element Plus 元件） |
| 路由 | Vue Router |
| 狀態管理 | Pinia |
| HTTP client | Axios |
| Mock | MSW（Mock Service Worker） |
| 圖表 | ECharts + vue-echarts |
| 日期 | dayjs |

## 目錄結構

```
src/
  views/         頁面元件，依業務模組分資料夾（cases、rides、masters、reports、settings…）
  api/           API client，依資源切檔（cases.ts、rides.ts、users.ts…），都走同一個 axios instance
  stores/        Pinia store（目前只有 auth.ts，管登入狀態跟目前使用者）
  router/        index.ts 是路由表，guards.ts 是權限守衛邏輯
  layouts/       版面骨架元件
  components/    跨頁共用元件
  composables/   可重用的 reactive 邏輯（useXxx）
  mocks/         MSW handler 跟假資料
  types/         TypeScript 型別；api.d.ts 是自動產生的，不要手改
  styles/        共用樣式，含 Element Plus 樣式覆寫
```

## 資料怎麼流

一個畫面的標準路徑：`路由(meta 定義權限) → view 元件 → 呼叫 src/api/*.ts → axios 走過攔截器 → 後端 → response 解包回 view`。

`src/api/client.ts` 是唯一的 axios instance，兩個攔截器做的事：

- **request**：自動帶 `Authorization: Bearer <token>`（從 `stores/auth.ts` 拿）；如果 `VITE_ENABLE_MSW=true` 且已登入，額外帶 `X-Mock-Role` / `X-Mock-User-ID`，讓後端本機環境也能吃到對應角色權限（這兩個 header 只在明確開 mock 時才加，正式 build 不會出現）。
- **response**：直接把 `response.data` 解出來（所以各個 `api/*.ts` 拿到的就是 `{ data, meta }` 那層，不用自己再 `.data.data`）；401 自動登出並導回登入頁；403 跳警告訊息；其他錯誤統一用 `ElMessage` / `ElNotification` 顯示，帶欄位級錯誤的話（後端回傳 `error.details`）會列出每個欄位的錯誤原因。

新增一支 API 就在 `src/api/` 對應資源的檔案加一個 function，回傳型別盡量用 `src/types/api.d.ts` 產生的型別，不要自己重複定義一份跟後端脫鉤的 interface。後端 API 改了要記得跑 `npm run gen:types` 重新產生型別。

## 路由與權限

路由表在 `router/index.ts`，每個路由的 `meta` 帶三個東西：

```ts
meta: { title: '個案管理', module: 'masters_cases', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
```

`roles` 只是給人看的標註，實際放不放行看 `module` 對應的模組權限表（角色預設值 ＋ 個人自訂覆蓋），判斷順序跟細節寫在獨立的 [frontend-permission-logic.md](frontend-permission-logic.md)，改權限規則前務必先讀那份，不要以為改 `meta.roles` 就會生效。加新頁面記得同步確認後端對應 API 的 `RequireRoles` 白名單有沒有涵蓋一致的角色（見 [backend-api-reference.md](backend-api-reference.md)），前後端角色定義要對得上，不然會出現「頁面看得到但每支 API 都 403」的狀況；也要注意前端的細粒度權限後端目前接不到，見 [frontend-permission-logic.md](frontend-permission-logic.md) 的落差說明。

## Mock（MSW）

`VITE_ENABLE_MSW=true` 時，`main.ts` 會啟動 MSW，攔截 `src/mocks/handlers/` 底下定義的請求並回傳假資料；沒被攔到的請求照常打真的後端（`onUnhandledRequest: 'bypass'`）。用途是本機開發、或前端功能要先做但後端 API 還沒好時，先用假資料把畫面跑起來——目前「使用者管理」「角色身分管理」兩個頁面完全靠 MSW 撐著在跑，後端還沒有對應實作（見 [backend-api-reference.md](backend-api-reference.md) 的 gap 標註）。

寫 mock handler 時盡量貼近真實 API 的 response 格式（同樣的 `{ data, meta }` envelope、同樣的分頁、同樣的 error 格式），不然開發時看起來沒問題，接上真後端才發現格式對不上。mock 資料放 `src/mocks/data/`，跟 handler 分開，方便共用同一份假資料給多個 handler 用。

正式 build（`npm run build`）不會把 MSW 打進去，`VITE_ENABLE_MSW` 正式環境必須是 `false` 或不設定。分類細節見 [`.agents/skills/mock-and-demo-boundaries/SKILL.md`](../../.agents/skills/mock-and-demo-boundaries/SKILL.md)。

### 正式環境的展示帳號（`src/lib/demoMode.ts`）

除了上面 build-time 的 `VITE_ENABLE_MSW`，正式環境還有另一條**登入時動態啟用 MSW** 的路徑，讓同一個正式部署輸入固定的展示帳密就能看假資料，其他帳密登入照常打真的後端：

- 帳號密碼**都輸入 `demo`**（`isDemoCredentials`，常數寫在 `demoMode.ts`）時，`LoginView.handleLogin` 完全略過 Supabase，直接動態 `import('@/mocks/browser')` 並 `worker.start()`，用一組寫死的假使用者（`role: 'admin'`）建立 session，之後這個分頁的 API 請求全部被 MSW 攔截。
- 非 `demo/demo` 的帳密才會照原本流程打 `supabase.auth.signInWithPassword`；登入成功後呼叫 `exitDemoModeIfActive()` 確保沒有殘留前一次展示模式的攔截（避免同一分頁先用 demo 登入過，殘留攔截到之後的真實帳號請求）。
- `main.ts` 開機時呼叫 `restoreDemoModeOnBoot()`，讀 `localStorage` 的 `ltc_demo_mode` 旗標決定要不要在重新整理頁面後還原這個攔截狀態。
- `stores/auth.ts` 的 `logout()` 會呼叫 `clearDemoModeOnLogout()` 清掉旗標並停用 worker。

要換展示帳密，直接改 `demoMode.ts` 的 `DEMO_ACCOUNT`／`DEMO_PASSWORD` 常數並重新部署；MSW 模組是動態 import，正式 build 不會因為這個機制把 mock 程式碼包進首屏 bundle，只有真的輸入 `demo/demo` 登入那一刻才會下載。因為帳密是任何人都能公開猜到的固定字串，等同公開的展示入口，展示模式的假資料裡不能放真實個資或任何敏感內容。

## 狀態管理

大部分頁面狀態就用元件內的 `ref`/`reactive`，不需要進 Pinia。只有真的要跨頁、跨元件共用、或要在整個 App 生命週期內持續存在的狀態（目前就是登入狀態，`stores/auth.ts`）才放 store。加新 store 前先想一下這個狀態是不是真的需要全域共用，還是其實用 props/emit 或一個 composable 就夠了。
