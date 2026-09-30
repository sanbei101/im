import type {
  ChatSDKOptions,
  ConnectionState,
  Message,
  SendMessageRequest,
  GatewayFrame,
  AckFrame,
  MessagePushFrame,
} from './types';
import { ChatEventType, ConnectionState as State } from './types';
import { EventEmitter, createError, createStateChange, generateUUID } from './utils';

interface PendingMessage {
  resolve: (ack: AckFrame) => void;
  reject: (err: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

/**
 * WebSocket 长连接管理器
 */
export class WebSocketManager {
  private ws: WebSocket | null = null;
  private options: Required<ChatSDKOptions>;
  private emitter: EventEmitter;
  private currentState: ConnectionState = State.Disconnected;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private messageQueue: { req: SendMessageRequest; resolve: (ack: AckFrame) => void; reject: (err: Error) => void }[] = [];
  private pendingRequests: Map<string, PendingMessage> = new Map();
  private token: string | null = null;
  private intentionalClose = false;
  private connectPromise: Promise<void> | null = null;

  constructor(options: ChatSDKOptions, emitter: EventEmitter) {
    this.options = {
      baseURL: options.baseURL,
      gatewayURL: options.gatewayURL,
      reconnectInterval: options.reconnectInterval ?? 3000,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
      heartbeatInterval: options.heartbeatInterval ?? 30000,
      messageBufferSize: options.messageBufferSize ?? 100,
      ackTimeout: options.ackTimeout ?? 10000,
    };
    this.emitter = emitter;
  }

  setToken(token: string): void {
    this.token = token;
  }

  clearToken(): void {
    this.token = null;
  }

  getState(): ConnectionState {
    return this.currentState;
  }

  isConnected(): boolean {
    return this.currentState === State.Connected && this.ws?.readyState === WebSocket.OPEN;
  }

  private setState(newState: ConnectionState): void {
    if (this.currentState === newState) {
      return;
    }
    const previousState = this.currentState;
    this.currentState = newState;
    this.emitter.emit(ChatEventType.ConnectionStateChange, createStateChange(newState, previousState));
  }

  /**
   * 建立 WebSocket 连接
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
          this.ws = null;
        }
        const wsUrl = new URL(this.options.gatewayURL);
        wsUrl.searchParams.set('token', this.token!);

        this.ws = new WebSocket(wsUrl.toString());
        this.ws.binaryType = 'arraybuffer';

        await this.setupWebSocketHandlers();
      } catch (error) {
        this.setState(State.Error);
        this.emitter.emit(
          ChatEventType.Error,
          createError('WS_CONNECT_FAILED', 'Failed to connect to WebSocket', error instanceof Error ? error : undefined)
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
   * 断开连接
   */
  disconnect(): void {
    this.intentionalClose = true;
    this.clearTimers();
    this.rejectAllPending(new Error('Client disconnected'));

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
   * 发送消息并返回 ACK Promise
   */
  sendMessage(req: SendMessageRequest): Promise<AckFrame> {
    const clientMsgId = req.client_msg_id || generateUUID();
    req.client_msg_id = clientMsgId;

    return new Promise<AckFrame>((resolve, reject) => {
      if (!this.isConnected()) {
        if (this.messageQueue.length >= this.options.messageBufferSize) {
          reject(new Error('WebSocket message buffer overflow'));
          return;
        }
        this.messageQueue.push({ req, resolve, reject });
        return;
      }

      this.dispatchMessage(req, resolve, reject);
    });
  }

  private dispatchMessage(
    req: SendMessageRequest,
    resolve: (ack: AckFrame) => void,
    reject: (err: Error) => void
  ): void {
    const clientMsgId = req.client_msg_id!;

    const timer = setTimeout(() => {
      this.pendingRequests.delete(clientMsgId);
      reject(new Error(`Message ACK timeout (${this.options.ackTimeout}ms) for client_msg_id: ${clientMsgId}`));
    }, this.options.ackTimeout);

    this.pendingRequests.set(clientMsgId, { resolve, reject, timer });

    const payload = {
      client_msg_id: clientMsgId,
      room_id: req.room_id,
      msg_type: req.msg_type,
      payload: req.payload,
      ...(req.reply_to_msg_id ? { reply_to_msg_id: req.reply_to_msg_id } : {}),
      ...(req.ext ? { ext: req.ext } : {}),
    };

    try {
      this.ws!.send(JSON.stringify(payload));
    } catch (err) {
      clearTimeout(timer);
      this.pendingRequests.delete(clientMsgId);
      reject(err instanceof Error ? err : new Error(String(err)));
    }
  }

  private setupWebSocketHandlers(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!this.ws) {
        reject(new Error('WebSocket instance is null'));
        return;
      }

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.setState(State.Connected);
        this.emitter.emit(ChatEventType.Connect, { timestamp: Date.now() });
        this.startHeartbeat();
        this.flushMessageQueue();
        resolve();
      };

      this.ws.onmessage = (event: MessageEvent) => {
        this.handleMessage(event.data);
      };

      this.ws.onclose = (event: CloseEvent) => {
        this.clearTimers();
        this.rejectAllPending(new Error(`WebSocket closed (code: ${event.code})`));

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
        this.emitter.emit(ChatEventType.Error, createError('WS_ERROR', 'WebSocket error occurred'));
        reject(new Error('WebSocket error occurred'));
      };
    });
  }

  /**
   * 解码收到的下行数据
   */
  private decodeData(data: unknown): string {
    if (typeof data === 'string') {
      return data;
    }
    if (data instanceof ArrayBuffer || ArrayBuffer.isView(data)) {
      return new TextDecoder().decode(data);
    }
    return String(data);
  }

  /**
   * 处理 WebSocket 下行帧
   */
  private handleMessage(data: unknown): void {
    const text = this.decodeData(data);

    let frame: GatewayFrame;
    try {
      frame = JSON.parse(text) as GatewayFrame;
    } catch (error) {
      this.emitter.emit(
        ChatEventType.Error,
        createError('MESSAGE_PARSE_ERROR', text, error instanceof Error ? error : undefined)
      );
      return;
    }

    if (!frame || typeof frame !== 'object' || !('type' in frame)) {
      this.emitter.emit(ChatEventType.Error, createError('INVALID_FRAME', 'Received frame without type field'));
      return;
    }

    switch (frame.type) {
      case 'message':
        this.handlePushMessage(frame);
        break;

      case 'ack':
        this.handleAck(frame);
        break;

      case 'pong':
        // 心跳应答正常
        break;

      case 'error':
        this.emitter.emit(
          ChatEventType.Error,
          createError('GATEWAY_ERROR', frame.error || 'Gateway reported an error')
        );
        break;

      default:
        break;
    }
  }

  private handlePushMessage(push: MessagePushFrame): void {
    const message: Message = {
      msg_id: push.msg_id,
      client_msg_id: push.client_msg_id,
      sender_id: push.sender_id,
      room_id: push.room_id,
      room_seq: push.room_seq,
      server_time: push.server_time,
      msg_type: push.msg_type,
      payload: push.payload,
      reply_to_msg_id: push.reply_to_msg_id,
      ext: push.ext,
    };

    this.emitter.emit(ChatEventType.MessageReceived, { message });
  }

  private handleAck(ack: AckFrame): void {
    const pending = this.pendingRequests.get(ack.client_msg_id);
    if (pending) {
      clearTimeout(pending.timer);
      this.pendingRequests.delete(ack.client_msg_id);

      if (ack.code === 0) {
        pending.resolve(ack);
      } else {
        pending.reject(new Error(ack.error || `Server returned error code: ${ack.code}`));
      }
    }

    if (ack.code === 0) {
      this.emitter.emit(ChatEventType.MessageSent, {
        client_msg_id: ack.client_msg_id,
        msg_id: ack.msg_id,
        room_id: ack.room_id,
        room_seq: ack.room_seq,
        server_time: ack.server_time,
      });
    }
  }

  private rejectAllPending(err: Error): void {
    for (const [, pending] of this.pendingRequests) {
      clearTimeout(pending.timer);
      pending.reject(err);
    }
    this.pendingRequests.clear();
  }

  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        try {
          this.ws!.send(JSON.stringify({ type: 'ping' }));
        } catch {
          // ignore write errors, onclose will trigger
        }
      }
    }, this.options.heartbeatInterval);
  }

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
    const delayMs = Math.min(
      this.options.reconnectInterval * Math.pow(1.5, this.reconnectAttempts - 1),
      30000
    );

    this.reconnectTimer = setTimeout(() => {
      this.connect().catch(() => {});
    }, delayMs);
  }

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

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0 && this.isConnected()) {
      const item = this.messageQueue.shift();
      if (item) {
        this.dispatchMessage(item.req, item.resolve, item.reject);
      }
    }
  }
}