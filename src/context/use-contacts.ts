import type { FriendApplication, FriendItem, PresenceResponse, UserProfile } from "go-chat-sdk";
import { useCallback, useRef } from "react";

import { isErrorWithMessage } from "@/types/chat";

import type { ChatDomainDeps } from "./domain-deps";

export interface ContactsState {
  readonly friends: readonly FriendItem[];
  readonly setFriends: (value: readonly FriendItem[]) => void;
  readonly friendApplications: readonly FriendApplication[];
  readonly setFriendApplications: (value: readonly FriendApplication[]) => void;
  readonly blacklist: readonly UserProfile[];
  readonly setBlacklist: (value: readonly UserProfile[]) => void;
  readonly isLoadingFriends: boolean;
  readonly setIsLoadingFriends: (value: boolean) => void;
  readonly presence: Readonly<Record<string, boolean>>;
  readonly setPresence: (value: Readonly<Record<string, boolean>>) => void;
}

export interface UseContactsOptions extends ChatDomainDeps {
  readonly sdk: {
    listFriends: () => Promise<FriendItem[]>;
    listFriendApplications: () => Promise<FriendApplication[]>;
    listBlacklist: () => Promise<UserProfile[]>;
    applyFriend: (req: { target_id: string; greeting?: string }) => Promise<void>;
    auditFriend: (req: { from_user_id: string; action: "accept" | "reject" }) => Promise<void>;
    deleteFriend: (userId: string) => Promise<void>;
    updateFriendRemark: (userId: string, req: { remark: string }) => Promise<void>;
    addBlacklist: (req: { target_id: string }) => Promise<void>;
    removeBlacklist: (userId: string) => Promise<void>;
    searchUsers: (keyword: string) => Promise<UserProfile[]>;
    getPresence: (req: { user_ids: string[] }) => Promise<PresenceResponse>;
  };
  readonly state: ContactsState;
}

/** Friends, pending applications, blacklist, user search and presence. */
export function useContacts(options: UseContactsOptions) {
  const { sdk, setError } = options;
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const inFlightRef = useRef(false);
  const queuedRefreshRef = useRef(false);

  const refreshFriends = useCallback(async () => {
    if (!optionsRef.current.isAuthenticated()) {
      return;
    }
    if (inFlightRef.current) {
      queuedRefreshRef.current = true;
      return;
    }
    inFlightRef.current = true;
    optionsRef.current.state.setIsLoadingFriends(true);
    try {
      // The three lists are independent, so one round trip covers them all.
      const [friendList, applications, blocked] = await Promise.all([
        sdk.listFriends(),
        sdk.listFriendApplications(),
        sdk.listBlacklist(),
      ]);
      optionsRef.current.state.setFriends(friendList);
      optionsRef.current.state.setFriendApplications(applications);
      optionsRef.current.state.setBlacklist(blocked);

      if (friendList.length > 0) {
        try {
          const presResp = await sdk.getPresence({ user_ids: friendList.map((f) => f.user_id) });
          optionsRef.current.state.setPresence(presResp.presence ?? {});
        } catch {
          // presence query is best-effort
        }
      }
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to load friends");
    } finally {
      optionsRef.current.state.setIsLoadingFriends(false);
      inFlightRef.current = false;
      if (queuedRefreshRef.current) {
        queuedRefreshRef.current = false;
        void refreshFriends();
      }
    }
  }, [sdk, setError]);

  const applyFriend = useCallback(
    async (targetId: string, greeting?: string) => {
      try {
        await sdk.applyFriend({ target_id: targetId, greeting });
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to send friend request");
        throw err;
      }
    },
    [sdk, setError],
  );

  const auditFriend = useCallback(
    async (fromUserId: string, action: "accept" | "reject") => {
      try {
        await sdk.auditFriend({ from_user_id: fromUserId, action });
        await refreshFriends();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to audit friend request");
        throw err;
      }
    },
    [sdk, refreshFriends, setError],
  );

  const deleteFriend = useCallback(
    async (userId: string) => {
      try {
        await sdk.deleteFriend(userId);
        await refreshFriends();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to delete friend");
        throw err;
      }
    },
    [sdk, refreshFriends, setError],
  );

  const updateFriendRemark = useCallback(
    async (userId: string, remark: string) => {
      try {
        await sdk.updateFriendRemark(userId, { remark });
        await refreshFriends();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update remark");
        throw err;
      }
    },
    [sdk, refreshFriends, setError],
  );

  const addBlacklist = useCallback(
    async (targetId: string) => {
      try {
        await sdk.addBlacklist({ target_id: targetId });
        await refreshFriends();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to block user");
        throw err;
      }
    },
    [sdk, refreshFriends, setError],
  );

  const removeBlacklist = useCallback(
    async (userId: string) => {
      try {
        await sdk.removeBlacklist(userId);
        await refreshFriends();
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to unblock user");
        throw err;
      }
    },
    [sdk, refreshFriends, setError],
  );

  const searchUsers = useCallback(
    async (keyword: string) => {
      try {
        return await sdk.searchUsers(keyword);
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to search users");
        throw err;
      }
    },
    [sdk, setError],
  );

  const queryPresence = useCallback(
    async (userIds: readonly string[]) => {
      if (userIds.length === 0) {
        optionsRef.current.state.setPresence({});
        return;
      }
      try {
        const resp = await sdk.getPresence({ user_ids: [...userIds] });
        optionsRef.current.state.setPresence(resp.presence ?? {});
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to query presence");
      }
    },
    [sdk, setError],
  );

  return {
    refreshFriends,
    applyFriend,
    auditFriend,
    deleteFriend,
    updateFriendRemark,
    addBlacklist,
    removeBlacklist,
    searchUsers,
    queryPresence,
  };
}
