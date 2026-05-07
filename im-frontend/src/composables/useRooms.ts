import { ref, computed } from 'vue'
import { ChatSDK, APIClient } from 'go-chat-sdk'
import { toast } from 'vue-sonner'

export interface Room {
  room_id: string
  name: string
  type: 'single' | 'group'
  members?: string[]
  last_message?: string
  unread_count?: number
  updated_at?: number
}

const rooms = ref<Room[]>([])
const currentRoomId = ref<string | null>(null)
const isLoading = ref(false)

export function useRooms() {
  const currentRoom = computed(() =>
    rooms.value.find(r => r.room_id === currentRoomId.value) || null
  )

  async function fetchRooms(_sdk: ChatSDK) {
    isLoading.value = true
    try {
      // TODO: 替换为实际的获取房间列表API
      // const resp = await sdk.getRooms()
      // rooms.value = resp.rooms
      rooms.value = []
    } catch (err: any) {
      toast.error(err.message || '获取房间列表失败')
    } finally {
      isLoading.value = false
    }
  }

  async function createRoom(sdk: ChatSDK, userId1: string, userId2: string) {
    try {
      const api = (sdk as any).api as APIClient
      const resp = await (api as any).createRoom({ user_id_1: userId1, user_id_2: userId2 })
      const newRoom: Room = {
        room_id: resp.room_id,
        name: `单聊 ${userId2}`,
        type: 'single',
      }
      rooms.value.unshift(newRoom)
      selectRoom(resp.room_id)
      toast.success('创建单聊成功')
      return resp.room_id
    } catch (err: any) {
      toast.error(err.message || '创建房间失败')
      throw err
    }
  }

  async function createGroupRoom(sdk: ChatSDK, memberIds: string[], name?: string) {
    try {
      const resp = await sdk.createGroupRoom({ member_ids: memberIds, name })
      const newRoom: Room = {
        room_id: resp.room_id,
        name: name || `群聊 ${resp.room_id.slice(0, 6)}`,
        type: 'group',
        members: memberIds,
      }
      rooms.value.unshift(newRoom)
      selectRoom(resp.room_id)
      toast.success('创建群聊成功')
      return resp.room_id
    } catch (err: any) {
      toast.error(err.message || '创建群聊失败')
      throw err
    }
  }

  function selectRoom(roomId: string | null) {
    currentRoomId.value = roomId
    const room = rooms.value.find(r => r.room_id === roomId)
    if (room) {
      room.unread_count = 0
    }
  }

  function addRoom(room: Room) {
    const exists = rooms.value.find(r => r.room_id === room.room_id)
    if (!exists) {
      rooms.value.unshift(room)
    }
  }

  function updateRoomLastMessage(roomId: string, message: string, time: number) {
    const room = rooms.value.find(r => r.room_id === roomId)
    if (room) {
      room.last_message = message
      room.updated_at = time
      if (room.room_id !== currentRoomId.value) {
        room.unread_count = (room.unread_count || 0) + 1
      }
      // Move to top
      rooms.value = rooms.value.sort((a, b) => (b.updated_at || 0) - (a.updated_at || 0))
    }
  }

  return {
    rooms,
    currentRoomId,
    currentRoom,
    isLoading,
    fetchRooms,
    createRoom,
    createGroupRoom,
    selectRoom,
    addRoom,
    updateRoomLastMessage,
  }
}
