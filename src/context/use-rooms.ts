import type { MemberInfo, Message, RoomDetail } from "go-chat-sdk";
import { useCallback, useRef } from "react";

import { isErrorWithMessage } from "@/types/chat";

import type { ChatDomainDeps } from "./domain-deps";

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
  const { sdk, setError } = options;
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const inFlightRef = useRef(false);

  // One detail load covers the room record, its members and its pins.
  const refreshRoomDetail = useCallback(async () => {
    const roomId = optionsRef.current.activeRoomId;
    if (!roomId) {
      optionsRef.current.state.setActiveRoomDetail(null);
      optionsRef.current.state.setMembers([]);
      optionsRef.current.state.setPinnedMessages([]);
      return;
    }
    if (inFlightRef.current) {
      return;
    }
    inFlightRef.current = true;
    optionsRef.current.state.setIsLoadingMembers(true);
    try {
      const detail = await sdk.getRoom(roomId);
      optionsRef.current.state.setActiveRoomDetail(detail);
      const memberList = await sdk.listMembers(roomId);
      optionsRef.current.state.setMembers(memberList);
      const pins = await sdk.getPinnedMessages(roomId);
      optionsRef.current.state.setPinnedMessages(pins);
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to load room details");
    } finally {
      optionsRef.current.state.setIsLoadingMembers(false);
      inFlightRef.current = false;
    }
  }, [sdk, setError]);

  const updateActiveRoom = useCallback(
    async (req: { name?: string; notice?: string }) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        const detail = await sdk.updateRoom(roomId, req);
        optionsRef.current.state.setActiveRoomDetail(detail);
        await optionsRef.current.refreshRooms();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update room");
        throw err;
      }
    },
    [sdk, setError],
  );

  const leaveActiveRoom = useCallback(async () => {
    const roomId = optionsRef.current.activeRoomId;
    if (!roomId) {
      return;
    }
    try {
      await sdk.leaveRoom(roomId);
      optionsRef.current.setActiveRoomId(null);
      optionsRef.current.state.setActiveRoomDetail(null);
      optionsRef.current.state.setMembers([]);
      optionsRef.current.state.setPinnedMessages([]);
      await optionsRef.current.refreshRooms();
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to leave room");
      throw err;
    }
  }, [sdk, setError]);

  const addMembers = useCallback(
    async (memberIds: readonly string[]) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId || memberIds.length === 0) {
        return;
      }
      try {
        await sdk.addMembers(roomId, { member_ids: [...memberIds] });
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to add members");
        throw err;
      }
    },
    [sdk, refreshRoomDetail, setError],
  );

  const updateMemberRole = useCallback(
    async (userId: string, role: "admin" | "member") => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        await sdk.updateMemberRole(roomId, userId, { role });
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update member role");
        throw err;
      }
    },
    [sdk, refreshRoomDetail, setError],
  );

  const removeMember = useCallback(
    async (userId: string) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        await sdk.removeMember(roomId, userId);
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to remove member");
        throw err;
      }
    },
    [sdk, refreshRoomDetail, setError],
  );

  const transferOwnership = useCallback(
    async (newOwnerId: string) => {
      const roomId = optionsRef.current.activeRoomId;
      if (!roomId) {
        return;
      }
      try {
        await sdk.transferOwner(roomId, { new_owner_id: newOwnerId });
        await refreshRoomDetail();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to transfer ownership");
        throw err;
      }
    },
    [sdk, refreshRoomDetail, setError],
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
