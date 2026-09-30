import type {
  RegisterRequest,
  LoginRequest,
  UserResponse,
  HistoryQueryParams,
  HistoryMessagesResponse,
  Message,
  CreateRoomRequest,
  CreateRoomResponse,
  CreateGroupRoomRequest,
  ListRoomsResponse,
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
      const errorData: unknown = await response.json().catch(() => ({}));
      const fields =
        errorData && typeof errorData === 'object' && !Array.isArray(errorData)
          ? (errorData as Record<string, unknown>)
          : {};
      const reason = fields.msg ?? fields.error;
      throw new APIError(
        response.status,
        typeof reason === 'string' ? reason : `HTTP ${response.status}: ${response.statusText}`,
        fields
      );
    }

    // 204 No Content
    if (response.status === 204) {
      return undefined as T;
    }

    const payload: unknown = await response.json();

    // 服务端统一返回 {code,msg,data} 信封；剥掉信封暴露 data，
    // 已是裸数据的响应原样返回。
    if (payload !== null && typeof payload === 'object' && !Array.isArray(payload)) {
      const candidate: Record<string, unknown> = payload as Record<string, unknown>;
      if (typeof candidate.code === 'number' && 'data' in candidate) {
        return candidate.data as T;
      }
    }

    return payload as T;
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

  // ==================== 消息相关 API ====================

  /**
   * 获取历史消息
   * 注意:后端目前只提供了按 conversation 查询的接口
   */
  async getHistoryMessages(params: HistoryQueryParams): Promise<HistoryMessagesResponse> {
    // 构建查询参数
    const queryParams = new URLSearchParams();
    queryParams.append('room_id', params.room_id);

    if (params.before_seq !== undefined) {
      queryParams.append('before_seq', params.before_seq.toString());
    }
    if (params.page_size !== undefined) {
      queryParams.append('page_size', params.page_size.toString());
    }

    const resp = await this.request<{ messages: Message[], hasMore: boolean }>(
      'GET',
      `/api/v1/messages/history?${queryParams.toString()}`
    );

    return {
      messages: resp?.messages || [],
      hasMore: resp?.hasMore ?? false,
    };
  }

  // ==================== 房间相关 API ====================

  /**
   * 创建或获取单聊房间
   */
  async createRoom(req: CreateRoomRequest): Promise<CreateRoomResponse> {
    return this.request<CreateRoomResponse>('POST', '/api/v1/rooms/single', req);
  }

  /**
   * 创建群聊房间
   */
  async createGroupRoom(req: CreateGroupRoomRequest): Promise<CreateRoomResponse> {
    return this.request<CreateRoomResponse>('POST', '/api/v1/rooms/group', req);
  }

  /**
   * 获取用户房间列表
   */
  async listRooms(): Promise<ListRoomsResponse> {
    return this.request<ListRoomsResponse>('POST', '/api/v1/rooms/list');
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
