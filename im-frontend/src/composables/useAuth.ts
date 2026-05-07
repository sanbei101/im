import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getSDK } from '@/lib/sdk'
import type { UserResponse } from 'go-chat-sdk'
import { toast } from 'vue-sonner'

export const useAuthStore = defineStore('auth', () => {
  const router = useRouter()
  const sdk = getSDK()

  const currentUser = ref<UserResponse | null>(null)
  const isLoading = ref(false)

  const isAuthenticated = computed(() => !!currentUser.value)

  function initAuth() {
    if (currentUser.value) {
      sdk.setAuth(currentUser.value)
    }
  }

  async function login(username: string, password: string) {
    isLoading.value = true
    try {
      const user = await sdk.login({ username, password })
      currentUser.value = user
      toast.success('登录成功')
      router.push('/chat')
    } catch (err) {
      const message = err instanceof Error ? err.message : '登录失败'
      toast.error(message)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function register(username: string, password: string) {
    isLoading.value = true
    try {
      const user = await sdk.register({ username, password })
      currentUser.value = user
      toast.success('注册成功')
      router.push('/chat')
    } catch (err) {
      const message = err instanceof Error ? err.message : '注册失败'
      toast.error(message)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  function logout() {
    sdk.disconnect()
    sdk.clearAuth()
    currentUser.value = null
    router.push('/login')
    toast.info('已退出登录')
  }

  return {
    currentUser,
    isAuthenticated,
    isLoading,
    initAuth,
    login,
    register,
    logout,
  }
}, {
  persist: {
    pick: ['currentUser'],
  },
})
