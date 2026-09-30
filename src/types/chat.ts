import type {
  Message,
  TextPayload,
  ImagePayload,
  FilePayload,
  VideoPayload,
  MessageType,
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
  };
}

export function getMessagePreviewText(message: UIMessage): string {
  if (message.msgType === "image") {
    return "[Image]";
  }
  if (message.msgType === "file") {
    if (isFilePayload(message.payload)) {
      return `[File] ${message.payload.name}`;
    }
    return "[File]";
  }
  if (message.msgType === "video") {
    return "[Video]";
  }
  if (isTextPayload(message.payload)) {
    return message.payload.text;
  }
  if (typeof message.payload === "string") {
    return message.payload;
  }
  return "[Message]";
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

export function getInitials(name: string): string {
  const trimmed = name.trim();
  if (trimmed.length === 0) {
    return "?";
  }
  return trimmed.slice(0, 2).toUpperCase();
}
