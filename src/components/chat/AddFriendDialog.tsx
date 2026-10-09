import type { UserProfile } from "go-chat-sdk";
import { AlertCircle, Check, Loader2, Search, UserCheck, UserPlus, X } from "lucide-react";
import { useCallback, useEffect, useState, type ReactNode } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { useChat } from "@/context/ChatContext";
import { getInitials, isErrorWithMessage } from "@/types/chat";

interface AddFriendDialogProps {
  readonly trigger?: ReactNode;
  readonly open?: boolean;
  readonly onOpenChange?: (open: boolean) => void;
  readonly initialTargetUser?: UserProfile | null;
}

export function AddFriendDialog({
  trigger,
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  initialTargetUser = null,
}: AddFriendDialogProps) {
  const { applyFriend, searchUsers, friends, blacklist, currentUser, profile, refreshFriends } =
    useChat();

  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const isControlled = controlledOpen !== undefined;
  const isOpen = isControlled ? controlledOpen : uncontrolledOpen;

  const [query, setQuery] = useState("");
  const [selectedUser, setSelectedUser] = useState<UserProfile | null>(initialTargetUser);
  const [greeting, setGreeting] = useState("");
  const [isSearching, setIsSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<readonly UserProfile[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (isControlled) {
        setControlledOpen?.(nextOpen);
      } else {
        setUncontrolledOpen(nextOpen);
      }
      if (!nextOpen) {
        // Reset state on close
        setQuery("");
        setSelectedUser(initialTargetUser);
        setErrorMsg(null);
        setSuccessMsg(null);
        setSearchResults([]);
        setIsSearching(false);
      }
    },
    [isControlled, setControlledOpen, initialTargetUser],
  );

  // Auto-fill default greeting when dialog opens
  useEffect(() => {
    if (isOpen) {
      const defaultName = profile?.nickname || currentUser?.username || "";
      setGreeting(defaultName ? `我是 ${defaultName}` : "你好，想添加你为好友");
      if (initialTargetUser) {
        setSelectedUser(initialTargetUser);
      }
    }
  }, [isOpen, initialTargetUser, profile?.nickname, currentUser?.username]);

  // Debounced user search
  useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed || selectedUser) {
      setSearchResults([]);
      setIsSearching(false);
      return;
    }

    let cancelled = false;
    setIsSearching(true);
    const timer = setTimeout(() => {
      void (async () => {
        try {
          const results = await searchUsers(trimmed);
          if (!cancelled) {
            setSearchResults(results);
          }
        } catch {
          if (!cancelled) {
            setSearchResults([]);
          }
        } finally {
          if (!cancelled) {
            setIsSearching(false);
          }
        }
      })();
    }, 280);

    return () => {
      cancelled = true;
      clearTimeout(timer);
      setIsSearching(false);
    };
  }, [query, selectedUser, searchUsers]);

  const handleSelectUser = (user: UserProfile) => {
    setSelectedUser(user);
    setQuery("");
    setSearchResults([]);
    setErrorMsg(null);
  };

  const handleClearSelected = () => {
    setSelectedUser(null);
    setErrorMsg(null);
  };

  const handleSend = async () => {
    const targetId = selectedUser?.user_id || query.trim();
    if (!targetId) {
      setErrorMsg("请选择或输入要添加的用户 ID");
      return;
    }

    if (currentUser && targetId === currentUser.user_id) {
      setErrorMsg("不能添加自己为好友");
      return;
    }

    const isAlreadyFriend = friends.some((f) => f.user_id === targetId);
    if (isAlreadyFriend) {
      setErrorMsg("你们已经是好友了");
      return;
    }

    const isBlocked = blacklist.some((b) => b.user_id === targetId);
    if (isBlocked) {
      setErrorMsg("该用户在黑名单中，请先解除屏蔽");
      return;
    }

    setIsSubmitting(true);
    setErrorMsg(null);
    setSuccessMsg(null);

    try {
      await applyFriend(targetId, greeting.trim() || undefined);
      setSuccessMsg("好友申请已发送，等待对方通过");
      void refreshFriends();
      setTimeout(() => {
        handleOpenChange(false);
      }, 1500);
    } catch (err) {
      setErrorMsg(isErrorWithMessage(err) ? err.message : "发送好友申请失败");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      {trigger && <DialogTrigger>{trigger}</DialogTrigger>}
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlus className="size-4.5 text-[#0099ff]" />
            <span>添加好友</span>
          </DialogTitle>
          <DialogDescription>搜索用户名或输入用户 ID，向对方发送好友申请</DialogDescription>
        </DialogHeader>

        {errorMsg && (
          <div className="bg-destructive/10 text-destructive flex items-center gap-2 rounded-lg p-2.5 text-xs">
            <AlertCircle className="size-4 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        {successMsg && (
          <div className="flex items-center gap-2 rounded-lg bg-emerald-500/10 p-2.5 text-xs text-emerald-600 dark:text-emerald-400">
            <Check className="size-4 shrink-0" />
            <span>{successMsg}</span>
          </div>
        )}

        <div className="flex flex-col gap-3 py-1">
          {/* Target user selection */}
          <div className="flex flex-col gap-1.5">
            <label className="text-muted-foreground text-xs font-medium">目标用户</label>

            {selectedUser ? (
              <div className="bg-muted/40 border-border/80 flex items-center justify-between rounded-lg border p-2.5">
                <div className="flex min-w-0 items-center gap-2.5">
                  <Avatar className="ring-border/50 size-9 ring-1">
                    <AvatarFallback className="text-xs font-semibold">
                      {getInitials(selectedUser.nickname || selectedUser.username)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-semibold">
                      {selectedUser.nickname || selectedUser.username}
                    </p>
                    <p className="text-muted-foreground truncate text-[11px]">
                      @{selectedUser.username}
                    </p>
                    <p className="text-muted-foreground truncate font-mono text-[10px]">
                      ID: {selectedUser.user_id}
                    </p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={handleClearSelected}
                  title="更换用户"
                  className="text-muted-foreground hover:text-foreground size-6"
                >
                  <X className="size-3.5" />
                </Button>
              </div>
            ) : (
              <div className="relative">
                <Search className="text-muted-foreground absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2" />
                <Input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="输入用户名或用户 ID (UUID)"
                  className="pl-8 font-mono text-xs"
                  disabled={isSubmitting}
                />
                {isSearching && (
                  <Loader2 className="text-muted-foreground absolute top-1/2 right-2.5 size-3.5 -translate-y-1/2 animate-spin" />
                )}
              </div>
            )}
          </div>

          {/* Search candidates dropdown / list */}
          {!selectedUser && query.trim() !== "" && (
            <div className="bg-card border-border/70 overflow-hidden rounded-lg border">
              <ScrollArea className="max-h-44">
                <div className="flex flex-col p-1">
                  {!isSearching && searchResults.length === 0 ? (
                    <div className="text-muted-foreground py-4 text-center text-xs">
                      未找到匹配的用户
                    </div>
                  ) : (
                    searchResults.map((user) => {
                      const isSelf = currentUser && user.user_id === currentUser.user_id;
                      const isFriend = friends.some((f) => f.user_id === user.user_id);

                      return (
                        <div
                          key={user.user_id}
                          className="hover:bg-muted/70 flex items-center justify-between gap-2 rounded-md p-2 transition-colors"
                        >
                          <div className="flex min-w-0 flex-1 items-center gap-2">
                            <Avatar className="size-7 shrink-0">
                              <AvatarFallback className="text-[10px]">
                                {getInitials(user.nickname || user.username)}
                              </AvatarFallback>
                            </Avatar>
                            <div className="min-w-0 flex-1">
                              <p className="truncate text-xs font-medium">
                                {user.nickname || user.username}
                              </p>
                              <p className="text-muted-foreground truncate text-[10px]">
                                @{user.username} · ID: {user.user_id.slice(0, 8)}...
                              </p>
                            </div>
                          </div>

                          {isSelf ? (
                            <span className="text-muted-foreground px-2 text-[10px]">自己</span>
                          ) : isFriend ? (
                            <span className="flex items-center gap-1 px-2 text-[10px] text-emerald-500">
                              <UserCheck className="size-3" /> 已是好友
                            </span>
                          ) : (
                            <Button
                              size="xs"
                              variant="outline"
                              onClick={() => handleSelectUser(user)}
                              className="h-6 text-xs"
                            >
                              选择
                            </Button>
                          )}
                        </div>
                      );
                    })
                  )}
                </div>
              </ScrollArea>
            </div>
          )}

          {/* Greeting message input */}
          <div className="flex flex-col gap-1.5">
            <label className="text-muted-foreground text-xs font-medium">
              验证消息 / 打招呼（选填）
            </label>
            <Textarea
              value={greeting}
              onChange={(e) => setGreeting(e.target.value)}
              placeholder="例如：你好，我是... 想添加你为好友"
              className="h-20 resize-none text-xs"
              disabled={isSubmitting}
            />
          </div>
        </div>

        <DialogFooter className="mt-1">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => handleOpenChange(false)}
            disabled={isSubmitting}
          >
            取消
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={() => void handleSend()}
            disabled={isSubmitting || (!selectedUser && !query.trim())}
            className="gap-1.5"
          >
            {isSubmitting ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <UserPlus className="size-3.5" />
            )}
            <span>发送申请</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
