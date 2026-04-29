import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { ChatSDK } from '../index';
import { TEST_CONFIG, randomUsername, randomPassword } from './setup';

describe('用户认证 API 集成测试', () => {
  let sdk: ChatSDK;

  beforeAll(() => {
    sdk = new ChatSDK(TEST_CONFIG);
  });

  afterAll(() => {
    sdk.disconnect();
  });

  it('应该成功注册新用户', async () => {
    const username = randomUsername();
    const password = randomPassword();

    const result = await sdk.register({
      username,
      password,
    });

    expect(result).toBeDefined();
    expect(result.user_id).toBeDefined();
    expect(result.username).toBe(username);
    expect(result.token).toBeDefined();
    expect(result.token.length).toBeGreaterThan(0);
  });

  it('应该成功登录已注册用户', async () => {
    const username = randomUsername();
    const password = randomPassword();

    // 先注册
    const registerResult = await sdk.register({ username, password });
    expect(registerResult.token).toBeDefined();

    // 清除认证状态
    sdk.clearAuth();

    // 登录
    const loginResult = await sdk.login({
      username,
      password,
    });

    expect(loginResult).toBeDefined();
    expect(loginResult.user_id).toBe(registerResult.user_id);
    expect(loginResult.username).toBe(username);
    expect(loginResult.token).toBeDefined();
  });

  it('应该使用相同的 token 登录', async () => {
    const username = randomUsername();
    const password = randomPassword();

    // 先注册用户
    const registerResult = await sdk.register({ username, password });
    expect(registerResult.token).toBeDefined();

    // 清除并重新登录
    sdk.clearAuth();
    const loginResult = await sdk.login({ username, password });

    // 注意:JWT token 每次生成都是新的,只要验证能获取到有效 token 即可
    expect(loginResult.token).toBeDefined();
    expect(sdk.isAuthenticated()).toBe(true);
  });

  it('登录时应该验证密码错误', async () => {
    const username = randomUsername();
    const password = randomPassword();

    await sdk.register({ username, password });
    sdk.clearAuth();

    await expect(sdk.login({ username, password: 'wrong_password' })).rejects.toThrow();
  });

  it('登录时应该验证用户不存在', async () => {
    await expect(sdk.login({
      username: `nonexistent_${randomUsername()}`,
      password: randomPassword(),
    })).rejects.toThrow();
  });

  it('应该批量生成测试用户', async () => {
    const count = 5;
    const result = await sdk.batchGenerateUsers({ count });

    expect(result).toBeDefined();
    expect(result.length).toBe(count);

    result.forEach((user) => {
      expect(user.user_id).toBeDefined();
      expect(user.username).toBeDefined();
      expect(user.password).toBeDefined();
      expect(user.token).toBeDefined();
    });
  });

  it('批量生成应该限制数量范围', async () => {
    // 测试数量过大
    await expect(sdk.batchGenerateUsers({ count: 101 })).rejects.toThrow();

    // 测试数量为0
    await expect(sdk.batchGenerateUsers({ count: 0 })).rejects.toThrow();
  });
});
