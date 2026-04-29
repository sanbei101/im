import type {
  RegisterRequest,
  LoginRequest,
  UserResponse,
  BatchGenerateRequest,
  BatchUserResponse,
  HistoryQueryParams,
  HistoryMessagesResponse,
  Message,
} from './types';

/**
 * API 客户端 - 处理所有 HTTP 请求
 */
export class APIClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL: string) {
    // 移除末尾的斜杠
    this.baseURL = baseURL.replace(/\/$/, '');
  }

  /**
   * 设置认证 Token
   */
  setToken(token: string): void {
    this.token = token;
  }

  /**
   * 清除认证 Token
   */
  clearToken(): void {
    this.token = null;
  }

  /**
   * 获取当前 Token
   */
  getToken(): string | null {
    return this.token;
  }

  /**
   * 构建请求头
   */
  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    return headers;
  }

  /**
   * 发送 HTTP 请求
   */
  private async request<T>(
    method: string,
    endpoint: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;

    const options: RequestInit = {
      method,
      headers: this.getHeaders(),
    };

    if (body !== undefined) {
      options.body = JSON.stringify(body);
    }

    const response = await fetch(url, options);

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new APIError(
        response.status,
        errorData.error || `HTTP ${response.status}: ${response.statusText}`,
        errorData
      );
    }

    // 204 No Content
    if (response.status === 204) {
      return undefined as T;
    }

    return response.json() as Promise<T>;
  }

  // ==================== 用户相关 API ====================

  /**
   * 用户注册
   */
  async register(req: RegisterRequest): Promise<UserResponse> {
    return this.request<UserResponse>('POST', '/api/v1/users/register', req);
  }

  /**
   * 用户登录
   */
  async login(req: LoginRequest): Promise<UserResponse> {
    const resp = await this.request<UserResponse>('POST', '/api/v1/users/login', req);
    this.setToken(resp.token);
    return resp;
  }

  /**
   * 批量生成用户
   */
  async batchGenerate(req: BatchGenerateRequest): Promise<{ users: BatchUserResponse[] }> {
    return this.request<{ users: BatchUserResponse[] }>('POST', '/api/v1/users/batch', req);
  }

  // ==================== 消息相关 API ====================

  /**
   * 获取历史消息
   * 注意:后端目前只提供了按 conversation 查询的接口
   */
  async getHistoryMessages(params: HistoryQueryParams): Promise<HistoryMessagesResponse> {
    // 构建查询参数
    const queryParams = new URLSearchParams();
    queryParams.append('room_id', params.room_id);
    queryParams.append('chat_type', params.chat_type);

    if (params.before_server_time !== undefined) {
      queryParams.append('before_server_time', params.before_server_time.toString());
    }
    if (params.page_size !== undefined) {
      queryParams.append('page_size', params.page_size.toString());
    }

    const messages = await this.request<Message[]>(
      'GET',
      `/api/v1/messages/history?${queryParams.toString()}`
    );

    return {
      messages: messages || [],
      hasMore: messages.length === (params.page_size || 20),
    };
  }
}

/**
 * API 错误类
 */
export class APIError extends Error {
  statusCode: number;
  data: Record<string, unknown>;

  constructor(statusCode: number, message: string, data: Record<string, unknown> = {}) {
    super(message);
    this.name = 'APIError';
    this.statusCode = statusCode;
    this.data = data;
  }
}
