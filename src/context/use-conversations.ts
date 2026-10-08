import { useCallback } from "react";
import type { ConversationInfo, MemberInfo, Message } from "go-chat-sdk";

import type { ChatDomainDeps } from "./domain-deps";
import { isErrorWithMessage, type UIMessage } from "@/types/chat";

/** State owned by the conversations domain. */
export interface ConversationsState {
  readonly conversations: readonly ConversationInfo[];
  readonly setConversations: (value: readonly ConversationInfo[]) => void;
  readonly isLoadingConversations: boolean;
  readonly setIsLoadingConversations: (value: boolean) => void;
}

/** Room-level actions that depend on the conversation list being fresh. */
export interface RoomsCrossTalk {
  readonly refreshRooms: () => Promise<void>;
}

export interface UseConversationsOptions extends ChatDomainDeps {
  readonly sdk: {
    listConversations: () => Promise<{ conversations: ConversationInfo[] }>;
    muteConversation: (roomId: string, req: { is_muted: boolean }) => Promise<void>;
    pinConversation: (roomId: string, req: { is_pinned: boolean }) => Promise<void>;
    clearUnread: (roomId: string) => Promise<void>;
    deleteRoom: (roomId: string) => Promise<void>;
  };
  readonly state: ConversationsState;
  readonly setMembers: (value: readonly MemberInfo[]) => void;
  readonly setActiveRoomDetail: (value: null) => void;
  readonly setPinnedMessages: (value: readonly Message[]) => void;
  readonly setMessagesByRoom: (
    fn: (prev: Readonly<Record<string, readonly UIMessage[]>>) => Record<string, readonly UIMessage[]>,
  ) => void;
  readonly rooms: RoomsCrossTalk;
}

/** Conversation list plus the mute/pin/unread toggles and room dissolution. */
export function useConversations(options: UseConversationsOptions) {
  const { sdk, state, activeRoomId, setError } = options;

  // The conversation list is the source of truth for unread counts, mute and
  // pin state, so every action below refreshes it.
  const refreshConversations = useCallback(async () => {
    if (!options.isAuthenticated()) {
      return;
    }
    state.setIsLoadingConversations(true);
    try {
      const resp = await sdk.listConversations();
      state.setConversations(resp.conversations ?? []);
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to load conversations");
    } finally {
      state.setIsLoadingConversations(false);
    }
  }, [sdk, state, setError, options.isAuthenticated]);

  const muteConversation = useCallback(
    async (roomId: string, muted: boolean) => {
      try {
        await sdk.muteConversation(roomId, { is_muted: muted });
        await refreshConversations();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update mute setting");
        throw err;
      }
    },
    [sdk, refreshConversations, setError],
  );

  const pinConversation = useCallback(
    async (roomId: string, pinned: boolean) => {
      try {
        await sdk.pinConversation(roomId, { is_pinned: pinned });
        await refreshConversations();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update pin setting");
        throw err;
      }
    },
    [sdk, refreshConversations, setError],
  );

  const clearUnread = useCallback(
    async (roomId: string) => {
      try {
        await sdk.clearUnread(roomId);
        await refreshConversations();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to clear unread");
        throw err;
      }
    },
    [sdk, refreshConversations, setError],
  );

  // Dissolving drops the room for every member, so local state must drop it too.
  const deleteRoom = useCallback(
    async (roomId: string) => {
      try {
        await sdk.deleteRoom(roomId);
        options.setMessagesByRoom((prev) => {
          const next = { ...prev };
          delete next[roomId];
          return next;
        });
        if (roomId === activeRoomId) {
          options.setActiveRoomId(null);
          options.setActiveRoomDetail(null);
          options.setMembers([]);
          options.setPinnedMessages([]);
        }
        await options.rooms.refreshRooms();
        await refreshConversations();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to dissolve room");
        throw err;
      }
    },
    [
      sdk,
      activeRoomId,
      refreshConversations,
      setError,
      options.setMessagesByRoom,
      options.setActiveRoomId,
      options.setActiveRoomDetail,
      options.setMembers,
      options.setPinnedMessages,
      options.rooms,
    ],
  );

  return {
    refreshConversations,
    muteConversation,
    pinConversation,
    clearUnread,
    deleteRoom,
  };
}
