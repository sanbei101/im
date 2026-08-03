import {
  create,
  fromBinary,
  toBinary,
} from '@bufbuild/protobuf';
import {
  MessageBodySchema,
  MessageSchema,
  MessageType as ProtoMessageType,
  SendMessageAckSchema,
  SendMessageReqSchema,
  type Message as ProtoMessage,
  type MessageBody,
  type SendMessageAck,
} from '../gen/ts/proto/im/v1/message_pb';
import {
  MessageType,
} from './types';
import type {
  FilePayload,
  ImagePayload,
  Message,
  SendMessageRequest,
  TextPayload,
  VideoPayload,
} from './types';

export type ServerAck = SendMessageAck;

export type DecodedServerFrame =
  | { kind: 'message'; message: Message }
  | { kind: 'ack'; ack: ServerAck };

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function toProtoMessageType(type: MessageType): ProtoMessageType {
  switch (type) {
    case MessageType.Text:
      return ProtoMessageType.TEXT;
    case MessageType.Image:
      return ProtoMessageType.IMAGE;
    case MessageType.Video:
      return ProtoMessageType.VIDEO;
    case MessageType.File:
      return ProtoMessageType.FILE;
    case MessageType.System:
      return ProtoMessageType.SYSTEM;
    default:
      throw new Error(`Unsupported message type: ${String(type)}`);
  }
}

function fromProtoMessageType(type: ProtoMessageType): MessageType {
  switch (type) {
    case ProtoMessageType.TEXT:
      return MessageType.Text;
    case ProtoMessageType.IMAGE:
      return MessageType.Image;
    case ProtoMessageType.VIDEO:
      return MessageType.Video;
    case ProtoMessageType.FILE:
      return MessageType.File;
    case ProtoMessageType.SYSTEM:
      return MessageType.System;
    default:
      throw new Error(`Unsupported protobuf message type: ${String(type)}`);
  }
}

function safeNumber(value: bigint, fieldName: string): number {
  const numberValue = Number(value);
  if (!Number.isSafeInteger(numberValue)) {
    throw new Error(`${fieldName} is outside JavaScript's safe integer range`);
  }
  return numberValue;
}

function asObject(payload: unknown, type: MessageType): Record<string, unknown> {
  if (payload === null || typeof payload !== 'object' || payload instanceof Uint8Array) {
    throw new Error(`${type} payload must be an object or serialized MessageBody`);
  }
  return payload as Record<string, unknown>;
}

function asString(value: unknown, fieldName: string): string {
  if (typeof value !== 'string') {
    throw new Error(`${fieldName} must be a string`);
  }
  return value;
}

function asOptionalNumber(value: unknown, fieldName: string): number | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
    throw new Error(`${fieldName} must be a non-negative number`);
  }
  return value;
}

function extensionFromName(name: string): string {
  const dot = name.lastIndexOf('.');
  return dot > -1 ? name.slice(dot + 1) : '';
}

function encodePayload(type: MessageType, payload: unknown): Uint8Array {
  if (payload instanceof Uint8Array) {
    return payload;
  }

  const value = asObject(payload, type);

  switch (type) {
    case MessageType.Text: {
      const text = asString(value.text, 'text');
      const atUserIds = Array.isArray(value.at_user_ids)
        ? value.at_user_ids.map((id) => asString(id, 'at_user_ids item'))
        : [];
      return toBinary(MessageBodySchema, create(MessageBodySchema, {
        content: {
          case: 'text',
          value: { text, atUserIds },
        },
      }));
    }
    case MessageType.Image: {
      const image = value as unknown as ImagePayload;
      return toBinary(MessageBodySchema, create(MessageBodySchema, {
        content: {
          case: 'image',
          value: {
            url: asString(image.url, 'url'),
            thumbnailUrl: typeof value.thumbnail_url === 'string' ? value.thumbnail_url : '',
            width: asOptionalNumber(image.width, 'width'),
            height: asOptionalNumber(image.height, 'height'),
            sizeBytes: BigInt(asOptionalNumber(image.size, 'size') ?? 0),
            mimeType: typeof value.mime_type === 'string' ? value.mime_type : '',
          },
        },
      }));
    }
    case MessageType.Video: {
      const video = value as unknown as VideoPayload;
      return toBinary(MessageBodySchema, create(MessageBodySchema, {
        content: {
          case: 'video',
          value: {
            url: asString(video.url, 'url'),
            coverUrl: typeof value.thumbnail_url === 'string' ? value.thumbnail_url : '',
            durationSec: asOptionalNumber(video.duration, 'duration'),
            width: asOptionalNumber(video.width, 'width'),
            height: asOptionalNumber(video.height, 'height'),
            sizeBytes: BigInt(asOptionalNumber(video.size, 'size') ?? 0),
          },
        },
      }));
    }
    case MessageType.File: {
      const file = value as unknown as FilePayload;
      const name = asString(file.name, 'name');
      return toBinary(MessageBodySchema, create(MessageBodySchema, {
        content: {
          case: 'file',
          value: {
            url: asString(file.url, 'url'),
            fileName: name,
            extension: typeof value.extension === 'string'
              ? value.extension
              : extensionFromName(name),
            sizeBytes: BigInt(asOptionalNumber(file.size, 'size') ?? 0),
          },
        },
      }));
    }
    case MessageType.System:
      return toBinary(MessageBodySchema, create(MessageBodySchema, {
        content: {
          case: 'system',
          value: {
            eventCode: typeof value.event_code === 'number' ? value.event_code : 0,
            content: typeof value.content === 'string' ? value.content : '',
            extraParams: value.extra_params as Record<string, string> | undefined,
          },
        },
      }));
    default:
      throw new Error(`Unsupported message type: ${String(type)}`);
  }
}

function encodeExt(ext: SendMessageRequest['ext']): Uint8Array {
  if (ext === undefined) {
    return new Uint8Array();
  }
  if (ext instanceof Uint8Array) {
    return ext;
  }
  return textEncoder.encode(JSON.stringify(ext));
}

export function encodeSendMessage(req: SendMessageRequest): Uint8Array {
  return toBinary(SendMessageReqSchema, create(SendMessageReqSchema, {
    clientMsgId: req.client_msg_id,
    roomId: req.room_id,
    msgType: toProtoMessageType(req.msg_type),
    payload: encodePayload(req.msg_type, req.payload),
    replyToMsgId: req.reply_to_msg_id,
    ext: encodeExt(req.ext),
  }));
}

export function encodeHeartbeat(): Uint8Array {
  return toBinary(SendMessageReqSchema, create(SendMessageReqSchema, {
    msgType: ProtoMessageType.UNSPECIFIED,
  }));
}

function decodeExt(ext: Uint8Array): Record<string, unknown> | Uint8Array | undefined {
  if (ext.length === 0) {
    return undefined;
  }
  try {
    const parsed: unknown = JSON.parse(textDecoder.decode(ext));
    if (parsed !== null && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // ext is allowed to contain arbitrary bytes.
  }
  return ext;
}

function decodeBody(body: MessageBody): unknown {
  switch (body.content.case) {
    case 'text':
      return {
        text: body.content.value.text,
        at_user_ids: body.content.value.atUserIds,
      } satisfies TextPayload & { at_user_ids: string[] };
    case 'image':
      return {
        url: body.content.value.url,
        thumbnail_url: body.content.value.thumbnailUrl || undefined,
        width: body.content.value.width || undefined,
        height: body.content.value.height || undefined,
        size: body.content.value.sizeBytes === 0n
          ? undefined
          : safeNumber(body.content.value.sizeBytes, 'image size'),
        mime_type: body.content.value.mimeType || undefined,
      } satisfies ImagePayload;
    case 'video':
      return {
        url: body.content.value.url,
        duration: body.content.value.durationSec || undefined,
        width: body.content.value.width || undefined,
        height: body.content.value.height || undefined,
        size: body.content.value.sizeBytes === 0n
          ? undefined
          : safeNumber(body.content.value.sizeBytes, 'video size'),
        thumbnail_url: body.content.value.coverUrl || undefined,
      } satisfies VideoPayload;
    case 'file':
      return {
        url: body.content.value.url,
        name: body.content.value.fileName,
        size: body.content.value.sizeBytes === 0n
          ? 0
          : safeNumber(body.content.value.sizeBytes, 'file size'),
        extension: body.content.value.extension || undefined,
      } satisfies FilePayload;
    case 'system':
      return {
        event_code: body.content.value.eventCode,
        content: body.content.value.content,
        extra_params: body.content.value.extraParams,
      };
    case undefined:
      return {};
  }
}

export function decodeMessage(protoMessage: ProtoMessage): Message {
  if (!protoMessage.msgId || !protoMessage.senderId || !protoMessage.roomId) {
    throw new Error('Received protobuf Message is missing required identifiers');
  }

  const body = fromBinary(MessageBodySchema, protoMessage.payload);

  return {
    msg_id: protoMessage.msgId,
    client_msg_id: protoMessage.clientMsgId,
    sender_id: protoMessage.senderId,
    room_id: protoMessage.roomId,
    server_time: safeNumber(protoMessage.serverTime, 'server_time'),
    reply_to_msg_id: protoMessage.replyToMsgId || undefined,
    msg_type: fromProtoMessageType(protoMessage.msgType),
    payload: decodeBody(body),
    ext: decodeExt(protoMessage.ext),
  };
}

export function decodeServerFrame(data: Uint8Array): DecodedServerFrame {
  try {
    const message = decodeMessage(fromBinary(MessageSchema, data));
    return { kind: 'message', message };
  } catch (messageError) {
    try {
      return { kind: 'ack', ack: fromBinary(SendMessageAckSchema, data) };
    } catch {
      throw new Error(
        `Unable to decode server frame: ${messageError instanceof Error ? messageError.message : String(messageError)}`
      );
    }
  }
}

export function decodeAck(data: Uint8Array): SendMessageAck {
  return fromBinary(SendMessageAckSchema, data);
}
