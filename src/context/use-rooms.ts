import { useCallback } from "react";
import type { MemberInfo, Message, RoomDetail } from "go-chat-sdk";

import type { ChatDomainDeps } from "./domain-deps";
import { isErrorWithMessage } from "@/types/chat";

export interface RoomsState {
  readonly activeRoomDetail: RoomDetail | null;
  readonly setActiveRoomDetail: (value: RoomDetail | null) => void;
  readonly members: readonly MemberInfo[];
  readonly setMembers: (value: readonly MemberInfo[]) => void;
  readonly isLoadingMembers: boolean;
  readonly setIsLoadingMembers: (value: boolean) => void;
  readonly pinnedMessages: readonly Message[];
  readonly setPinnedMessages: (value: readonly Message[]) => void;
}

export interface UseRoomsOptions extends ChatDomainDeps {
  readonly sdk: {
    getRoom: (roomId: string) => Promise<RoomDetail>;
    listMembers: (roomId: string) => Promise<MemberInfo[]>;
    getPinnedMessages: (roomId: string) => Promise<Message[]>;
    updateRoom: (roomId: string, req: { name?: string; notice?: string }) => Promise<RoomDetail>;
    leaveRoom: (roomId: string) => Promise<void>;
    addMembers: (roomId: string, req: { member_ids: string[] }) => Promise<void>;
    updateMemberRole: (
      roomId: string,
      userId: string,
      req: { role: "admin" | "member" },
    ) => Promise<void>;
    removeMember: (roomId: string, userId: string) => Promise<void>;
    transferOwner: (roomId: string, req: { new_owner_id: string }) => Promise<void>;
  };
  readonly state: RoomsState;
  readonly refreshRooms: () => Promise<void>;
}

/** Active-room detail, member management and the leave flow. */
export function useRooms(options: UseRoomsOptions) {
  const { sdk, state, activeRoomId, setError } = options;

  // One detail load covers the room record, its members and its pins.
  const refreshRoomDetail = useCallback(async () => {
    if (!activeRoomId) {
      state.setActiveRoomDetail(null);
      state.setMembers([]);
      state.setPinnedMessages([]);
      return;
    }
    state.setIsLoadingMembers(true);
    try {
      const detail = await sdk.getRoom(activeRoomId);
      state.setActiveRoomDetail(detail);
      const memberList = await sdk.listMembers(activeRoomId);
      state.setMembers(memberList);
      const pins = await sdk.getPinnedMessages(activeRoomId);
      state.setPinnedMessages(pins);
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to load room details");
    } finally {
      state.setIsLoadingMembers(false);
    }
  }, [sdk, state, activeRoomId, setError]);

  const updateActiveRoom = useCallback(
    async (req: { name?: string; notice?: string }) => {
      if (!activeRoomId) {
        return;
      }
      try {
        const detail = await sdk.updateRoom(activeRoomId, req);
        state.setActiveRoomDetail(detail);
        await options.refreshRooms();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update room");
        throw err;
      }
    },
    [sdk, state, activeRoomId, setError, options.refreshRooms],
  );

  const leaveActiveRoom = useCallback(async () => {
    if (!activeRoomId) {
      return;
    }
    try {
      await sdk.leaveRoom(activeRoomId);
      options.setActiveRoomId(null);
      state.setActiveRoomDetail(null);
      state.setMembers([]);
      state.setPinnedMessages([]);
      await options.refreshRooms();
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to leave room");
      throw err;
    }
  }, [sdk, state, activeRoomId, setError, options.setActiveRoomId, options.refreshRooms]);

  const addMembers = useCallback(
    async (memberIds: readonly string[]) => {
      if (!activeRoomId || memberIds.length === 0) {
        return;
      }
      try {
        await sdk.addMembers(activeRoomId, { member_ids: [...memberIds] });
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to add members");
        throw err;
      }
    },
    [sdk, activeRoomId, refreshRoomDetail, setError],
  );

  const updateMemberRole = useCallback(
    async (userId: string, role: "admin" | "member") => {
      if (!activeRoomId) {
        return;
      }
      try {
        await sdk.updateMemberRole(activeRoomId, userId, { role });
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update member role");
        throw err;
      }
    },
    [sdk, activeRoomId, refreshRoomDetail, setError],
  );

  const removeMember = useCallback(
    async (userId: string) => {
      if (!activeRoomId) {
        return;
      }
      try {
        await sdk.removeMember(activeRoomId, userId);
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to remove member");
        throw err;
      }
    },
    [sdk, activeRoomId, refreshRoomDetail, setError],
  );

  const transferOwnership = useCallback(
    async (newOwnerId: string) => {
      if (!activeRoomId) {
        return;
      }
      try {
        await sdk.transferOwner(activeRoomId, { new_owner_id: newOwnerId });
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to transfer ownership");
        throw err;
      }
    },
    [sdk, activeRoomId, refreshRoomDetail, setError],
  );

  return {
    refreshRoomDetail,
    updateActiveRoom,
    leaveActiveRoom,
    addMembers,
    updateMemberRole,
    removeMember,
    transferOwnership,
  };
}
