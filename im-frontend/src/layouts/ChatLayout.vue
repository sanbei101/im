<script setup lang="ts">
import { inject, onMounted, watch } from 'vue'
import { ChatSDK } from 'go-chat-sdk'
import { useAuth } from '@/composables/useAuth'
import { useChat } from '@/composables/useChat'
import { useRooms } from '@/composables/useRooms'
import RoomSidebar from '@/components/room/RoomSidebar.vue'
import ChatWindow from '@/components/chat/ChatWindow.vue'

const sdk = inject<ChatSDK>('sdk')!
const { currentUser, initAuth, logout } = useAuth()
const { initChat, connect, clearMessages } = useChat()
const { currentRoomId, selectRoom } = useRooms()

initAuth(sdk)
initChat(sdk)

onMounted(async () => {
  if (currentUser.value && !sdk.isConnected()) {
    try {
      await connect(sdk)
    } catch {
      // connection error handled by useChat
    }
  }
})

watch(currentRoomId, () => {
  clearMessages()
})
</script>

<template>
  <div class="flex h-full w-full">
    <RoomSidebar
      :current-user="currentUser"
      :sdk="sdk"
      @logout="logout(sdk)"
    />
    <div class="flex-1 flex flex-col min-w-0">
      <ChatWindow
        v-if="currentRoomId"
        :sdk="sdk"
      />
      <div v-else class="flex-1 flex items-center justify-center text-muted-foreground">
        <div class="text-center space-y-2">
          <p class="text-lg font-medium">选择一个房间开始聊天</p>
          <p class="text-sm">或创建一个新的聊天</p>
        </div>
      </div>
    </div>
  </div>
</template>
