import { APIClient } from "./api";
import type {
  AddMembersRequest,
  ApplyFriendRequest,
  AuditFriendRequest,
  BlacklistRequest,
  ChatSDKOptions,
  ConnectionState,
  DeviceTokenRequest,
  EventListener,
  FilePayload,
  HistoryMessagesResponse,
  HistoryQueryParams,
  ImagePayload,
  ListConversationsResponse,
  ListRoomsResponse,
  LoginRequest,
  MarkReadRequest,
  MemberInfo,
  Message,
  MuteConversationRequest,
  PinConversationRequest,
  PinMessageRequest,
  PresenceRequest,
  PresenceResponse,
  PresignUploadRequest,
  PresignUploadResponse,
  ReactionGroup,
  ReactionRequest,
  ReadUsersResponse,
  RecallMessageRequest,
  RegisterRequest,
  RoomDetail,
  SearchMessagesParams,
  SendMessageRequest,
  TextPayload,
  TransferOwnerRequest,
  UpdateMemberRoleRequest,
  UpdatePasswordRequest,
  UpdateProfileRequest,
  UpdateRemarkRequest,
  UpdateRoomRequest,
  UserProfile,
  UserResponse,
  VideoPayload,
  FriendApplication,
  FriendItem,
  CreateRoomRequest,
  CreateRoomResponse,
  CreateGroupRoomRequest,
  AckFrame,
} from "./types";
import { MessageType, type ChatEventType } from "./types";
import { EventEmitter, generateUUID, isValidUUID } from "./utils";
import { WebSocketManager } from "./websocket";

export * from "./types";
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

  /**
   * 发送正在输入状态
   */
  sendTyping(roomId: string): void {
    this.wsManager.sendTyping(roomId);
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

  async getProfile(userId?: string): Promise<UserProfile> {
    return this.api.getProfile(userId);
  }

  async updateProfile(req: UpdateProfileRequest): Promise<UserProfile> {
    return this.api.updateProfile(req);
  }

  async searchUsers(keyword: string): Promise<UserProfile[]> {
    return this.api.searchUsers(keyword);
  }

  async getPresence(req: PresenceRequest): Promise<PresenceResponse> {
    return this.api.getPresence(req);
  }

  async updatePassword(req: UpdatePasswordRequest): Promise<void> {
    return this.api.updatePassword(req);
  }

  async logout(): Promise<void> {
    try {
      await this.api.logout();
    } finally {
      this.disconnect();
      this.clearAuth();
    }
  }

  async saveDeviceToken(req: DeviceTokenRequest): Promise<void> {
    return this.api.saveDeviceToken(req);
  }

  async searchMessages(params: SearchMessagesParams): Promise<Message[]> {
    return this.api.searchMessages(params);
  }

  async searchRoomMessages(
    roomId: string,
    params: Omit<SearchMessagesParams, "room_id">,
  ): Promise<Message[]> {
    return this.api.searchRoomMessages(roomId, params);
  }

  async recallMessage(req: RecallMessageRequest): Promise<Message> {
    return this.api.recallMessage(req);
  }

  async addReaction(messageId: string, req: ReactionRequest): Promise<void> {
    return this.api.addReaction(messageId, req);
  }

  async removeReaction(messageId: string, req: ReactionRequest): Promise<void> {
    return this.api.removeReaction(messageId, req);
  }

  async getReactions(messageId: string, roomId: string): Promise<ReactionGroup[]> {
    return this.api.getReactions(messageId, roomId);
  }

  async getReadUsers(messageId: string, roomId: string): Promise<ReadUsersResponse> {
    return this.api.getReadUsers(messageId, roomId);
  }

  async getRoom(roomId: string): Promise<RoomDetail> {
    return this.api.getRoom(roomId);
  }

  async updateRoom(roomId: string, req: UpdateRoomRequest): Promise<RoomDetail> {
    return this.api.updateRoom(roomId, req);
  }

  async deleteRoom(roomId: string): Promise<void> {
    return this.api.deleteRoom(roomId);
  }

  async listMembers(roomId: string): Promise<MemberInfo[]> {
    return this.api.listMembers(roomId);
  }

  async addMembers(roomId: string, req: AddMembersRequest): Promise<void> {
    return this.api.addMembers(roomId, req);
  }

  async updateMemberRole(
    roomId: string,
    userId: string,
    req: UpdateMemberRoleRequest,
  ): Promise<void> {
    return this.api.updateMemberRole(roomId, userId, req);
  }

  async removeMember(roomId: string, userId: string): Promise<void> {
    return this.api.removeMember(roomId, userId);
  }

  async leaveRoom(roomId: string): Promise<void> {
    return this.api.leaveRoom(roomId);
  }

  async transferOwner(roomId: string, req: TransferOwnerRequest): Promise<void> {
    return this.api.transferOwner(roomId, req);
  }

  async pinMessage(roomId: string, req: PinMessageRequest): Promise<void> {
    return this.api.pinMessage(roomId, req);
  }

  async unpinMessage(roomId: string, messageId: string): Promise<void> {
    return this.api.unpinMessage(roomId, messageId);
  }

  async getPinnedMessages(roomId: string): Promise<Message[]> {
    return this.api.getPinnedMessages(roomId);
  }

  async listFriends(): Promise<FriendItem[]> {
    return this.api.listFriends();
  }

  async applyFriend(req: ApplyFriendRequest): Promise<void> {
    return this.api.applyFriend(req);
  }

  async auditFriend(req: AuditFriendRequest): Promise<void> {
    return this.api.auditFriend(req);
  }

  async listFriendApplications(): Promise<FriendApplication[]> {
    return this.api.listFriendApplications();
  }

  async deleteFriend(userId: string): Promise<void> {
    return this.api.deleteFriend(userId);
  }

  async updateFriendRemark(userId: string, req: UpdateRemarkRequest): Promise<void> {
    return this.api.updateFriendRemark(userId, req);
  }

  async listBlacklist(): Promise<UserProfile[]> {
    return this.api.listBlacklist();
  }

  async addBlacklist(req: BlacklistRequest): Promise<void> {
    return this.api.addBlacklist(req);
  }

  async removeBlacklist(userId: string): Promise<void> {
    return this.api.removeBlacklist(userId);
  }

  async listConversations(): Promise<ListConversationsResponse> {
    return this.api.listConversations();
  }

  async markRead(roomId: string, req: MarkReadRequest): Promise<void> {
    return this.api.markRead(roomId, req);
  }

  async clearUnread(roomId: string): Promise<void> {
    return this.api.clearUnread(roomId);
  }

  async pinConversation(roomId: string, req: PinConversationRequest): Promise<void> {
    return this.api.pinConversation(roomId, req);
  }

  async muteConversation(roomId: string, req: MuteConversationRequest): Promise<void> {
    return this.api.muteConversation(roomId, req);
  }

  async presignUpload(req: PresignUploadRequest): Promise<PresignUploadResponse> {
    return this.api.presignUpload(req);
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
