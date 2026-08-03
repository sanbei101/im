import { describe, expect, it } from 'vitest';
import { create, fromBinary, toBinary } from '@bufbuild/protobuf';
import {
  MessageBodySchema,
  MessageSchema,
  MessageType as ProtoMessageType,
  SendMessageAckSchema,
  SendMessageReqSchema,
} from '../../gen/ts/proto/im/v1/message_pb';
import { MessageType } from '../types';
import {
  decodeServerFrame,
  encodeHeartbeat,
  encodeSendMessage,
} from '../protobuf';

describe('Protobuf WebSocket codec', () => {
  it('编码文本 SendMessageReq 和 MessageBody', () => {
    const bytes = encodeSendMessage({
      client_msg_id: '11111111-1111-7111-8111-111111111111',
      room_id: '22222222-2222-7222-8222-222222222222',
      msg_type: MessageType.Text,
      payload: { text: 'hello', at_user_ids: ['user-1'] },
    });

    const request = fromBinary(SendMessageReqSchema, bytes);
    const body = fromBinary(MessageBodySchema, request.payload);

    expect(request.clientMsgId).toBe('11111111-1111-7111-8111-111111111111');
    expect(request.roomId).toBe('22222222-2222-7222-8222-222222222222');
    expect(request.msgType).toBe(ProtoMessageType.TEXT);
    expect(body.content.case).toBe('text');
    if (body.content.case === 'text') {
      expect(body.content.value.text).toBe('hello');
      expect(body.content.value.atUserIds).toEqual(['user-1']);
    }
  });

  it('编码 UNSPECIFIED 心跳', () => {
    const heartbeat = fromBinary(SendMessageReqSchema, encodeHeartbeat());

    expect(heartbeat.msgType).toBe(ProtoMessageType.UNSPECIFIED);
    expect(heartbeat.roomId).toBe('');
  });

  it('解析下行 Message 并将正文转换为 SDK payload', () => {
    const body = create(MessageBodySchema, {
      content: {
        case: 'text',
        value: { text: 'from server', atUserIds: [] },
      },
    });
    const message = create(MessageSchema, {
      msgId: '33333333-3333-7333-8333-333333333333',
      clientMsgId: '44444444-4444-7444-8444-444444444444',
      senderId: '55555555-5555-7555-8555-555555555555',
      roomId: '22222222-2222-7222-8222-222222222222',
      serverTime: 1700000000000000n,
      msgType: ProtoMessageType.TEXT,
      payload: toBinary(MessageBodySchema, body),
    });

    const frame = decodeServerFrame(toBinary(MessageSchema, message));

    expect(frame.kind).toBe('message');
    if (frame.kind === 'message') {
      expect(frame.message.msg_id).toBe(message.msgId);
      expect(frame.message.sender_id).toBe(message.senderId);
      expect(frame.message.payload).toEqual({
        text: 'from server',
        at_user_ids: [],
      });
    }
  });

  it('解析成功和失败 ACK', () => {
    const success = create(SendMessageAckSchema, {
      clientMsgId: '11111111-1111-7111-8111-111111111111',
      msgId: '33333333-3333-7333-8333-333333333333',
      serverTime: 1700000000000000n,
      code: 0,
    });
    const failure = create(SendMessageAckSchema, {
      clientMsgId: '11111111-1111-7111-8111-111111111111',
      code: -1,
      errMsg: 'enqueue failed',
    });

    const successFrame = decodeServerFrame(toBinary(SendMessageAckSchema, success));
    const failureFrame = decodeServerFrame(toBinary(SendMessageAckSchema, failure));

    expect(successFrame.kind).toBe('ack');
    expect(failureFrame.kind).toBe('ack');
    if (successFrame.kind === 'ack') {
      expect(successFrame.ack.code).toBe(0);
      expect(successFrame.ack.msgId).toBe(success.msgId);
    }
    if (failureFrame.kind === 'ack') {
      expect(failureFrame.ack.code).toBe(-1);
      expect(failureFrame.ack.errMsg).toBe('enqueue failed');
    }
  });
});
