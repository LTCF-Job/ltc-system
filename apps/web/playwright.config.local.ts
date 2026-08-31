import { defineConfig } from '@playwright/test'

// 本機驗證專用設定：與 playwright.config.ts 相同，只是換一個 port（5183）避免跟同一台機器上
// 其他 session 已經在跑的 5173 dev server 互搶。純本機暫存檔，不提交進版控。
export default defineConfig({
  testDir: '.',
  testMatch: 'tests/e2e/**/*.spec.ts',
  fullyParallel: false,
  retries: 0,
  workers: 1,
  reporter: 'list',
  timeout: 30000,
  expect: {
    timeout: 5000
  },
  use: {
    baseURL: 'http://localhost:5183',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    headless: true,
    viewport: { width: 1280, height: 800 }
  },
  webServer: {
    command: 'npm run dev -- --port 5183 --strictPort',
    url: 'http://localhost:5183',
    reuseExistingServer: false,
    timeout: 60000,
    env: {
      VITE_ENABLE_MSW: 'true',
      VITE_E2E: 'true'
    }
  }
})
