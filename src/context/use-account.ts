import type { UserProfile, UserResponse } from "go-chat-sdk";
import { useCallback } from "react";

import { isErrorWithMessage } from "@/types/chat";

import type { ChatDomainDeps } from "./domain-deps";

export interface AccountState {
  readonly profile: UserProfile | null;
  readonly setProfile: (value: UserProfile | null) => void;
  readonly isLoadingProfile: boolean;
  readonly setIsLoadingProfile: (value: boolean) => void;
}

export interface UseAccountOptions extends ChatDomainDeps {
  readonly sdk: {
    getProfile: () => Promise<UserProfile>;
    updateProfile: (req: { nickname?: string; avatar_url?: string }) => Promise<UserProfile>;
    updatePassword: (req: { old_password: string; new_password: string }) => Promise<void>;
    saveDeviceToken: (req: { token: string; platform: string }) => Promise<void>;
    logout: () => Promise<void>;
    disconnect: () => void;
    clearAuth: () => void;
  };
  readonly state: AccountState;
  readonly currentUser: UserResponse | null;
  readonly setCurrentUser: (value: UserResponse | null) => void;
  /** Drops every cached slice so a fresh login starts clean. */
  readonly resetSessionState: () => void;
  readonly removeStoredUser: () => void;
}

/** Self profile, password, device token and sign-out. */
export function useAccount(options: UseAccountOptions) {
  const { sdk, state, setError, currentUser } = options;

  const refreshProfile = useCallback(async () => {
    if (!options.isAuthenticated()) {
      return;
    }
    state.setIsLoadingProfile(true);
    try {
      const me = await sdk.getProfile();
      state.setProfile(me);
    } catch (err) {
      setError(isErrorWithMessage(err) ? err.message : "Failed to load profile");
    } finally {
      state.setIsLoadingProfile(false);
    }
  }, [sdk, state, setError, options.isAuthenticated]);

  const updateProfile = useCallback(
    async (req: { nickname?: string; avatar_url?: string }) => {
      try {
        const me = await sdk.updateProfile(req);
        state.setProfile(me);
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update profile");
        throw err;
      }
    },
    [sdk, state, setError],
  );

  const updatePassword = useCallback(
    async (oldPassword: string, newPassword: string) => {
      try {
        await sdk.updatePassword({ old_password: oldPassword, new_password: newPassword });
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to update password");
        throw err;
      }
    },
    [sdk, setError],
  );

  const saveDeviceToken = useCallback(
    async (token: string, platform: string) => {
      try {
        await sdk.saveDeviceToken({ token, platform });
      } catch (err) {
        setError(isErrorWithMessage(err) ? err.message : "Failed to save device token");
      }
    },
    [sdk, setError],
  );

  // logout() on the SDK already tears down the socket, but a failed request
  // must still clear local state, hence the finally block.
  const logout = useCallback(() => {
    void sdk
      .logout()
      .catch(() => undefined)
      .finally(() => {
        sdk.disconnect();
        sdk.clearAuth();
        options.setCurrentUser(null);
        options.resetSessionState();
        options.removeStoredUser();
      });
  }, [sdk, options]);

  return {
    refreshProfile,
    updateProfile,
    updatePassword,
    saveDeviceToken,
    logout,
    currentUser,
  };
}
