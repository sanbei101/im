import { APIClient } from "./api";
import type {
  ChatSDKOptions,
  ConnectionState,
  SendMessageRequest,
  RegisterRequest,
  LoginRequest,
  UserResponse,
  HistoryQueryParams,
  HistoryMessagesResponse,
  EventListener,
  TextPayload,
  ImagePayload,
  VideoPayload,
  FilePayload,
  CreateRoomRequest,
  CreateRoomResponse,
  CreateGroupRoomRequest,
  ListRoomsResponse,
  AckFrame,
} from "./types";
import { MessageType, type ChatEventType } from "./types";
import { EventEmitter, generateUUID, isValidUUID } from "./utils";
import { WebSocketManager } from "./websocket";

export type {
  ChatSDKOptions,
  SendMessageRequest,
  RegisterRequest,
  LoginRequest,
  UserResponse,
  HistoryQueryParams,
  HistoryMessagesResponse,
  EventListener,
  TextPayload,
  ImagePayload,
  VideoPayload,
  FilePayload,
  CreateRoomRequest,
  CreateRoomResponse,
  CreateGroupRoomRequest,
  ListRoomsResponse,
  AckFrame,
  Message,
  MessagePushFrame,
  PongFrame,
  ErrorFrame,
  GatewayFrame,
  MessageReceivedData,
  MessageSentData,
  ConnectionStateChangeData,
  ErrorData,
  ConnectData,
  DisconnectData,
  ChatEventDataMap,
  ChatEvent,
  RoomInfo,
} from "./types";
export { ChatType, MessageType, ConnectionState, ChatEventType } from "./types";
export * from "./utils";
export { APIClient, APIError } from "./api";
export { WebSocketManager } from "./websocket";

/**
 * ChatSDK - 类型安全的高性能即时通讯 SDK
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
      ackTimeout: options.ackTimeout ?? 10000,
    };

    this.emitter = new EventEmitter();
    this.api = new APIClient(this.options.baseURL);
    this.wsManager = new WebSocketManager(this.options, this.emitter);
  }

  // ==================== 事件监听 ====================

  /**
   * 监听指定事件 (返回取消订阅函数)
   */
  on<T extends ChatEventType>(event: T, listener: EventListener<T>): () => void {
    return this.emitter.on(event, listener);
  }

  /**
   * 监听一次性事件
   */
  once<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    this.emitter.once(event, listener);
  }

  /**
   * 取消事件监听
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
   * 用户注册并自动登录
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
   * 获取当前已认证用户信息
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
      throw new Error("Must be authenticated before connecting to WebSocket");
    }
    return this.wsManager.connect();
  }

  /**
   * 断开连接
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
   * 检查连接是否可用
   */
  isConnected(): boolean {
    return this.wsManager.isConnected();
  }

  // ==================== 消息发送 ====================

  /**
   * 发送原始消息 (返回服务端持久化 ACK Promise)
   */
  sendMessage(req: SendMessageRequest): Promise<AckFrame> {
    if (!req.client_msg_id) {
      req.client_msg_id = generateUUID();
    }
    return this.wsManager.sendMessage(req);
  }

  /**
   * 发送文本消息
   */
  sendTextMessage(
    params: Omit<SendMessageRequest, "msg_type" | "payload"> & { text: string },
  ): Promise<AckFrame> {
    const { text, ...rest } = params;
    const payload: TextPayload = { text };
    return this.sendMessage({
      ...rest,
      msg_type: MessageType.Text,
      payload,
    });
  }

  /**
   * 发送图片消息
   */
  sendImageMessage(
    params: Omit<SendMessageRequest, "msg_type" | "payload"> & ImagePayload,
  ): Promise<AckFrame> {
    const { url, width, height, size, ...rest } = params;
    const payload: ImagePayload = { url, width, height, size };
    return this.sendMessage({
      ...rest,
      msg_type: MessageType.Image,
      payload,
    });
  }

  /**
   * 发送视频消息
   */
  sendVideoMessage(
    params: Omit<SendMessageRequest, "msg_type" | "payload"> & VideoPayload,
  ): Promise<AckFrame> {
    const { url, duration, width, height, size, thumbnail_url, ...rest } = params;
    const payload: VideoPayload = { url, duration, width, height, size, thumbnail_url };
    return this.sendMessage({
      ...rest,
      msg_type: MessageType.Video,
      payload,
    });
  }

  /**
   * 发送文件消息
   */
  sendFileMessage(
    params: Omit<SendMessageRequest, "msg_type" | "payload"> & FilePayload,
  ): Promise<AckFrame> {
    const { url, name, size, mime_type, ...rest } = params;
    const payload: FilePayload = { url, name, size, mime_type };
    return this.sendMessage({
      ...rest,
      msg_type: MessageType.File,
      payload,
    });
  }

  // ==================== 历史消息 ====================

  /**
   * 获取房间历史消息
   */
  async getHistoryMessages(params: HistoryQueryParams): Promise<HistoryMessagesResponse> {
    return this.api.getHistoryMessages(params);
  }

  // ==================== 房间管理 ====================

  /**
   * 创建或获取单聊房间
   */
  async createRoom(req: CreateRoomRequest): Promise<CreateRoomResponse> {
    return this.api.createRoom(req);
  }

  /**
   * 创建群聊房间
   */
  async createGroupRoom(req: CreateGroupRoomRequest): Promise<CreateRoomResponse> {
    return this.api.createGroupRoom(req);
  }

  /**
   * 获取当前用户的所有房间列表
   */
  async listRooms(): Promise<ListRoomsResponse> {
    return this.api.listRooms();
  }

  // ==================== 辅助工具 ====================

  generateMessageId(): string {
    return generateUUID();
  }

  validateMessageId(id: string): boolean {
    return isValidUUID(id);
  }
}

export default ChatSDK;
