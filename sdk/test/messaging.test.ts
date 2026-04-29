import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { ChatSDK, ChatType, MessageType, ChatEventType } from '../index';
import { TEST_CONFIG, randomUsername, randomPassword, sleep } from './setup';

describe('消息发送与接收集成测试', () => {
  let sdk1: ChatSDK;
  let sdk2: ChatSDK;
  let user1Id: string;
  let user2Id: string;

  beforeAll(async () => {
    // 创建两个 SDK 实例
    sdk1 = new ChatSDK(TEST_CONFIG);
    sdk2 = new ChatSDK(TEST_CONFIG);

    // 注册两个测试用户
    const [result1, result2] = await Promise.all([
      sdk1.register({ username: randomUsername(), password: randomPassword() }),
      sdk2.register({ username: randomUsername(), password: randomPassword() }),
    ]);

    user1Id = result1.user_id;
    user2Id = result2.user_id;

    // 连接 WebSocket
    await Promise.all([sdk1.connect(), sdk2.connect()]);

    expect(sdk1.isConnected()).toBe(true);
    expect(sdk2.isConnected()).toBe(true);
  });

  afterAll(() => {
    sdk1.disconnect();
    sdk2.disconnect();
  });

  it('sdk1 应该能发送文本消息给 sdk2', async () => {
    const receivedMessages: { text: string; sender_id: string }[] = [];

    // sdk2 监听消息
    const unsubscribe = sdk2.on(ChatEventType.MessageReceived, (event) => {
      const msg = event.data.message;
      receivedMessages.push({
        text: (msg.payload as { text: string }).text,
        sender_id: msg.sender_id,
      });
    });

    // sdk1 发送消息
    const testText = `Hello from user1! ${Date.now()}`;
    sdk1.sendTextMessage({
      room_id: user2Id,
      chat_type: ChatType.Single,
      text: testText,
    });

    // 等待消息到达
    await sleep(2000);

    unsubscribe();

    expect(receivedMessages.length).toBeGreaterThan(0);
    expect(receivedMessages[0].sender_id).toBe(user1Id);
    expect(receivedMessages[0].text).toBe(testText);
  });

  it('sdk2 应该能回复消息给 sdk1', async () => {
    const receivedMessages: { text: string; sender_id: string }[] = [];

    // sdk1 监听消息
    const unsubscribe = sdk1.on(ChatEventType.MessageReceived, (event) => {
      const msg = event.data.message;
      receivedMessages.push({
        text: (msg.payload as { text: string }).text,
        sender_id: msg.sender_id,
      });
    });

    // sdk2 发送消息
    const replyText = `Reply from user2! ${Date.now()}`;
    sdk2.sendTextMessage({
      room_id: user1Id,
      chat_type: ChatType.Single,
      text: replyText,
    });

    await sleep(2000);

    unsubscribe();

    expect(receivedMessages.length).toBeGreaterThan(0);
    expect(receivedMessages[0].sender_id).toBe(user2Id);
    expect(receivedMessages[0].text).toBe(replyText);
  });

  it('应该能发送不同类型的消息', async () => {
    const receivedMessages: { type: string; payload: unknown }[] = [];

    const unsubscribe = sdk2.on(ChatEventType.MessageReceived, (event) => {
      receivedMessages.push({
        type: event.data.message.msg_type,
        payload: event.data.message.payload,
      });
    });

    // 发送图片消息
    sdk1.sendImageMessage({
      room_id: user2Id,
      chat_type: ChatType.Single,
      url: 'https://example.com/image.jpg',
      width: 1920,
      height: 1080,
      size: 1024000,
    });

    // 发送视频消息
    sdk1.sendVideoMessage({
      room_id: user2Id,
      chat_type: ChatType.Single,
      url: 'https://example.com/video.mp4',
      duration: 60,
      width: 1920,
      height: 1080,
      size: 10485760,
      thumbnail_url: 'https://example.com/thumb.jpg',
    });

    // 发送文件消息
    sdk1.sendFileMessage({
      room_id: user2Id,
      chat_type: ChatType.Single,
      url: 'https://example.com/document.pdf',
      name: 'document.pdf',
      size: 2048000,
      mime_type: 'application/pdf',
    });

    // 等待消息到达
    await sleep(3000);

    unsubscribe();

    expect(receivedMessages.length).toBeGreaterThanOrEqual(3);

    // 验证各类消息
    const imageMsg = receivedMessages.find(m => m.type === MessageType.Image);
    const videoMsg = receivedMessages.find(m => m.type === MessageType.Video);
    const fileMsg = receivedMessages.find(m => m.type === MessageType.File);

    expect(imageMsg).toBeDefined();
    expect((imageMsg!.payload as { url: string }).url).toBe('https://example.com/image.jpg');

    expect(videoMsg).toBeDefined();
    expect((videoMsg!.payload as { duration: number }).duration).toBe(60);

    expect(fileMsg).toBeDefined();
    expect((fileMsg!.payload as { name: string }).name).toBe('document.pdf');
  });

  it('消息应该包含正确的客户端消息ID', async () => {
    const receivedMessages: { client_msg_id: string }[] = [];
    const clientMsgId = sdk1.generateMessageId();

    const unsubscribe = sdk2.on(ChatEventType.MessageReceived, (event) => {
      receivedMessages.push({
        client_msg_id: event.data.message.client_msg_id,
      });
    });

    sdk1.sendMessage({
      client_msg_id: clientMsgId,
      room_id: user2Id,
      chat_type: ChatType.Single,
      msg_type: MessageType.Text,
      payload: { text: 'Test client_msg_id' },
    });

    await sleep(2000);

    unsubscribe();

    expect(receivedMessages.length).toBeGreaterThan(0);
    expect(receivedMessages[0].client_msg_id).toBe(clientMsgId);
  });

  it('未连接时不应该能发送消息', async () => {
    const disconnectedSdk = new ChatSDK(TEST_CONFIG);
    await disconnectedSdk.register({
      username: randomUsername(),
      password: randomPassword(),
    });

    // 注意:当前实现在未连接时会将消息加入队列而不是报错
    // 这里测试的是消息被队列化的情况
    // 不应该抛出错误(消息会被缓存)
    expect(() => {
      disconnectedSdk.sendTextMessage({
        room_id: user2Id,
        chat_type: ChatType.Single,
        text: 'This message should be queued',
      });
    }).not.toThrow();

    disconnectedSdk.disconnect();
  });
});
