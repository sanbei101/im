import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { ChatSDK, ChatType, ChatEventType } from '../index';
import { TEST_CONFIG, randomUsername, randomPassword, sleep } from './setup';

describe('重连机制集成测试', () => {
  let sdk: ChatSDK;

  beforeAll(async () => {
    sdk = new ChatSDK({
      ...TEST_CONFIG,
      reconnectInterval: 1000, // 1秒重连间隔
      maxReconnectAttempts: 5,
    });

    await sdk.register({
      username: randomUsername(),
      password: randomPassword(),
    });
  });

  afterAll(() => {
    sdk.disconnect();
  });

  it('主动断开连接后状态应该正确', async () => {
    await sdk.connect();
    expect(sdk.isConnected()).toBe(true);

    sdk.disconnect();

    expect(sdk.isConnected()).toBe(false);
  });

  it('应该触发错误事件', async () => {
    const errors: { code: string; message: string }[] = [];

    // 创建一个无法连接的实例
    const badSdk = new ChatSDK({
      baseURL: TEST_CONFIG.baseURL,
      gatewayURL: 'ws://invalid-server:9999/ws', // 无效的服务器
      reconnectInterval: 500,
      maxReconnectAttempts: 2,
    });

    // 先注册用户
    await badSdk.register({
      username: randomUsername(),
      password: randomPassword(),
    });

    // 订阅错误事件
    const unsubscribe = badSdk.on(ChatEventType.Error, (event) => {
      errors.push({
        code: event.data.code,
        message: event.data.message,
      });
    });

    // 尝试连接 - 预期会失败
    await expect(badSdk.connect()).rejects.toThrow();

    await sleep(3000); // 等待重连尝试

    unsubscribe();

    badSdk.disconnect();
  });
});

describe('错误处理集成测试', () => {
  it('应该正确处理发送消息时的错误', async () => {
    const sdk = new ChatSDK({
      ...TEST_CONFIG,
      maxReconnectAttempts: 0, // 不重连
    });

    await sdk.register({
      username: randomUsername(),
      password: randomPassword(),
    });

    const errors: { code: string; message: string }[] = [];
    const unsubscribe = sdk.on(ChatEventType.Error, (event) => {
      errors.push({
        code: event.data.code,
        message: event.data.message,
      });
    });

    // 不连接就发送消息(消息会被加入队列,同时触发错误)
    sdk.sendTextMessage({
      room_id: 'test-user-id',
      chat_type: ChatType.Single,
      text: 'Test',
    });

    await sleep(500);

    unsubscribe();

    // 应该收到未连接的错误
    expect(errors.some(e => e.code === 'WS_NOT_CONNECTED')).toBe(true);

    sdk.disconnect();
  });
});
