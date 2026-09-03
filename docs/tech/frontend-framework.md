# 前端框架與專案結構

給要動 `apps/web` 程式碼的人看。技術棧、目錄結構、資料流、路由與權限、狀態管理原則。頁面完整清單另見 [frontend-pages.md](frontend-pages.md)，功能流程另見 [frontend-flows.md](frontend-flows.md)。

## 技術棧

| 項目 | 選擇 |
|---|---|
| 框架 | Vue 3（Composition API）+ TypeScript |
| 建置工具 | Vite |
| UI 元件庫 | Element Plus（自動 import，見 `vite.config.ts` 的 `unplugin-vue-components`，畫面裡不需要手動 `import` Element Plus 元件） |
| 路由 | Vue Router |
| 狀態管理 | Pinia |
| HTTP client | Axios |
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
  types/         TypeScript 型別；api.d.ts 是自動產生的，不要手改
  styles/        共用樣式，含 Element Plus 樣式覆寫
```

## 資料怎麼流

一個畫面的標準路徑：`路由(meta 定義權限) → view 元件 → 呼叫 src/api/*.ts → axios 走過攔截器 → 後端 → response 解包回 view`。

`src/api/client.ts` 是唯一的 axios instance，兩個攔截器做的事：

- **request**：自動帶 `Authorization: Bearer <token>`（從 `stores/auth.ts` 拿）。
- **response**：直接把 `response.data` 解出來（所以各個 `api/*.ts` 拿到的就是 `{ data, meta }` 那層，不用自己再 `.data.data`）；401 自動登出並導回登入頁；403 跳警告訊息；其他錯誤統一用 `ElMessage` / `ElNotification` 顯示，帶欄位級錯誤的話（後端回傳 `error.details`）會列出每個欄位的錯誤原因。

新增一支 API 就在 `src/api/` 對應資源的檔案加一個 function，回傳型別盡量用 `src/types/api.d.ts` 產生的型別，不要自己重複定義一份跟後端脫鉤的 interface。後端 API 改了要記得跑 `npm run gen:types` 重新產生型別。

## 路由與權限

路由表在 `router/index.ts`，每個路由的 `meta` 帶三個東西：

```ts
meta: { title: '個案管理', module: 'masters_cases', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
```

`roles` 只是給人看的標註，實際放不放行看 `module` 對應的模組權限表（角色預設值 ＋ 個人自訂覆蓋），判斷順序跟細節寫在獨立的 [frontend-permission-logic.md](frontend-permission-logic.md)，改權限規則前務必先讀那份，不要以為改 `meta.roles` 就會生效。加新頁面記得同步確認後端對應 API 的 `RequireRoles` 白名單有沒有涵蓋一致的角色（見 [backend-api-reference.md](backend-api-reference.md)），前後端角色定義要對得上，不然會出現「頁面看得到但每支 API 都 403」的狀況；也要注意前端的細粒度權限後端目前接不到，見 [frontend-permission-logic.md](frontend-permission-logic.md) 的落差說明。

## 環境模型（local／production）

前端**沒有任何 mock 或假資料層**（MSW 已整個移除）。兩個環境跑的是同一份程式碼、同一條功能路徑，差別只在連到哪個資料庫：

| 環境 | 資料庫 | 登入 |
|---|---|---|
| local | 本機 PostgreSQL | 未設定 `VITE_SUPABASE_URL`／`VITE_SUPABASE_ANON_KEY` 時不驗證密碼，改發 `mock_jwt_<role>` 給後端的 local 分支解析 |
| production | 正式資料庫 | Supabase Auth 真實驗證 |

local 的登入表單與其他環境完全一樣：照常輸入帳號密碼、照常按「登入系統」，只是不會呼叫 Supabase。角色由輸入的帳號推斷——含 `viewer` 字樣即以檢視人員登入，其餘一律管理員（`LoginView.handleLogin`）。這條路徑只有在 `import.meta.env.DEV` 或 `VITE_APP_ENV=local` 時成立；其餘環境沒接上 Supabase 就直接拒絕登入，不存在任何略過驗證的入口。

後端對應的放行條件見 `apps/api/internal/platform/auth/auth.go`：只有 `APP_ENV=local` 才接受 `mock_jwt_` 前綴的憑證。

### 登入帳號代稱

登入表單接受帳號代稱，`LoginView.handleLogin` 會在送出前把它換成 Supabase Auth 真正使用的 email。**密碼一律照常送進 `supabase.auth.signInWithPassword` 做真實驗證，沒有任何略過 Supabase 的登入路徑**：

| 輸入的帳號 | 實際送出的 email | 判斷依據 |
|---|---|---|
| `ltcf-admin` | `ltcf-admin@ltc.example.com` | `LoginView.handleLogin` 內的字串比對 |

代稱只是輸入上的方便，不代表帳號存在，也不代表有預設密碼：

- **沒有任何 migration 會建立登入帳號。** `apps/api/migrations/000002_seed_reference_data.up.sql` 現在只寫入 22 個縣市的 `regions`，`000011_backfill_admin_identity.up.sql` 已改成 `SELECT 1;` 的 no-op。管理員帳號必須另行 bootstrap（見 [environment-bootstrap.md](environment-bootstrap.md)），密碼只存在於建立當下使用的 secret 或密碼管理工具，不寫進程式碼、環境變數範本或任何文件。
- 權限來自 JWT 的 `app_metadata.role`。`LoginView` 把這個欄位存進 session。

## 狀態管理

大部分頁面狀態就用元件內的 `ref`/`reactive`，不需要進 Pinia。只有真的要跨頁、跨元件共用、或要在整個 App 生命週期內持續存在的狀態（目前就是登入狀態，`stores/auth.ts`）才放 store。加新 store 前先想一下這個狀態是不是真的需要全域共用，還是其實用 props/emit 或一個 composable 就夠了。
