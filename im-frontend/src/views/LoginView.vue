<script setup lang="ts">
import { ref, inject } from 'vue'
import { useRouter } from 'vue-router'
import { ChatSDK } from 'go-chat-sdk'
import { useAuth } from '@/composables/useAuth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

const sdk = inject<ChatSDK>('sdk')!
const { login, isLoading } = useAuth()
const router = useRouter()

const username = ref('')
const password = ref('')

async function handleSubmit() {
  if (!username.value || !password.value) return
  await login(sdk, username.value, password.value)
}
</script>

<template>
  <div class="flex min-h-full items-center justify-center p-4">
    <Card class="w-full max-w-sm">
      <CardHeader class="space-y-1">
        <CardTitle class="text-2xl font-bold">登录</CardTitle>
        <CardDescription>输入你的账号信息以继续</CardDescription>
      </CardHeader>
      <form @submit.prevent="handleSubmit">
        <CardContent class="space-y-4">
          <div class="space-y-2">
            <Label for="username">用户名</Label>
            <Input
              id="username"
              v-model="username"
              placeholder="请输入用户名"
              required
              autocomplete="username"
            />
          </div>
          <div class="space-y-2">
            <Label for="password">密码</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="请输入密码"
              required
              autocomplete="current-password"
            />
          </div>
        </CardContent>
        <CardFooter class="flex flex-col gap-4">
          <Button type="submit" class="w-full" :disabled="isLoading">
            {{ isLoading ? '登录中...' : '登录' }}
          </Button>
          <p class="text-sm text-muted-foreground">
            还没有账号?
            <Button variant="link" class="p-0 h-auto" @click="router.push('/register')">
              去注册
            </Button>
          </p>
        </CardFooter>
      </form>
    </Card>
  </div>
</template>
