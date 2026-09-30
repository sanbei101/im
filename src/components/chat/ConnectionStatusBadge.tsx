import { ConnectionState } from "go-chat-sdk";
import { RefreshCw } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";

export function ConnectionStatusBadge() {
  const { connectionState, connect } = useChat();

  switch (connectionState) {
    case ConnectionState.Connected:
      return (
        <Badge
          variant="outline"
          className="gap-1.5 border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
        >
          <span className="size-1.5 rounded-full bg-emerald-500" />
          <span>Connected</span>
        </Badge>
      );
    case ConnectionState.Connecting:
    case ConnectionState.Reconnecting:
      return (
        <Badge
          variant="outline"
          className="gap-1.5 border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400"
        >
          <span className="size-1.5 animate-pulse rounded-full bg-amber-500" />
          <span>
            {connectionState === ConnectionState.Reconnecting ? "Reconnecting..." : "Connecting..."}
          </span>
        </Badge>
      );
    case ConnectionState.Error:
    case ConnectionState.Disconnected:
    default:
      return (
        <Tooltip>
          <TooltipTrigger>
            <Button
              variant="outline"
              size="xs"
              onClick={() => {
                void connect();
              }}
              className="gap-1.5 border-rose-500/30 bg-rose-500/10 text-rose-600 hover:bg-rose-500/20 dark:text-rose-400"
            >
              <span className="size-1.5 rounded-full bg-rose-500" />
              <span>Offline</span>
              <RefreshCw className="ml-0.5 size-3" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <span>Click to reconnect WebSocket gateway</span>
          </TooltipContent>
        </Tooltip>
      );
  }
}
