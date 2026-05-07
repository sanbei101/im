import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guest: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { guest: true },
    },
    {
      path: '/chat',
      name: 'chat',
      component: () => import('@/layouts/ChatLayout.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/',
      redirect: '/chat',
    },
  ],
})

router.beforeEach((to) => {
  const { isAuthenticated } = useAuth()
  if (to.meta.requiresAuth && !isAuthenticated.value) {
    return '/login'
  }
  if (to.meta.guest && isAuthenticated.value) {
    return '/chat'
  }
})

export default router
