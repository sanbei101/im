<script setup lang="ts">
import { ref } from 'vue'
import { ChatSDK } from 'go-chat-sdk'
import { useRooms } from '@/composables/useRooms'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const props = defineProps<{
  sdk: ChatSDK
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const { createRoom, createGroupRoom } = useRooms()

const activeTab = ref('single')
const singleUserId = ref('')
const groupName = ref('')
const groupMembers = ref('')
const isSubmitting = ref(false)

async function handleCreateSingle() {
  if (!singleUserId.value.trim()) return
  isSubmitting.value = true
  try {
    const currentUserId = props.sdk.getCurrentUser()?.user_id
    if (!currentUserId) return
    await createRoom(props.sdk, currentUserId, singleUserId.value.trim())
    emit('update:open', false)
    singleUserId.value = ''
  } finally {
    isSubmitting.value = false
  }
}

async function handleCreateGroup() {
  const members = groupMembers.value.split(',').map(s => s.trim()).filter(Boolean)
  if (members.length < 2) return
  isSubmitting.value = true
  try {
    await createGroupRoom(props.sdk, members, groupName.value.trim() || undefined)
    emit('update:open', false)
    groupName.value = ''
    groupMembers.value = ''
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>创建聊天</DialogTitle>
        <DialogDescription>选择创建单聊或群聊房间</DialogDescription>
      </DialogHeader>

      <Tabs v-model="activeTab" class="w-full">
        <TabsList class="grid w-full grid-cols-2">
          <TabsTrigger value="single">单聊</TabsTrigger>
          <TabsTrigger value="group">群聊</TabsTrigger>
        </TabsList>

        <TabsContent value="single" class="space-y-4 mt-4">
          <div class="space-y-2">
            <Label for="single-user">对方用户ID</Label>
            <Input
              id="single-user"
              v-model="singleUserId"
              placeholder="输入对方的用户ID"
            />
          </div>
          <DialogFooter>
            <Button
              type="submit"
              :disabled="!singleUserId.trim() || isSubmitting"
              @click="handleCreateSingle"
            >
              {{ isSubmitting ? '创建中...' : '创建单聊' }}
            </Button>
          </DialogFooter>
        </TabsContent>

        <TabsContent value="group" class="space-y-4 mt-4">
          <div class="space-y-2">
            <Label for="group-name">群名称（可选）</Label>
            <Input
              id="group-name"
              v-model="groupName"
              placeholder="输入群名称"
            />
          </div>
          <div class="space-y-2">
            <Label for="group-members">成员用户ID</Label>
            <Input
              id="group-members"
              v-model="groupMembers"
              placeholder="用逗号分隔多个用户ID，至少2人"
            />
            <p class="text-xs text-muted-foreground">例如: user1, user2, user3</p>
          </div>
          <DialogFooter>
            <Button
              type="submit"
              :disabled="groupMembers.split(',').filter(Boolean).length < 2 || isSubmitting"
              @click="handleCreateGroup"
            >
              {{ isSubmitting ? '创建中...' : '创建群聊' }}
            </Button>
          </DialogFooter>
        </TabsContent>
      </Tabs>
    </DialogContent>
  </Dialog>
</template>
