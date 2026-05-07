import { ChatSDK } from 'go-chat-sdk'

let _sdk: ChatSDK | null = null

export function getSDK(): ChatSDK {
  if (!_sdk) {
    _sdk = new ChatSDK({
      baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
      gatewayURL: import.meta.env.VITE_WS_GATEWAY_URL || 'ws://localhost:8081/ws',
    })
  }
  return _sdk
}

export type { ChatSDK }
