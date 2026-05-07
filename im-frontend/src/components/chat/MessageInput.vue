<script setup lang="ts">
import { ref, inject } from 'vue'
import { ChatSDK } from 'go-chat-sdk'
import { useRooms } from '@/composables/useRooms'
import { useChat } from '@/composables/useChat'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Send } from 'lucide-vue-next'

const props = defineProps<{
  sdk: ChatSDK
}>()

const { currentRoomId } = useRooms()
const { sendTextMessage } = useChat()

const text = ref('')
const isSending = ref(false)

async function handleSend() {
  const trimmed = text.value.trim()
  if (!trimmed || !currentRoomId.value) return

  sendTextMessage(props.sdk, currentRoomId.value, trimmed)
  text.value = ''
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}
</script>

<template>
  <div class="flex items-end gap-2 p-3 shrink-0">
    <Textarea
      v-model="text"
      placeholder="输入消息... (Enter发送, Shift+Enter换行)"
      class="min-h-[44px] max-h-32 resize-none"
      rows="1"
      @keydown="handleKeydown"
    />
    <Button
      size="icon"
      class="shrink-0 h-11 w-11"
      :disabled="!text.trim() || !currentRoomId"
      @click="handleSend"
    >
      <Send class="h-4 w-4" />
    </Button>
  </div>
</template>
