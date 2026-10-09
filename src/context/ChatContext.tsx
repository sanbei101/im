import {
  ChatSDK,
  ConnectionState,
  ChatEventType,
  type UserResponse,
  type RoomInfo,
  type FriendItem,
  type FriendApplication,
  type UserProfile,
  type MemberInfo,
  type RoomDetail,
  type ConversationInfo,
  type Message,
} from "go-chat-sdk";
import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  useRef,
  type ReactNode,
} from "react";

import {
  type UIMessage,
  type ServerConfig,
  mapSdkMessageToUIMessage,
  updateMessageAck,
  isErrorWithMessage,
  sortConversations,
} from "@/types/chat";

import { type ChatContextValue } from "./chat-context-value";
import { useAccount } from "./use-account";
import { useContacts } from "./use-contacts";
import { useConversations } from "./use-conversations";
import { useMessages } from "./use-messages";
import { useRooms } from "./use-rooms";
import { useSenders } from "./use-senders";

const DEFAULT_CONFIG: ServerConfig = {
  baseURL: "http://127.0.0.1:8801",
  gatewayURL: "ws://127.0.0.1:8800/ws",
};

interface StorageKeys {
  readonly CONFIG: string;
  readonly USER: string;
}

const STORAGE_KEYS: StorageKeys = {
  CONFIG: "im_server_config",
  USER: "im_current_user",
};

function getStoredString(key: string): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  return window.localStorage.getItem(key);
}

function getSessionString(key: string): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  return window.sessionStorage.getItem(key);
}

function setSessionString(key: string, value: string): void {
  if (typeof window !== "undefined") {
    window.sessionStorage.setItem(key, value);
  }
}

function removeSessionString(key: string): void {
  if (typeof window !== "undefined") {
    window.sessionStorage.removeItem(key);
  }
}

function parseStoredConfig(): ServerConfig {
  const json = getStoredString(STORAGE_KEYS.CONFIG);
  if (!json) {
    return DEFAULT_CONFIG;
  }
  try {
    const parsed: unknown = JSON.parse(json);
    if (
      typeof parsed === "object" &&
      parsed !== null &&
      "baseURL" in parsed &&
      typeof parsed.baseURL === "string" &&
      "gatewayURL" in parsed &&
      typeof parsed.gatewayURL === "string"
    ) {
      return {
        baseURL: parsed.baseURL,
        gatewayURL: parsed.gatewayURL,
      };
    }
  } catch {
    return DEFAULT_CONFIG;
  }
  return DEFAULT_CONFIG;
}

function parseStoredUser(): UserResponse | null {
  // Clean up legacy shared localStorage user if any
  if (typeof window !== "undefined") {
    window.localStorage.removeItem(STORAGE_KEYS.USER);
  }

  const json = getSessionString(STORAGE_KEYS.USER);
  if (!json) {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(json);
    if (
      typeof parsed === "object" &&
      parsed !== null &&
      "user_id" in parsed &&
      typeof parsed.user_id === "string" &&
      "username" in parsed &&
      typeof parsed.username === "string" &&
      "token" in parsed &&
      typeof parsed.token === "string"
    ) {
      return {
        user_id: parsed.user_id,
        username: parsed.username,
        token: parsed.token,
      };
    }
  } catch {
    return null;
  }
  return null;
}

const ChatContext = createContext<ChatContextValue | null>(null);

export function ChatProvider({ children }: { readonly children: ReactNode }) {
  const [config, setConfig] = useState<ServerConfig>(parseStoredConfig);
  const [currentUser, setCurrentUser] = useState<UserResponse | null>(parseStoredUser);
  const [connectionState, setConnectionState] = useState<ConnectionState>(
    ConnectionState.Disconnected,
  );
  const [rooms, setRooms] = useState<readonly RoomInfo[]>([]);
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);
  const [messagesByRoom, setMessagesByRoom] = useState<
    Readonly<Record<string, readonly UIMessage[]>>
  >({});
  const [isLoadingRooms, setIsLoadingRooms] = useState<boolean>(false);
  const [isLoadingHistory, setIsLoadingHistory] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [replyingToMessage, setReplyingToMessage] = useState<UIMessage | null>(null);

  // Conversations, friends, presence and profile.
  const [conversations, setConversations] = useState<readonly ConversationInfo[]>([]);
  const [isLoadingConversations, setIsLoadingConversations] = useState<boolean>(false);
  const [friends, setFriends] = useState<readonly FriendItem[]>([]);
  const [friendApplications, setFriendApplications] = useState<readonly FriendApplication[]>([]);
  const [blacklist, setBlacklist] = useState<readonly UserProfile[]>([]);
  const [isLoadingFriends, setIsLoadingFriends] = useState<boolean>(false);
  const [presence, setPresence] = useState<Readonly<Record<string, boolean>>>({});
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [isLoadingProfile, setIsLoadingProfile] = useState<boolean>(false);

  // Active room detail: members, my role and pinned messages.
  const [activeRoomDetail, setActiveRoomDetail] = useState<RoomDetail | null>(null);
  const [members, setMembers] = useState<readonly MemberInfo[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState<boolean>(false);
  const [pinnedMessages, setPinnedMessages] = useState<readonly Message[]>([]);

  // History pagination: the lowest room_seq already rendered.
  const [hasMoreHistory, setHasMoreHistory] = useState<boolean>(false);
  const [isLoadingMoreHistory, setIsLoadingMoreHistory] = useState<boolean>(false);

  // Message search results.
  const [searchResults, setSearchResults] = useState<readonly Message[]>([]);
  const [isSearching, setIsSearching] = useState<boolean>(false);

  // Real-time typing indicators (roomId -> userId)
  const [typingRooms, setTypingRooms] = useState<Readonly<Record<string, string>>>({});
  const typingTimersRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({});

  const sdkRef = useRef<ChatSDK | null>(null);

  if (sdkRef.current === null) {
    const newSdk = new ChatSDK({
      baseURL: config.baseURL,
      gatewayURL: config.gatewayURL,
      reconnectInterval: 2500,
      maxReconnectAttempts: 8,
    });
    if (currentUser) {
      newSdk.setAuth(currentUser);
    }
    sdkRef.current = newSdk;
  }

  const sdk = sdkRef.current;

  const isAuthenticated = useCallback(() => sdk.isAuthenticated(), [sdk]);

  // Reads that back the message senders and the reaction author check.
  const currentUserId = currentUser?.user_id ?? null;

  // Several domains need to reload the room list, but it is defined further
  // down because it depends on selectRoom. This ref breaks that cycle without
  // leaking a stale closure.
  const refreshRoomsRef = useRef<() => Promise<void>>(async () => undefined);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const sendTyping = useCallback(
    (roomId: string) => {
      sdk.sendTyping(roomId);
    },
    [sdk],
  );

  const updateConfig = useCallback(
    (newConfig: ServerConfig) => {
      setConfig(newConfig);
      if (typeof window !== "undefined") {
        window.localStorage.setItem(STORAGE_KEYS.CONFIG, JSON.stringify(newConfig));
      }
      if (sdkRef.current) {
        sdkRef.current.disconnect();
      }
      const newSdk = new ChatSDK({
        baseURL: newConfig.baseURL,
        gatewayURL: newConfig.gatewayURL,
        reconnectInterval: 2500,
        maxReconnectAttempts: 8,
      });
      if (currentUser) {
        newSdk.setAuth(currentUser);
      }
      sdkRef.current = newSdk;
      setConnectionState(ConnectionState.Disconnected);
    },
    [currentUser],
  );

  // Connect handler
  const connect = useCallback(async () => {
    if (!sdk.isAuthenticated()) {
      return;
    }
    try {
      await sdk.connect();
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to connect to gateway";
      setError(msg);
    }
  }, [sdk]);

  // Disconnect handler
  const disconnect = useCallback(() => {
    sdk.disconnect();
    setConnectionState(ConnectionState.Disconnected);
  }, [sdk]);

  // Domain hooks. Each owns one slice of state; ChatContext only wires them.
  const conversationsDomain = useConversations({
    sdk,
    activeRoomId,
    setActiveRoomId,
    setMembers,
    setActiveRoomDetail,
    setPinnedMessages,
    setMessagesByRoom,
    setError,
    isAuthenticated,
    state: {
      conversations,
      setConversations,
      isLoadingConversations,
      setIsLoadingConversations,
    },
    rooms: { refreshRooms: () => refreshRoomsRef.current() },
  });

  const roomsDomain = useRooms({
    sdk,
    activeRoomId,
    setActiveRoomId,
    setError,
    isAuthenticated,
    state: {
      activeRoomDetail,
      setActiveRoomDetail,
      members,
      setMembers,
      isLoadingMembers,
      setIsLoadingMembers,
      pinnedMessages,
      setPinnedMessages,
    },
    refreshRooms: () => refreshRoomsRef.current(),
  });

  const contactsDomain = useContacts({
    sdk,
    activeRoomId,
    setActiveRoomId,
    setError,
    isAuthenticated,
    state: {
      friends,
      setFriends,
      friendApplications,
      setFriendApplications,
      blacklist,
      setBlacklist,
      isLoadingFriends,
      setIsLoadingFriends,
      presence,
      setPresence,
    },
  });

  const messagesDomain = useMessages({
    sdk,
    activeRoomId,
    setActiveRoomId,
    setError,
    isAuthenticated,
    currentUserId,
    state: {
      messagesByRoom,
      setMessagesByRoom,
      searchResults,
      setSearchResults,
      isSearching,
      setIsSearching,
    },
    refreshRoomDetail: () => roomsDomain.refreshRoomDetail(),
  });

  const accountDomain = useAccount({
    sdk,
    activeRoomId,
    setActiveRoomId,
    setError,
    isAuthenticated,
    state: { profile, setProfile, isLoadingProfile, setIsLoadingProfile },
    currentUser,
    setCurrentUser,
    resetSessionState: () => {
      setRooms([]);
      setActiveRoomId(null);
      setReplyingToMessage(null);
      setMessagesByRoom({});
      setConnectionState(ConnectionState.Disconnected);
    },
    removeStoredUser: removeSessionString.bind(null, STORAGE_KEYS.USER),
  });

  const conversationsDomainRef = useRef(conversationsDomain);
  conversationsDomainRef.current = conversationsDomain;

  const roomsDomainRef = useRef(roomsDomain);
  roomsDomainRef.current = roomsDomain;

  const contactsDomainRef = useRef(contactsDomain);
  contactsDomainRef.current = contactsDomain;

  const accountDomainRef = useRef(accountDomain);
  accountDomainRef.current = accountDomain;

  // Select room & load history
  const selectRoom = useCallback(
    async (roomId: string) => {
      setActiveRoomId(roomId);
      setReplyingToMessage(null);
      if (!roomId) {
        return;
      }

      setIsLoadingHistory(true);
      try {
        const resp = await sdk.getHistoryMessages({
          room_id: roomId,
          page_size: 50,
        });

        const historyMsgs: readonly UIMessage[] = resp.messages.map((m) =>
          mapSdkMessageToUIMessage(m, "sent"),
        );

        // Sort by room_seq ascending, falling back to server_time.
        const sorted = [...historyMsgs].sort((a, b) => {
          if (a.roomSeq !== b.roomSeq) {
            return a.roomSeq - b.roomSeq;
          }
          return a.serverTime - b.serverTime;
        });

        setMessagesByRoom((prev) => ({
          ...prev,
          [roomId]: sorted,
        }));
        setHasMoreHistory(resp.has_more);
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to fetch room history";
        setError(msg);
      } finally {
        setIsLoadingHistory(false);
      }
    },
    [sdk],
  );

  const activeRoomIdRef = useRef<string | null>(activeRoomId);
  activeRoomIdRef.current = activeRoomId;

  const refreshRooms = useCallback(async () => {
    if (!sdk.isAuthenticated()) {
      return;
    }
    setIsLoadingRooms(true);
    try {
      const resp = await sdk.listRooms();
      const loadedRooms = resp.rooms ?? [];
      setRooms(loadedRooms);
      // Only auto-select when nothing is selected yet: re-selecting the active
      // room would refetch its history and drop locally merged state such as
      // reactions.
      if (loadedRooms.length > 0 && !activeRoomIdRef.current) {
        const first = loadedRooms[0]?.room_id;
        if (first) {
          void selectRoom(first);
        }
      }
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to load rooms";
      setError(msg);
    } finally {
      setIsLoadingRooms(false);
    }
  }, [sdk, selectRoom]);

  refreshRoomsRef.current = refreshRooms;

  // Login handler
  const login = useCallback(
    async (req: { username: string; password: string }) => {
      try {
        setError(null);
        const user = await sdk.login(req);
        setCurrentUser(user);
        setSessionString(STORAGE_KEYS.USER, JSON.stringify(user));
        try {
          await sdk.connect();
        } catch {
          // Connection can be retried automatically
        }
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Login failed";
        setError(msg);
        throw err;
      }
    },
    [sdk],
  );

  // Register handler
  const register = useCallback(
    async (req: { username: string; password: string }) => {
      try {
        setError(null);
        const user = await sdk.register(req);
        setCurrentUser(user);
        setSessionString(STORAGE_KEYS.USER, JSON.stringify(user));
        try {
          await sdk.connect();
        } catch {
          // Connection can be retried automatically
        }
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Registration failed";
        setError(msg);
        throw err;
      }
    },
    [sdk],
  );

  // Create single room
  const createSingleRoom = useCallback(
    async (targetUserId: string): Promise<string> => {
      try {
        const resp = await sdk.createRoom({ user_id_2: targetUserId });
        await refreshRooms();
        await selectRoom(resp.room_id);
        return resp.room_id;
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to create single chat";
        setError(msg);
        throw err;
      }
    },
    [sdk, refreshRooms, selectRoom],
  );

  // Create group room
  const createGroupRoom = useCallback(
    async (name: string, memberIds: readonly string[]): Promise<string> => {
      try {
        const resp = await sdk.createGroupRoom({
          name: name.trim() ? name : undefined,
          member_ids: [...memberIds],
        });
        await refreshRooms();
        await selectRoom(resp.room_id);
        return resp.room_id;
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to create group chat";
        setError(msg);
        throw err;
      }
    },
    [sdk, refreshRooms, selectRoom],
  );

  // History pagination: pull the page below the oldest rendered room_seq.
  const loadMoreHistory = useCallback(async () => {
    if (!activeRoomId || isLoadingMoreHistory) {
      return;
    }
    const rendered = messagesByRoom[activeRoomId] ?? [];
    const oldestSeq = rendered.reduce<number | null>(
      (min, m) => (min === null || m.roomSeq < min ? m.roomSeq : min),
      null,
    );
    if (oldestSeq === null || oldestSeq <= 1) {
      return;
    }
    setIsLoadingMoreHistory(true);
    try {
      const resp = await sdk.getHistoryMessages({
        room_id: activeRoomId,
        before_seq: oldestSeq,
        page_size: 50,
      });
      const older: readonly UIMessage[] = resp.messages.map((m) =>
        mapSdkMessageToUIMessage(m, "sent"),
      );
      setMessagesByRoom((prev) => {
        const existing = prev[activeRoomId] ?? [];
        const known = new Set(existing.map((m) => m.id));
        const merged = [...older.filter((m) => !known.has(m.id)), ...existing];
        return { ...prev, [activeRoomId]: merged };
      });
      setHasMoreHistory(resp.has_more);
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to load older messages";
      setError(msg);
    } finally {
      setIsLoadingMoreHistory(false);
    }
  }, [sdk, activeRoomId, isLoadingMoreHistory, messagesByRoom]);

  // Marking read needs the conversation's last_seq, refreshed after every
  // message so the watermark never regresses.
  const markActiveRoomRead = useCallback(async () => {
    if (!activeRoomId) {
      return;
    }
    const roomSeq = conversations.find((c) => c.room.room_id === activeRoomId)?.room.last_seq ?? 0;
    if (roomSeq <= 0) {
      return;
    }
    try {
      await sdk.markRead(activeRoomId, { read_seq: roomSeq });
      await conversationsDomain.refreshConversations();
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to mark room read";
      setError(msg);
    }
  }, [sdk, activeRoomId, conversations, conversationsDomain]);

  // Message sending: one optimistic/ack pipeline behind typed wrappers.
  const senders = useSenders({
    sdk,
    activeRoomId,
    setActiveRoomId,
    setError,
    isAuthenticated,
    currentUser,
    replyingToMessage,
    setReplyingToMessage,
    setMessagesByRoom,
    onSent: (sentInfo) => {
      if (sentInfo) {
        setConversations((prev) => {
          const idx = prev.findIndex((c) => c.room.room_id === sentInfo.roomId);
          if (idx >= 0) {
            const old = prev[idx];
            const updated: ConversationInfo = {
              ...old,
              last_message: {
                msg_id: sentInfo.msgId,
                client_msg_id: sentInfo.clientMsgId,
                sender_id: currentUser?.user_id || "",
                room_id: sentInfo.roomId,
                room_seq: sentInfo.roomSeq,
                server_time: sentInfo.serverTime,
                msg_type: sentInfo.msgType,
                payload: sentInfo.payload,
              },
              room: {
                ...old.room,
                last_seq: sentInfo.roomSeq,
                updated_at: new Date().toISOString(),
              },
            };
            const next = [...prev];
            next.splice(idx, 1);
            return [updated, ...next].sort(sortConversations);
          }
          return prev;
        });
      }
      void conversationsDomain.refreshConversations();
    },
  });

  // Uploads go through the backend presign, then a raw PUT to object storage.
  const uploadFile = useCallback(
    async (file: File) => {
      const presign = await sdk.presignUpload({
        file_name: file.name,
        content_type: file.type || "application/octet-stream",
      });
      const upload = await fetch(presign.upload_url, {
        method: "PUT",
        body: file,
        headers: file.type ? { "Content-Type": file.type } : undefined,
      });
      if (!upload.ok) {
        throw new Error(`upload failed: HTTP ${upload.status}`);
      }
      return {
        url: presign.download_url,
        name: file.name,
        size: file.size,
      };
    },
    [sdk],
  );

  // Attach SDK listeners
  useEffect(() => {
    const unsubState = sdk.on(ChatEventType.ConnectionStateChange, (event) => {
      setConnectionState(event.data.state);
    });

    const unsubMsg = sdk.on(ChatEventType.MessageReceived, (event) => {
      const incoming = event.data.message;
      const targetRoomId = incoming.room_id;

      // Clear typing indicator for this room
      if (typingTimersRef.current[targetRoomId]) {
        clearTimeout(typingTimersRef.current[targetRoomId]);
        delete typingTimersRef.current[targetRoomId];
      }
      setTypingRooms((prev) => {
        if (!prev[targetRoomId]) return prev;
        const next = { ...prev };
        delete next[targetRoomId];
        return next;
      });

      setRooms((currentRooms) => {
        if (!currentRooms.some((r) => r.room_id === targetRoomId)) {
          void refreshRoomsRef.current();
        }
        return currentRooms;
      });

      setMessagesByRoom((prev) => {
        const roomMessages = prev[targetRoomId] ?? [];

        // A message is already present when echo matches on msg_id or client_msg_id.
        const alreadyExists = roomMessages.some(
          (m) =>
            (incoming.msg_id && m.id === incoming.msg_id) ||
            (incoming.client_msg_id && m.clientMsgId === incoming.client_msg_id),
        );

        if (alreadyExists) {
          return {
            ...prev,
            [targetRoomId]: roomMessages.map((m) => {
              if (
                (incoming.msg_id && m.id === incoming.msg_id) ||
                (incoming.client_msg_id && m.clientMsgId === incoming.client_msg_id)
              ) {
                const mapped = mapSdkMessageToUIMessage(incoming, "sent");
                return {
                  ...mapped,
                  replyToMsgId: mapped.replyToMsgId || m.replyToMsgId,
                  reactions: m.reactions ?? mapped.reactions,
                };
              }
              return m;
            }),
          };
        }

        const newMsg = mapSdkMessageToUIMessage(incoming, "sent");
        return {
          ...prev,
          [targetRoomId]: [...roomMessages, newMsg],
        };
      });

      // Optimistically update conversations in local state immediately
      setConversations((prev) => {
        const existingIndex = prev.findIndex((c) => c.room.room_id === targetRoomId);
        const isCurrentActive = targetRoomId === activeRoomIdRef.current;
        if (existingIndex >= 0) {
          const old = prev[existingIndex];
          const updated: ConversationInfo = {
            ...old,
            last_message: incoming,
            unread_count: isCurrentActive ? 0 : (old.unread_count || 0) + 1,
            room: {
              ...old.room,
              last_seq: incoming.room_seq,
              updated_at: new Date(
                incoming.server_time
                  ? incoming.server_time > 10_000_000_000_000
                    ? Math.floor(incoming.server_time / 1000)
                    : incoming.server_time
                  : Date.now(),
              ).toISOString(),
            },
          };
          const next = [...prev];
          next.splice(existingIndex, 1);
          return [updated, ...next].sort(sortConversations);
        } else {
          const roomObj = rooms.find((r) => r.room_id === targetRoomId);
          const newConv: ConversationInfo = {
            room: {
              room_id: targetRoomId,
              chat_type: roomObj?.chat_type || "single",
              name: roomObj?.name ?? "",
              avatar_url: roomObj?.avatar_url ?? "",
              notice: roomObj?.notice ?? "",
              last_seq: incoming.room_seq,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
            member: {
              room_id: targetRoomId,
              user_id: currentUserId || "",
              role: "member",
              is_hidden: false,
              is_muted: false,
              is_pinned: false,
            },
            unread_count: isCurrentActive ? 0 : 1,
            last_message: incoming,
          };
          return [newConv, ...prev].sort(sortConversations);
        }
      });

      // Unread counts and the sidebar preview come from the conversation list.
      void conversationsDomainRef.current.refreshConversations();
    });

    const unsubTyping = sdk.on(ChatEventType.Typing, (event) => {
      const { room_id, user_id } = event.data;
      if (!room_id || !user_id || user_id === currentUserId) {
        return;
      }
      if (typingTimersRef.current[room_id]) {
        clearTimeout(typingTimersRef.current[room_id]);
      }
      setTypingRooms((prev) => ({ ...prev, [room_id]: user_id }));
      typingTimersRef.current[room_id] = setTimeout(() => {
        setTypingRooms((prev) => {
          if (prev[room_id] !== user_id) return prev;
          const next = { ...prev };
          delete next[room_id];
          return next;
        });
        delete typingTimersRef.current[room_id];
      }, 3500);
    });

    const unsubAck = sdk.on(ChatEventType.MessageSent, (event) => {
      const ack = event.data;
      setMessagesByRoom((prev) => {
        const roomMessages = prev[ack.room_id] ?? [];
        return {
          ...prev,
          [ack.room_id]: roomMessages.map((m) => {
            if (m.clientMsgId === ack.client_msg_id) {
              return updateMessageAck(m, ack.msg_id, ack.room_seq, ack.server_time);
            }
            return m;
          }),
        };
      });
      void conversationsDomainRef.current.refreshConversations();
    });

    const unsubErr = sdk.on(ChatEventType.Error, (event) => {
      console.warn("[ChatSDK Event Error]", event.data.code, event.data.message);
    });

    return () => {
      unsubState();
      unsubMsg();
      unsubTyping();
      unsubAck();
      unsubErr();
    };
  }, [sdk]);

  const loggedInUserIdRef = useRef<string | null>(null);

  // Initial connect & room load on mount if user authenticated.
  useEffect(() => {
    if (currentUser) {
      if (loggedInUserIdRef.current === currentUser.user_id) {
        return;
      }
      loggedInUserIdRef.current = currentUser.user_id;

      sdk.setAuth(currentUser);
      void connect();
      void refreshRoomsRef.current();
      void conversationsDomainRef.current.refreshConversations();
      void contactsDomainRef.current.refreshFriends();
      void accountDomainRef.current.refreshProfile();
    } else {
      loggedInUserIdRef.current = null;
    }
  }, [currentUser, sdk, connect]);

  // Room detail (members, role, pins) follows the active room.
  useEffect(() => {
    if (activeRoomId) {
      void roomsDomainRef.current.refreshRoomDetail();
    }
  }, [activeRoomId]);

  // Update document title for multi-tab testing
  useEffect(() => {
    if (typeof document !== "undefined") {
      if (currentUser) {
        document.title = `${currentUser.username} - Go IM`;
      } else {
        document.title = "Go IM";
      }
    }
  }, [currentUser]);

  // Periodic refresh for friends & applications (every 15s) and window focus sync
  useEffect(() => {
    if (!currentUser) return;
    const interval = setInterval(() => {
      void contactsDomainRef.current.refreshFriends();
    }, 15000);
    const handleFocus = () => {
      void contactsDomainRef.current.refreshFriends();
      void conversationsDomainRef.current.refreshConversations();
    };
    window.addEventListener("focus", handleFocus);
    return () => {
      clearInterval(interval);
      window.removeEventListener("focus", handleFocus);
    };
  }, [currentUser]);

  // Find active room object
  const activeRoom = rooms.find((r) => r.room_id === activeRoomId) ?? null;
  const activeMessages = activeRoomId ? (messagesByRoom[activeRoomId] ?? []) : [];

  const contextValue: ChatContextValue = {
    sdk,
    config,
    currentUser,
    connectionState,
    rooms,
    activeRoomId,
    activeRoom,
    messages: activeMessages,
    replyingToMessage,
    setReplyingToMessage,
    isLoadingRooms,
    isLoadingHistory,
    error,
    updateConfig,
    login,
    register,
    logout: accountDomain.logout,
    connect,
    disconnect,
    selectRoom,
    refreshRooms,
    createSingleRoom,
    createGroupRoom,
    conversations,
    isLoadingConversations,
    refreshConversations: conversationsDomain.refreshConversations,
    muteConversation: conversationsDomain.muteConversation,
    pinConversation: conversationsDomain.pinConversation,
    clearUnread: conversationsDomain.clearUnread,
    deleteRoom: conversationsDomain.deleteRoom,
    activeRoomDetail,
    members,
    isLoadingMembers,
    pinnedMessages,
    refreshRoomDetail: roomsDomain.refreshRoomDetail,
    updateActiveRoom: roomsDomain.updateActiveRoom,
    leaveActiveRoom: roomsDomain.leaveActiveRoom,
    addMembers: roomsDomain.addMembers,
    updateMemberRole: roomsDomain.updateMemberRole,
    removeMember: roomsDomain.removeMember,
    transferOwnership: roomsDomain.transferOwnership,
    friends,
    friendApplications,
    blacklist,
    isLoadingFriends,
    refreshFriends: contactsDomain.refreshFriends,
    applyFriend: contactsDomain.applyFriend,
    auditFriend: contactsDomain.auditFriend,
    deleteFriend: contactsDomain.deleteFriend,
    updateFriendRemark: contactsDomain.updateFriendRemark,
    addBlacklist: contactsDomain.addBlacklist,
    removeBlacklist: contactsDomain.removeBlacklist,
    searchUsers: contactsDomain.searchUsers,
    presence,
    queryPresence: contactsDomain.queryPresence,
    hasMoreHistory,
    isLoadingMoreHistory,
    loadMoreHistory,
    searchResults,
    isSearching,
    searchMessages: messagesDomain.searchMessages,
    clearSearchResults: messagesDomain.clearSearchResults,
    recallMessage: messagesDomain.recallMessage,
    getReadUsers: messagesDomain.getReadUsers,
    toggleReaction: messagesDomain.toggleReaction,
    pinMessage: messagesDomain.pinMessage,
    unpinMessage: messagesDomain.unpinMessage,
    markActiveRoomRead,
    uploadFile,
    profile,
    isLoadingProfile,
    refreshProfile: accountDomain.refreshProfile,
    updateProfile: accountDomain.updateProfile,
    updatePassword: accountDomain.updatePassword,
    saveDeviceToken: accountDomain.saveDeviceToken,
    sendTextMessage: senders.sendTextMessage,
    sendImageMessage: senders.sendImageMessage,
    sendVideoMessage: senders.sendVideoMessage,
    sendFileMessage: senders.sendFileMessage,
    clearError,
    typingRooms,
    sendTyping,
  };

  return <ChatContext.Provider value={contextValue}>{children}</ChatContext.Provider>;
}

export function useChat(): ChatContextValue {
  const context = useContext(ChatContext);
  if (!context) {
    throw new Error("useChat must be used within a ChatProvider");
  }
  return context;
}
