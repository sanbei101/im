import {
  ChatSDK,
  ConnectionState,
  ChatEventType,
  MessageType,
  type UserResponse,
  type RoomInfo,
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
  updateMessageStatus,
  updateMessageAck,
  isErrorWithMessage,
} from "@/types/chat";

interface ChatContextValue {
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
  readonly createGroupRoom: (name: string, memberIds: readonly string[]) => Promise<string>;
  readonly sendTextMessage: (text: string) => Promise<void>;
  readonly sendImageMessage: (
    url: string,
    width?: number,
    height?: number,
    size?: number,
  ) => Promise<void>;
  readonly sendFileMessage: (
    url: string,
    name: string,
    size: number,
    mimeType?: string,
  ) => Promise<void>;
  readonly clearError: () => void;
}

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
  const [messagesByRoom, setMessagesByRoom] = useState<Record<string, readonly UIMessage[]>>({});
  const [isLoadingRooms, setIsLoadingRooms] = useState<boolean>(false);
  const [isLoadingHistory, setIsLoadingHistory] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

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

  const clearError = useCallback(() => {
    setError(null);
  }, []);

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

  // Select room & load history
  const selectRoom = useCallback(
    async (roomId: string) => {
      setActiveRoomId(roomId);
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

        // Sort by server_time or room_seq ascending
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
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to fetch room history";
        setError(msg);
      } finally {
        setIsLoadingHistory(false);
      }
    },
    [sdk],
  );

  // Refresh rooms list
  const refreshRooms = useCallback(async () => {
    if (!sdk.isAuthenticated()) {
      return;
    }
    setIsLoadingRooms(true);
    try {
      const resp = await sdk.listRooms();
      const loadedRooms = resp.rooms ?? [];
      setRooms(loadedRooms);
      if (loadedRooms.length > 0) {
        const targetId = activeRoomId ?? loadedRooms[0]?.room_id;
        if (targetId) {
          void selectRoom(targetId);
        }
      }
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to load rooms";
      setError(msg);
    } finally {
      setIsLoadingRooms(false);
    }
  }, [sdk, activeRoomId, selectRoom]);

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
    [sdk]
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
    [sdk]
  );

  // Logout handler
  const logout = useCallback(() => {
    sdk.disconnect();
    sdk.clearAuth();
    setCurrentUser(null);
    setRooms([]);
    setActiveRoomId(null);
    setMessagesByRoom({});
    setConnectionState(ConnectionState.Disconnected);
    removeSessionString(STORAGE_KEYS.USER);
  }, [sdk]);

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

  // Send Text Message
  const sendTextMessage = useCallback(
    async (text: string) => {
      if (!activeRoomId || !currentUser || !text.trim()) {
        return;
      }
      const clientMsgId = sdk.generateMessageId();
      const optimisticMsg: UIMessage = {
        id: clientMsgId,
        clientMsgId,
        senderId: currentUser.user_id,
        roomId: activeRoomId,
        roomSeq: 0,
        serverTime: Date.now(),
        msgType: MessageType.Text,
        payload: { text },
        status: "sending",
      };

      setMessagesByRoom((prev) => {
        const existing = prev[activeRoomId] ?? [];
        return {
          ...prev,
          [activeRoomId]: [...existing, optimisticMsg],
        };
      });

      try {
        await sdk.sendTextMessage({
          room_id: activeRoomId,
          client_msg_id: clientMsgId,
          text,
        });

        // Update status to sent
        setMessagesByRoom((prev) => {
          const list = prev[activeRoomId] ?? [];
          return {
            ...prev,
            [activeRoomId]: list.map((m) =>
              m.clientMsgId === clientMsgId ? updateMessageStatus(m, "sent") : m,
            ),
          };
        });
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to send message";
        setMessagesByRoom((prev) => {
          const list = prev[activeRoomId] ?? [];
          return {
            ...prev,
            [activeRoomId]: list.map((m) =>
              m.clientMsgId === clientMsgId ? updateMessageStatus(m, "error", msg) : m,
            ),
          };
        });
      }
    },
    [activeRoomId, currentUser, sdk],
  );

  // Send Image Message
  const sendImageMessage = useCallback(
    async (url: string, width?: number, height?: number, size?: number) => {
      if (!activeRoomId || !currentUser || !url.trim()) {
        return;
      }
      const clientMsgId = sdk.generateMessageId();
      const optimisticMsg: UIMessage = {
        id: clientMsgId,
        clientMsgId,
        senderId: currentUser.user_id,
        roomId: activeRoomId,
        roomSeq: 0,
        serverTime: Date.now(),
        msgType: MessageType.Image,
        payload: { url, width, height, size },
        status: "sending",
      };

      setMessagesByRoom((prev) => {
        const existing = prev[activeRoomId] ?? [];
        return {
          ...prev,
          [activeRoomId]: [...existing, optimisticMsg],
        };
      });

      try {
        await sdk.sendImageMessage({
          room_id: activeRoomId,
          client_msg_id: clientMsgId,
          url,
          width,
          height,
          size,
        });

        setMessagesByRoom((prev) => {
          const list = prev[activeRoomId] ?? [];
          return {
            ...prev,
            [activeRoomId]: list.map((m) =>
              m.clientMsgId === clientMsgId ? updateMessageStatus(m, "sent") : m,
            ),
          };
        });
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to send image";
        setMessagesByRoom((prev) => {
          const list = prev[activeRoomId] ?? [];
          return {
            ...prev,
            [activeRoomId]: list.map((m) =>
              m.clientMsgId === clientMsgId ? updateMessageStatus(m, "error", msg) : m,
            ),
          };
        });
      }
    },
    [activeRoomId, currentUser, sdk],
  );

  // Send File Message
  const sendFileMessage = useCallback(
    async (url: string, name: string, size: number, mimeType?: string) => {
      if (!activeRoomId || !currentUser || !url.trim()) {
        return;
      }
      const clientMsgId = sdk.generateMessageId();
      const optimisticMsg: UIMessage = {
        id: clientMsgId,
        clientMsgId,
        senderId: currentUser.user_id,
        roomId: activeRoomId,
        roomSeq: 0,
        serverTime: Date.now(),
        msgType: MessageType.File,
        payload: { url, name, size, mime_type: mimeType },
        status: "sending",
      };

      setMessagesByRoom((prev) => {
        const existing = prev[activeRoomId] ?? [];
        return {
          ...prev,
          [activeRoomId]: [...existing, optimisticMsg],
        };
      });

      try {
        await sdk.sendFileMessage({
          room_id: activeRoomId,
          client_msg_id: clientMsgId,
          url,
          name,
          size,
          mime_type: mimeType,
        });

        setMessagesByRoom((prev) => {
          const list = prev[activeRoomId] ?? [];
          return {
            ...prev,
            [activeRoomId]: list.map((m) =>
              m.clientMsgId === clientMsgId ? updateMessageStatus(m, "sent") : m,
            ),
          };
        });
      } catch (err) {
        const msg = isErrorWithMessage(err) ? err.message : "Failed to send file";
        setMessagesByRoom((prev) => {
          const list = prev[activeRoomId] ?? [];
          return {
            ...prev,
            [activeRoomId]: list.map((m) =>
              m.clientMsgId === clientMsgId ? updateMessageStatus(m, "error", msg) : m,
            ),
          };
        });
      }
    },
    [activeRoomId, currentUser, sdk],
  );

  // Attach SDK listeners
  useEffect(() => {
    const unsubState = sdk.on(ChatEventType.ConnectionStateChange, (event) => {
      setConnectionState(event.data.state);
    });

    const unsubMsg = sdk.on(ChatEventType.MessageReceived, (event) => {
      const incoming = event.data.message;
      const targetRoomId = incoming.room_id;

      setRooms((currentRooms) => {
        if (!currentRooms.some((r) => r.room_id === targetRoomId)) {
          void refreshRooms();
        }
        return currentRooms;
      });

      setMessagesByRoom((prev) => {
        const roomMessages = prev[targetRoomId] ?? [];

        // Check if message is already present by clientMsgId or msg_id
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
                return mapSdkMessageToUIMessage(incoming, "sent");
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
    });

    const unsubErr = sdk.on(ChatEventType.Error, (event) => {
      console.warn("[ChatSDK Event Error]", event.data.code, event.data.message);
    });

    return () => {
      unsubState();
      unsubMsg();
      unsubAck();
      unsubErr();
    };
  }, [sdk]);

  // Initial connect & room load on mount if user authenticated
  useEffect(() => {
    if (currentUser) {
      sdk.setAuth(currentUser);
      void connect();
      void refreshRooms();
    }
  }, [currentUser, sdk, connect, refreshRooms]);

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
    isLoadingRooms,
    isLoadingHistory,
    error,
    updateConfig,
    login,
    register,
    logout,
    connect,
    disconnect,
    selectRoom,
    refreshRooms,
    createSingleRoom,
    createGroupRoom,
    sendTextMessage,
    sendImageMessage,
    sendFileMessage,
    clearError,
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
