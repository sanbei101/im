import { createApp } from 'vue'
import { ChatSDK } from 'go-chat-sdk'
import { Toaster } from '@/components/ui/sonner'
import router from './router'
import App from './App.vue'
import './style.css'

const sdk = new ChatSDK({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  gatewayURL: import.meta.env.VITE_GATEWAY_URL || 'ws://localhost:8081/ws',
})

const app = createApp(App)
app.use(router)
app.provide('sdk', sdk)
app.component('Toaster', Toaster)
app.mount('#app')
