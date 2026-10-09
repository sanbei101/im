import { ConnectionState, type ConversationInfo } from "go-chat-sdk";
import {
  MessageSquare,
  Search,
  Users,
  LogOut,
  RefreshCw,
  PanelLeftClose,
  PanelLeftOpen,
  MessageSquarePlus,
  BellOff,
  Pin,
  Settings,
  UserPlus,
} from "lucide-react";
import { useMemo, useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { cn } from "@/lib/utils";
import {
  formatRelativeTime,
  getInitials,
  getMessagePreviewText,
  sortConversations,
} from "@/types/chat";

import { AddFriendDialog } from "./AddFriendDialog";
import { ContactPanel } from "./ContactPanel";
import { CreateRoomDialog } from "./CreateRoomDialog";
import { SearchPanel } from "./SearchPanel";
import { SettingsDialog } from "./SettingsDialog";

export type NavTab = "chats" | "contacts" | "search";

interface ChatSidebarProps {
  readonly isCollapsed: boolean;
  readonly onToggleCollapse: () => void;
  readonly onShowProfile: () => void;
}

export function ChatSidebar({ isCollapsed, onToggleCollapse, onShowProfile }: ChatSidebarProps) {
  const {
    rooms,
    activeRoomId,
    selectRoom,
    currentUser,
    logout,
    refreshRooms,
    isLoadingRooms,
    conversations,
    refreshConversations,
    friendApplications,
    connectionState,
    connect,
    typingRooms,
  } = useChat();

  const [navTab, setNavTab] = useState<NavTab>("chats");
  const [searchQuery, setSearchQuery] = useState("");

  // Merge full conversations with any rooms not yet in conversations, sorted by activity
  const allConversations: readonly ConversationInfo[] = useMemo(() => {
    const list = [...conversations];
    const knownRoomIds = new Set(list.map((c) => c.room.room_id));
    for (const r of rooms) {
      if (!knownRoomIds.has(r.room_id)) {
        list.push({
          room: {
            room_id: r.room_id,
            chat_type: r.chat_type,
            name: r.name ?? "",
            avatar_url: r.avatar_url ?? "",
            notice: r.notice ?? "",
            last_seq: 0,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
          member: {
            room_id: r.room_id,
            user_id: currentUser?.user_id ?? "",
            role: "member",
            is_hidden: false,
            is_muted: false,
            is_pinned: false,
          },
          unread_count: 0,
        });
      }
    }
    return list.sort(sortConversations);
  }, [conversations, rooms, currentUser]);

  // Total unread count for the navigation icon badge
  const totalUnread = useMemo(
    () => allConversations.reduce((sum, c) => sum + (c.unread_count || 0), 0),
    [allConversations],
  );

  const pendingRequests = friendApplications.filter((a) => a.status === "pending").length;

  const filteredConversations = useMemo(() => {
    if (!searchQuery.trim()) {
      return allConversations;
    }
    const q = searchQuery.toLowerCase();
    return allConversations.filter((c) => {
      const name = c.room.name || (c.room.chat_type === "group" ? "群聊" : "私聊");
      const preview = c.last_message ? getMessagePreviewText(c.last_message) : "";
      return (
        name.toLowerCase().includes(q) ||
        c.room.room_id.toLowerCase().includes(q) ||
        preview.toLowerCase().includes(q)
      );
    });
  }, [allConversations, searchQuery]);

  const isConnected = connectionState === ConnectionState.Connected;

  return (
    <div className="flex h-full shrink-0 select-none">
      {/* 1. Left Dock Navigation Rail (Classic QQ style, ~60px) */}
      <aside className="border-border/70 z-10 flex h-full w-15 shrink-0 flex-col items-center justify-between border-r bg-[#f0f2f5] py-3.5 dark:bg-[#16171a]">
        {/* Top: User Avatar with online indicator */}
        <div className="flex flex-col items-center gap-4">
          <Tooltip>
            <TooltipTrigger>
              <button
                type="button"
                onClick={onShowProfile}
                className="relative rounded-full transition-transform hover:scale-105 focus-visible:outline-none"
              >
                <Avatar className="ring-background size-10 shadow-xs ring-2">
                  <AvatarFallback className="bg-primary/10 text-primary text-xs font-semibold">
                    {currentUser ? getInitials(currentUser.username) : "QQ"}
                  </AvatarFallback>
                </Avatar>
                {/* Online presence badge */}
                <span
                  className={cn(
                    "absolute -bottom-0.5 -right-0.5 size-3 rounded-full border-2 border-background",
                    isConnected ? "bg-emerald-500" : "bg-zinc-400",
                  )}
                />
              </button>
            </TooltipTrigger>
            <TooltipContent side="right">
              <span className="font-medium">{currentUser?.username || "个人资料"}</span>
              {currentUser && (
                <span className="text-muted-foreground block font-mono text-[10px]">
                  ID: {currentUser.user_id}
                </span>
              )}
              <span className="text-muted-foreground block text-[10px]">
                {isConnected ? "在线" : "离线"} · 点击查看设置
              </span>
            </TooltipContent>
          </Tooltip>

          {/* Navigation Tab Icons */}
          <div className="mt-1 flex flex-col items-center gap-2">
            {/* Chats tab */}
            <Tooltip>
              <TooltipTrigger>
                <button
                  type="button"
                  onClick={() => {
                    setNavTab("chats");
                    if (isCollapsed) {
                      onToggleCollapse();
                    }
                  }}
                  className={cn(
                    "relative flex size-10 items-center justify-center rounded-xl transition-all",
                    navTab === "chats"
                      ? "bg-[#0099ff] text-white shadow-xs"
                      : "text-muted-foreground hover:bg-muted/80 hover:text-foreground",
                  )}
                >
                  <MessageSquare className="size-5" />
                  {totalUnread > 0 && (
                    <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-bold text-white shadow-2xs">
                      {totalUnread > 99 ? "99+" : totalUnread}
                    </span>
                  )}
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">消息列表</TooltipContent>
            </Tooltip>

            {/* Contacts tab */}
            <Tooltip>
              <TooltipTrigger>
                <button
                  type="button"
                  onClick={() => {
                    setNavTab("contacts");
                    if (isCollapsed) {
                      onToggleCollapse();
                    }
                  }}
                  className={cn(
                    "relative flex size-10 items-center justify-center rounded-xl transition-all",
                    navTab === "contacts"
                      ? "bg-[#0099ff] text-white shadow-xs"
                      : "text-muted-foreground hover:bg-muted/80 hover:text-foreground",
                  )}
                >
                  <Users className="size-5" />
                  {pendingRequests > 0 && (
                    <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-bold text-white shadow-2xs">
                      {pendingRequests}
                    </span>
                  )}
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">通讯录 / 好友</TooltipContent>
            </Tooltip>

            {/* Search tab */}
            <Tooltip>
              <TooltipTrigger>
                <button
                  type="button"
                  onClick={() => {
                    setNavTab("search");
                    if (isCollapsed) {
                      onToggleCollapse();
                    }
                  }}
                  className={cn(
                    "relative flex size-10 items-center justify-center rounded-xl transition-all",
                    navTab === "search"
                      ? "bg-[#0099ff] text-white shadow-xs"
                      : "text-muted-foreground hover:bg-muted/80 hover:text-foreground",
                  )}
                >
                  <Search className="size-5" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">消息搜索</TooltipContent>
            </Tooltip>
          </div>
        </div>

        {/* Bottom Actions */}
        <div className="flex flex-col items-center gap-2">
          {/* Collapse/Expand Sidebar */}
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={onToggleCollapse}
                className="text-muted-foreground hover:text-foreground size-8 rounded-lg"
              >
                {isCollapsed ? (
                  <PanelLeftOpen className="size-4" />
                ) : (
                  <PanelLeftClose className="size-4" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">{isCollapsed ? "展开面板" : "折叠面板"}</TooltipContent>
          </Tooltip>

          {/* Refresh data */}
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => {
                  void refreshRooms();
                  void refreshConversations();
                  if (!isConnected) {
                    void connect();
                  }
                }}
                disabled={isLoadingRooms}
                className="text-muted-foreground hover:text-foreground size-8 rounded-lg"
              >
                <RefreshCw
                  className={isLoadingRooms ? "text-primary size-4 animate-spin" : "size-4"}
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">刷新数据</TooltipContent>
          </Tooltip>

          {/* Settings Dialog */}
          <SettingsDialog
            trigger={
              <Button
                variant="ghost"
                size="icon-xs"
                className="text-muted-foreground hover:text-foreground size-8 rounded-lg"
              >
                <Settings className="size-4" />
              </Button>
            }
          />

          {/* Sign Out */}
          {currentUser && (
            <Tooltip>
              <TooltipTrigger>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={logout}
                  className="text-muted-foreground size-8 rounded-lg hover:text-rose-600"
                >
                  <LogOut className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">退出登录 ({currentUser.username})</TooltipContent>
            </Tooltip>
          )}
        </div>
      </aside>

      {/* 2. Middle Sub-Panel (Conversations / Contacts / Search, ~280px-300px) */}
      {!isCollapsed && (
        <div className="bg-card border-border/70 flex h-full w-72 flex-col border-r transition-all duration-200 sm:w-80 dark:bg-[#1c1d22]">
          {navTab === "chats" && (
            <>
              {/* Header with Search + Add Button */}
              <div className="border-border/70 flex h-14 items-center justify-between gap-2 border-b px-3.5">
                <div className="relative flex-1">
                  <Search className="text-muted-foreground absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2" />
                  <Input
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="搜索"
                    className="bg-muted/50 focus-visible:border-primary/50 focus-visible:bg-background h-8 rounded-lg border-transparent pl-8 text-xs"
                  />
                </div>

                <div className="flex items-center gap-0.5">
                  <AddFriendDialog
                    trigger={
                      <Tooltip>
                        <TooltipTrigger>
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            className="text-muted-foreground hover:text-foreground hover:bg-muted size-8 shrink-0 rounded-lg"
                          >
                            <UserPlus className="size-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>添加好友</TooltipContent>
                      </Tooltip>
                    }
                  />

                  <CreateRoomDialog
                    trigger={
                      <Tooltip>
                        <TooltipTrigger>
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            className="text-muted-foreground hover:text-foreground hover:bg-muted size-8 shrink-0 rounded-lg"
                          >
                            <MessageSquarePlus className="size-4.5" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>发起新聊天 / 创建群聊</TooltipContent>
                      </Tooltip>
                    }
                  />
                </div>
              </div>

              {/* Conversation List */}
              <ScrollArea className="h-[calc(100%-3.5rem)]">
                <div className="flex flex-col gap-0.5 p-2">
                  {filteredConversations.length === 0 ? (
                    <div className="text-muted-foreground p-8 text-center text-xs">
                      {searchQuery ? "未找到匹配会话" : "暂无会话，点击右上角发起聊天"}
                    </div>
                  ) : (
                    filteredConversations.map((conv) => {
                      const room = conv.room;
                      const isActive = room.room_id === activeRoomId;
                      const isGroup = room.chat_type === "group";
                      const displayName = room.name || (isGroup ? "群聊" : "私聊");
                      const initials = getInitials(displayName);
                      const unread = conv.unread_count || 0;
                      const convMuted = conv.member.is_muted;
                      const convPinned = conv.member.is_pinned;
                      const isTyping = typingRooms[room.room_id] !== undefined;
                      const lastTime = formatRelativeTime(
                        conv.last_message?.server_time ?? conv.room.updated_at,
                      );
                      const previewText = conv.last_message
                        ? getMessagePreviewText(conv.last_message)
                        : "暂无消息";

                      return (
                        <button
                          key={room.room_id}
                          type="button"
                          onClick={() => {
                            selectRoom(room.room_id);
                          }}
                          className={cn(
                            "group/item relative flex w-full items-center gap-3 rounded-xl p-2.5 text-left transition-colors cursor-pointer",
                            isActive
                              ? "bg-[#0099ff]/10 text-primary font-medium"
                              : convPinned
                                ? "bg-muted/40 hover:bg-muted/70 text-foreground"
                                : "hover:bg-muted/60 text-foreground",
                          )}
                        >
                          {/* Room Avatar */}
                          <div className="relative shrink-0">
                            <Avatar className="ring-border/40 size-10 shadow-2xs ring-1">
                              <AvatarFallback
                                className={cn(
                                  "text-xs font-semibold",
                                  isActive
                                    ? "bg-[#0099ff] text-white"
                                    : "bg-primary/10 text-primary",
                                )}
                              >
                                {initials}
                              </AvatarFallback>
                            </Avatar>
                            {convPinned && (
                              <div className="absolute -top-1 -left-1 flex size-3.5 items-center justify-center rounded-full bg-[#0099ff] text-white shadow-2xs">
                                <Pin className="size-2 fill-white" />
                              </div>
                            )}
                          </div>

                          {/* Room Details */}
                          <div className="flex min-w-0 flex-1 flex-col">
                            <div className="flex items-center justify-between gap-1">
                              <span className="truncate text-xs leading-tight font-semibold">
                                {displayName}
                              </span>

                              <div className="flex shrink-0 items-center gap-1">
                                {lastTime && (
                                  <span className="text-muted-foreground text-[10px] leading-none">
                                    {lastTime}
                                  </span>
                                )}
                                {convMuted && <BellOff className="text-muted-foreground size-3" />}
                              </div>
                            </div>

                            <div className="mt-1 flex items-center justify-between gap-1">
                              {isTyping ? (
                                <span className="animate-pulse truncate text-[11px] leading-tight font-medium text-[#0099ff]">
                                  对方正在输入...
                                </span>
                              ) : (
                                <span className="text-muted-foreground truncate text-[11px] leading-tight">
                                  {previewText}
                                </span>
                              )}

                              {unread > 0 && (
                                <span className="flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-bold text-white shadow-2xs">
                                  {unread > 99 ? "99+" : unread}
                                </span>
                              )}
                            </div>
                          </div>
                        </button>
                      );
                    })
                  )}
                </div>
              </ScrollArea>
            </>
          )}

          {navTab === "contacts" && <ContactPanel onClose={() => setNavTab("chats")} embedded />}

          {navTab === "search" && <SearchPanel onClose={() => setNavTab("chats")} embedded />}
        </div>
      )}
    </div>
  );
}
