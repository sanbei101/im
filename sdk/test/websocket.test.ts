import { describe, it, expect, beforeAll, afterEach } from 'vitest';
import { ChatSDK, ChatEventType, ConnectionState } from '../index';
import { TEST_CONFIG, randomUsername, randomPassword, sleep } from './setup';

describe('WebSocket 连接集成测试', () => {
  afterEach(() => {
    sdk?.disconnect();
  });

  let sdk: ChatSDK;

  beforeAll(async () => {
    sdk = new ChatSDK(TEST_CONFIG);
    await sdk.register({
      username: randomUsername(),
      password: randomPassword(),
    });
  });

  it('应该成功连接到 WebSocket 网关', async () => {
    expect(sdk.isAuthenticated()).toBe(true);

    await sdk.connect();

    expect(sdk.isConnected()).toBe(true);
    expect(sdk.getConnectionState()).toBe(ConnectionState.Connected);
  });

  it('应该触发连接状态变更事件', async () => {
    const stateChanges: ConnectionState[] = [];

    const unsubscribe = sdk.on(ChatEventType.ConnectionStateChange, (event) => {
      stateChanges.push(event.data.state);
    });

    sdk.disconnect();
    await sleep(500);

    await sdk.connect();
    await sleep(500);

    unsubscribe();

    expect(stateChanges.length).toBeGreaterThan(0);
    expect(stateChanges).toContain(ConnectionState.Connected);
  });

  it('应该触发 connect 事件', async () => {
    let connected = false;

    const unsubscribe = sdk.on(ChatEventType.Connect, () => {
      connected = true;
    });

    await sdk.connect();

    unsubscribe();

    expect(connected).toBe(true);
  });

  it('断开连接后状态应该正确', async () => {
    await sdk.connect();
    expect(sdk.isConnected()).toBe(true);

    sdk.disconnect();
    await sleep(500);

    expect(sdk.isConnected()).toBe(false);
    expect(sdk.getConnectionState()).toBe(ConnectionState.Disconnected);
  });

  it('未认证时不应该能连接', async () => {
    const unauthSdk = new ChatSDK(TEST_CONFIG);

    await expect(unauthSdk.connect()).rejects.toThrow(/authenticated/i);
  });

  it('应该支持多次连接断开', async () => {
    for (let i = 0; i < 3; i++) {
      await sdk.connect();
      expect(sdk.isConnected()).toBe(true);
      await sleep(200);

      sdk.disconnect();
      await sleep(200);
      expect(sdk.isConnected()).toBe(false);
    }
  });
});
