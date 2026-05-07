<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { getSDK } from '@/lib/sdk'
import { useAuthStore } from '@/composables/useAuth'
import { useChatStore } from '@/composables/useChat'
import { useRoomsStore } from '@/composables/useRooms'
import RoomSidebar from '@/components/room/RoomSidebar.vue'
import ChatWindow from '@/components/chat/ChatWindow.vue'

const authStore = useAuthStore()
const chatStore = useChatStore()
const roomsStore = useRoomsStore()

authStore.initAuth()
chatStore.initChat()

onMounted(async () => {
  if (authStore.currentUser && !getSDK().isConnected()) {
    try {
      await chatStore.connect()
    } catch {
      // connection error handled by useChat
    }
  }
})

watch(() => roomsStore.currentRoomId, () => {
  chatStore.clearMessages()
})
</script>

<template>
  <div class="flex h-full w-full">
    <RoomSidebar
      :current-user="authStore.currentUser"
      @logout="authStore.logout()"
    />
    <div class="flex-1 flex flex-col min-w-0">
      <ChatWindow
        v-if="roomsStore.currentRoomId"
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
