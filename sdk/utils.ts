import type {
  ChatEventType,
  EventListener,
  ChatEvent,
  ErrorData,
  ConnectionState,
  ConnectionStateChangeData,
} from './types';

/**
 * 事件发射器 - 用于SDK内部事件管理
 */
export class EventEmitter {
  private listeners: Map<ChatEventType, Set<EventListener<ChatEventType>>> = new Map();

  /**
   * 监听事件
   */
  on<T extends ChatEventType>(event: T, listener: EventListener<T>): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(listener as EventListener<ChatEventType>);

    // 返回取消订阅函数
    return () => {
      this.off(event, listener);
    };
  }

  /**
   * 监听一次性事件
   */
  once<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    const onceWrapper = (e: ChatEvent<T>) => {
      this.off(event, onceWrapper as EventListener<T>);
      listener(e);
    };
    this.on(event, onceWrapper as EventListener<T>);
  }

  /**
   * 取消监听
   */
  off<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    const set = this.listeners.get(event);
    if (set) {
      set.delete(listener as EventListener<ChatEventType>);
      if (set.size === 0) {
        this.listeners.delete(event);
      }
    }
  }

  /**
   * 触发事件
   */
  emit<T extends ChatEventType>(event: T, data: ChatEvent<T>['data']): void {
    const set = this.listeners.get(event);
    if (set) {
      const eventObj: ChatEvent<T> = {
        type: event,
        data,
        timestamp: Date.now(),
      };
      set.forEach((listener) => {
        try {
          listener(eventObj as ChatEvent<ChatEventType>);
        } catch (err) {
          console.error(`Event listener error for ${event}:`, err);
        }
      });
    }
  }

  /**
   * 移除所有监听器
   */
  removeAllListeners(event?: ChatEventType): void {
    if (event) {
      this.listeners.delete(event);
    } else {
      this.listeners.clear();
    }
  }
}

/**
 * 生成UUID v4
 */
export function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * 检查字符串是否为有效的UUID
 */
export function isValidUUID(str: string): boolean {
  const uuidRegex =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
  return uuidRegex.test(str);
}

/**
 * 延迟函数
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * 创建错误数据对象
 */
export function createError(
  code: string,
  message: string,
  originalError?: Error
): ErrorData {
  return {
    code,
    message,
    originalError,
  };
}

/**
 * 创建连接状态变更数据
 */
export function createStateChange(
  state: ConnectionState,
  previousState: ConnectionState
): ConnectionStateChangeData {
  return {
    state,
    previousState,
  };
}