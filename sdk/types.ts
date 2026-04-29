// 聊天类型枚举
export enum ChatType {
  Single = 'single',
  Group = 'group',
}

// 消息类型枚举
export enum MessageType {
  Text = 'text',
  Image = 'image',
  Video = 'video',
  File = 'file',
}

// 消息数据结构
export interface Message {
  /** 服务器生成的消息ID */
  msg_id: string;
  /** 客户端生成的消息ID(用于去重) */
  client_msg_id: string;
  /** 发送者ID */
  sender_id: string;
  /** 接收者ID(用户ID或群ID) */
  room_id: string;
  /** 聊天类型: single(单聊) / group(群聊) */
  chat_type: ChatType;
  /** 服务器时间戳(微秒级) */
  server_time: number;
  /** 回复的消息ID(可选) */
  reply_to_msg_id?: string;
  /** 消息类型: text/image/video/file */
  msg_type: MessageType;
  /** 消息内容负载 */
  payload: unknown;
  /** 扩展字段 */
  ext?: Record<string, unknown>;
  /** 创建时间 */
  created_at?: string;
}

// 发送消息的请求结构(客户端需要构造的)
export interface SendMessageRequest {
  /** 客户端生成的唯一消息ID */
  client_msg_id?: string;
  /** 接收者ID */
  room_id: string;
  /** 聊天类型 */
  chat_type: ChatType;
  /** 消息类型 */
  msg_type: MessageType;
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

// 批量生成用户请求
export interface BatchGenerateRequest {
  count: number;
}

// 批量生成用户响应
export interface BatchUserResponse {
  user_id: string;
  username: string;
  password: string;
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
}

// 连接状态
export enum ConnectionState {
  Disconnected = 'disconnected',
  Connecting = 'connecting',
  Connected = 'connected',
  Reconnecting = 'reconnecting',
  Error = 'error',
}

// 事件类型
export enum ChatEventType {
  MessageReceived = 'message:received',
  MessageSent = 'message:sent',
  ConnectionStateChange = 'connection:state:change',
  Error = 'error',
  Connect = 'connect',
  Disconnect = 'disconnect',
}

// 消息接收事件数据
export interface MessageReceivedData {
  message: Message;
}

// 消息发送成功事件数据
export interface MessageSentData {
  client_msg_id: string;
  server_msg_id?: string;
  server_time?: number;
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

// 事件数据映射表 - 用于类型推导
export interface ChatEventDataMap {
  [ChatEventType.MessageReceived]: MessageReceivedData;
  [ChatEventType.MessageSent]: MessageSentData;
  [ChatEventType.ConnectionStateChange]: ConnectionStateChangeData;
  [ChatEventType.Error]: ErrorData;
  [ChatEventType.Connect]: ConnectData;
  [ChatEventType.Disconnect]: DisconnectData;
}

// 聊天事件 - 使用映射表实现类型安全
export type ChatEvent<T extends ChatEventType = ChatEventType> = {
  type: T;
  data: T extends keyof ChatEventDataMap ? ChatEventDataMap[T] : unknown;
  timestamp: number;
};

// 事件监听器类型
export type EventListener<T extends ChatEventType = ChatEventType> = (
  event: ChatEvent<T>
) => void;

// 历史消息查询参数
export interface HistoryQueryParams {
  /** 接收者ID */
  room_id: string;
  /** 聊天类型 */
  chat_type: ChatType;
  /** 查询此时间戳之前的消息(微秒级,默认为当前时间) */
  before_server_time?: number;
  /** 每页数量(默认20) */
  page_size?: number;
}

// 历史消息响应
export interface HistoryMessagesResponse {
  messages: Message[];
  hasMore: boolean;
}
