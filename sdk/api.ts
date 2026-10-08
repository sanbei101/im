import type {
  AddMembersRequest,
  ApplyFriendRequest,
  AuditFriendRequest,
  BlacklistRequest,
  CreateGroupRoomRequest,
  CreateRoomRequest,
  CreateRoomResponse,
  DeviceTokenRequest,
  FriendApplication,
  FriendItem,
  HistoryMessagesResponse,
  HistoryQueryParams,
  ListConversationsResponse,
  ListRoomsResponse,
  LoginRequest,
  MarkReadRequest,
  MemberInfo,
  Message,
  MuteConversationRequest,
  PinConversationRequest,
  PinMessageRequest,
  PresenceRequest,
  PresenceResponse,
  PresignUploadRequest,
  PresignUploadResponse,
  ReactionGroup,
  ReactionRequest,
  ReadUsersResponse,
  RecallMessageRequest,
  RegisterRequest,
  RoomDetail,
  SearchMessagesParams,
  UpdateMemberRoleRequest,
  UpdatePasswordRequest,
  UpdateProfileRequest,
  UpdateRemarkRequest,
  UserProfile,
  UserResponse,
  TransferOwnerRequest,
  UpdateRoomRequest,
} from "./types";

interface ApiResponse<T = unknown> {
  code: number;
  msg: string;
  data?: T;
}

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
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (this.token) headers.Authorization = `Bearer ${this.token}`;
    return headers;
  }

  private async request<T>(method: string, endpoint: string, body?: unknown): Promise<T> {
    const options: RequestInit = { method, headers: this.getHeaders() };
    if (body !== undefined) options.body = JSON.stringify(body);
    const response = await fetch(`${this.baseURL}${endpoint}`, options);

    if (!response.ok) {
      const errorJson = (await response.json().catch(() => ({}))) as Record<string, unknown>;
      const reason =
        (typeof errorJson.msg === "string" && errorJson.msg) ||
        (typeof errorJson.error === "string" && errorJson.error) ||
        `HTTP ${response.status}: ${response.statusText}`;
      throw new APIError(response.status, reason, errorJson);
    }
    if (response.status === 204) return undefined as T;
    const result = (await response.json()) as ApiResponse<T>;
    return result && typeof result === "object" && "data" in result
      ? (result.data as T)
      : (result as unknown as T);
  }

  private query(params: Record<string, string | number | undefined>): string {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined) query.set(key, String(value));
    }
    const encoded = query.toString();
    return encoded ? `?${encoded}` : "";
  }

  // 用户
  async register(req: RegisterRequest): Promise<UserResponse> {
    return this.request("POST", "/api/v1/users/register", req);
  }
  async login(req: LoginRequest): Promise<UserResponse> {
    const response = await this.request<UserResponse>("POST", "/api/v1/users/login", req);
    this.setToken(response.token);
    return response;
  }
  async getProfile(userId?: string): Promise<UserProfile> {
    return this.request("GET", userId ? `/api/v1/users/${userId}` : "/api/v1/users/profile");
  }
  async updateProfile(req: UpdateProfileRequest): Promise<UserProfile> {
    return this.request("PUT", "/api/v1/users/profile", req);
  }
  async searchUsers(keyword: string): Promise<UserProfile[]> {
    return this.request("GET", `/api/v1/users/search${this.query({ keyword })}`);
  }
  async getPresence(req: PresenceRequest): Promise<PresenceResponse> {
    return this.request("POST", "/api/v1/users/presence", req);
  }
  async updatePassword(req: UpdatePasswordRequest): Promise<void> {
    await this.request("PUT", "/api/v1/users/password", req);
  }
  async logout(): Promise<void> {
    await this.request("POST", "/api/v1/users/logout");
    this.clearToken();
  }
  async saveDeviceToken(req: DeviceTokenRequest): Promise<void> {
    await this.request("POST", "/api/v1/devices/token", req);
  }

  // 消息
  async getHistoryMessages(params: HistoryQueryParams): Promise<HistoryMessagesResponse> {
    const response = await this.request<{ messages?: Message[]; has_more?: boolean }>(
      "GET",
      `/api/v1/messages/history${this.query({ ...params })}`,
    );
    return { messages: response?.messages ?? [], has_more: response?.has_more ?? false };
  }
  async searchMessages(params: SearchMessagesParams): Promise<Message[]> {
    const { room_id, ...query } = params;
    return this.request("GET", `/api/v1/messages/search${this.query({ room_id, ...query })}`);
  }
  async searchRoomMessages(
    roomId: string,
    params: Omit<SearchMessagesParams, "room_id">,
  ): Promise<Message[]> {
    return this.request("GET", `/api/v1/rooms/${roomId}/messages/search${this.query(params)}`);
  }
  async recallMessage(req: RecallMessageRequest): Promise<Message> {
    return this.request("POST", "/api/v1/messages/recall", req);
  }
  async addReaction(messageId: string, req: ReactionRequest): Promise<void> {
    await this.request("POST", `/api/v1/messages/${messageId}/reactions`, req);
  }
  async removeReaction(messageId: string, req: ReactionRequest): Promise<void> {
    await this.request("DELETE", `/api/v1/messages/${messageId}/reactions`, req);
  }
  async getReactions(messageId: string, roomId: string): Promise<ReactionGroup[]> {
    return this.request(
      "GET",
      `/api/v1/messages/${messageId}/reactions${this.query({ room_id: roomId })}`,
    );
  }
  async getReadUsers(messageId: string, roomId: string): Promise<ReadUsersResponse> {
    return this.request(
      "GET",
      `/api/v1/messages/${messageId}/read_users${this.query({ room_id: roomId })}`,
    );
  }

  // 房间
  async createRoom(req: CreateRoomRequest): Promise<CreateRoomResponse> {
    return this.request("POST", "/api/v1/rooms/single", req);
  }
  async createGroupRoom(req: CreateGroupRoomRequest): Promise<CreateRoomResponse> {
    return this.request("POST", "/api/v1/rooms/group", req);
  }
  async listRooms(): Promise<ListRoomsResponse> {
    return this.request("POST", "/api/v1/rooms/list");
  }
  async getRoom(roomId: string): Promise<RoomDetail> {
    return this.request("GET", `/api/v1/rooms/${roomId}`);
  }
  async updateRoom(roomId: string, req: UpdateRoomRequest): Promise<RoomDetail> {
    return this.request("PUT", `/api/v1/rooms/${roomId}`, req);
  }
  async deleteRoom(roomId: string): Promise<void> {
    await this.request("DELETE", `/api/v1/rooms/${roomId}`);
  }
  async listMembers(roomId: string): Promise<MemberInfo[]> {
    return this.request("GET", `/api/v1/rooms/${roomId}/members`);
  }
  async addMembers(roomId: string, req: AddMembersRequest): Promise<void> {
    await this.request("POST", `/api/v1/rooms/${roomId}/members`, req);
  }
  async updateMemberRole(
    roomId: string,
    userId: string,
    req: UpdateMemberRoleRequest,
  ): Promise<void> {
    await this.request("PUT", `/api/v1/rooms/${roomId}/members/${userId}/role`, req);
  }
  async removeMember(roomId: string, userId: string): Promise<void> {
    await this.request("DELETE", `/api/v1/rooms/${roomId}/members/${userId}`);
  }
  async leaveRoom(roomId: string): Promise<void> {
    await this.request("POST", `/api/v1/rooms/${roomId}/leave`);
  }
  async transferOwner(roomId: string, req: TransferOwnerRequest): Promise<void> {
    await this.request("POST", `/api/v1/rooms/${roomId}/transfer`, req);
  }
  async pinMessage(roomId: string, req: PinMessageRequest): Promise<void> {
    await this.request("POST", `/api/v1/rooms/${roomId}/pins`, req);
  }
  async unpinMessage(roomId: string, messageId: string): Promise<void> {
    await this.request("DELETE", `/api/v1/rooms/${roomId}/pins/${messageId}`);
  }
  async getPinnedMessages(roomId: string): Promise<Message[]> {
    return this.request("GET", `/api/v1/rooms/${roomId}/pins`);
  }

  // 好友
  async listFriends(): Promise<FriendItem[]> {
    return this.request("GET", "/api/v1/friends/");
  }
  async applyFriend(req: ApplyFriendRequest): Promise<void> {
    await this.request("POST", "/api/v1/friends/apply", req);
  }
  async auditFriend(req: AuditFriendRequest): Promise<void> {
    await this.request("POST", "/api/v1/friends/audit", req);
  }
  async listFriendApplications(): Promise<FriendApplication[]> {
    return this.request("GET", "/api/v1/friends/applications");
  }
  async deleteFriend(userId: string): Promise<void> {
    await this.request("DELETE", `/api/v1/friends/${userId}`);
  }
  async updateFriendRemark(userId: string, req: UpdateRemarkRequest): Promise<void> {
    await this.request("PUT", `/api/v1/friends/${userId}/remark`, req);
  }
  async listBlacklist(): Promise<UserProfile[]> {
    return this.request("GET", "/api/v1/friends/blacklist");
  }
  async addBlacklist(req: BlacklistRequest): Promise<void> {
    await this.request("POST", "/api/v1/friends/blacklist", req);
  }
  async removeBlacklist(userId: string): Promise<void> {
    await this.request("DELETE", `/api/v1/friends/blacklist/${userId}`);
  }

  // 会话
  async listConversations(): Promise<ListConversationsResponse> {
    return this.request("GET", "/api/v1/conversations/");
  }
  async markRead(roomId: string, req: MarkReadRequest): Promise<void> {
    await this.request("POST", `/api/v1/rooms/${roomId}/read`, req);
  }
  async clearUnread(roomId: string): Promise<void> {
    await this.request("PUT", `/api/v1/rooms/${roomId}/clear_unread`);
  }
  async pinConversation(roomId: string, req: PinConversationRequest): Promise<void> {
    await this.request("PUT", `/api/v1/conversations/${roomId}/pin`, req);
  }
  async muteConversation(roomId: string, req: MuteConversationRequest): Promise<void> {
    await this.request("PUT", `/api/v1/conversations/${roomId}/mute`, req);
  }

  // 文件
  async presignUpload(req: PresignUploadRequest): Promise<PresignUploadResponse> {
    return this.request("POST", "/api/v1/files/presign", req);
  }
}
