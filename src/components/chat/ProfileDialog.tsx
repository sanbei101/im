import { KeyRound, Loader2, LogOut, Smartphone, UserCog } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useChat } from "@/context/ChatContext";

interface ProfileDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

/** Self profile editing, password change and device-token registration. */
export function ProfileDialog({ open, onOpenChange }: ProfileDialogProps) {
  const {
    profile,
    isLoadingProfile,
    currentUser,
    refreshProfile,
    updateProfile,
    updatePassword,
    saveDeviceToken,
    logout,
  } = useChat();

  const [tab, setTab] = useState<"profile" | "password" | "device">("profile");

  const [nickname, setNickname] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [isSavingProfile, setIsSavingProfile] = useState(false);

  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [isSavingPassword, setIsSavingPassword] = useState(false);

  const [deviceToken, setDeviceToken] = useState("");
  const [devicePlatform, setDevicePlatform] = useState("web");
  const [isSavingDevice, setIsSavingDevice] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setNotice(null);
      void refreshProfile();
    }
  }, [open, refreshProfile]);

  useEffect(() => {
    if (profile) {
      setNickname(profile.nickname);
      setAvatarUrl(profile.avatar_url);
    }
  }, [profile]);

  const handleSaveProfile = async () => {
    setIsSavingProfile(true);
    setNotice(null);
    try {
      await updateProfile({ nickname: nickname.trim(), avatar_url: avatarUrl.trim() });
      setNotice("Profile updated.");
    } catch {
      // Error banner is set by the context action.
    } finally {
      setIsSavingProfile(false);
    }
  };

  const handleSavePassword = async () => {
    setIsSavingPassword(true);
    setNotice(null);
    try {
      await updatePassword(oldPassword, newPassword);
      setNotice("Password updated.");
      setOldPassword("");
      setNewPassword("");
    } catch {
      // Error banner is set by the context action.
    } finally {
      setIsSavingPassword(false);
    }
  };

  const handleSaveDevice = async () => {
    setIsSavingDevice(true);
    setNotice(null);
    try {
      await saveDeviceToken(deviceToken.trim(), devicePlatform.trim());
      setNotice("Device token saved.");
      setDeviceToken("");
    } catch {
      // Error banner is set by the context action.
    } finally {
      setIsSavingDevice(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserCog className="size-4" />
            Account
          </DialogTitle>
          <DialogDescription className="text-xs">
            {currentUser ? `Signed in as @${currentUser.username}` : "Not signed in"}
          </DialogDescription>
        </DialogHeader>

        <Tabs value={tab} onValueChange={(next) => setTab(next as typeof tab)}>
          <TabsList className="w-full">
            <TabsTrigger value="profile" className="flex-1">
              Profile
            </TabsTrigger>
            <TabsTrigger value="password" className="flex-1">
              Password
            </TabsTrigger>
            <TabsTrigger value="device" className="flex-1">
              Device
            </TabsTrigger>
          </TabsList>

          <TabsContent value="profile" className="flex flex-col gap-3 pt-2">
            <div className="flex flex-col gap-1.5">
              <label
                className="text-muted-foreground text-xs font-medium"
                htmlFor="profile-username"
              >
                Username
              </label>
              <Input id="profile-username" value={currentUser?.username ?? ""} disabled />
            </div>
            <div className="flex flex-col gap-1.5">
              <label
                className="text-muted-foreground text-xs font-medium"
                htmlFor="profile-nickname"
              >
                Nickname
              </label>
              <Input
                id="profile-nickname"
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                placeholder={isLoadingProfile ? "Loading…" : "Display name"}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium" htmlFor="profile-avatar">
                Avatar URL
              </label>
              <Input
                id="profile-avatar"
                value={avatarUrl}
                onChange={(e) => setAvatarUrl(e.target.value)}
                placeholder="https://…"
              />
            </div>
            <Button onClick={() => void handleSaveProfile()} disabled={isSavingProfile}>
              {isSavingProfile && <Loader2 className="size-3.5 animate-spin" />}
              Save profile
            </Button>
          </TabsContent>

          <TabsContent value="password" className="flex flex-col gap-3 pt-2">
            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium" htmlFor="old-password">
                Current password
              </label>
              <Input
                id="old-password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium" htmlFor="new-password">
                New password
              </label>
              <Input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
            <Button
              onClick={() => void handleSavePassword()}
              disabled={isSavingPassword || oldPassword === "" || newPassword === ""}
            >
              {isSavingPassword ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <KeyRound className="size-3.5" />
              )}
              Change password
            </Button>
          </TabsContent>

          <TabsContent value="device" className="flex flex-col gap-3 pt-2">
            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium" htmlFor="device-token">
                Push token
              </label>
              <Input
                id="device-token"
                value={deviceToken}
                onChange={(e) => setDeviceToken(e.target.value)}
                placeholder="FCM/APNs token"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label
                className="text-muted-foreground text-xs font-medium"
                htmlFor="device-platform"
              >
                Platform
              </label>
              <Input
                id="device-platform"
                value={devicePlatform}
                onChange={(e) => setDevicePlatform(e.target.value)}
              />
            </div>
            <Button
              onClick={() => void handleSaveDevice()}
              disabled={isSavingDevice || deviceToken.trim() === ""}
            >
              {isSavingDevice ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Smartphone className="size-3.5" />
              )}
              Register device
            </Button>
          </TabsContent>
        </Tabs>

        {notice && <p className="text-success text-xs">{notice}</p>}

        <DialogFooter>
          <Button
            variant="destructive"
            onClick={() => {
              onOpenChange(false);
              logout();
            }}
          >
            <LogOut className="size-3.5" />
            Sign out
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
