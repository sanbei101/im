import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  Ban,
  Bell,
  Check,
  MessageSquare,
  Search,
  Trash2,
  UserPlus,
  UserRound,
  Users,
  X,
} from "lucide-react";
import type { UserProfile } from "go-chat-sdk";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useChat } from "@/context/ChatContext";

interface ContactPanelProps {
  readonly onClose: () => void;
}

type ContactTab = "friends" | "requests" | "blacklist" | "discover";

/** Friends, friend requests, blacklist and user discovery in one drawer. */
export function ContactPanel({ onClose }: ContactPanelProps) {
  const {
    friends,
    friendApplications,
    blacklist,
    isLoadingFriends,
    refreshFriends,
    applyFriend,
    auditFriend,
    deleteFriend,
    updateFriendRemark,
    addBlacklist,
    removeBlacklist,
    createSingleRoom,
    searchUsers,
    presence,
  } = useChat();

  const [tab, setTab] = useState<ContactTab>("friends");
  const [keyword, setKeyword] = useState("");
  const [results, setResults] = useState<readonly UserProfile[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [busyUserId, setBusyUserId] = useState<string | null>(null);
  const [remarkDraft, setRemarkDraft] = useState("");

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

  const run = useCallback(
    async (userId: string, action: () => Promise<unknown>) => {
      setBusyUserId(userId);
      try {
        await action();
      } catch {
        // Error banner is already set by the context action.
      } finally {
        setBusyUserId(null);
      }
    },
    [],
  );

  return (
    <aside className="bg-background/50 flex h-full w-80 shrink-0 flex-col border-l">
      <div className="flex h-14 shrink-0 items-center justify-between border-b px-4">
        <span className="flex items-center gap-2 text-sm font-semibold">
          <Users className="size-4" />
          Contacts
        </span>
        <Button variant="ghost" size="icon-sm" aria-label="Close contacts" onClick={onClose}>
          <X className="size-4" />
        </Button>
      </div>

      <Tabs
        value={tab}
        onValueChange={(next) => setTab(next as ContactTab)}
        className="flex min-h-0 flex-1 flex-col"
      >
        <div className="px-3 pt-3">
          <TabsList className="w-full">
            <TabsTrigger value="friends" className="flex-1">
              Friends
            </TabsTrigger>
            <TabsTrigger value="requests" className="flex-1">
              Requests
              {pendingRequests.length > 0 && (
                <Badge variant="destructive" className="ml-1">
                  {pendingRequests.length}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="blacklist" className="flex-1">
              Blocked
            </TabsTrigger>
            <TabsTrigger value="discover" className="flex-1">
              Find
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="friends" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {friends.length === 0 && (
                <EmptyState
                  icon={<UserRound className="size-5" />}
                  title="No friends yet"
                  hint="Use “Find” to add people."
                />
              )}
              {friends.map((friend) => (
                <div key={friend.user_id} className="group/friend flex flex-col gap-1 rounded-md p-2 hover:bg-muted/60">
                  <div className="flex items-center gap-2">
                    <span className="relative">
                      <Avatar className="size-8">
                        <AvatarFallback className="text-xs">
                          {(friend.remark || friend.nickname || friend.username).slice(0, 2)}
                        </AvatarFallback>
                      </Avatar>
                      {presence[friend.user_id] && (
                        <span className="absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full border-2 border-background bg-success" />
                      )}
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {friend.remark || friend.nickname || friend.username}
                      </p>
                      <p className="text-muted-foreground truncate text-xs">@{friend.username}</p>
                    </div>
                    <div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover/friend:opacity-100">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Message ${friend.username}`}
                        onClick={() => void run(friend.user_id, async () => {
                          await createSingleRoom(friend.user_id);
                        })}
                      >
                        <MessageSquare className="size-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Block ${friend.username}`}
                        disabled={busyUserId === friend.user_id}
                        onClick={() => void run(friend.user_id, () => addBlacklist(friend.user_id))}
                      >
                        <Ban className="size-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Delete ${friend.username}`}
                        disabled={busyUserId === friend.user_id}
                        onClick={() => void run(friend.user_id, () => deleteFriend(friend.user_id))}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </div>
                  </div>
                  <form
                    className="flex items-center gap-1"
                    onSubmit={(e) => {
                      e.preventDefault();
                      void run(friend.user_id, () => updateFriendRemark(friend.user_id, remarkDraft.trim()));
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
                    <Button type="submit" size="icon-sm" variant="outline" aria-label="Save remark">
                      <Check className="size-3.5" />
                    </Button>
                  </form>
                </div>
              ))}
              {isLoadingFriends && friends.length === 0 && <LoadingRows />}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="requests" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {pendingRequests.length === 0 && (
                <EmptyState
                  icon={<Bell className="size-5" />}
                  title="No pending requests"
                  hint="Friend requests show up here."
                />
              )}
              {pendingRequests.map((request) => (
                <div key={request.from_user_id} className="flex flex-col gap-1 rounded-md p-2 hover:bg-muted/60">
                  <p className="truncate text-sm font-medium">{request.from_user_id.slice(0, 12)}</p>
                  <p className="text-muted-foreground text-xs">{request.greeting || "Wants to add you"}</p>
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
                      Accept
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
                      Reject
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="blacklist" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {blacklist.length === 0 && (
                <EmptyState
                  icon={<Ban className="size-5" />}
                  title="No blocked users"
                  hint="Block from a friend row."
                />
              )}
              {blacklist.map((user) => (
                <div key={user.user_id} className="flex items-center gap-2 rounded-md p-2 hover:bg-muted/60">
                  <Avatar className="size-8">
                    <AvatarFallback className="text-xs">
                      {(user.nickname || user.username).slice(0, 2)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{user.nickname || user.username}</p>
                    <p className="text-muted-foreground truncate text-xs">@{user.username}</p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busyUserId === user.user_id}
                    onClick={() => void run(user.user_id, () => removeBlacklist(user.user_id))}
                  >
                    Unblock
                  </Button>
                </div>
              ))}
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
                  placeholder="Search by username"
                  aria-label="Search users"
                  className="pl-8"
                />
              </div>
            </div>
            <ScrollArea className="min-h-0 flex-1">
              <div className="flex flex-col gap-1 px-3 pb-3">
                {keyword.trim() !== "" && !isSearching && results.length === 0 && (
                  searchError ? (
                    <EmptyState
                      icon={<X className="size-5" />}
                      title="Search failed"
                      hint="Please try again."
                    />
                  ) : (
                    <EmptyState
                      icon={<Search className="size-5" />}
                      title="No users found"
                      hint="Try a different username."
                    />
                  )
                )}
                {results.map((user) => (
                  <div key={user.user_id} className="flex items-center gap-2 rounded-md p-2 hover:bg-muted/60">
                    <Avatar className="size-8">
                      <AvatarFallback className="text-xs">
                        {(user.nickname || user.username).slice(0, 2)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{user.nickname || user.username}</p>
                      <p className="text-muted-foreground truncate text-xs">@{user.username}</p>
                    </div>
                    <Button
                      size="sm"
                      disabled={busyUserId === user.user_id}
                      onClick={() => void run(user.user_id, () => applyFriend(user.user_id))}
                    >
                      <UserPlus className="size-3" />
                      Add
                    </Button>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </div>
        </TabsContent>
      </Tabs>
    </aside>
  );
}

interface EmptyStateProps {
  readonly icon: ReactNode;
  readonly title: string;
  readonly hint: string;
}

function EmptyState({ icon, title, hint }: EmptyStateProps) {
  return (
    <div className="text-muted-foreground flex flex-col items-center gap-2 px-4 py-10 text-center">
      <div className="bg-muted flex size-10 items-center justify-center rounded-full">{icon}</div>
      <p className="text-foreground text-sm font-medium">{title}</p>
      <p className="text-xs">{hint}</p>
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
