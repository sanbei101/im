import { ConnectionState } from "go-chat-sdk";
import {
  MessageSquare,
  Search,
  Users,
  User,
  LogOut,
  RefreshCw,
  PanelLeftClose,
  PanelLeftOpen,
  MessageSquarePlus,
  BellOff,
  Pin,
  Settings,
} from "lucide-react";
import { useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { cn } from "@/lib/utils";
import { getInitials } from "@/types/chat";

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
  } = useChat();

  const [navTab, setNavTab] = useState<NavTab>("chats");
  const [searchQuery, setSearchQuery] = useState("");

  // Calculate unread counts
  const unreadCountByRoom: Record<string, number> = {};
  let totalUnread = 0;
  for (const conversation of conversations) {
    const unread = conversation.unread_count || 0;
    unreadCountByRoom[conversation.room.room_id] = unread;
    totalUnread += unread;
  }

  const pendingRequests = friendApplications.filter((a) => a.status === "pending").length;

  const filteredRooms = rooms.filter((room) => {
    if (!searchQuery.trim()) {
      return true;
    }
    const q = searchQuery.toLowerCase();
    const nameMatch = room.name ? room.name.toLowerCase().includes(q) : false;
    const idMatch = room.room_id.toLowerCase().includes(q);
    return nameMatch || idMatch;
  });

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

              {/* Conversation List */}
              <ScrollArea className="h-[calc(100%-3.5rem)]">
                <div className="flex flex-col gap-0.5 p-2">
                  {filteredRooms.length === 0 ? (
                    <div className="text-muted-foreground p-8 text-center text-xs">
                      {searchQuery ? "未找到匹配会话" : "暂无会话，点击右上角发起聊天"}
                    </div>
                  ) : (
                    filteredRooms.map((room) => {
                      const isActive = room.room_id === activeRoomId;
                      const isGroup = room.chat_type === "group";
                      const displayName = room.name || (isGroup ? "群聊" : "私聊");
                      const initials = getInitials(displayName);
                      const unread = unreadCountByRoom[room.room_id] ?? 0;
                      const conv = conversations.find((c) => c.room.room_id === room.room_id);
                      const convMuted = conv?.member.is_muted ?? false;
                      const convPinned = conv?.member.is_pinned ?? false;

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
                          </div>

                          {/* Room Details */}
                          <div className="flex min-w-0 flex-1 flex-col">
                            <div className="flex items-center justify-between gap-1">
                              <span className="truncate text-xs leading-tight font-semibold">
                                {displayName}
                              </span>

                              <div className="flex shrink-0 items-center gap-1">
                                {convMuted && <BellOff className="text-muted-foreground size-3" />}
                                {convPinned && <Pin className="size-3 text-[#0099ff]" />}
                                <Badge
                                  variant="secondary"
                                  className="h-3.5 shrink-0 px-1 py-0 text-[9px] font-normal"
                                >
                                  {isGroup ? (
                                    <Users className="mr-0.5 size-2.5" />
                                  ) : (
                                    <User className="mr-0.5 size-2.5" />
                                  )}
                                  {isGroup ? "群" : "私"}
                                </Badge>
                              </div>
                            </div>

                            <div className="mt-1 flex items-center justify-between gap-1">
                              <span className="text-muted-foreground max-w-[150px] truncate font-mono text-[11px] leading-tight">
                                {room.room_id.slice(0, 10)}...
                              </span>

                              {unread > 0 && (
                                <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-bold text-white shadow-2xs">
                                  {unread}
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
