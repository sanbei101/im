<script setup lang="ts">
import { inject, onMounted, watch, nextTick } from 'vue'
import { ChatSDK } from 'go-chat-sdk'
import { useRooms } from '@/composables/useRooms'
import { useChat } from '@/composables/useChat'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import MessageItem from './MessageItem.vue'
import MessageInput from './MessageInput.vue'

const props = defineProps<{
  sdk: ChatSDK
}>()

const { currentRoom } = useRooms()
const { messages, isLoadingHistory, hasMoreHistory, loadHistory, clearMessages } = useChat()

async function handleLoadMore() {
  if (!currentRoom.value) return
  await loadHistory(props.sdk, currentRoom.value.room_id)
}

watch(() => currentRoom.value?.room_id, async (newRoomId) => {
  if (newRoomId) {
    clearMessages()
    await loadHistory(props.sdk, newRoomId)
  }
})

onMounted(async () => {
  if (currentRoom.value) {
    await loadHistory(props.sdk, currentRoom.value.room_id)
  }
})
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center gap-3 px-4 py-3 border-b shrink-0">
      <Avatar class="h-9 w-9">
        <AvatarFallback>{{ currentRoom?.name?.[0]?.toUpperCase() || '#' }}</AvatarFallback>
      </Avatar>
      <div class="flex flex-col">
        <span class="text-sm font-medium">{{ currentRoom?.name || '聊天' }}</span>
        <span class="text-xs text-muted-foreground">
          {{ currentRoom?.type === 'group' ? '群聊' : '单聊' }}
        </span>
      </div>
    </div>

    <!-- Messages -->
    <ScrollArea class="flex-1">
      <div class="flex flex-col p-4 space-y-1">
        <div v-if="hasMoreHistory" class="flex justify-center py-2">
          <Button
            v-if="!isLoadingHistory"
            variant="ghost"
            size="sm"
            @click="handleLoadMore"
          >
            加载更多
          </Button>
          <Skeleton v-else class="h-8 w-24" />
        </div>

        <MessageItem
          v-for="msg in messages"
          :key="msg.client_msg_id || msg.msg_id"
          :message="msg"
          :sdk="sdk"
        />
      </div>
    </ScrollArea>

    <Separator />

    <!-- Input -->
    <MessageInput :sdk="sdk" />
  </div>
</template>

<script lang="ts">
import { Button } from '@/components/ui/button'
export { Button }
</script>
