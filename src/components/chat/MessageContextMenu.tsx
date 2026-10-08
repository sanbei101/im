import { Copy, Eye, Pin, PinOff, Reply, Trash2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

import { Separator } from "@/components/ui/separator";
import { useChat } from "@/context/ChatContext";
import { isTextPayload, type UIMessage } from "@/types/chat";

/** Emoji set offered by the reaction row. */
const REACTIONS = ["👍", "❤️", "😂", "🎉", "😮", "😢", "👀", "🙏"] as const;

interface MessageContextMenuProps {
  readonly message: UIMessage;
  readonly isSelf: boolean;
  /** Anchor point in viewport coordinates. */
  readonly anchor: { readonly x: number; readonly y: number };
  readonly onClose: () => void;
}

/**
 * Right-click menu for a message bubble, positioned near the pointer the way
 * desktop IM clients do it.
 */
export function MessageContextMenu({ message, isSelf, anchor, onClose }: MessageContextMenuProps) {
  const {
    setReplyingToMessage,
    toggleReaction,
    pinMessage,
    unpinMessage,
    recallMessage,
    pinnedMessages,
    getReadUsers,
    currentUser,
  } = useChat();

  const [pending, setPending] = useState(false);
  const [readUsers, setReadUsers] = useState<readonly string[]>([]);
  const [copied, setCopied] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const isPinned = pinnedMessages.some((p) => p.msg_id === message.id);
  const canRecall = isSelf && !message.recalled;

  // Close on any outside interaction, matching native context menus.
  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      if (!ref.current?.contains(event.target as Node)) {
        onClose();
      }
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  // Flip the menu back on-screen when it would overflow the viewport.
  const style = {
    left: Math.min(anchor.x, window.innerWidth - 220),
    top: Math.min(anchor.y, window.innerHeight - 280),
  } as const;

  const run = useCallback(
    async (action: () => Promise<unknown>) => {
      setPending(true);
      try {
        await action();
      } catch {
        // Errors surface through the context error banner.
      } finally {
        setPending(false);
        onClose();
      }
    },
    [onClose],
  );

  const previewText = extractText(message);

  const handleReadUsers = async () => {
    try {
      const ids = await getReadUsers(message.id, message.roomId);
      setReadUsers(ids);
    } catch {
      // Error banner is set by the context action.
    }
  };

  return (
    <div
      ref={ref}
      role="menu"
      aria-label="Message actions"
      data-message-context-menu=""
      className="bg-popover text-popover-foreground fixed z-50 w-52 rounded-lg border p-1 text-xs shadow-lg"
      style={style}
    >
      <div className="flex items-center justify-between gap-1 px-1.5 py-1">
        <span className="text-muted-foreground text-[10px] font-medium">Reactions</span>
        {message.reactions && message.reactions.length > 0 && (
          <span className="text-muted-foreground text-[10px] tabular-nums">
            {message.reactions.reduce((sum, g) => sum + g.count, 0)}
          </span>
        )}
      </div>

      <div className="grid grid-cols-4 gap-0.5 px-0.5 pb-1">
        {REACTIONS.map((emoji) => {
          const mine =
            currentUser !== null &&
            (message.reactions ?? []).some(
              (g) => g.emoji === emoji && g.user_ids.includes(currentUser.user_id),
            );
          return (
            <button
              key={emoji}
              type="button"
              role="menuitem"
              aria-label={`React with ${emoji}`}
              aria-pressed={mine}
              disabled={pending}
              className={
                "hover:bg-muted flex size-8 items-center justify-center rounded-md text-base transition-colors" +
                (mine ? " bg-muted" : "")
              }
              onClick={() => void run(() => toggleReaction(message.id, emoji))}
            >
              {emoji}
            </button>
          );
        })}
      </div>

      <Separator />

      <MenuItem
        icon={<Reply className="size-3.5" />}
        label="Reply"
        disabled={pending}
        onClick={() => {
          setReplyingToMessage(message);
          onClose();
        }}
      />

      <MenuItem
        icon={isPinned ? <PinOff className="size-3.5" /> : <Pin className="size-3.5" />}
        label={isPinned ? "Unpin" : "Pin"}
        disabled={pending}
        onClick={() =>
          void run(() => (isPinned ? unpinMessage(message.id) : pinMessage(message.id)))
        }
      />

      <MenuItem
        icon={<Eye className="size-3.5" />}
        label="Read receipts"
        disabled={pending || message.recalled}
        hint={readUsers.length > 0 ? String(readUsers.length) : undefined}
        onClick={() => void handleReadUsers()}
      />

      <MenuItem
        icon={<Copy className="size-3.5" />}
        label={copied ? "Copied" : "Copy text"}
        disabled={pending || message.recalled || previewText === ""}
        onClick={() => {
          void navigator.clipboard?.writeText(previewText);
          setCopied(true);
          onClose();
        }}
      />

      {readUsers.length > 0 && (
        <div className="text-muted-foreground px-2 py-1 text-[10px]">
          Read by {readUsers.length}: {readUsers.slice(0, 3).join(", ")}
          {readUsers.length > 3 ? "…" : ""}
        </div>
      )}

      {canRecall && (
        <>
          <Separator />
          <MenuItem
            icon={<Trash2 className="size-3.5" />}
            label="Recall"
            destructive
            disabled={pending}
            onClick={() => void run(() => recallMessage(message.id))}
          />
        </>
      )}
    </div>
  );
}

function MenuItem(props: {
  readonly icon: ReactNode;
  readonly label: string;
  readonly onClick: () => void;
  readonly disabled?: boolean;
  readonly destructive?: boolean;
  readonly hint?: string;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      disabled={props.disabled}
      onClick={props.onClick}
      className={
        "flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-left transition-colors disabled:pointer-events-none disabled:opacity-50" +
        (props.destructive
          ? " text-destructive hover:bg-destructive/10"
          : " hover:bg-accent hover:text-accent-foreground")
      }
    >
      {props.icon}
      <span className="flex-1">{props.label}</span>
      {props.hint && <span className="text-muted-foreground tabular-nums">{props.hint}</span>}
    </button>
  );
}

function extractText(message: UIMessage): string {
  if (isTextPayload(message.payload)) {
    return message.payload.text;
  }
  if (typeof message.payload === "string") {
    return message.payload;
  }
  return "";
}
