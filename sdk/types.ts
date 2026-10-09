// 聊天类型枚举
export enum ChatType {
  Single = "single",
  Group = "group",
}

// 消息类型枚举
export enum MessageType {
  Text = "text",
  Image = "image",
  Video = "video",
  File = "file",
  System = "system",
  Recall = "recall",
}

// 消息数据结构
export interface Message {
  /** 服务器生成的消息ID */
  msg_id: string;
  /** 客户端生成的消息ID(用于去重与ACK对齐) */
  client_msg_id: string;
  /** 发送者ID */
  sender_id: string;
  /** 房间ID */
  room_id: string;
  /** 房间内的自增单调递增消息序号 */
  room_seq: number;
  /** 服务器时间戳(微秒级) */
  server_time: number;
  /** 回复的消息ID(可选) */
  reply_to_msg_id?: string;
  /** 消息类型: text/image/video/file/system */
  msg_type: MessageType | string;
  /** 消息内容负载 */
  payload: unknown;
  /** 扩展字段 */
  ext?: Record<string, unknown>;
  /** 表情表态聚合列表 */
  reactions?: ReactionGroup[];
}

// WebSocket 下行帧判别联合 (Discriminated Union)
export interface MessagePushFrame {
  type: "message";
  msg_id: string;
  client_msg_id: string;
  sender_id: string;
  room_id: string;
  room_seq: number;
  server_time: number;
  msg_type: string;
  payload: unknown;
  reply_to_msg_id?: string;
  ext?: Record<string, unknown>;
}

export interface AckFrame {
  type: "ack";
  request_id: string;
  client_msg_id: string;
  msg_id: string;
  room_id: string;
  room_seq: number;
  server_time: number;
  code: number;
  error?: string;
}

export interface PongFrame {
  type: "pong";
}

export interface TypingFrame {
  type: "typing";
  room_id: string;
  user_id?: string;
}

export interface ErrorFrame {
  type: "error";
  error: string;
}

export type GatewayFrame = MessagePushFrame | AckFrame | PongFrame | ErrorFrame | TypingFrame;

// 发送消息的请求结构
export interface SendMessageRequest {
  /** 客户端生成的唯一消息ID(可选,不提供时自动生成) */
  client_msg_id?: string;
  /** 房间ID */
  room_id: string;
  /** 消息类型 */
  msg_type: MessageType | string;
  /** 消息内容负载 */
  payload: unknown;
  /** 回复的消息ID(可选) */
  reply_to_msg_id?: string;
  /** 扩展字段(可选) */
  ext?: Record<string, unknown>;
}

// 文本消息负载
export interface TextPayload {
  text: string;
}

// 图片消息负载
export interface ImagePayload {
  url: string;
  width?: number;
  height?: number;
  size?: number;
}

// 视频消息负载
export interface VideoPayload {
  url: string;
  duration?: number;
  width?: number;
  height?: number;
  size?: number;
  thumbnail_url?: string;
}

// 文件消息负载
export interface FilePayload {
  url: string;
  name: string;
  size: number;
  mime_type?: string;
}

// 用户注册请求
export interface RegisterRequest {
  username: string;
  password: string;
}

// 登录请求
export interface LoginRequest {
  username: string;
  password: string;
}

// 用户响应
export interface UserResponse {
  user_id: string;
  username: string;
  token: string;
}

// SDK配置选项
export interface ChatSDKOptions {
  /** API基础URL */
  baseURL: string;
  /** WebSocket网关URL */
  gatewayURL: string;
  /** 自动重连间隔(毫秒),默认3000ms */
  reconnectInterval?: number;
  /** 最大重连次数,默认10次 */
  maxReconnectAttempts?: number;
  /** 心跳间隔(毫秒),默认30000ms */
  heartbeatInterval?: number;
  /** 消息缓冲区大小,默认100 */
  messageBufferSize?: number;
  /** 消息 ACK 超时时间(毫秒),默认10000ms */
  ackTimeout?: number;
}

// 连接状态
export enum ConnectionState {
  Disconnected = "disconnected",
  Connecting = "connecting",
  Connected = "connected",
  Reconnecting = "reconnecting",
  Error = "error",
}

// 事件类型
export enum ChatEventType {
  MessageReceived = "message:received",
  MessageSent = "message:sent",
  ConnectionStateChange = "connection:state:change",
  Error = "error",
  Connect = "connect",
  Disconnect = "disconnect",
  Typing = "typing",
}

// 消息接收事件数据
export interface MessageReceivedData {
  message: Message;
}

// 正在输入事件数据
export interface TypingReceivedData {
  room_id: string;
  user_id: string;
  timestamp: number;
}

// 消息发送成功事件数据 (ACK)
export interface MessageSentData {
  client_msg_id: string;
  msg_id: string;
  room_id: string;
  room_seq: number;
  server_time: number;
}

// 连接状态变更事件数据
export interface ConnectionStateChangeData {
  state: ConnectionState;
  previousState: ConnectionState;
}

// 错误事件数据
export interface ErrorData {
  code: string;
  message: string;
  originalError?: Error;
}

// 连接事件数据
export interface ConnectData {
  timestamp: number;
}

// 断开连接事件数据
export interface DisconnectData {
  code?: number;
  reason?: string;
}

// 事件数据映射表
export interface ChatEventDataMap {
  [ChatEventType.MessageReceived]: MessageReceivedData;
  [ChatEventType.MessageSent]: MessageSentData;
  [ChatEventType.ConnectionStateChange]: ConnectionStateChangeData;
  [ChatEventType.Error]: ErrorData;
  [ChatEventType.Connect]: ConnectData;
  [ChatEventType.Disconnect]: DisconnectData;
  [ChatEventType.Typing]: TypingReceivedData;
}

// 聊天事件
export type ChatEvent<T extends ChatEventType = ChatEventType> = {
  type: T;
  data: ChatEventDataMap[T];
  timestamp: number;
};

// 事件监听器类型
export type EventListener<T extends ChatEventType = ChatEventType> = (event: ChatEvent<T>) => void;

// 历史消息查询参数
export interface HistoryQueryParams {
  /** 房间ID */
  room_id: string;
  /** 只返回 room_seq 小于该值的消息（排他上界），用于按本地最新序号向前补拉 */
  before_seq?: number;
  /** 每页数量(默认20) */
  page_size?: number;
}

// 历史消息响应
export interface HistoryMessagesResponse {
  messages: Message[];
  has_more: boolean;
}

// 创建房间请求
export interface CreateRoomRequest {
  user_id_2: string;
}

// 创建房间响应
export interface CreateRoomResponse {
  room_id: string;
}

// 创建群聊房间请求
export interface CreateGroupRoomRequest {
  /** 群名称(可选,空则自动生成) */
  name?: string;
  /** 成员用户ID列表(至少2人) */
  member_ids: string[];
}

// 房间信息
export interface RoomInfo {
  room_id: string;
  chat_type: string;
  name: string;
  avatar_url: string;
  notice: string;
}

// 列出用户房间响应
export interface ListRoomsResponse {
  rooms: RoomInfo[];
}

// 用户资料与账号管理
export interface UserProfile {
  user_id: string;
  username: string;
  nickname: string;
  avatar_url: string;
}

export interface UpdateProfileRequest {
  /** 省略或空字符串保持原值;昵称会先去除首尾空白 */
  nickname?: string;
  avatar_url?: string;
}

export interface UpdatePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface PresenceRequest {
  /** 最多100个用户 */
  user_ids: string[];
}

export interface PresenceResponse {
  presence: Record<string, boolean>;
}

export interface DeviceTokenRequest {
  token: string;
  platform: string;
}

// 群聊管理
export type MemberRole = "owner" | "admin" | "member";

export interface RoomDetail extends RoomInfo {
  my_role: MemberRole;
}

export interface UpdateRoomRequest {
  /** 仅非空字段会更新,空字符串或省略均保持原值 */
  name?: string;
  avatar_url?: string;
  notice?: string;
}

export interface MemberInfo extends UserProfile {
  role: MemberRole;
  is_pinned: boolean;
  is_muted: boolean;
}

export interface AddMembersRequest {
  member_ids: string[];
}

export interface UpdateMemberRoleRequest {
  role: "admin" | "member";
}

export interface TransferOwnerRequest {
  new_owner_id: string;
}

export interface PinMessageRequest {
  msg_id: string;
}

// 消息搜索、撤回、表态与已读
export interface SearchMessagesParams extends HistoryQueryParams {
  keyword: string;
}

export interface RecallMessageRequest {
  room_id: string;
  msg_id: string;
}

export interface ReactionRequest {
  room_id: string;
  emoji: string;
}

export interface ReactionGroup {
  emoji: string;
  count: number;
  user_ids: string[];
}

export interface ReadUsersResponse {
  read_user_ids: string[];
  count: number;
}

// 好友与黑名单
export interface FriendItem extends UserProfile {
  remark: string;
  /** RFC3339时间字符串 */
  created_at: string;
}

export interface ApplyFriendRequest {
  target_id: string;
  greeting?: string;
}

export interface AuditFriendRequest {
  from_user_id: string;
  action: "accept" | "reject";
}

export interface UpdateRemarkRequest {
  remark: string;
}

export interface BlacklistRequest {
  target_id: string;
}

export interface FriendApplication {
  from_user_id: string;
  to_user_id: string;
  greeting: string;
  status: "pending" | "accepted" | "rejected";
  created_at: string;
}

// 会话列表返回嵌套的房间、成员和最新消息
export interface ConversationInfo {
  room: RoomInfo & {
    last_seq: number;
    created_at: string;
    updated_at: string;
  };
  member: {
    room_id: string;
    user_id: string;
    role: MemberRole;
    is_hidden: boolean;
    is_muted: boolean;
    is_pinned: boolean;
  };
  unread_count: number;
  last_message?: Message;
}

export interface ListConversationsResponse {
  conversations: ConversationInfo[];
}

export interface MarkReadRequest {
  /** 已读的房间序号,必须大于0 */
  read_seq: number;
}

export interface PinConversationRequest {
  is_pinned: boolean;
}

export interface MuteConversationRequest {
  is_muted: boolean;
}

// 预签名上传:客户端向 upload_url 执行 PUT 上传二进制文件
export interface PresignUploadRequest {
  file_name: string;
  content_type?: string;
}

export interface PresignUploadResponse {
  upload_url: string;
  download_url: string;
  file_key: string;
}
