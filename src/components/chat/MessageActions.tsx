import { MoreVertical, Pin, PinOff, Reply, SmilePlus, Trash2 } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import type { UIMessage } from "@/types/chat";

/** Emoji set offered by the reaction picker. */
const REACTIONS = ["👍", "❤️", "😂", "🎉", "👀", "🙏"] as const;

interface MessageActionsProps {
  readonly message: UIMessage;
  readonly isSelf: boolean;
  readonly onDismiss?: () => void;
}

/** Hover actions for a message: reply, react, pin, recall. */
export function MessageActions({ message, isSelf, onDismiss }: MessageActionsProps) {
  const {
    setReplyingToMessage,
    toggleReaction,
    pinMessage,
    unpinMessage,
    recallMessage,
    pinnedMessages,
    currentUser,
  } = useChat();
  const [reactorOpen, setReactorOpen] = useState(false);
  const [pending, setPending] = useState(false);

  const isPinned = pinnedMessages.some((p) => p.msg_id === message.id);
  const canPin = !!currentUser;
  const canRecall = isSelf && !message.recalled;

  const run = async (action: () => Promise<unknown>) => {
    setPending(true);
    try {
      await action();
    } catch {
      // Errors surface through the context error banner.
    } finally {
      setPending(false);
      onDismiss?.();
    }
  };

  const summary = message.reactions ?? [];
  const myReactions = new Set(
    currentUser
      ? summary.filter((r) => r.user_ids.includes(currentUser.user_id)).map((r) => r.emoji)
      : [],
  );

  return (
    <div className="flex items-center gap-0.5">
      <Popover open={reactorOpen} onOpenChange={setReactorOpen}>
        <PopoverTrigger
          render={<Button variant="ghost" size="icon-sm" aria-label="React" disabled={pending} />}
        >
          <SmilePlus className="size-3.5" />
        </PopoverTrigger>
        <PopoverContent className="w-auto p-1" align="center">
          <div className="flex gap-0.5">
            {REACTIONS.map((emoji) => (
              <button
                key={emoji}
                type="button"
                aria-label={`React with ${emoji}`}
                aria-pressed={myReactions.has(emoji)}
                className={
                  "hover:bg-muted flex size-8 items-center justify-center rounded-md text-base transition-colors" +
                  (myReactions.has(emoji) ? " bg-muted" : "")
                }
                onClick={() => {
                  setReactorOpen(false);
                  void run(() => toggleReaction(message.id, emoji));
                }}
              >
                {emoji}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>

      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Reply"
              disabled={pending}
              onClick={() => {
                setReplyingToMessage(message);
                onDismiss?.();
              }}
            />
          }
        >
          <Reply className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>Reply</TooltipContent>
      </Tooltip>

      {canRecall && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="Recall message"
                disabled={pending}
                className="text-muted-foreground hover:text-destructive"
                onClick={() => void run(() => recallMessage(message.id))}
              />
            }
          >
            <Trash2 className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent>Recall</TooltipContent>
        </Tooltip>
      )}

      {canPin && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<Button variant="ghost" size="icon-sm" aria-label="More" disabled={pending} />}
          >
            <MoreVertical className="size-3.5" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() =>
                void run(() => (isPinned ? unpinMessage(message.id) : pinMessage(message.id)))
              }
            >
              {isPinned ? <PinOff /> : <Pin />}
              {isPinned ? "Unpin" : "Pin"}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}

/** Compact reaction chips rendered under a message bubble. */
export function MessageReactions({ message }: { readonly message: UIMessage }) {
  const { toggleReaction, currentUser } = useChat();
  const reactions = message.reactions ?? [];
  if (reactions.length === 0) {
    return null;
  }

  return (
    <div className="mt-1 flex flex-wrap gap-1">
      {reactions.map((group) => {
        const mine = currentUser ? group.user_ids.includes(currentUser.user_id) : false;
        return (
          <button
            key={group.emoji}
            type="button"
            aria-label={`${group.emoji} ${group.count}`}
            aria-pressed={mine}
            className={
              "bg-muted/60 hover:bg-muted inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs transition-colors" +
              (mine ? " border-primary/40" : " border-transparent")
            }
            onClick={() => void toggleReaction(message.id, group.emoji)}
          >
            <span>{group.emoji}</span>
            <span className="text-muted-foreground tabular-nums">{group.count}</span>
          </button>
        );
      })}
    </div>
  );
}
