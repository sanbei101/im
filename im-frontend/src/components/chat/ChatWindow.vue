<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRoomsStore } from '@/composables/useRooms'
import { useChatStore } from '@/composables/useChat'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import MessageItem from './MessageItem.vue'
import MessageInput from './MessageInput.vue'

const roomsStore = useRoomsStore()
const chatStore = useChatStore()

async function handleLoadMore() {
  if (!roomsStore.currentRoom) return
  await chatStore.loadHistory(roomsStore.currentRoom.room_id)
}

watch(() => roomsStore.currentRoom?.room_id, async (newRoomId) => {
  if (newRoomId) {
    chatStore.clearMessages()
    await chatStore.loadHistory(newRoomId)
  }
})

onMounted(async () => {
  if (roomsStore.currentRoom) {
    await chatStore.loadHistory(roomsStore.currentRoom.room_id)
  }
})
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center gap-3 px-4 py-3 border-b shrink-0">
      <Avatar class="h-9 w-9">
        <AvatarFallback>{{ roomsStore.currentRoom?.name?.[0]?.toUpperCase() || '#' }}</AvatarFallback>
      </Avatar>
      <div class="flex flex-col">
        <span class="text-sm font-medium">{{ roomsStore.currentRoom?.name || '聊天' }}</span>
        <span class="text-xs text-muted-foreground">
          {{ roomsStore.currentRoom?.type === 'group' ? '群聊' : '单聊' }}
        </span>
      </div>
    </div>

    <!-- Messages -->
    <ScrollArea class="flex-1">
      <div class="flex flex-col p-4 space-y-1">
        <div v-if="chatStore.hasMoreHistory" class="flex justify-center py-2">
          <Button
            v-if="!chatStore.isLoadingHistory"
            variant="ghost"
            size="sm"
            @click="handleLoadMore"
          >
            加载更多
          </Button>
          <Skeleton v-else class="h-8 w-24" />
        </div>

        <MessageItem
          v-for="msg in chatStore.messages"
          :key="msg.client_msg_id || msg.msg_id"
          :message="msg"
        />
      </div>
    </ScrollArea>

    <Separator />

    <!-- Input -->
    <MessageInput />
  </div>
</template>

