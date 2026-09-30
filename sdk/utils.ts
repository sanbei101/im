import type {
  ChatEventType,
  EventListener,
  ChatEvent,
  ChatEventDataMap,
  ErrorData,
  ConnectionState,
  ConnectionStateChangeData,
  TextPayload,
  ImagePayload,
  VideoPayload,
  FilePayload,
} from "./types";

function isCallable(val: unknown): val is (event: unknown) => void {
  return typeof val === "function";
}

/**
 * 类型安全的事件发射器
 */
export class EventEmitter {
  private listeners: Map<ChatEventType, Set<unknown>> = new Map();

  /**
   * 监听指定事件
   */
  on<T extends ChatEventType>(event: T, listener: EventListener<T>): () => void {
    let set = this.listeners.get(event);
    if (!set) {
      set = new Set<unknown>();
      this.listeners.set(event, set);
    }
    set.add(listener);

    return () => {
      this.off(event, listener);
    };
  }

  /**
   * 监听一次性事件
   */
  once<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    const onceWrapper: EventListener<T> = (e) => {
      this.off(event, onceWrapper);
      listener(e);
    };
    this.on(event, onceWrapper);
  }

  /**
   * 取消事件监听
   */
  off<T extends ChatEventType>(event: T, listener: EventListener<T>): void {
    const set = this.listeners.get(event);
    if (set) {
      set.delete(listener);
      if (set.size === 0) {
        this.listeners.delete(event);
      }
    }
  }

  /**
   * 触发事件
   */
  emit<T extends ChatEventType>(event: T, data: ChatEventDataMap[T]): void {
    const set = this.listeners.get(event);
    if (set && set.size > 0) {
      const eventObj: ChatEvent<T> = {
        type: event,
        data,
        timestamp: Date.now(),
      };
      set.forEach((fn) => {
        try {
          if (isCallable(fn)) {
            fn(eventObj);
          }
        } catch (err) {
          console.error(`[ChatSDK] Event listener error for ${event}:`, err);
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
 * 生成 UUID
 */
export function generateUUID(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

/**
 * 校验是否为合法的 UUID 字符串
 */
export function isValidUUID(str: string): boolean {
  return UUID_REGEX.test(str);
}

/**
 * 延迟等待
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export const sleep = delay;

/**
 * 构建规范的错误对象
 */
export function createError(code: string, message: string, originalError?: Error): ErrorData {
  return {
    code,
    message,
    originalError,
  };
}

/**
 * 构建连接状态变更数据
 */
export function createStateChange(
  state: ConnectionState,
  previousState: ConnectionState,
): ConnectionStateChangeData {
  return {
    state,
    previousState,
  };
}

/**
 * 文本负载类型守卫
 */
export function isTextPayload(payload: unknown): payload is TextPayload {
  return (
    typeof payload === "object" &&
    payload !== null &&
    "text" in payload &&
    typeof Reflect.get(payload, "text") === "string"
  );
}

/**
 * 图片负载类型守卫
 */
export function isImagePayload(payload: unknown): payload is ImagePayload {
  return (
    typeof payload === "object" &&
    payload !== null &&
    "url" in payload &&
    typeof Reflect.get(payload, "url") === "string"
  );
}

/**
 * 视频负载类型守卫
 */
export function isVideoPayload(payload: unknown): payload is VideoPayload {
  return (
    typeof payload === "object" &&
    payload !== null &&
    "url" in payload &&
    typeof Reflect.get(payload, "url") === "string"
  );
}

/**
 * 文件负载类型守卫
 */
export function isFilePayload(payload: unknown): payload is FilePayload {
  return (
    typeof payload === "object" &&
    payload !== null &&
    "url" in payload &&
    "name" in payload &&
    typeof Reflect.get(payload, "url") === "string" &&
    typeof Reflect.get(payload, "name") === "string"
  );
}
