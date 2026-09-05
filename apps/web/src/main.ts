import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import '@/styles/element-overrides.scss'
import { useAuthStore } from '@/stores/auth'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)

async function bootstrap() {
  const authStore = useAuthStore(pinia)
  await authStore.initializeAuth()
  app.use(router)
  app.use(ElementPlus)
  app.mount('#app')
}

void bootstrap()
