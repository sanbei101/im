import type {
  ChatSDKOptions,
  ConnectionState,
  Message,
  SendMessageRequest,
  MessageReceivedData,
} from './types';
import { ChatEventType, ConnectionState as State } from './types';
import { EventEmitter, createError, createStateChange } from './utils';
import WebSocket from 'ws';

/**
 * WebSocket 连接管理器
 */
export class WebSocketManager {
  private ws: WebSocket | null = null;
  private options: Required<ChatSDKOptions>;
  private emitter: EventEmitter;
  private currentState: ConnectionState = State.Disconnected;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private messageQueue: SendMessageRequest[] = [];
  private token: string | null = null;
  private intentionalClose = false;

  constructor(options: ChatSDKOptions, emitter: EventEmitter) {
    this.options = {
      baseURL: options.baseURL,
      gatewayURL: options.gatewayURL,
      reconnectInterval: options.reconnectInterval ?? 3000,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
      heartbeatInterval: options.heartbeatInterval ?? 30000,
      messageBufferSize: options.messageBufferSize ?? 100,
    };
    this.emitter = emitter;
  }

  /**
   * 设置认证 Token
   */
  setToken(token: string): void {
    this.token = token;
  }

  /**
   * 清除认证 Token
   */
  clearToken(): void {
    this.token = null;
  }

  /**
   * 获取当前连接状态
   */
  getState(): ConnectionState {
    return this.currentState;
  }

  /**
   * 是否已连接
   */
  isConnected(): boolean {
    return this.currentState === State.Connected && this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * 更新连接状态并触发事件
   */
  private setState(newState: ConnectionState): void {
    const previousState = this.currentState;
    this.currentState = newState;

    this.emitter.emit(ChatEventType.ConnectionStateChange, createStateChange(newState, previousState));
  }

  /**
   * 连接到 WebSocket 网关
   */
  async connect(): Promise<void> {
    if (this.isConnected()) {
      return;
    }

    if (!this.token) {
      throw new Error('Token is required before connecting to WebSocket');
    }

    this.intentionalClose = false;
    this.setState(State.Connecting);

    try {
      const wsUrl = new URL(this.options.gatewayURL);

      this.ws = new WebSocket(wsUrl.toString(), {
        headers: {
          Authorization: `Bearer ${this.token}`,
        },
      });

      await this.setupWebSocketHandlers();
    } catch (error) {
      this.setState(State.Error);
      this.emitter.emit(
        ChatEventType.Error,
        createError(
          'WS_CONNECT_FAILED',
          'Failed to connect to WebSocket',
          error instanceof Error ? error : undefined
        )
      );
      this.scheduleReconnect();
      throw error;
    }
  }

  /**
   * 断开 WebSocket 连接
   */
  disconnect(): void {
    this.intentionalClose = true;
    this.clearTimers();

    if (this.ws) {
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        this.ws.close(1000, 'Client disconnect');
      }
      this.ws = null;
    }

    this.reconnectAttempts = 0;
    this.setState(State.Disconnected);
    this.emitter.emit(ChatEventType.Disconnect, { code: 1000, reason: 'Client disconnect' });
  }

  /**
   * 发送消息
   */
  sendMessage(req: SendMessageRequest): void {
    if (!this.isConnected()) {
      // 如果未连接,将消息加入队列
      this.messageQueue.push(req);
      this.emitter.emit(
        ChatEventType.Error,
        createError('WS_NOT_CONNECTED', 'WebSocket not connected, message queued')
      );
      return;
    }

    const message = this.buildMessage(req);
    this.ws!.send(JSON.stringify(message));
  }

  /**
   * 构建发送的消息对象
   */
  private buildMessage(req: SendMessageRequest): Record<string, unknown> {
    return {
      client_msg_id: req.client_msg_id,
      room_id: req.room_id,
      chat_type: req.chat_type,
      msg_type: req.msg_type,
      payload: req.payload,
      ...(req.reply_to_msg_id && { reply_to_msg_id: req.reply_to_msg_id }),
      ...(req.ext && { ext: req.ext }),
    };
  }

  /**
   * 设置 WebSocket 事件处理器
   */
  private setupWebSocketHandlers(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!this.ws) {
        reject(new Error('WebSocket instance is null'));
        return;
      }

      // 连接打开
      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.setState(State.Connected);
        this.emitter.emit(ChatEventType.Connect, { timestamp: Date.now() });
        this.startHeartbeat();
        this.flushMessageQueue();
        resolve();
      };

      // 接收消息
      this.ws.onmessage = (event) => {
        const data = typeof event.data === 'string' ? event.data : event.data.toString();
        this.handleMessage(data);
      };

      // 连接关闭
      this.ws.onclose = (event) => {
        this.clearTimers();

        if (this.intentionalClose) {
          this.setState(State.Disconnected);
          this.emitter.emit(ChatEventType.Disconnect, {
            code: event.code,
            reason: event.reason,
          });
        } else {
          this.setState(State.Reconnecting);
          this.scheduleReconnect();
        }
      };

      // 连接错误
      this.ws.onerror = (error) => {
        const err = error as unknown as Error;
        this.emitter.emit(
          ChatEventType.Error,
          createError('WS_ERROR', err.message || 'WebSocket error occurred', err)
        );
        reject(err);
      };
    });
  }

  /**
   * 处理接收到的消息
   */
  private handleMessage(data: string): void {
    try {
      const message = JSON.parse(data) as Message;

      // 验证消息格式
      if (!message.msg_id || !message.sender_id) {
        this.emitter.emit(
          ChatEventType.Error,
          createError('INVALID_MESSAGE', 'Received invalid message format')
        );
        return;
      }

      this.emitter.emit(ChatEventType.MessageReceived, {
        message,
      } as MessageReceivedData);
    } catch (error) {
      // 可能是服务器返回的错误文本
      this.emitter.emit(
        ChatEventType.Error,
        createError(
          'MESSAGE_PARSE_ERROR',
          data,
          error instanceof Error ? error : undefined
        )
      );
    }
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        // 发送 ping 帧(WebSocket 原生支持)
        // 或者可以发送自定义心跳消息
        this.ws!.send(JSON.stringify({ type: 'ping' }));
      }
    }, this.options.heartbeatInterval);
  }

  /**
   * 安排重连
   */
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this.setState(State.Error);
      this.emitter.emit(
        ChatEventType.Error,
        createError('MAX_RECONNECT_REACHED', 'Maximum reconnection attempts reached')
      );
      return;
    }

    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      this.connect().catch(() => {
        // 重连失败,会继续触发 onclose 事件,从而再次安排重连
      });
    }, this.options.reconnectInterval * this.reconnectAttempts); // 指数退避
  }

  /**
   * 清空定时器
   */
  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /**
   * 刷新消息队列(连接成功后发送缓存的消息)
   */
  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const req = this.messageQueue.shift();
      if (req) {
        this.sendMessage(req);
      }
    }
  }
}
