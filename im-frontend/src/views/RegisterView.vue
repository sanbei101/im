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
const { register, isLoading } = useAuth()
const router = useRouter()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')

async function handleSubmit() {
  error.value = ''
  if (!username.value || !password.value) {
    error.value = '请填写所有字段'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  await register(sdk, username.value, password.value)
}
</script>

<template>
  <div class="flex min-h-full items-center justify-center p-4">
    <Card class="w-full max-w-sm">
      <CardHeader class="space-y-1">
        <CardTitle class="text-2xl font-bold">注册</CardTitle>
        <CardDescription>创建一个新账号</CardDescription>
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
              autocomplete="new-password"
            />
          </div>
          <div class="space-y-2">
            <Label for="confirm">确认密码</Label>
            <Input
              id="confirm"
              v-model="confirmPassword"
              type="password"
              placeholder="请再次输入密码"
              required
              autocomplete="new-password"
            />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
        </CardContent>
        <CardFooter class="flex flex-col gap-4">
          <Button type="submit" class="w-full" :disabled="isLoading">
            {{ isLoading ? '注册中...' : '注册' }}
          </Button>
          <p class="text-sm text-muted-foreground">
            已有账号?
            <Button variant="link" class="p-0 h-auto" @click="router.push('/login')">
              去登录
            </Button>
          </p>
        </CardFooter>
      </form>
    </Card>
  </div>
</template>
