import { useCallback, useEffect, useState } from "react";
import { Search, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChat } from "@/context/ChatContext";
import { formatTime, getMessagePreviewText, mapSdkMessageToUIMessage } from "@/types/chat";

interface SearchPanelProps {
  readonly onClose: () => void;
}

/** Room-scoped message search; the backend requires a room_id. */
export function SearchPanel({ onClose }: SearchPanelProps) {
  const { activeRoomId, activeRoom, searchResults, isSearching, searchMessages, selectRoom } =
    useChat();
  const [keyword, setKeyword] = useState("");
  // The backend requires a room_id, so scope is fixed to the active room.
  const scope = "room" as const;

  const submit = useCallback(
    (event: { preventDefault: () => void }) => {
      event.preventDefault();
      void searchMessages(keyword, scope);
    },
    [keyword, scope, searchMessages],
  );

  // Drop stale results whenever the active room changes.
  useEffect(() => {
    setKeyword("");
  }, [activeRoomId]);

  return (
    <aside className="bg-background/50 flex h-full w-80 shrink-0 flex-col border-l">
      <div className="flex h-14 shrink-0 items-center justify-between border-b px-4">
        <span className="flex items-center gap-2 text-sm font-semibold">
          <Search className="size-4" />
          Search
        </span>
        <Button variant="ghost" size="icon-sm" aria-label="Close search" onClick={onClose}>
          <X className="size-4" />
        </Button>
      </div>

      <form className="flex items-center gap-1.5 border-b p-3" onSubmit={submit}>
        <Input
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          placeholder="Search in this room"
          aria-label="Search keyword"
          className="h-8 text-xs"
        />
        <Button type="submit" size="sm" disabled={isSearching || keyword.trim() === ""}>
          <Search className="size-3.5" />
        </Button>
      </form>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-1 p-2">
          {!activeRoomId && (
            <p className="text-muted-foreground px-2 py-8 text-center text-xs">
              Select a room to search its messages.
            </p>
          )}
          {activeRoomId && isSearching && (
            <div className="flex flex-col gap-2 p-2">
              {[0, 1, 2].map((i) => (
                <div key={i} className="bg-muted/50 h-14 animate-pulse rounded-md" />
              ))}
            </div>
          )}
          {activeRoomId && !isSearching && keyword.trim() !== "" && searchResults.length === 0 && (
            <p className="text-muted-foreground px-2 py-8 text-center text-xs">
              No messages match “{keyword.trim()}”.
            </p>
          )}
          {searchResults.map((msg) => {
            const ui = mapSdkMessageToUIMessage(msg);
            return (
              <button
                key={msg.msg_id}
                type="button"
                className="hover:bg-muted/60 flex flex-col gap-0.5 rounded-md p-2 text-left"
                onClick={() => {
                  selectRoom(msg.room_id);
                  const target = document.getElementById(`msg-${msg.msg_id}`);
                  target?.scrollIntoView({ behavior: "smooth", block: "center" });
                }}
              >
                <span className="text-muted-foreground flex items-center gap-1 text-[10px]">
                  {msg.room_id === activeRoomId ? (
                    activeRoom?.name || "This room"
                  ) : (
                    <span className="font-mono">{msg.room_id.slice(0, 8)}</span>
                  )}
                  <span>·</span>
                  <span>{formatTime(msg.server_time)}</span>
                  {msg.room_seq > 0 && <span>#{msg.room_seq}</span>}
                </span>
                <span className="line-clamp-2 text-xs break-words">
                  {getMessagePreviewText(ui)}
                </span>
              </button>
            );
          })}
        </div>
      </ScrollArea>

      <div className="text-muted-foreground shrink-0 border-t p-2 text-[10px]">
        Search runs server-side across the active room.
      </div>
    </aside>
  );
}
