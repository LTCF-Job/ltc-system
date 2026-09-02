import { defineConfig } from '@playwright/test'

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
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    headless: true,
    viewport: { width: 1280, height: 800 }
  },
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
    timeout: 60000,
    env: {
      VITE_ENABLE_MSW: 'true',
      VITE_E2E: 'true',
      // 供 src/mocks/handlers/supabaseAuth.ts 攔截的假 Supabase 專案設定
      VITE_SUPABASE_URL: 'http://mock.supabase.local',
      VITE_SUPABASE_ANON_KEY: 'mock-anon-key'
    }
  }
})
