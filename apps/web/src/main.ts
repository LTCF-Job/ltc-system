import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import '@/styles/element-overrides.scss'
import { isMockRuntimeEnabled, restoreDemoModeOnBoot } from '@/lib/demoMode'
import { onUnhandledRequest } from '@/mocks/onUnhandledRequest'

async function prepareApp() {
  if (isMockRuntimeEnabled()) {
    const { worker } = await import('./mocks/browser')
    await worker.start({ onUnhandledRequest })
    return
  }
  // 重新整理頁面時還原展示帳號的 mock 攔截狀態（見 syncDemoModeForLogin）
  await restoreDemoModeOnBoot()
}

prepareApp().then(() => {
  const app = createApp(App)

  app.use(createPinia())
  app.use(router)
  app.use(ElementPlus)

  app.mount('#app')
})
