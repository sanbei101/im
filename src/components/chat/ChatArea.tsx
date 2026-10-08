import { MessageSquareDashed, RefreshCw, PanelLeftOpen } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChat } from "@/context/ChatContext";
import { getInitials } from "@/types/chat";

import { ChatHeader } from "./ChatHeader";
import { ChatInput } from "./ChatInput";
import { ChatMessageItem } from "./ChatMessageItem";
import { RoomDetailsPanel } from "./RoomDetailsPanel";

interface ChatAreaProps {
  readonly isSidebarCollapsed?: boolean;
  readonly onToggleSidebar?: () => void;
}

function formatDateDivider(timestamp: number): string {
  if (!timestamp) return "";
  const ms = timestamp > 10_000_000_000_000 ? Math.floor(timestamp / 1000) : timestamp;
  const date = new Date(ms);
  const now = new Date();
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();
  const hours = date.getHours().toString().padStart(2, "0");
  const minutes = date.getMinutes().toString().padStart(2, "0");
  if (isToday) {
    return `${hours}:${minutes}`;
  }
  const month = (date.getMonth() + 1).toString().padStart(2, "0");
  const day = date.getDate().toString().padStart(2, "0");
  return `${month}-${day} ${hours}:${minutes}`;
}

function shouldShowDivider(prevTime: number | undefined, currTime: number): boolean {
  if (!prevTime) return true;
  const p = prevTime > 10_000_000_000_000 ? Math.floor(prevTime / 1000) : prevTime;
  const c = currTime > 10_000_000_000_000 ? Math.floor(currTime / 1000) : currTime;
  return Math.abs(c - p) > 5 * 60 * 1000;
}

export function ChatArea({ isSidebarCollapsed = false, onToggleSidebar }: ChatAreaProps) {
  const {
    activeRoomId,
    activeRoom,
    messages,
    currentUser,
    isLoadingHistory,
    hasMoreHistory,
    isLoadingMoreHistory,
    loadMoreHistory,
    markActiveRoomRead,
  } = useChat();
  const [showDetails, setShowDetails] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto scroll to bottom on new message
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages.length]);

  // Mark the active room read once its history is loaded.
  useEffect(() => {
    if (activeRoomId && !isLoadingHistory) {
      void markActiveRoomRead();
    }
  }, [activeRoomId, isLoadingHistory, markActiveRoomRead]);

  if (!activeRoomId) {
    return (
      <main className="relative flex flex-1 flex-col items-center justify-center bg-[#f4f5f8] p-8 text-center select-none dark:bg-[#151619]">
        {isSidebarCollapsed && onToggleSidebar && (
          <div className="absolute top-3 left-4">
            <Button
              variant="outline"
              size="sm"
              onClick={onToggleSidebar}
              className="text-muted-foreground gap-1.5 text-xs shadow-2xs"
            >
              <PanelLeftOpen className="size-4" />
              <span>Expand Sidebar</span>
            </Button>
          </div>
        )}
        <div className="bg-card text-muted-foreground mb-4 flex size-20 items-center justify-center rounded-2xl border shadow-xs dark:bg-zinc-800/60">
          <MessageSquareDashed className="text-primary/60 size-10" />
        </div>
        <h3 className="text-foreground mb-1 text-base font-semibold">QQ 在线聊天</h3>
        <p className="text-muted-foreground max-w-xs text-xs leading-relaxed">
          从左侧列表选择会话，或点击创建新聊天开始交流。
        </p>
      </main>
    );
  }

  const displayName = activeRoom?.name || (activeRoom?.chat_type === "group" ? "群聊" : "私聊");

  return (
    <main className="bg-background flex h-full flex-1 overflow-hidden">
      <div className="flex h-full min-w-0 flex-1 flex-col">
        <ChatHeader
          isSidebarCollapsed={isSidebarCollapsed}
          onToggleSidebar={onToggleSidebar}
          showDetails={showDetails}
          onToggleDetails={() => {
            setShowDetails((prev) => !prev);
          }}
        />

        {/* Message area with QQ-style soft light background */}
        <div className="relative min-h-0 flex-1 bg-[#f4f5f8] dark:bg-[#151619]">
          <ScrollArea className="h-full">
            <div className="py-3">
              {hasMoreHistory && (
                <div className="mb-3 flex justify-center">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={isLoadingMoreHistory}
                    onClick={() => void loadMoreHistory()}
                    className="text-muted-foreground bg-card/80 h-7 gap-1.5 rounded-full px-3 text-xs shadow-2xs backdrop-blur-xs"
                  >
                    <RefreshCw
                      className={isLoadingMoreHistory ? "size-3 animate-spin" : "size-3"}
                    />
                    {isLoadingMoreHistory ? "加载中…" : "查看更多历史消息"}
                  </Button>
                </div>
              )}

              {isLoadingHistory && messages.length === 0 && (
                <div className="text-muted-foreground flex items-center justify-center gap-2 py-16">
                  <RefreshCw className="text-primary size-4 animate-spin" />
                  <span className="text-xs">正在同步聊天记录...</span>
                </div>
              )}

              {!isLoadingHistory && messages.length === 0 && (
                <div className="flex flex-col items-center justify-center py-20 text-center select-none">
                  <Avatar className="ring-border/50 mb-3 size-14 shadow-xs ring-2">
                    <AvatarFallback className="bg-primary/10 text-primary text-base font-bold">
                      {getInitials(displayName)}
                    </AvatarFallback>
                  </Avatar>
                  <span className="text-foreground mb-1 text-sm font-semibold">{displayName}</span>
                  <p className="text-muted-foreground max-w-xs text-xs">
                    你们已经是好友了，快发一条消息打个招呼吧！
                  </p>
                </div>
              )}

              {messages.map((message, index) => {
                const isSelf = currentUser !== null && message.senderId === currentUser.user_id;
                const prevMessage = index > 0 ? messages[index - 1] : undefined;
                const showDivider = shouldShowDivider(prevMessage?.serverTime, message.serverTime);

                return (
                  <div key={message.id || message.clientMsgId}>
                    {showDivider && (
                      <div className="my-3 flex justify-center select-none">
                        <span className="text-muted-foreground rounded-full bg-black/5 px-2.5 py-0.5 text-[11px] font-medium dark:bg-white/10">
                          {formatDateDivider(message.serverTime)}
                        </span>
                      </div>
                    )}
                    <ChatMessageItem message={message} isSelf={isSelf} />
                  </div>
                );
              })}

              <div ref={messagesEndRef} className="h-3" />
            </div>
          </ScrollArea>
        </div>

        <ChatInput />
      </div>

      {showDetails && activeRoom && (
        <RoomDetailsPanel
          onClose={() => {
            setShowDetails(false);
          }}
        />
      )}
    </main>
  );
}
