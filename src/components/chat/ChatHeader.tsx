import {
  Copy,
  Check,
  Info,
  Users,
  User,
  RefreshCw,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { getInitials } from "@/types/chat";

import { ConnectionStatusBadge } from "./ConnectionStatusBadge";

interface ChatHeaderProps {
  readonly isSidebarCollapsed?: boolean;
  readonly onToggleSidebar?: () => void;
  readonly onToggleDetails: () => void;
  readonly showDetails: boolean;
}

export function ChatHeader({
  isSidebarCollapsed = false,
  onToggleSidebar,
  onToggleDetails,
  showDetails,
}: ChatHeaderProps) {
  const { activeRoom, activeRoomId, selectRoom, isLoadingHistory, typingRooms } = useChat();
  const [copied, setCopied] = useState(false);
  const isTyping = activeRoomId ? typingRooms[activeRoomId] !== undefined : false;

  if (!activeRoom || !activeRoomId) {
    return (
      <header className="bg-background/95 flex h-14 items-center justify-between border-b px-4 backdrop-blur-xs">
        <div className="flex items-center gap-2">
          {onToggleSidebar && (
            <Tooltip>
              <TooltipTrigger>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={onToggleSidebar}
                  className="text-muted-foreground hover:text-foreground shrink-0"
                >
                  {isSidebarCollapsed ? (
                    <PanelLeftOpen className="size-4" />
                  ) : (
                    <PanelLeftClose className="size-4" />
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {isSidebarCollapsed ? "Expand Sidebar" : "Collapse Sidebar"}
              </TooltipContent>
            </Tooltip>
          )}
          <span className="text-muted-foreground text-sm">No conversation selected</span>
        </div>
        <ConnectionStatusBadge />
      </header>
    );
  }

  const isGroup = activeRoom.chat_type === "group";
  const displayName = activeRoom.name || (isGroup ? "Group Conversation" : "Direct Message");
  const initials = getInitials(displayName);

  const handleCopyRoomId = async () => {
    if (typeof navigator !== "undefined" && navigator.clipboard) {
      await navigator.clipboard.writeText(activeRoomId);
      setCopied(true);
      setTimeout(() => {
        setCopied(false);
      }, 2000);
    }
  };

  const handleRefreshHistory = () => {
    selectRoom(activeRoomId);
  };

  return (
    <header className="bg-background/95 flex h-14 shrink-0 items-center justify-between border-b px-4 backdrop-blur-xs">
      <div className="flex min-w-0 items-center gap-2 sm:gap-3">
        {onToggleSidebar && (
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={onToggleSidebar}
                className="text-muted-foreground hover:text-foreground shrink-0"
              >
                {isSidebarCollapsed ? (
                  <PanelLeftOpen className="size-4" />
                ) : (
                  <PanelLeftClose className="size-4" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {isSidebarCollapsed ? "Expand Sidebar" : "Collapse Sidebar"}
            </TooltipContent>
          </Tooltip>
        )}
        <Avatar className="size-9 shrink-0">
          <AvatarFallback className="bg-primary/10 text-primary text-xs font-medium">
            {initials}
          </AvatarFallback>
        </Avatar>

        <div className="flex min-w-0 flex-col">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-sm leading-none font-semibold">{displayName}</h2>
            <Badge variant="secondary" className="h-4 px-1.5 py-0 text-[10px]">
              {isGroup ? (
                <span className="flex items-center gap-1">
                  <Users className="size-2.5" /> Group
                </span>
              ) : (
                <span className="flex items-center gap-1">
                  <User className="size-2.5" /> Direct
                </span>
              )}
            </Badge>
          </div>

          {isTyping ? (
            <div className="mt-0.5 flex items-center gap-1.5">
              <span className="flex animate-pulse items-center gap-1 text-[11px] font-medium text-[#0099ff]">
                <span className="size-1.5 rounded-full bg-[#0099ff]" />
                对方正在输入...
              </span>
            </div>
          ) : (
            <div className="mt-0.5 flex items-center gap-1.5">
              <span className="text-muted-foreground max-w-[180px] truncate font-mono text-[11px]">
                {activeRoomId}
              </span>
              <Tooltip>
                <TooltipTrigger>
                  <button
                    type="button"
                    onClick={handleCopyRoomId}
                    className="text-muted-foreground hover:text-foreground rounded-sm p-0.5 transition-colors"
                  >
                    {copied ? (
                      <Check className="size-3 text-emerald-500" />
                    ) : (
                      <Copy className="size-3" />
                    )}
                  </button>
                </TooltipTrigger>
                <TooltipContent>
                  <span>{copied ? "Copied!" : "Copy Room ID"}</span>
                </TooltipContent>
              </Tooltip>
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <ConnectionStatusBadge />

        <Tooltip>
          <TooltipTrigger>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleRefreshHistory}
              disabled={isLoadingHistory}
              title="Refresh conversation history"
            >
              <RefreshCw className={isLoadingHistory ? "size-4 animate-spin" : "size-4"} />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Sync History</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger>
            <Button
              variant={showDetails ? "secondary" : "ghost"}
              size="icon-sm"
              onClick={onToggleDetails}
              title="Toggle room info"
            >
              <Info className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Room Details</TooltipContent>
        </Tooltip>
      </div>
    </header>
  );
}
