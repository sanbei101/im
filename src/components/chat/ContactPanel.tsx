import type { UserProfile } from "go-chat-sdk";
import {
  Ban,
  Bell,
  Check,
  Copy,
  MessageSquare,
  Search,
  Trash2,
  UserPlus,
  UserRound,
  Users,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useChat } from "@/context/ChatContext";
import { cn } from "@/lib/utils";
import { getInitials } from "@/types/chat";

import { AddFriendDialog } from "./AddFriendDialog";

interface ContactPanelProps {
  readonly onClose: () => void;
  readonly embedded?: boolean;
}

type ContactTab = "friends" | "requests" | "blacklist" | "discover";

/** Friends, friend requests, blacklist and user discovery in one drawer. */
export function ContactPanel({ onClose, embedded = false }: ContactPanelProps) {
  const {
    friends,
    friendApplications,
    blacklist,
    isLoadingFriends,
    refreshFriends,
    auditFriend,
    deleteFriend,
    updateFriendRemark,
    addBlacklist,
    removeBlacklist,
    createSingleRoom,
    searchUsers,
    presence,
    currentUser,
    profile,
  } = useChat();

  const [tab, setTab] = useState<ContactTab>("friends");
  const [keyword, setKeyword] = useState("");
  const [results, setResults] = useState<readonly UserProfile[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [busyUserId, setBusyUserId] = useState<string | null>(null);
  const [remarkDraft, setRemarkDraft] = useState("");
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [candidateForAdd, setCandidateForAdd] = useState<UserProfile | null>(null);
  const [isAddFriendOpen, setIsAddFriendOpen] = useState(false);

  const copyToClipboard = useCallback(async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedId(id);
      setTimeout(() => {
        setCopiedId((curr) => (curr === id ? null : curr));
      }, 2000);
    } catch {
      // Ignore clipboard write failure
    }
  }, []);

  useEffect(() => {
    void refreshFriends();
  }, [refreshFriends]);

  useEffect(() => {
    if (tab !== "discover") {
      return;
    }
    const trimmed = keyword.trim();
    if (trimmed === "") {
      setResults([]);
      return;
    }
    let cancelled = false;
    setIsSearching(true);
    const timer = setTimeout(() => {
      void (async () => {
        try {
          const found = await searchUsers(trimmed);
          if (!cancelled) {
            setResults(found);
            setSearchError(null);
          }
        } catch {
          if (!cancelled) {
            setResults([]);
            setSearchError("Search failed");
          }
        } finally {
          if (!cancelled) {
            setIsSearching(false);
          }
        }
      })();
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(timer);
      setIsSearching(false);
    };
  }, [keyword, tab, searchUsers]);

  const pendingRequests = useMemo(
    () => friendApplications.filter((a) => a.status === "pending"),
    [friendApplications],
  );

  const run = useCallback(async (userId: string, action: () => Promise<unknown>) => {
    setBusyUserId(userId);
    try {
      await action();
    } catch {
      // Error banner is already set by the context action.
    } finally {
      setBusyUserId(null);
    }
  }, []);

  return (
    <aside
      className={cn(
        "bg-background/50 flex h-full flex-col",
        embedded ? "w-full" : "w-80 shrink-0 border-l",
      )}
    >
      <div className="flex h-14 shrink-0 items-center justify-between border-b px-4">
        <span className="flex items-center gap-2 text-sm font-semibold">
          <Users className="size-4" />
          通讯录
        </span>
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="添加好友"
            title="添加好友"
            onClick={() => setIsAddFriendOpen(true)}
            className="text-muted-foreground hover:text-foreground"
          >
            <UserPlus className="size-4" />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label="Close contacts" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>
      </div>

      {/* Current User Info & ID Card */}
      {currentUser && (
        <div className="bg-muted/30 border-b px-3.5 py-2.5">
          <div className="flex items-center gap-2.5">
            <Avatar className="ring-border/50 size-9 shrink-0 ring-1">
              <AvatarFallback className="bg-primary/10 text-primary text-xs font-semibold">
                {getInitials(profile?.nickname || currentUser.username)}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="truncate text-xs font-semibold">
                  {profile?.nickname || currentUser.username}
                </span>
                <span className="text-muted-foreground truncate text-[11px]">
                  @{currentUser.username}
                </span>
              </div>
              <div className="mt-0.5 flex items-center gap-1">
                <span className="text-muted-foreground shrink-0 text-[10px] font-medium">ID:</span>
                <code
                  className="text-muted-foreground truncate font-mono text-[10px] select-all"
                  title={currentUser.user_id}
                >
                  {currentUser.user_id}
                </code>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground size-5 shrink-0"
                  onClick={() => void copyToClipboard(currentUser.user_id, "my-user-id")}
                  title="复制我的用户ID"
                  aria-label="Copy my user ID"
                >
                  {copiedId === "my-user-id" ? (
                    <Check className="size-3 text-emerald-500" />
                  ) : (
                    <Copy className="size-3" />
                  )}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      <Tabs
        value={tab}
        onValueChange={(next) => setTab(next as ContactTab)}
        className="flex min-h-0 flex-1 flex-col"
      >
        <div className="px-3 pt-3">
          <TabsList className="w-full">
            <TabsTrigger value="friends" className="flex-1">
              好友
            </TabsTrigger>
            <TabsTrigger value="requests" className="flex-1">
              申请
              {pendingRequests.length > 0 && (
                <Badge variant="destructive" className="ml-1">
                  {pendingRequests.length}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="blacklist" className="flex-1">
              黑名单
            </TabsTrigger>
            <TabsTrigger value="discover" className="flex-1">
              找人
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="friends" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {isLoadingFriends && friends.length === 0 ? (
                <LoadingRows />
              ) : friends.length === 0 ? (
                <EmptyState
                  icon={<UserRound className="size-5" />}
                  title="暂无好友"
                  hint="搜索并添加好友，开始即时沟通。"
                  action={
                    <Button size="sm" onClick={() => setIsAddFriendOpen(true)} className="gap-1.5">
                      <UserPlus className="size-3.5" />
                      <span>添加好友</span>
                    </Button>
                  }
                />
              ) : (
                friends.map((friend) => (
                  <div
                    key={friend.user_id}
                    className="group/friend hover:bg-muted/60 flex flex-col gap-1 rounded-md p-2"
                  >
                    <div className="flex items-center gap-2">
                      <span className="relative">
                        <Avatar className="size-8">
                          <AvatarFallback className="text-xs">
                            {(friend.remark || friend.nickname || friend.username).slice(0, 2)}
                          </AvatarFallback>
                        </Avatar>
                        <span
                          className={cn(
                            "border-background absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full border-2",
                            presence[friend.user_id]
                              ? "bg-emerald-500"
                              : "bg-zinc-400 dark:bg-zinc-600",
                          )}
                          title={presence[friend.user_id] ? "在线" : "离线"}
                        />
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <p className="truncate text-sm font-medium">
                            {friend.remark || friend.nickname || friend.username}
                          </p>
                          <span
                            className={cn(
                              "text-[10px]",
                              presence[friend.user_id]
                                ? "font-medium text-emerald-500"
                                : "text-muted-foreground",
                            )}
                          >
                            ({presence[friend.user_id] ? "在线" : "离线"})
                          </span>
                        </div>
                        <div className="text-muted-foreground flex items-center gap-1 text-xs">
                          <span className="truncate">@{friend.username}</span>
                          <span>·</span>
                          <span
                            className="max-w-[80px] truncate font-mono text-[10px]"
                            title={friend.user_id}
                          >
                            ID: {friend.user_id.slice(0, 8)}...
                          </span>
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            className="text-muted-foreground hover:text-foreground size-4 shrink-0 p-0"
                            onClick={() => void copyToClipboard(friend.user_id, friend.user_id)}
                            title={`复制 ${friend.username} 的用户ID`}
                            aria-label={`Copy user ID for ${friend.username}`}
                          >
                            {copiedId === friend.user_id ? (
                              <Check className="size-2.5 text-emerald-500" />
                            ) : (
                              <Copy className="size-2.5" />
                            )}
                          </Button>
                        </div>
                      </div>
                      <div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover/friend:opacity-100">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label={`Message ${friend.username}`}
                          onClick={() =>
                            void run(friend.user_id, async () => {
                              await createSingleRoom(friend.user_id);
                            })
                          }
                        >
                          <MessageSquare className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label={`Block ${friend.username}`}
                          disabled={busyUserId === friend.user_id}
                          onClick={() =>
                            void run(friend.user_id, () => addBlacklist(friend.user_id))
                          }
                        >
                          <Ban className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label={`Delete ${friend.username}`}
                          disabled={busyUserId === friend.user_id}
                          onClick={() =>
                            void run(friend.user_id, () => deleteFriend(friend.user_id))
                          }
                        >
                          <Trash2 className="size-3.5" />
                        </Button>
                      </div>
                    </div>
                    <form
                      className="flex items-center gap-1"
                      onSubmit={(e) => {
                        e.preventDefault();
                        void run(friend.user_id, () =>
                          updateFriendRemark(friend.user_id, remarkDraft.trim()),
                        );
                        setRemarkDraft("");
                      }}
                    >
                      <Input
                        value={remarkDraft}
                        onChange={(e) => setRemarkDraft(e.target.value)}
                        placeholder="Set remark"
                        aria-label={`Remark for ${friend.username}`}
                        className="h-7 text-xs"
                      />
                      <Button
                        type="submit"
                        size="icon-sm"
                        variant="outline"
                        aria-label="Save remark"
                      >
                        <Check className="size-3.5" />
                      </Button>
                    </form>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="requests" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {isLoadingFriends && pendingRequests.length === 0 ? (
                <LoadingRows />
              ) : pendingRequests.length === 0 ? (
                <EmptyState
                  icon={<Bell className="size-5" />}
                  title="暂无待处理的好友申请"
                  hint="收到的好友申请将显示在这里。"
                />
              ) : (
                pendingRequests.map((request) => (
                  <div
                    key={request.from_user_id}
                    className="hover:bg-muted/60 flex flex-col gap-1.5 rounded-md p-2"
                  >
                    <div className="flex items-center justify-between gap-1">
                      <div className="flex min-w-0 items-center gap-1 font-mono text-xs">
                        <span className="text-muted-foreground text-[10px]">ID:</span>
                        <span className="truncate text-xs font-medium" title={request.from_user_id}>
                          {request.from_user_id}
                        </span>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        className="text-muted-foreground hover:text-foreground size-5 shrink-0"
                        onClick={() =>
                          void copyToClipboard(request.from_user_id, request.from_user_id)
                        }
                        title="复制用户ID"
                        aria-label="Copy applicant user ID"
                      >
                        {copiedId === request.from_user_id ? (
                          <Check className="size-3 text-emerald-500" />
                        ) : (
                          <Copy className="size-3" />
                        )}
                      </Button>
                    </div>
                    <p className="text-muted-foreground text-xs">
                      {request.greeting || "请求添加你为好友"}
                    </p>
                    <div className="flex items-center gap-1">
                      <Button
                        size="xs"
                        disabled={busyUserId === request.from_user_id}
                        onClick={() =>
                          void run(request.from_user_id, () =>
                            auditFriend(request.from_user_id, "accept"),
                          )
                        }
                      >
                        <Check className="size-3" />
                        通过
                      </Button>
                      <Button
                        size="xs"
                        variant="outline"
                        disabled={busyUserId === request.from_user_id}
                        onClick={() =>
                          void run(request.from_user_id, () =>
                            auditFriend(request.from_user_id, "reject"),
                          )
                        }
                      >
                        <X className="size-3" />
                        拒绝
                      </Button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="blacklist" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {isLoadingFriends && blacklist.length === 0 ? (
                <LoadingRows />
              ) : blacklist.length === 0 ? (
                <EmptyState
                  icon={<Ban className="size-5" />}
                  title="黑名单为空"
                  hint="可在好友操作中屏蔽不良用户。"
                />
              ) : (
                blacklist.map((user) => (
                  <div
                    key={user.user_id}
                    className="hover:bg-muted/60 flex items-center gap-2 rounded-md p-2"
                  >
                    <Avatar className="size-8">
                      <AvatarFallback className="text-xs">
                        {(user.nickname || user.username).slice(0, 2)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {user.nickname || user.username}
                      </p>
                      <div className="text-muted-foreground flex items-center gap-1 text-xs">
                        <span className="truncate">@{user.username}</span>
                        <span>·</span>
                        <span
                          className="max-w-[80px] truncate font-mono text-[10px]"
                          title={user.user_id}
                        >
                          ID: {user.user_id.slice(0, 8)}...
                        </span>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          className="text-muted-foreground hover:text-foreground size-4 shrink-0 p-0"
                          onClick={() => void copyToClipboard(user.user_id, user.user_id)}
                          title="复制用户ID"
                          aria-label={`Copy user ID for ${user.username}`}
                        >
                          {copiedId === user.user_id ? (
                            <Check className="size-2.5 text-emerald-500" />
                          ) : (
                            <Copy className="size-2.5" />
                          )}
                        </Button>
                      </div>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={busyUserId === user.user_id}
                      onClick={() => void run(user.user_id, () => removeBlacklist(user.user_id))}
                    >
                      解除屏蔽
                    </Button>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="discover" className="min-h-0 flex-1">
          <div className="flex h-full flex-col">
            <div className="p-3 pb-2">
              <div className="relative">
                <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                <Input
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  placeholder="搜索用户名或用户 ID"
                  aria-label="搜索用户"
                  className="pl-8 text-xs"
                />
              </div>
            </div>
            <ScrollArea className="min-h-0 flex-1">
              <div className="flex flex-col gap-1 px-3 pb-3">
                {keyword.trim() !== "" &&
                  !isSearching &&
                  results.length === 0 &&
                  (searchError ? (
                    <EmptyState
                      icon={<X className="size-5" />}
                      title="搜索失败"
                      hint="请稍后重试。"
                    />
                  ) : (
                    <EmptyState
                      icon={<Search className="size-5" />}
                      title="未找到匹配用户"
                      hint="尝试搜索其他用户名或完整用户 ID。"
                    />
                  ))}
                {results.map((user) => {
                  const isSelf = currentUser && user.user_id === currentUser.user_id;
                  const isFriend = friends.some((f) => f.user_id === user.user_id);

                  return (
                    <div
                      key={user.user_id}
                      className="hover:bg-muted/60 flex items-center gap-2 rounded-md p-2"
                    >
                      <Avatar className="size-8">
                        <AvatarFallback className="text-xs">
                          {(user.nickname || user.username).slice(0, 2)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">
                          {user.nickname || user.username}
                        </p>
                        <div className="text-muted-foreground flex items-center gap-1 text-xs">
                          <span className="truncate">@{user.username}</span>
                          <span>·</span>
                          <span
                            className="max-w-[80px] truncate font-mono text-[10px]"
                            title={user.user_id}
                          >
                            ID: {user.user_id.slice(0, 8)}...
                          </span>
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            className="text-muted-foreground hover:text-foreground size-4 shrink-0 p-0"
                            onClick={() => void copyToClipboard(user.user_id, user.user_id)}
                            title="复制用户ID"
                            aria-label={`Copy user ID for ${user.username}`}
                          >
                            {copiedId === user.user_id ? (
                              <Check className="size-2.5 text-emerald-500" />
                            ) : (
                              <Copy className="size-2.5" />
                            )}
                          </Button>
                        </div>
                      </div>

                      {isSelf ? (
                        <span className="text-muted-foreground px-2 text-xs">自己</span>
                      ) : isFriend ? (
                        <span className="px-2 text-xs text-emerald-500">已是好友</span>
                      ) : (
                        <Button
                          size="sm"
                          onClick={() => setCandidateForAdd(user)}
                          className="h-7 gap-1 text-xs"
                        >
                          <UserPlus className="size-3" />
                          添加
                        </Button>
                      )}
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </div>
        </TabsContent>
      </Tabs>

      {/* Add Friend Dialogs */}
      <AddFriendDialog open={isAddFriendOpen} onOpenChange={setIsAddFriendOpen} />
      <AddFriendDialog
        open={candidateForAdd !== null}
        onOpenChange={(open) => !open && setCandidateForAdd(null)}
        initialTargetUser={candidateForAdd}
      />
    </aside>
  );
}

interface EmptyStateProps {
  readonly icon: ReactNode;
  readonly title: string;
  readonly hint: string;
  readonly action?: ReactNode;
}

function EmptyState({ icon, title, hint, action }: EmptyStateProps) {
  return (
    <div className="text-muted-foreground flex flex-col items-center gap-2 px-4 py-10 text-center">
      <div className="bg-muted flex size-10 items-center justify-center rounded-full">{icon}</div>
      <p className="text-foreground text-sm font-medium">{title}</p>
      <p className="text-xs">{hint}</p>
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

function LoadingRows() {
  return (
    <div className="flex flex-col gap-2 p-2">
      {[0, 1, 2].map((i) => (
        <div key={i} className="bg-muted/50 h-12 animate-pulse rounded-md" />
      ))}
    </div>
  );
}
