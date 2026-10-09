import type {
  ChatSDK,
  ConnectionState,
  ConversationInfo,
  FriendApplication,
  FriendItem,
  MemberInfo,
  Message,
  RoomDetail,
  RoomInfo,
  UserProfile,
  UserResponse,
} from "go-chat-sdk";

import type { ServerConfig, UIMessage } from "@/types/chat";

export interface ChatContextValue {
  readonly sdk: ChatSDK;
  readonly config: ServerConfig;
  readonly currentUser: UserResponse | null;
  readonly connectionState: ConnectionState;
  readonly rooms: readonly RoomInfo[];
  readonly activeRoomId: string | null;
  readonly activeRoom: RoomInfo | null;
  readonly messages: readonly UIMessage[];
  readonly isLoadingRooms: boolean;
  readonly isLoadingHistory: boolean;
  readonly error: string | null;
  readonly updateConfig: (config: ServerConfig) => void;
  readonly login: (req: { username: string; password: string }) => Promise<void>;
  readonly register: (req: { username: string; password: string }) => Promise<void>;
  readonly logout: () => void;
  readonly connect: () => Promise<void>;
  readonly disconnect: () => void;
  readonly selectRoom: (roomId: string) => void;
  readonly refreshRooms: () => Promise<void>;
  readonly createSingleRoom: (targetUserId: string) => Promise<string>;

  // Conversations
  readonly conversations: readonly ConversationInfo[];
  readonly isLoadingConversations: boolean;
  readonly refreshConversations: () => Promise<void>;
  readonly muteConversation: (roomId: string, muted: boolean) => Promise<void>;
  readonly pinConversation: (roomId: string, pinned: boolean) => Promise<void>;
  readonly clearUnread: (roomId: string) => Promise<void>;
  readonly deleteRoom: (roomId: string) => Promise<void>;

  // Rooms & members
  readonly activeRoomDetail: RoomDetail | null;
  readonly members: readonly MemberInfo[];
  readonly isLoadingMembers: boolean;
  readonly pinnedMessages: readonly Message[];
  readonly refreshRoomDetail: () => Promise<void>;
  readonly updateActiveRoom: (req: { name?: string; notice?: string }) => Promise<void>;
  readonly leaveActiveRoom: () => Promise<void>;
  readonly addMembers: (memberIds: readonly string[]) => Promise<void>;
  readonly updateMemberRole: (userId: string, role: "admin" | "member") => Promise<void>;
  readonly removeMember: (userId: string) => Promise<void>;
  readonly transferOwnership: (newOwnerId: string) => Promise<void>;

  // Friends & contacts
  readonly friends: readonly FriendItem[];
  readonly friendApplications: readonly FriendApplication[];
  readonly blacklist: readonly UserProfile[];
  readonly isLoadingFriends: boolean;
  readonly refreshFriends: () => Promise<void>;
  readonly applyFriend: (targetId: string, greeting?: string) => Promise<void>;
  readonly auditFriend: (fromUserId: string, action: "accept" | "reject") => Promise<void>;
  readonly deleteFriend: (userId: string) => Promise<void>;
  readonly updateFriendRemark: (userId: string, remark: string) => Promise<void>;
  readonly addBlacklist: (targetId: string) => Promise<void>;
  readonly removeBlacklist: (userId: string) => Promise<void>;
  readonly searchUsers: (keyword: string) => Promise<readonly UserProfile[]>;

  // Presence
  readonly presence: Readonly<Record<string, boolean>>;
  readonly queryPresence: (userIds: readonly string[]) => Promise<void>;

  // History
  readonly hasMoreHistory: boolean;
  readonly isLoadingMoreHistory: boolean;
  readonly loadMoreHistory: () => Promise<void>;

  // Search
  readonly searchResults: readonly Message[];
  readonly isSearching: boolean;
  readonly searchMessages: (keyword: string, scope: "room" | "all") => Promise<void>;
  readonly clearSearchResults: () => void;

  // Message actions
  readonly recallMessage: (messageId: string) => Promise<void>;
  readonly getReadUsers: (messageId: string, roomId: string) => Promise<readonly string[]>;
  readonly toggleReaction: (messageId: string, emoji: string) => Promise<void>;
  readonly pinMessage: (messageId: string) => Promise<void>;
  readonly unpinMessage: (messageId: string) => Promise<void>;
  readonly markActiveRoomRead: () => Promise<void>;

  // Media upload
  readonly uploadFile: (file: File) => Promise<{ url: string; name: string; size: number }>;

  // Profile & account
  readonly profile: UserProfile | null;
  readonly isLoadingProfile: boolean;
  readonly refreshProfile: () => Promise<void>;
  readonly updateProfile: (req: { nickname?: string; avatar_url?: string }) => Promise<void>;
  readonly updatePassword: (oldPassword: string, newPassword: string) => Promise<void>;
  readonly saveDeviceToken: (token: string, platform: string) => Promise<void>;

  // Room creation
  readonly createGroupRoom: (name: string, memberIds: readonly string[]) => Promise<string>;

  readonly replyingToMessage: UIMessage | null;
  readonly setReplyingToMessage: (msg: UIMessage | null) => void;
  readonly sendTextMessage: (text: string, replyToMsgId?: string) => Promise<void>;
  readonly sendImageMessage: (
    url: string,
    width?: number,
    height?: number,
    size?: number,
    replyToMsgId?: string,
  ) => Promise<void>;
  readonly sendVideoMessage: (
    url: string,
    duration?: number,
    width?: number,
    height?: number,
    size?: number,
    thumbnailUrl?: string,
    replyToMsgId?: string,
  ) => Promise<void>;
  readonly sendFileMessage: (
    url: string,
    name: string,
    size: number,
    mimeType?: string,
    replyToMsgId?: string,
  ) => Promise<void>;
  readonly clearError: () => void;

  // Typing state
  readonly typingRooms: Readonly<Record<string, string>>;
  readonly sendTyping: (roomId: string) => void;
}
