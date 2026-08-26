import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import '@/styles/element-overrides.scss'
import { restoreDemoModeOnBoot } from '@/lib/demoMode'

async function prepareApp() {
  if (import.meta.env.VITE_ENABLE_MSW === 'true') {
    const { worker } = await import('./mocks/browser')
    await worker.start({
      onUnhandledRequest: 'bypass'
    })
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

  // 全域註冊所有 Element Plus 圖示元件
  for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
  }

  app.mount('#app')
})
