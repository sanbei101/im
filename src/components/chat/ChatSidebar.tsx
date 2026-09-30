import {
  MessageSquare,
  Search,
  Users,
  User,
  LogOut,
  Copy,
  Check,
  RefreshCw,
  PanelLeftClose,
  PanelLeftOpen,
  MessageSquarePlus,
} from "lucide-react";
import { useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { getInitials } from "@/types/chat";

import { CreateRoomDialog } from "./CreateRoomDialog";
import { SettingsDialog } from "./SettingsDialog";

interface ChatSidebarProps {
  readonly isCollapsed: boolean;
  readonly onToggleCollapse: () => void;
}

export function ChatSidebar({ isCollapsed, onToggleCollapse }: ChatSidebarProps) {
  const { rooms, activeRoomId, selectRoom, currentUser, logout, refreshRooms, isLoadingRooms } =
    useChat();

  const [searchQuery, setSearchQuery] = useState("");
  const [copiedUserId, setCopiedUserId] = useState(false);

  const filteredRooms = rooms.filter((room) => {
    if (!searchQuery.trim()) {
      return true;
    }
    const q = searchQuery.toLowerCase();
    const nameMatch = room.name ? room.name.toLowerCase().includes(q) : false;
    const idMatch = room.room_id.toLowerCase().includes(q);
    return nameMatch || idMatch;
  });

  const handleCopyUserId = async () => {
    if (currentUser && typeof navigator !== "undefined" && navigator.clipboard) {
      await navigator.clipboard.writeText(currentUser.user_id);
      setCopiedUserId(true);
      setTimeout(() => {
        setCopiedUserId(false);
      }, 2000);
    }
  };

  // Render collapsed mini-rail
  if (isCollapsed) {
    return (
      <aside className="bg-muted/20 flex h-full w-16 shrink-0 flex-col items-center border-r py-2 select-none transition-[width] duration-200">
        <div className="flex flex-col items-center gap-2 pb-2">
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon"
                onClick={onToggleCollapse}
                className="hover:bg-accent size-9"
              >
                <PanelLeftOpen className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">Expand Sidebar</TooltipContent>
          </Tooltip>

          <CreateRoomDialog
            trigger={
              <Tooltip>
                <TooltipTrigger>
                  <Button variant="default" size="icon" className="size-9 rounded-lg shadow-xs">
                    <MessageSquarePlus className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="right">New Conversation</TooltipContent>
              </Tooltip>
            }
          />
        </div>

        <Separator className="w-8 my-1" />

        <div className="relative min-h-0 flex-1 w-full">
          <ScrollArea className="h-full w-full">
            <div className="flex flex-col items-center gap-2 px-2 py-1">
              {filteredRooms.map((room) => {
                const isActive = room.room_id === activeRoomId;
                const isGroup = room.chat_type === "group";
                const displayName = room.name || (isGroup ? "Group Chat" : "Direct Chat");
                const initials = getInitials(displayName);

                return (
                  <Tooltip key={room.room_id}>
                    <TooltipTrigger>
                      <button
                        type="button"
                        onClick={() => {
                          selectRoom(room.room_id);
                        }}
                        className={
                          isActive
                            ? "ring-2 ring-primary bg-primary/10 rounded-full transition-all"
                            : "hover:opacity-80 rounded-full transition-all"
                        }
                      >
                        <Avatar className="size-9">
                          <AvatarFallback className="text-xs font-semibold">
                            {initials}
                          </AvatarFallback>
                        </Avatar>
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="right" className="flex items-center gap-1.5 text-xs">
                      <span>{displayName}</span>
                      <span className="text-[10px] text-muted-foreground font-mono">
                        ({isGroup ? "Group" : "Direct"})
                      </span>
                    </TooltipContent>
                  </Tooltip>
                );
              })}
            </div>
          </ScrollArea>
        </div>

        <Separator className="w-8 my-1" />

        <div className="flex flex-col items-center gap-2 pt-1">
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => {
                  void refreshRooms();
                }}
                disabled={isLoadingRooms}
                className="size-8 text-muted-foreground hover:text-foreground"
              >
                <RefreshCw className={isLoadingRooms ? "size-3.5 animate-spin" : "size-3.5"} />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">Refresh</TooltipContent>
          </Tooltip>

          <SettingsDialog
            trigger={
              <Button
                variant="ghost"
                size="icon"
                className="size-8 text-muted-foreground hover:text-foreground"
              >
                <MessageSquare className="size-3.5" />
              </Button>
            }
          />

          {currentUser && (
            <Tooltip>
              <TooltipTrigger>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={logout}
                  className="size-8 text-muted-foreground hover:text-destructive"
                >
                  <LogOut className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">
                <span>Sign Out ({currentUser.username})</span>
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      </aside>
    );
  }

  // Render expanded sidebar
  return (
    <aside className="bg-muted/20 flex h-full w-80 shrink-0 flex-col border-r select-none transition-[width] duration-200">
      {/* Top Header */}
      <div className="bg-background/50 flex h-14 items-center justify-between border-b px-3.5">
        <div className="flex items-center gap-2">
          <div className="bg-primary text-primary-foreground flex size-8 items-center justify-center rounded-lg font-bold">
            <MessageSquare className="size-4" />
          </div>
          <div className="flex flex-col">
            <span className="text-sm leading-tight font-semibold">Go IM</span>
            <span className="text-muted-foreground text-[10px] leading-tight">
              Instant Messenger
            </span>
          </div>
        </div>

        <div className="flex items-center gap-1">
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => {
                  void refreshRooms();
                }}
                disabled={isLoadingRooms}
                className="text-muted-foreground hover:text-foreground"
              >
                <RefreshCw className={isLoadingRooms ? "size-3.5 animate-spin" : "size-3.5"} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Refresh Rooms</TooltipContent>
          </Tooltip>

          <SettingsDialog />

          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={onToggleCollapse}
                className="text-muted-foreground hover:text-foreground"
              >
                <PanelLeftClose className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Collapse Sidebar</TooltipContent>
          </Tooltip>
        </div>
      </div>

      {/* Action button & Search */}
      <div className="bg-background/30 flex flex-col gap-2 border-b p-3">
        <CreateRoomDialog />

        <div className="relative">
          <Search className="text-muted-foreground absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2" />
          <Input
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
            }}
            placeholder="Search conversations..."
            className="bg-background/60 h-8 pl-8 text-xs"
          />
        </div>
      </div>

      {/* Rooms list */}
      <div className="relative min-h-0 flex-1">
        <ScrollArea className="h-full">
          <div className="flex flex-col gap-1 p-2">
            {filteredRooms.length === 0 ? (
              <div className="text-muted-foreground p-6 text-center text-xs">
                {searchQuery ? "No matching conversations" : "No conversations yet"}
              </div>
            ) : (
              filteredRooms.map((room) => {
                const isActive = room.room_id === activeRoomId;
                const isGroup = room.chat_type === "group";
                const displayName = room.name || (isGroup ? "Group Chat" : "Direct Chat");
                const initials = getInitials(displayName);

                return (
                  <button
                    key={room.room_id}
                    type="button"
                    onClick={() => {
                      selectRoom(room.room_id);
                    }}
                    className={
                      isActive
                        ? "bg-accent text-accent-foreground flex w-full items-center gap-3 rounded-lg p-2.5 text-left shadow-2xs transition-colors"
                        : "hover:bg-muted/60 text-foreground flex w-full items-center gap-3 rounded-lg p-2.5 text-left transition-colors"
                    }
                  >
                    <Avatar className="size-9 shrink-0">
                      <AvatarFallback className="bg-primary/10 text-primary text-xs font-medium">
                        {initials}
                      </AvatarFallback>
                    </Avatar>

                    <div className="flex min-w-0 flex-1 flex-col">
                      <div className="flex items-center justify-between gap-1">
                        <span className="truncate text-xs leading-tight font-semibold">
                          {displayName}
                        </span>
                        <Badge
                          variant={isActive ? "default" : "secondary"}
                          className="h-3.5 shrink-0 px-1 py-0 text-[9px]"
                        >
                          {isGroup ? <Users className="size-2.5" /> : <User className="size-2.5" />}
                        </Badge>
                      </div>

                      <span className="text-muted-foreground mt-0.5 truncate font-mono text-[11px]">
                        {room.room_id}
                      </span>
                    </div>
                  </button>
                );
              })
            )}
          </div>
        </ScrollArea>
      </div>

      <Separator />

      {/* User profile footer */}
      {currentUser && (
        <div className="bg-background/50 flex items-center justify-between p-3">
          <div className="flex min-w-0 items-center gap-2.5">
            <Avatar className="size-8 shrink-0">
              <AvatarFallback className="bg-secondary text-secondary-foreground text-xs font-medium">
                {getInitials(currentUser.username)}
              </AvatarFallback>
            </Avatar>

            <div className="flex min-w-0 flex-col">
              <span className="truncate text-xs leading-tight font-semibold">
                {currentUser.username}
              </span>
              <div className="mt-0.5 flex items-center gap-1">
                <span className="text-muted-foreground max-w-[120px] truncate font-mono text-[10px]">
                  {currentUser.user_id}
                </span>
                <Tooltip>
                  <TooltipTrigger>
                    <button
                      type="button"
                      onClick={handleCopyUserId}
                      className="text-muted-foreground hover:text-foreground rounded-xs p-0.5 transition-colors"
                    >
                      {copiedUserId ? (
                        <Check className="size-2.5 text-emerald-500" />
                      ) : (
                        <Copy className="size-2.5" />
                      )}
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <span>{copiedUserId ? "Copied!" : "Copy My User ID"}</span>
                  </TooltipContent>
                </Tooltip>
              </div>
            </div>
          </div>

          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={logout}
                className="text-muted-foreground hover:text-destructive"
              >
                <LogOut className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Sign Out</TooltipContent>
          </Tooltip>
        </div>
      )}
    </aside>
  );
}
