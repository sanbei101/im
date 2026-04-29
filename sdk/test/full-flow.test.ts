import { describe, it, expect } from 'vitest';
import { ChatSDK, ChatType, ChatEventType } from '../index';
import { TEST_CONFIG, randomUsername, randomPassword, sleep } from './setup';

describe('SDK 完整流程集成测试', () => {
  it('应该完成完整的聊天流程', async () => {
    // 创建两个 SDK 实例
    const sdkA = new ChatSDK(TEST_CONFIG);
    const sdkB = new ChatSDK(TEST_CONFIG);

    // 注册两个用户
    const [userA, userB] = await Promise.all([
      sdkA.register({ username: randomUsername(), password: randomPassword() }),
      sdkB.register({ username: randomUsername(), password: randomPassword() }),
    ]);

    // 连接 WebSocket
    await Promise.all([sdkA.connect(), sdkB.connect()]);
    expect(sdkA.isConnected()).toBe(true);
    expect(sdkB.isConnected()).toBe(true);

    // 设置消息监听
    const userAMessages: string[] = [];
    const userBMessages: string[] = [];

    sdkA.on(ChatEventType.MessageReceived, (event) => {
      const text = (event.data.message.payload as { text: string }).text;
      userAMessages.push(text);
    });

    sdkB.on(ChatEventType.MessageReceived, (event) => {
      const text = (event.data.message.payload as { text: string }).text;
      userBMessages.push(text);
    });

    // 双向发送消息
    sdkA.sendTextMessage({
      room_id: userB.user_id,
      chat_type: ChatType.Single,
      text: 'Hello from User A!',
    });

    sdkB.sendTextMessage({
      room_id: userA.user_id,
      chat_type: ChatType.Single,
      text: 'Hello from User B!',
    });

    // 等待消息到达
    await sleep(2000);

    // 验证消息
    expect(userBMessages.length).toBeGreaterThan(0);
    expect(userAMessages.length).toBeGreaterThan(0);
    expect(userBMessages[0]).toBe('Hello from User A!');
    expect(userAMessages[0]).toBe('Hello from User B!');

    // 断开连接
    sdkA.disconnect();
    sdkB.disconnect();

    expect(sdkA.isConnected()).toBe(false);
    expect(sdkB.isConnected()).toBe(false);
  }, 30000);
});
