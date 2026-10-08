import { MessageSquareDashed, RefreshCw, PanelLeftOpen } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChat } from "@/context/ChatContext";

import { ChatHeader } from "./ChatHeader";
import { ChatInput } from "./ChatInput";
import { ChatMessageItem } from "./ChatMessageItem";
import { RoomDetailsPanel } from "./RoomDetailsPanel";

interface ChatAreaProps {
  readonly isSidebarCollapsed?: boolean;
  readonly onToggleSidebar?: () => void;
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
      <main className="bg-muted/10 relative flex flex-1 flex-col items-center justify-center p-8 text-center">
        {isSidebarCollapsed && onToggleSidebar && (
          <div className="absolute top-3 left-4">
            <Button
              variant="outline"
              size="sm"
              onClick={onToggleSidebar}
              className="text-muted-foreground gap-1.5 text-xs"
            >
              <PanelLeftOpen className="size-4" />
              <span>Expand Sidebar</span>
            </Button>
          </div>
        )}
        <div className="bg-muted text-muted-foreground mb-4 flex size-16 items-center justify-center rounded-full">
          <MessageSquareDashed className="size-8" />
        </div>
        <h3 className="mb-1 text-lg font-semibold">No Active Conversation</h3>
        <p className="text-muted-foreground max-w-sm text-sm">
          Select a room from the sidebar or click &ldquo;New Conversation&rdquo; to start chatting.
        </p>
      </main>
    );
  }

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

        <div className="relative min-h-0 flex-1">
          <ScrollArea className="h-full px-4 py-3">
            {hasMoreHistory && (
              <div className="mb-2 flex justify-center">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={isLoadingMoreHistory}
                  onClick={() => void loadMoreHistory()}
                  className="text-muted-foreground gap-1.5 text-xs"
                >
                  <RefreshCw className={isLoadingMoreHistory ? "size-3.5 animate-spin" : "size-3.5"} />
                  {isLoadingMoreHistory ? "Loading…" : "Load older messages"}
                </Button>
              </div>
            )}

            {isLoadingHistory && messages.length === 0 && (
              <div className="text-muted-foreground flex items-center justify-center gap-2 py-12">
                <RefreshCw className="size-4 animate-spin" />
                <span className="text-xs">Loading message history...</span>
              </div>
            )}

            {!isLoadingHistory && messages.length === 0 && (
              <div className="text-muted-foreground flex flex-col items-center justify-center py-16 text-center">
                <p className="text-xs">
                  This conversation is just beginning. Send a message below!
                </p>
              </div>
            )}

            {messages.map((message) => {
              const isSelf = currentUser !== null && message.senderId === currentUser.user_id;
              return (
                <ChatMessageItem
                  key={message.id || message.clientMsgId}
                  message={message}
                  isSelf={isSelf}
                />
              );
            })}

            <div ref={messagesEndRef} className="h-2" />
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
