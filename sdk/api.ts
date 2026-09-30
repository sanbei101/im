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
} from "./types";

interface ApiResponse<T = unknown> {
  code: number;
  msg: string;
  data?: T;
}

/**
 * API 错误类
 */
export class APIError extends Error {
  constructor(
    public readonly statusCode: number,
    message: string,
    public readonly data?: unknown,
  ) {
    super(message);
    this.name = "APIError";
  }
}

/**
 * HTTP API 客户端
 */
export class APIClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL: string) {
    this.baseURL = baseURL.replace(/\/+$/, "");
  }

  setToken(token: string): void {
    this.token = token;
  }

  clearToken(): void {
    this.token = null;
  }

  getToken(): string | null {
    return this.token;
  }

  private getHeaders(): HeadersInit {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (this.token) {
      headers["Authorization"] = `Bearer ${this.token}`;
    }
    return headers;
  }

  private async request<T>(method: string, endpoint: string, body?: unknown): Promise<T> {
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
      const errorJson = (await response.json().catch(() => ({}))) as Record<string, unknown>;
      const reason =
        (typeof errorJson.msg === "string" && errorJson.msg) ||
        (typeof errorJson.error === "string" && errorJson.error) ||
        `HTTP ${response.status}: ${response.statusText}`;
      throw new APIError(response.status, reason, errorJson);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    const res = (await response.json()) as ApiResponse<T>;
    if (res && typeof res === "object" && "data" in res) {
      return res.data as T;
    }
    return res as unknown as T;
  }

  // ==================== 用户相关 API ====================

  /**
   * 用户注册
   */
  async register(req: RegisterRequest): Promise<UserResponse> {
    return this.request<UserResponse>("POST", "/api/v1/users/register", req);
  }

  /**
   * 用户登录
   */
  async login(req: LoginRequest): Promise<UserResponse> {
    const resp = await this.request<UserResponse>("POST", "/api/v1/users/login", req);
    this.setToken(resp.token);
    return resp;
  }

  // ==================== 消息相关 API ====================

  /**
   * 获取房间历史消息
   */
  async getHistoryMessages(params: HistoryQueryParams): Promise<HistoryMessagesResponse> {
    const query = new URLSearchParams();
    query.set("room_id", params.room_id);
    if (params.before_seq !== undefined) {
      query.set("before_seq", params.before_seq.toString());
    }
    if (params.page_size !== undefined) {
      query.set("page_size", params.page_size.toString());
    }

    const resp = await this.request<{ messages?: Message[]; hasMore?: boolean }>(
      "GET",
      `/api/v1/messages/history?${query.toString()}`,
    );

    return {
      messages: resp?.messages ?? [],
      hasMore: resp?.hasMore ?? false,
    };
  }

  // ==================== 房间相关 API ====================

  /**
   * 创建或获取单聊房间
   */
  async createRoom(req: CreateRoomRequest): Promise<CreateRoomResponse> {
    return this.request<CreateRoomResponse>("POST", "/api/v1/rooms/single", req);
  }

  /**
   * 创建群聊房间
   */
  async createGroupRoom(req: CreateGroupRoomRequest): Promise<CreateRoomResponse> {
    return this.request<CreateRoomResponse>("POST", "/api/v1/rooms/group", req);
  }

  /**
   * 获取用户房间列表
   */
  async listRooms(): Promise<ListRoomsResponse> {
    return this.request<ListRoomsResponse>("POST", "/api/v1/rooms/list");
  }
}
