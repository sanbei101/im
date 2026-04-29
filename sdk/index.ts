import {
  ChatSDKOptions,
  ConnectionState,
  SendMessageRequest,
  RegisterRequest,
  LoginRequest,
  UserResponse,
  BatchGenerateRequest,
  BatchUserResponse,
  HistoryQueryParams,
  HistoryMessagesResponse,
  ChatEventType,
  EventListener,
  MessageSentData,
  TextPayload,
  ImagePayload,
  VideoPayload,
  FilePayload,
} from './types';

import {
  MessageType,
} from './types';

import { APIClient } from './api';
import { WebSocketManager } from './websocket';
import { EventEmitter, generateUUID, isValidUUID } from './utils';

export * from './types';
export * from './utils';
export { APIClient, APIError } from './api';
export { WebSocketManager } from './websocket';

/**
 * ChatSDK - 类型安全的聊天 SDK
 *
 * 使用示例:
 * ```typescript
 * const sdk = new ChatSDK({
 *   baseURL: 'http://localhost:8080',
 *   gatewayURL: 'ws://localhost:8081/ws',
 * });
 *
 * // 注册/登录
 * await sdk.register({ username: 'test', password: '123456' });
 * await sdk.login({ username: 'test', password: '123456' });
 *
 * // 连接 WebSocket
 * await sdk.connect();
 *
 * // 监听消息
 * sdk.on(ChatEventType.MessageReceived, (event) => {
 *   console.log('收到消息:', event.data.message);
 * });
 *
 * // 发送文本消息
 * sdk.sendTextMessage({
 *   room_id: 'xxx',
 *   chat_type: ChatType.Single,
 *   text: 'Hello!',
 * });
 * ```
 */
export class ChatSDK {
  private api: APIClient;
  private wsManager: WebSocketManager;
  private emitter: EventEmitter;
  private options: Required<ChatSDKOptions>;
  private currentUser: UserResponse | null = null;

  constructor(options: ChatSDKOptions) {
    this.options = {
      baseURL: options.baseURL,
      gatewayURL: options.gatewayURL,
      reconnectInterval: options.reconnectInterval ?? 3000,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
      heartbeatInterval: options.heartbeatInterval ?? 30000,
      messageBufferSize: options.messageBufferSize ?? 100,
    };

    this.emitter = new EventEmitter();
    this.api = new APIClient(this.options.baseURL);
    this.wsManager = new WebSocketManager(this.options, this.emitter);

    // 转发 WebSocket 连接事件到 SDK 层
    this.setupEventForwarding();
  }

  // ==================== 事件监听 ====================

  /**
   * 监听事件 - 类型安全的事件监听
   */
  on<T extends ChatEventType>(event: T, listener: EventListener<T>): () => void {
    return this.emitter.on(event, listener);
  }

  /**
   * 监听一次性事件 - 类型安全的事件监听
   */
  once<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    this.emitter.once(event, listener);
  }

  /**
   * 取消监听
   */
  off<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    this.emitter.off(event, listener);
  }

  /**
   * 移除所有监听器
   */
  removeAllListeners(event?: ChatEventType): void {
    this.emitter.removeAllListeners(event);
  }

  // ==================== 用户认证 ====================

  /**
   * 用户注册
   */
  async register(req: RegisterRequest): Promise<UserResponse> {
    const resp = await this.api.register(req);
    this.setAuth(resp);
    return resp;
  }

  /**
   * 用户登录
   */
  async login(req: LoginRequest): Promise<UserResponse> {
    const resp = await this.api.login(req);
    this.setAuth(resp);
    return resp;
  }

  /**
   * 批量生成用户(测试/管理用途)
   */
  async batchGenerateUsers(
    req: BatchGenerateRequest
  ): Promise<BatchUserResponse[]> {
    const resp = await this.api.batchGenerate(req);
    return resp.users;
  }

  /**
   * 设置认证信息
   */
  setAuth(user: UserResponse): void {
    this.currentUser = user;
    this.api.setToken(user.token);
    this.wsManager.setToken(user.token);
  }

  /**
   * 清除认证信息
   */
  clearAuth(): void {
    this.currentUser = null;
    this.api.clearToken();
    this.wsManager.clearToken();
  }

  /**
   * 获取当前用户信息
   */
  getCurrentUser(): UserResponse | null {
    return this.currentUser;
  }

  /**
   * 检查是否已认证
   */
  isAuthenticated(): boolean {
    return !!this.currentUser && !!this.api.getToken();
  }

  // ==================== WebSocket 连接 ====================

  /**
   * 连接到消息网关
   */
  async connect(): Promise<void> {
    if (!this.isAuthenticated()) {
      throw new Error('Must be authenticated before connecting');
    }
    return this.wsManager.connect();
  }

  /**
   * 断开消息网关连接
   */
  disconnect(): void {
    this.wsManager.disconnect();
  }

  /**
   * 获取连接状态
   */
  getConnectionState(): ConnectionState {
    return this.wsManager.getState();
  }

  /**
   * 是否已连接
   */
  isConnected(): boolean {
    return this.wsManager.isConnected();
  }

  // ==================== 消息发送 ====================

  /**
   * 发送原始消息
   */
  sendMessage(req: SendMessageRequest): void {
    if (!req.client_msg_id) {
      req.client_msg_id = generateUUID();
    }
    this.wsManager.sendMessage(req);

    // 触发发送事件
    this.emitter.emit(ChatEventType.MessageSent, {
      client_msg_id: req.client_msg_id,
    } as MessageSentData);
  }

  /**
   * 发送文本消息
   */
  sendTextMessage(
    params: Omit<SendMessageRequest, 'client_msg_id' | 'msg_type' | 'payload'> & { text: string }
  ): void {
    const { text, ...rest } = params;
    this.sendMessage({
      ...rest,
      msg_type: MessageType.Text,
      payload: { text } as TextPayload,
    });
  }

  /**
   * 发送图片消息
   */
  sendImageMessage(
    params: Omit<SendMessageRequest, 'client_msg_id' | 'msg_type' | 'payload'> & ImagePayload
  ): void {
    const { url, width, height, size, ...rest } = params;
    this.sendMessage({
      ...rest,
      msg_type: MessageType.Image,
      payload: { url, width, height, size } as ImagePayload,
    });
  }

  /**
   * 发送视频消息
   */
  sendVideoMessage(
    params: Omit<SendMessageRequest, 'client_msg_id' | 'msg_type' | 'payload'> & VideoPayload
  ): void {
    const { url, duration, width, height, size, thumbnail_url, ...rest } =
      params;
    this.sendMessage({
      ...rest,
      msg_type: MessageType.Video,
      payload: { url, duration, width, height, size, thumbnail_url } as VideoPayload,
    });
  }

  /**
   * 发送文件消息
   */
  sendFileMessage(
    params: Omit<SendMessageRequest, 'client_msg_id' | 'msg_type' | 'payload'> & FilePayload
  ): void {
    const { url, name, size, mime_type, ...rest } = params;
    this.sendMessage({
      ...rest,
      msg_type: MessageType.File,
      payload: { url, name, size, mime_type } as FilePayload,
    });
  }

  // ==================== 历史消息 ====================

  /**
   * 获取历史消息
   */
  async getHistoryMessages(
    params: HistoryQueryParams
  ): Promise<HistoryMessagesResponse> {
    return this.api.getHistoryMessages(params);
  }

  // ==================== 工具方法 ====================

  /**
   * 生成 UUID(用于 client_msg_id)
   */
  generateMessageId(): string {
    return generateUUID();
  }

  /**
   * 验证 UUID 格式
   */
  validateMessageId(id: string): boolean {
    return isValidUUID(id);
  }

  // ==================== 私有方法 ====================

  /**
   * 设置事件转发
   */
  private setupEventForwarding(): void {
    // WebSocketManager 已经通过同一个 EventEmitter 触发事件
    // 所以不需要额外转发,SDK 层直接监听即可
  }
}

export default ChatSDK;
