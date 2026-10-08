import type { FriendApplication, FriendItem, PresenceResponse, UserProfile } from "go-chat-sdk";
import { useCallback } from "react";

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
  const { sdk, state, setError } = options;

  const refreshFriends = useCallback(async () => {
    if (!options.isAuthenticated()) {
      return;
    }
    state.setIsLoadingFriends(true);
    try {
      // The three lists are independent, so one round trip covers them all.
      const [friendList, applications, blocked] = await Promise.all([
        sdk.listFriends(),
        sdk.listFriendApplications(),
        sdk.listBlacklist(),
      ]);
      state.setFriends(friendList);
      state.setFriendApplications(applications);
      state.setBlacklist(blocked);
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to load friends");
    } finally {
      state.setIsLoadingFriends(false);
    }
  }, [sdk, state, setError, options.isAuthenticated]);

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
        state.setPresence({});
        return;
      }
      try {
        const resp = await sdk.getPresence({ user_ids: [...userIds] });
        state.setPresence(resp.presence ?? {});
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to query presence");
      }
    },
    [sdk, state, setError],
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
