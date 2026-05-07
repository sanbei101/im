<script setup lang="ts">
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import type { Room } from '@/composables/useRooms'

const props = defineProps<{
  room: Room
  isActive: boolean
}>()
</script>

<template>
  <div
    class="flex items-center gap-3 rounded-lg px-3 py-2 cursor-pointer transition-colors"
    :class="isActive ? 'bg-sidebar-accent text-sidebar-accent-foreground' : 'hover:bg-sidebar-accent/50'"
  >
    <Avatar class="h-10 w-10 shrink-0">
      <AvatarFallback>{{ room.name?.[0]?.toUpperCase() || '#' }}</AvatarFallback>
    </Avatar>
    <div class="flex-1 min-w-0">
      <div class="flex items-center justify-between">
        <span class="text-sm font-medium truncate">{{ room.name }}</span>
        <Badge v-if="room.unread_count" variant="default" class="h-5 min-w-5 px-1 text-xs">
          {{ room.unread_count }}
        </Badge>
      </div>
      <p class="text-xs text-muted-foreground truncate">
        {{ room.last_message || '暂无消息' }}
      </p>
    </div>
  </div>
</template>
