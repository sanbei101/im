import { X, Users, User, Server, Radio, Hash, ShieldCheck } from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useChat } from "@/context/ChatContext";
import { getInitials } from "@/types/chat";

interface ChatRoomDetailsProps {
  readonly onClose: () => void;
}

export function ChatRoomDetails({ onClose }: ChatRoomDetailsProps) {
  const { activeRoom, activeRoomId, config, currentUser } = useChat();

  if (!activeRoom || !activeRoomId) {
    return null;
  }

  const isGroup = activeRoom.chat_type === "group";
  const displayName = activeRoom.name || (isGroup ? "Group Conversation" : "Direct Message");
  const initials = getInitials(displayName);

  return (
    <aside className="bg-background/50 flex h-full w-72 shrink-0 flex-col border-l">
      <div className="flex h-14 items-center justify-between border-b px-4">
        <span className="text-sm font-semibold">Conversation Info</span>
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={onClose}
          className="text-muted-foreground hover:text-foreground"
        >
          <X className="size-4" />
        </Button>
      </div>

      <div className="flex flex-col items-center p-6 text-center">
        <Avatar className="mb-3 size-16">
          <AvatarFallback className="bg-primary/10 text-primary text-xl font-bold">
            {initials}
          </AvatarFallback>
        </Avatar>

        <h3 className="max-w-full truncate text-base font-semibold">{displayName}</h3>

        <div className="mt-1 flex items-center gap-1.5">
          <Badge variant="secondary" className="text-xs">
            {isGroup ? (
              <span className="flex items-center gap-1">
                <Users className="size-3" /> Group Room
              </span>
            ) : (
              <span className="flex items-center gap-1">
                <User className="size-3" /> Direct Chat
              </span>
            )}
          </Badge>
        </div>
      </div>

      <Separator />

      <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4 text-xs">
        <div className="flex flex-col gap-1.5">
          <span className="text-muted-foreground flex items-center gap-1 font-medium">
            <Hash className="size-3.5" /> Room ID
          </span>
          <div className="bg-muted/50 rounded-md p-2 font-mono text-[11px] break-all select-all">
            {activeRoomId}
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <span className="text-muted-foreground flex items-center gap-1 font-medium">
            <Radio className="size-3.5" /> Gateway Endpoint
          </span>
          <div className="bg-muted/50 rounded-md p-2 font-mono text-[11px] break-all select-all">
            {config.gatewayURL}
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <span className="text-muted-foreground flex items-center gap-1 font-medium">
            <Server className="size-3.5" /> HTTP API Base
          </span>
          <div className="bg-muted/50 rounded-md p-2 font-mono text-[11px] break-all select-all">
            {config.baseURL}
          </div>
        </div>

        {currentUser && (
          <div className="mt-auto flex flex-col gap-1.5 border-t pt-4">
            <span className="text-muted-foreground flex items-center gap-1 font-medium">
              <ShieldCheck className="size-3.5" /> Active Session
            </span>
            <div className="bg-muted/50 flex items-center gap-2 rounded-md p-2">
              <div className="flex min-w-0 flex-1 flex-col">
                <span className="text-foreground truncate font-semibold">
                  {currentUser.username}
                </span>
                <span className="text-muted-foreground truncate font-mono text-[10px]">
                  {currentUser.user_id}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>
    </aside>
  );
}
