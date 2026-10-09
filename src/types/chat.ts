import { MessageType } from "go-chat-sdk";
import type {
  ConversationInfo,
  Message,
  TextPayload,
  ImagePayload,
  FilePayload,
  VideoPayload,
  ReactionGroup,
} from "go-chat-sdk";

export type MessageStatus = "sending" | "sent" | "error";

export interface UIMessage {
  readonly id: string;
  readonly clientMsgId: string;
  readonly senderId: string;
  readonly roomId: string;
  readonly roomSeq: number;
  readonly serverTime: number;
  readonly msgType: MessageType | string;
  readonly payload: unknown;
  readonly replyToMsgId?: string;
  readonly status: MessageStatus;
  readonly errorMessage?: string;
  /** Set once the server confirms the message was recalled. */
  readonly recalled?: boolean;
  /** True when this message is pinned in its room. */
  readonly pinned?: boolean;
  /** Latest reaction summary for this message, if loaded. */
  readonly reactions?: readonly ReactionGroup[];
}

export interface ServerConfig {
  readonly baseURL: string;
  readonly gatewayURL: string;
}

export function isTextPayload(payload: unknown): payload is TextPayload {
  if (typeof payload !== "object" || payload === null) {
    return false;
  }
  return "text" in payload && typeof payload.text === "string";
}

export function isImagePayload(payload: unknown): payload is ImagePayload {
  if (typeof payload !== "object" || payload === null) {
    return false;
  }
  return "url" in payload && typeof payload.url === "string";
}

export function isFilePayload(payload: unknown): payload is FilePayload {
  if (typeof payload !== "object" || payload === null) {
    return false;
  }
  return (
    "url" in payload &&
    typeof payload.url === "string" &&
    "name" in payload &&
    typeof payload.name === "string"
  );
}

export function isVideoPayload(payload: unknown): payload is VideoPayload {
  if (typeof payload !== "object" || payload === null) {
    return false;
  }
  return "url" in payload && typeof payload.url === "string";
}

export function isErrorWithMessage(error: unknown): error is { message: string } {
  if (typeof error !== "object" || error === null) {
    return false;
  }
  return "message" in error && typeof error.message === "string";
}

export function mapSdkMessageToUIMessage(msg: Message, status: MessageStatus = "sent"): UIMessage {
  return {
    id: msg.msg_id || msg.client_msg_id,
    clientMsgId: msg.client_msg_id,
    senderId: msg.sender_id,
    roomId: msg.room_id,
    roomSeq: msg.room_seq,
    serverTime: msg.server_time,
    msgType: msg.msg_type,
    payload: msg.payload,
    replyToMsgId: msg.reply_to_msg_id || undefined,
    status,
    reactions: msg.reactions,
    // The server rewrites a recalled message to msg_type=recall, so both the
    // optimistic flag and the server type mark a recalled message.
    recalled: msg.msg_type === MessageType.Recall || msg.msg_type === "recall",
  };
}

export function getMessagePreviewText(message?: UIMessage | Message | null): string {
  if (!message) {
    return "";
  }
  const msgType = "msgType" in message ? message.msgType : message.msg_type;
  const isRecalled =
    ("recalled" in message && message.recalled) ||
    msgType === MessageType.Recall ||
    msgType === "recall";

  if (isRecalled) {
    return "[撤回了一条消息]";
  }

  if (msgType === "image" || msgType === MessageType.Image) {
    return "[图片]";
  }
  if (msgType === "file" || msgType === MessageType.File) {
    if (isFilePayload(message.payload)) {
      return `[文件] ${message.payload.name}`;
    }
    return "[文件]";
  }
  if (msgType === "video" || msgType === MessageType.Video) {
    return "[视频]";
  }
  if (isTextPayload(message.payload)) {
    return message.payload.text;
  }
  if (typeof message.payload === "string") {
    return message.payload;
  }
  return "[消息]";
}

export function formatTime(timestamp: number): string {
  if (!timestamp || timestamp <= 0) {
    return "";
  }
  // Server timestamps might be in microseconds (16 digits) or milliseconds (13 digits)
  const ms = timestamp > 10_000_000_000_000 ? Math.floor(timestamp / 1000) : timestamp;
  const date = new Date(ms);
  const hours = date.getHours().toString().padStart(2, "0");
  const minutes = date.getMinutes().toString().padStart(2, "0");
  return `${hours}:${minutes}`;
}

export function formatRelativeTime(time?: number | string | null): string {
  if (!time) {
    return "";
  }
  let ms: number;
  if (typeof time === "string") {
    ms = new Date(time).getTime();
  } else {
    ms = time > 10_000_000_000_000 ? Math.floor(time / 1000) : time;
  }
  if (Number.isNaN(ms) || ms <= 0) {
    return "";
  }
  const date = new Date(ms);
  const now = new Date();
  const diffMs = now.getTime() - ms;

  if (diffMs >= 0 && diffMs < 60_000) {
    return "刚刚";
  }

  const isSameDay =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  if (isSameDay) {
    const hours = date.getHours().toString().padStart(2, "0");
    const minutes = date.getMinutes().toString().padStart(2, "0");
    return `${hours}:${minutes}`;
  }

  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  const isYesterday =
    date.getFullYear() === yesterday.getFullYear() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getDate() === yesterday.getDate();

  if (isYesterday) {
    return "昨天";
  }

  if (date.getFullYear() === now.getFullYear()) {
    const month = (date.getMonth() + 1).toString().padStart(2, "0");
    const day = date.getDate().toString().padStart(2, "0");
    return `${month}-${day}`;
  }

  const year = date.getFullYear();
  const month = (date.getMonth() + 1).toString().padStart(2, "0");
  const day = date.getDate().toString().padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function sortConversations(a: ConversationInfo, b: ConversationInfo): number {
  if (a.member.is_pinned !== b.member.is_pinned) {
    return a.member.is_pinned ? -1 : 1;
  }
  const timeA =
    a.last_message?.server_time ??
    new Date(a.room.updated_at || a.room.created_at || 0).getTime() * 1000;
  const timeB =
    b.last_message?.server_time ??
    new Date(b.room.updated_at || b.room.created_at || 0).getTime() * 1000;
  return timeB - timeA;
}

export function formatFileSize(bytes?: number): string {
  if (typeof bytes !== "number" || bytes <= 0) {
    return "0 B";
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function updateMessageStatus(
  message: UIMessage,
  status: MessageStatus,
  errorMessage?: string,
): UIMessage {
  return {
    ...message,
    status,
    errorMessage,
  };
}

export function updateMessageAck(
  message: UIMessage,
  serverMsgId: string,
  roomSeq: number,
  serverTime: number,
): UIMessage {
  return {
    ...message,
    id: serverMsgId,
    roomSeq,
    serverTime,
    status: "sent",
  };
}

/** Marks a message as recalled, which the bubble renders as a placeholder. */
export function markRecalled(message: UIMessage): UIMessage {
  return { ...message, recalled: true };
}

/** Applies a reaction summary on top of an existing message. */
export function applyReactions(
  messages: readonly UIMessage[],
  messageId: string,
  reactions: readonly ReactionGroup[],
): readonly UIMessage[] {
  return messages.map((m) =>
    m.id === messageId || m.clientMsgId === messageId ? { ...m, reactions } : m,
  );
}

export function getInitials(name: string): string {
  const trimmed = name.trim();
  if (trimmed.length === 0) {
    return "?";
  }
  return trimmed.slice(0, 2).toUpperCase();
}
