import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ChatSDK, type UserResponse } from 'go-chat-sdk'
import { toast } from 'vue-sonner'

const currentUser = ref<UserResponse | null>(null)
const isLoading = ref(false)

export function useAuth() {
  const router = useRouter()

  const isAuthenticated = computed(() => !!currentUser.value)

  function initAuth(sdk: ChatSDK) {
    const saved = localStorage.getItem('go-chat-user')
    if (saved) {
      try {
        const user: UserResponse = JSON.parse(saved)
        currentUser.value = user
        sdk.setAuth(user)
      } catch {
        localStorage.removeItem('go-chat-user')
      }
    }
  }

  async function login(sdk: ChatSDK, username: string, password: string) {
    isLoading.value = true
    try {
      const user = await sdk.login({ username, password })
      currentUser.value = user
      localStorage.setItem('go-chat-user', JSON.stringify(user))
      toast.success('登录成功')
      router.push('/chat')
    } catch (err: any) {
      toast.error(err.message || '登录失败')
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function register(sdk: ChatSDK, username: string, password: string) {
    isLoading.value = true
    try {
      const user = await sdk.register({ username, password })
      currentUser.value = user
      localStorage.setItem('go-chat-user', JSON.stringify(user))
      toast.success('注册成功')
      router.push('/chat')
    } catch (err: any) {
      toast.error(err.message || '注册失败')
      throw err
    } finally {
      isLoading.value = false
    }
  }

  function logout(sdk: ChatSDK) {
    sdk.disconnect()
    sdk.clearAuth()
    currentUser.value = null
    localStorage.removeItem('go-chat-user')
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
}
