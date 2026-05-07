import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getSDK } from '@/lib/sdk'
import { ChatEventType, ConnectionState, MessageType, type Message } from 'go-chat-sdk'
import { toast } from 'vue-sonner'

export interface ChatMessage extends Message {
  status?: 'sending' | 'sent' | 'failed'
}

export const useChatStore = defineStore('chat', () => {
  const sdk = getSDK()

  const messages = ref<ChatMessage[]>([])
  const connectionState = ref<ConnectionState>(ConnectionState.Disconnected)
  const isLoadingHistory = ref(false)
  const hasMoreHistory = ref(true)

  const isConnected = computed(() => connectionState.value === 'connected')

  function initChat() {
    sdk.on(ChatEventType.ConnectionStateChange, (event) => {
      connectionState.value = event.data.state
    })

    sdk.on(ChatEventType.MessageReceived, (event) => {
      const msg = event.data.message as ChatMessage
      msg.status = 'sent'
      messages.value.push(msg)
    })

    sdk.on(ChatEventType.MessageSent, (event) => {
      const clientMsgId = event.data.client_msg_id
      const existing = messages.value.find(m => m.client_msg_id === clientMsgId)
      if (existing) {
        existing.status = 'sent'
        if (event.data.server_msg_id) {
          existing.msg_id = event.data.server_msg_id
        }
        if (event.data.server_time) {
          existing.server_time = event.data.server_time
        }
      }
    })

    sdk.on(ChatEventType.Error, (event) => {
      toast.error(event.data.message || '连接错误')
    })
  }

  async function connect() {
    try {
      await sdk.connect()
    } catch (err) {
      const message = err instanceof Error ? err.message : '连接失败'
      toast.error(message)
      throw err
    }
  }

  function sendTextMessage(roomId: string, text: string) {
    const clientMsgId = sdk.generateMessageId()
    const tempMessage: ChatMessage = {
      msg_id: '',
      client_msg_id: clientMsgId,
      sender_id: sdk.getCurrentUser()?.user_id || '',
      room_id: roomId,
      server_time: Date.now() * 1000,
      msg_type: MessageType.Text,
      payload: { text },
      status: 'sending',
    }
    messages.value.push(tempMessage)
    sdk.sendTextMessage({ room_id: roomId, text })
  }

  async function loadHistory(roomId: string) {
    if (isLoadingHistory.value || !hasMoreHistory.value) return
    isLoadingHistory.value = true
    try {
      const beforeTime = messages.value.length > 0
        ? messages.value[0].server_time
        : Date.now() * 1000

      const resp = await sdk.getHistoryMessages({
        room_id: roomId,
        before_server_time: beforeTime,
        page_size: 20,
      })

      const historyMessages = resp.messages.map(m => ({ ...m, status: 'sent' as const }))
      messages.value.unshift(...historyMessages)
      hasMoreHistory.value = resp.hasMore
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载历史消息失败'
      toast.error(message)
    } finally {
      isLoadingHistory.value = false
    }
  }

  function clearMessages() {
    messages.value = []
    hasMoreHistory.value = true
  }

  function setMessagesForRoom(msgs: ChatMessage[]) {
    messages.value = msgs
  }

  return {
    messages,
    connectionState,
    isConnected,
    isLoadingHistory,
    hasMoreHistory,
    initChat,
    connect,
    sendTextMessage,
    loadHistory,
    clearMessages,
    setMessagesForRoom,
  }
})
