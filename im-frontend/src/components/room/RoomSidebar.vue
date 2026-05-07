<script setup lang="ts">
import { ref } from 'vue'
import { useRoomsStore } from '@/composables/useRooms'
import { getSDK } from '@/lib/sdk'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import { Plus, LogOut, Menu, MessageSquare } from 'lucide-vue-next'
import RoomListItem from './RoomListItem.vue'
import CreateRoomDialog from './CreateRoomDialog.vue'
import type { UserResponse } from 'go-chat-sdk'

const props = defineProps<{
  currentUser: UserResponse | null
}>()

const emit = defineEmits<{
  logout: []
}>()

const roomsStore = useRoomsStore()

const createDialogOpen = ref(false)
const mobileMenuOpen = ref(false)

function handleSelectRoom(roomId: string) {
  roomsStore.selectRoom(roomId)
  mobileMenuOpen.value = false
}
</script>

<template>
  <!-- Desktop Sidebar -->
  <div class="hidden md:flex w-72 flex-col border-r bg-sidebar">
    <!-- Header -->
    <div class="flex items-center justify-between p-4">
      <div class="flex items-center gap-3">
        <Avatar class="h-9 w-9">
          <AvatarFallback>{{ currentUser?.username?.[0]?.toUpperCase() || 'U' }}</AvatarFallback>
        </Avatar>
        <div class="flex flex-col">
          <span class="text-sm font-medium">{{ currentUser?.username || '未登录' }}</span>
          <span class="text-xs text-muted-foreground">在线</span>
        </div>
      </div>
      <Button variant="ghost" size="icon" @click="createDialogOpen = true">
        <Plus class="h-4 w-4" />
      </Button>
    </div>

    <Separator />

    <!-- Room List -->
    <ScrollArea class="flex-1">
      <div v-if="roomsStore.isLoading" class="p-4 space-y-3">
        <div v-for="i in 5" :key="i" class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-full bg-muted animate-pulse" />
          <div class="flex-1 space-y-1">
            <div class="h-4 w-24 bg-muted animate-pulse rounded" />
            <div class="h-3 w-16 bg-muted animate-pulse rounded" />
          </div>
        </div>
      </div>
      <div v-else-if="roomsStore.rooms.length === 0" class="flex flex-col items-center justify-center p-8 text-muted-foreground">
        <MessageSquare class="h-8 w-8 mb-2 opacity-50" />
        <p class="text-sm">暂无聊天房间</p>
        <p class="text-xs mt-1">点击 + 创建新聊天</p>
      </div>
      <div v-else class="p-2 space-y-1">
        <RoomListItem
          v-for="room in roomsStore.rooms"
          :key="room.room_id"
          :room="room"
          :is-active="room.room_id === roomsStore.currentRoomId"
          @click="handleSelectRoom(room.room_id)"
        />
      </div>
    </ScrollArea>

    <Separator />

    <!-- Footer -->
    <div class="p-2">
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="ghost" class="w-full justify-start gap-2">
            <LogOut class="h-4 w-4" />
            <span>退出登录</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" class="w-56">
          <DropdownMenuItem @click="emit('logout')">
            <LogOut class="h-4 w-4 mr-2" />
            确认退出
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>

  <!-- Mobile Sidebar (Sheet) -->
  <div class="md:hidden flex items-center border-b p-2 gap-2">
    <Sheet v-model:open="mobileMenuOpen">
      <SheetTrigger as-child>
        <Button variant="ghost" size="icon">
          <Menu class="h-5 w-5" />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" class="w-72 p-0 flex flex-col">
        <div class="flex items-center justify-between p-4">
          <div class="flex items-center gap-3">
            <Avatar class="h-9 w-9">
              <AvatarFallback>{{ currentUser?.username?.[0]?.toUpperCase() || 'U' }}</AvatarFallback>
            </Avatar>
            <div class="flex flex-col">
              <span class="text-sm font-medium">{{ currentUser?.username || '未登录' }}</span>
              <span class="text-xs text-muted-foreground">在线</span>
            </div>
          </div>
          <Button variant="ghost" size="icon" @click="createDialogOpen = true">
            <Plus class="h-4 w-4" />
          </Button>
        </div>
        <Separator />
        <ScrollArea class="flex-1">
          <div v-if="roomsStore.rooms.length === 0" class="flex flex-col items-center justify-center p-8 text-muted-foreground">
            <MessageSquare class="h-8 w-8 mb-2 opacity-50" />
            <p class="text-sm">暂无聊天房间</p>
          </div>
          <div v-else class="p-2 space-y-1">
            <RoomListItem
              v-for="room in roomsStore.rooms"
              :key="room.room_id"
              :room="room"
              :is-active="room.room_id === roomsStore.currentRoomId"
              @click="handleSelectRoom(room.room_id)"
            />
          </div>
        </ScrollArea>
        <Separator />
        <div class="p-2">
          <Button variant="ghost" class="w-full justify-start gap-2" @click="emit('logout')">
            <LogOut class="h-4 w-4" />
            <span>退出登录</span>
          </Button>
        </div>
      </SheetContent>
    </Sheet>
    <span class="font-medium">{{ currentUser?.username }}</span>
  </div>

  <CreateRoomDialog v-model:open="createDialogOpen" />
</template>
