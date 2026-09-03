import { defineConfig } from '@playwright/test'

// 真實 Supabase／Demo API 的 API-level 驗證，不需要本機前端 dev server。
// 缺少必要環境變數時各測試會各自 test.skip，不會讓整條 CI pipeline 因為本機沒有
// 真實憑證而失敗；只有在 CI 設好對應 secrets 後才會真的執行。
export default defineConfig({
  testDir: '.',
  testMatch: 'tests/e2e-live/**/*.spec.ts',
  fullyParallel: false,
  retries: 0,
  workers: 1,
  reporter: 'list',
  timeout: 30000
})
