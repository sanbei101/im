import type { Message, ReactionGroup } from "go-chat-sdk";
import { useCallback, useRef } from "react";

import { applyReactions, isErrorWithMessage, markRecalled, type UIMessage } from "@/types/chat";

import type { ChatDomainDeps } from "./domain-deps";

export interface MessagesState {
  readonly messagesByRoom: Readonly<Record<string, readonly UIMessage[]>>;
  readonly setMessagesByRoom: (
    fn: (
      prev: Readonly<Record<string, readonly UIMessage[]>>,
    ) => Record<string, readonly UIMessage[]>,
  ) => void;
  readonly searchResults: readonly Message[];
  readonly setSearchResults: (value: readonly Message[]) => void;
  readonly isSearching: boolean;
  readonly setIsSearching: (value: boolean) => void;
}

export interface UseMessagesOptions extends ChatDomainDeps {
  readonly sdk: {
    recallMessage: (req: { room_id: string; msg_id: string }) => Promise<Message>;
    getReadUsers: (messageId: string, roomId: string) => Promise<{ read_user_ids: string[] }>;
    addReaction: (messageId: string, req: { room_id: string; emoji: string }) => Promise<void>;
    removeReaction: (messageId: string, req: { room_id: string; emoji: string }) => Promise<void>;
    getReactions: (messageId: string, roomId: string) => Promise<ReactionGroup[]>;
    pinMessage: (roomId: string, req: { msg_id: string }) => Promise<void>;
    unpinMessage: (roomId: string, messageId: string) => Promise<void>;
    searchRoomMessages: (
      roomId: string,
      params: { keyword: string; page_size: number },
    ) => Promise<Message[]>;
    markRead: (roomId: string, req: { read_seq: number }) => Promise<void>;
  };
  readonly state: MessagesState;
  readonly currentUserId: string | null;
  /** Refresh the room detail so the pins list stays in sync. */
  readonly refreshRoomDetail: () => Promise<void>;
}

/** Message-level mutations: recall, reactions, pins, receipts, search. */
export function useMessages(options: UseMessagesOptions) {
  const { sdk, setError, currentUserId } = options;
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const recallMessage = useCallback(
    async (messageId: string) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        await sdk.recallMessage({ room_id: roomId, msg_id: messageId });
        optionsRef.current.state.setMessagesByRoom((prev) => {
          const list = prev[roomId] ?? [];
          return {
            ...prev,
            [roomId]: list.map((m) =>
              m.id === messageId || m.clientMsgId === messageId ? markRecalled(m) : m,
            ),
          };
        });
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to recall message");
        throw err;
      }
    },
    [sdk, setError],
  );

  const getReadUsers = useCallback(
    async (messageId: string, roomId: string) => {
      try {
        const resp = await sdk.getReadUsers(messageId, roomId);
        return resp.read_user_ids ?? [];
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to load read receipts");
        throw err;
      }
    },
    [sdk, setError],
  );

  const toggleReaction = useCallback(
    async (messageId: string, emoji: string) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        const target = (optionsRef.current.state.messagesByRoom[roomId] ?? []).find(
          (m) => m.id === messageId || m.clientMsgId === messageId,
        );
        // The same emoji from this user again removes the reaction.
        const mine = target?.reactions?.find((r) =>
          currentUserId ? r.user_ids.includes(currentUserId) : false,
        );
        if (mine?.emoji === emoji) {
          await sdk.removeReaction(messageId, { room_id: roomId, emoji });
        } else {
          await sdk.addReaction(messageId, { room_id: roomId, emoji });
        }
        const groups = await sdk.getReactions(messageId, roomId);
        optionsRef.current.state.setMessagesByRoom((prev) => {
          const list = prev[roomId] ?? [];
          const next = { ...prev, [roomId]: applyReactions(list, messageId, groups) };
          return next;
        });
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to react to message");
      }
    },
    [sdk, currentUserId, setError],
  );

  const pinMessage = useCallback(
    async (messageId: string) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        await sdk.pinMessage(roomId, { msg_id: messageId });
        await optionsRef.current.refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to pin message");
        throw err;
      }
    },
    [sdk, setError],
  );

  const unpinMessage = useCallback(
    async (messageId: string) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        await sdk.unpinMessage(roomId, messageId);
        await optionsRef.current.refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to unpin message");
        throw err;
      }
    },
    [sdk, setError],
  );

  // The backend requires a room_id, so search is always room-scoped.
  const searchMessages = useCallback(
    async (keyword: string) => {
      const trimmed = keyword.trim();
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        optionsRef.current.state.setSearchResults([]);
        setError("Select a room before searching.");
        return;
      }
      if (trimmed === "") {
        optionsRef.current.state.setSearchResults([]);
        return;
      }
      optionsRef.current.state.setIsSearching(true);
      try {
        const results = await sdk.searchRoomMessages(roomId, {
          keyword: trimmed,
          page_size: 50,
        });
        optionsRef.current.state.setSearchResults(results);
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to search messages");
        optionsRef.current.state.setSearchResults([]);
      } finally {
        optionsRef.current.state.setIsSearching(false);
      }
    },
    [sdk, setError],
  );

  const clearSearchResults = useCallback(() => {
    optionsRef.current.state.setSearchResults([]);
  }, []);

  return {
    recallMessage,
    getReadUsers,
    toggleReaction,
    pinMessage,
    unpinMessage,
    searchMessages,
    clearSearchResults,
  };
}
