<script setup lang="ts">
import { computed } from 'vue'
import { ChatSDK } from 'go-chat-sdk'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Card, CardContent } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type { ChatMessage } from '@/composables/useChat'

const props = defineProps<{
  message: ChatMessage
  sdk: ChatSDK
}>()

const isSelf = computed(() => props.message.sender_id === props.sdk.getCurrentUser()?.user_id)

const displayTime = computed(() => {
  const time = props.message.server_time
  if (!time) return ''
  // server_time is in microseconds
  const date = new Date(time / 1000)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
})

const textContent = computed(() => {
  if (props.message.msg_type === 'text') {
    return (props.message.payload as { text: string }).text
  }
  return '[其他消息类型]'
})
</script>

<template>
  <TooltipProvider>
    <div
      class="flex gap-2 py-1"
      :class="isSelf ? 'flex-row-reverse' : 'flex-row'"
    >
      <Avatar class="h-8 w-8 shrink-0 mt-1">
        <AvatarFallback>{{ message.sender_id?.[0]?.toUpperCase() || '?' }}</AvatarFallback>
      </Avatar>

      <div class="flex flex-col max-w-[70%]"
        :class="isSelf ? 'items-end' : 'items-start'"
      >
        <span class="text-xs text-muted-foreground px-1">
          {{ message.sender_id }}
        </span>

        <Tooltip>
          <TooltipTrigger as-child>
            <Card
              class="border-0 shadow-sm"
              :class="isSelf ? 'bg-primary text-primary-foreground' : 'bg-muted'"
            >
              <CardContent class="p-2.5 text-sm">
                <span>{{ textContent }}</span>
                <span
                  v-if="message.status === 'sending'"
                  class="ml-1 text-xs opacity-60"
                >发送中...</span>
              </CardContent>
            </Card>
          </TooltipTrigger>
          <TooltipContent side="bottom">
            <p class="text-xs">{{ displayTime }}</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  </TooltipProvider>
</template>
