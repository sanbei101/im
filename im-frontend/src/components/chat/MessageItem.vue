<script setup lang="ts">
import { computed } from 'vue'
import { getSDK } from '@/lib/sdk'
import type { ChatMessage } from '@/composables/useChat'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { prepareWithSegments, measureLineStats } from '@chenglou/pretext'

const props = defineProps<{
  message: ChatMessage
}>()

const isSelf = computed(() => props.message.sender_id === getSDK().getCurrentUser()?.user_id)

const displayTime = computed(() => {
  const time = props.message.server_time
  if (!time) return ''
  const date = new Date(time / 1000)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
})

const textContent = computed(() => {
  if (props.message.msg_type === 'text') {
    return (props.message.payload as { text: string }).text
  }
  return '[其他消息类型]'
})

const CHAT_FONT = '400 14px "Inter", "PingFang SC", "Microsoft YaHei", "Noto Sans CJK SC", sans-serif'
const MAX_TEXT_WIDTH = 300

const bubbleStyle = computed(() => {
  if (!textContent.value || props.message.msg_type !== 'text') {
    return {}
  }

  const prepared = prepareWithSegments(textContent.value, CHAT_FONT, { whiteSpace: 'pre-wrap' })
  const { maxLineWidth } = measureLineStats(prepared, MAX_TEXT_WIDTH)

  const finalWidth = Math.ceil(maxLineWidth) + 25

  return {
    width: `${finalWidth}px`,
  }
})
</script>

<template>
  <TooltipProvider>
    <div class="flex gap-2 py-1" :class="isSelf ? 'flex-row-reverse' : 'flex-row'">
      <Avatar class="h-8 w-8 shrink-0 mt-1">
        <AvatarFallback>{{ message.sender_id?.[0]?.toUpperCase() || '?' }}</AvatarFallback>
      </Avatar>

      <div class="flex flex-col max-w-[70%]" :class="isSelf ? 'items-end' : 'items-start'">
        <span class="text-xs text-muted-foreground px-1 mb-1">
          {{ message.sender_id }}
        </span>

        <Tooltip>
          <TooltipTrigger as-child>
            <div class="px-3 py-2 text-sm rounded-2xl shadow-sm" :class="[
              isSelf
                ? 'bg-primary text-primary-foreground rounded-tr-sm'
                : 'bg-muted rounded-tl-sm'
            ]" :style="bubbleStyle">
              <span class="whitespace-pre-wrap break-words leading-relaxed">{{ textContent }}</span>
              <span v-if="message.status === 'sending'" class="ml-1 text-xs opacity-60">发送中...</span>
            </div>
          </TooltipTrigger>
          <TooltipContent side="bottom">
            <p class="text-xs">{{ displayTime }}</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  </TooltipProvider>
</template>