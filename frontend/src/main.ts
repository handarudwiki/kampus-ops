import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { DesignSystem } from './design-system'
import { setUnauthorizedHandler } from './api/client'
import { useAuthStore } from './stores/auth'
import './styles/main.css'

const app = createApp(App).use(createPinia()).use(router).use(DesignSystem)

setUnauthorizedHandler(() => {
  useAuthStore().expire()
  router.push({ path: '/login', query: { next: router.currentRoute.value.fullPath } })
})

app.mount('#app')
