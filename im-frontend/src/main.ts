import { createApp } from 'vue'
import { Toaster } from '@/components/ui/sonner'
import router from './router'
import App from './App.vue'
import { createPinia } from 'pinia'
import './style.css'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)
const app = createApp(App)
app.use(pinia)
app.use(router)
app.component('Toaster', Toaster)
app.mount('#app')
