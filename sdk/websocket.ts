import type {
  ChatSDKOptions,
  ConnectionState,
  SendMessageRequest,
  MessageReceivedData,
  MessageSentData,
} from './types';
import { ChatEventType, ConnectionState as State } from './types';
import { EventEmitter, createError, createStateChange } from './utils';
import {
  decodeServerFrame,
  encodeHeartbeat,
  encodeSendMessage,
} from './protobuf';

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

  private connectPromise: Promise<void> | null = null;

  /**
   * 连接到 WebSocket 网关
   */
  async connect(): Promise<void> {
    if (this.isConnected()) {
      return;
    }

    if (this.currentState === State.Connecting && this.connectPromise) {
      return this.connectPromise;
    }

    if (!this.token) {
      throw new Error('Token is required before connecting to WebSocket');
    }

    this.intentionalClose = false;
    this.setState(State.Connecting);

    this.connectPromise = (async () => {
      try {
        if (this.ws) {
          this.ws.close();
        }
        const wsUrl = new URL(this.options.gatewayURL);
        wsUrl.searchParams.append('token', this.token!);
        this.ws = new WebSocket(wsUrl.toString());
        this.ws.binaryType = 'arraybuffer';
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
      } finally {
        this.connectPromise = null;
      }
    })();

    return this.connectPromise;
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
   * 发送 Protobuf 二进制消息
   */
  sendMessage(req: SendMessageRequest): void {
    const encoded = encodeSendMessage(req);

    if (!this.isConnected()) {
      this.messageQueue.push(req);
      this.emitter.emit(
        ChatEventType.Error,
        createError('WS_NOT_CONNECTED', 'WebSocket not connected, message queued')
      );
      return;
    }

    this.ws!.send(encoded as unknown as BufferSource);
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

      this.ws.onmessage = (event: MessageEvent) => {
        this.handleMessage(event.data).catch((error: unknown) => {
          this.emitter.emit(
            ChatEventType.Error,
            createError(
              'MESSAGE_PARSE_ERROR',
              error instanceof Error ? error.message : String(error),
              error instanceof Error ? error : undefined
            )
          );
        });
      };

      this.ws.onclose = (event: CloseEvent) => {
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

      this.ws.onerror = (_event: Event) => {
        this.emitter.emit(
          ChatEventType.Error,
          createError('WS_ERROR', 'WebSocket error occurred')
        );
        reject(new Error('WebSocket error occurred'));
      };
    });
  }

  /**
   * 处理服务端 Protobuf 二进制帧
   */
  private async handleMessage(data: unknown): Promise<void> {
    const bytes = await this.toBytes(data);
    const frame = decodeServerFrame(bytes);

    if (frame.kind === 'message') {
      this.emitter.emit(ChatEventType.MessageReceived, {
        message: frame.message,
      } as MessageReceivedData);
      return;
    }

    const ack = frame.ack;
    const serverTime = Number(ack.serverTime);
    if (ack.code === 0) {
      this.emitter.emit(ChatEventType.MessageSent, {
        client_msg_id: ack.clientMsgId,
        server_msg_id: ack.msgId || undefined,
        server_time: Number.isSafeInteger(serverTime) ? serverTime : undefined,
      } as MessageSentData);
      return;
    }

    this.emitter.emit(
      ChatEventType.Error,
      createError(
        'SEND_FAILED',
        ack.errMsg || `Message send failed with code ${ack.code}`
      )
    );
  }

  private async toBytes(data: unknown): Promise<Uint8Array> {
    if (data instanceof ArrayBuffer) {
      return new Uint8Array(data);
    }

    if (ArrayBuffer.isView(data)) {
      return new Uint8Array(data.buffer, data.byteOffset, data.byteLength);
    }

    if (typeof Blob !== 'undefined' && data instanceof Blob) {
      return new Uint8Array(await data.arrayBuffer());
    }

    throw new Error(`Unsupported WebSocket data type: ${Object.prototype.toString.call(data)}`);
  }

  /**
   * 启动 Protobuf 心跳
   */
  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.ws!.send(encodeHeartbeat() as unknown as BufferSource);
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
