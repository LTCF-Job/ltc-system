import { setupWorker } from 'msw/browser'
import { handlers } from './handlers'
import { persistDemoData, restoreDemoData } from './data/demoStore'

// 還原上次展示模式登出前寫入的資料，確保重新整理頁面不會遺失
restoreDemoData()

export const worker = setupWorker(...handlers)

// 任何寫入類請求被 mock 攔截後即落地目前展示資料
worker.events.on('response:mocked', ({ request }) => {
  if (request.method !== 'GET') {
    persistDemoData()
  }
})
