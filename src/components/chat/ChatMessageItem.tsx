import { MessageType } from "go-chat-sdk";
import { Check, CheckCheck, Clock, AlertCircle, FileText, Download, Reply } from "lucide-react";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { cn } from "@/lib/utils";
import {
  type UIMessage,
  isTextPayload,
  isImagePayload,
  isFilePayload,
  formatTime,
  formatFileSize,
  getInitials,
  getMessagePreviewText,
} from "@/types/chat";

import { MessageActions, MessageReactions } from "./MessageActions";
import { MessageContextMenu } from "./MessageContextMenu";

interface ChatMessageItemProps {
  readonly message: UIMessage;
  readonly isSelf: boolean;
  readonly showAvatar?: boolean;
}

export function ChatMessageItem({ message, isSelf, showAvatar = true }: ChatMessageItemProps) {
  const { activeRoom, messages, currentUser, setReplyingToMessage } = useChat();
  const [actionsOpen, setActionsOpen] = useState(false);
  const [menuAnchor, setMenuAnchor] = useState<{ x: number; y: number } | null>(null);
  const isGroup = activeRoom ? activeRoom.chat_type === "group" : false;

  // Clicking anywhere else collapses the action bar again.
  useEffect(() => {
    if (!actionsOpen) {
      return;
    }
    const close = (event: globalThis.MouseEvent) => {
      const target = event.target;
      if (target instanceof Element && target.closest("[data-message-actions]")) {
        return;
      }
      setActionsOpen(false);
    };
    document.addEventListener("click", close);
    return () => {
      document.removeEventListener("click", close);
    };
  }, [actionsOpen]);

  // System notice
  if (message.msgType === MessageType.System || message.msgType === "system") {
    let systemText = "System notice";
    if (isTextPayload(message.payload)) {
      systemText = message.payload.text;
    } else if (typeof message.payload === "string") {
      systemText = message.payload;
    }
    return (
      <div className="my-3 flex justify-center select-none">
        <span className="bg-muted/80 dark:bg-muted/40 text-muted-foreground rounded-full px-3 py-0.5 text-xs font-normal">
          {systemText}
        </span>
      </div>
    );
  }

  // Recalled message notice (QQ style centered notice)
  if (message.recalled) {
    const senderDisplay = isSelf ? "你" : message.senderId.slice(0, 8) || "对方";
    return (
      <div className="my-3 flex justify-center select-none">
        <span className="bg-muted/60 dark:bg-muted/30 text-muted-foreground rounded-full px-3 py-1 text-xs font-normal">
          {senderDisplay} 撤回了一条消息
        </span>
      </div>
    );
  }

  const renderStatus = () => {
    if (!isSelf) {
      return null;
    }
    switch (message.status) {
      case "sending":
        return (
          <Tooltip>
            <TooltipTrigger>
              <Clock className="text-muted-foreground size-3 animate-spin" />
            </TooltipTrigger>
            <TooltipContent>Sending...</TooltipContent>
          </Tooltip>
        );
      case "error":
        return (
          <Tooltip>
            <TooltipTrigger>
              <AlertCircle className="text-destructive size-3" />
            </TooltipTrigger>
            <TooltipContent>{message.errorMessage ?? "Failed to send message"}</TooltipContent>
          </Tooltip>
        );
      case "sent":
      default:
        return (
          <Tooltip>
            <TooltipTrigger>
              {message.roomSeq > 0 ? (
                <CheckCheck className="size-3 text-emerald-500" />
              ) : (
                <Check className="text-muted-foreground size-3" />
              )}
            </TooltipTrigger>
            <TooltipContent>
              {message.roomSeq > 0 ? `Delivered (#${message.roomSeq})` : "Sent"}
            </TooltipContent>
          </Tooltip>
        );
    }
  };

  const renderBody = () => {
    if (message.msgType === MessageType.Image || message.msgType === "image") {
      if (isImagePayload(message.payload)) {
        return (
          <div
            className={cn(
              "overflow-hidden rounded-2xl shadow-xs border border-border/40 transition-transform active:scale-[0.99]",
              isSelf ? "rounded-tr-xs bg-[#0099ff]/10" : "rounded-tl-xs bg-card",
            )}
          >
            <img
              src={message.payload.url}
              alt="Sent attachment"
              className="max-h-72 max-w-xs rounded-xl object-cover transition-opacity hover:opacity-95 sm:max-w-sm"
              loading="lazy"
            />
          </div>
        );
      }
    }

    if (message.msgType === MessageType.File || message.msgType === "file") {
      if (isFilePayload(message.payload)) {
        return (
          <div
            className={cn(
              "flex items-center gap-3 rounded-2xl p-3 border shadow-xs max-w-xs sm:max-w-sm transition-all",
              isSelf
                ? "rounded-tr-xs bg-primary/10 border-primary/25 text-foreground"
                : "rounded-tl-xs bg-card border-border/80 text-foreground",
            )}
          >
            <div className="bg-primary/10 text-primary flex size-10 shrink-0 items-center justify-center rounded-xl">
              <FileText className="size-5" />
            </div>
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-xs leading-tight font-semibold">
                {message.payload.name}
              </span>
              <span className="text-muted-foreground mt-0.5 text-[11px]">
                {formatFileSize(message.payload.size)}
              </span>
            </div>
            <a
              href={message.payload.url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:bg-muted text-muted-foreground hover:text-foreground shrink-0 rounded-md p-1.5 transition-colors"
              title="Download file"
              onClick={(e) => e.stopPropagation()}
            >
              <Download className="size-4" />
            </a>
          </div>
        );
      }
    }

    // Default to Text
    let text = "";
    if (isTextPayload(message.payload)) {
      text = message.payload.text;
    } else if (typeof message.payload === "string") {
      text = message.payload;
    } else {
      text = JSON.stringify(message.payload);
    }

    return (
      <div
        className={cn(
          "rounded-2xl px-3.5 py-2.5 text-sm leading-relaxed shadow-xs whitespace-pre-wrap break-words [word-break:break-word] select-text",
          isSelf
            ? "rounded-tr-xs bg-[#0099ff] text-white hover:bg-[#008de6]"
            : "rounded-tl-xs bg-card border border-border/70 dark:border-zinc-700/60 text-foreground hover:bg-card/90",
        )}
      >
        {text}
      </div>
    );
  };

  const quotedMessage = message.replyToMsgId
    ? (messages.find(
        (m) => m.id === message.replyToMsgId || m.clientMsgId === message.replyToMsgId,
      ) ?? null)
    : null;

  const handleScrollToOriginal = () => {
    if (!message.replyToMsgId) {
      return;
    }
    const elem =
      document.getElementById(`msg-${message.replyToMsgId}`) ||
      (quotedMessage?.clientMsgId
        ? document.getElementById(`msg-${quotedMessage.clientMsgId}`)
        : null) ||
      (quotedMessage?.id ? document.getElementById(`msg-${quotedMessage.id}`) : null);
    if (elem) {
      elem.scrollIntoView({ behavior: "smooth", block: "center" });
      elem.classList.add("ring-2", "ring-primary/40", "bg-primary/5");
      setTimeout(() => {
        elem.classList.remove("ring-2", "ring-primary/40", "bg-primary/5");
      }, 1500);
    }
  };

  const renderQuotedMessage = () => {
    if (!message.replyToMsgId) {
      return null;
    }

    const quotedText = quotedMessage ? getMessagePreviewText(quotedMessage) : "Original message";
    const isQuotedSelf = currentUser !== null && quotedMessage?.senderId === currentUser.user_id;
    const quotedSender = quotedMessage
      ? isQuotedSelf
        ? "You"
        : quotedMessage.senderId.slice(0, 8)
      : "Message";

    return (
      <button
        type="button"
        onClick={handleScrollToOriginal}
        title="Click to locate original message"
        className={cn(
          "mb-1.5 flex max-w-full items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs transition-colors",
          isSelf
            ? "border-r-2 border-[#0099ff] bg-[#0099ff]/10 hover:bg-[#0099ff]/20 text-right self-end"
            : "border-l-2 border-primary/70 bg-muted/60 hover:bg-muted/90 text-left self-start",
        )}
      >
        <Reply className={cn("size-3 text-primary shrink-0", !isSelf && "rotate-180")} />
        <span className="text-foreground/80 shrink-0 text-[11px] font-semibold">
          {quotedSender}:
        </span>
        <span className="text-muted-foreground truncate text-[11px]">{quotedText}</span>
      </button>
    );
  };

  const formattedTime = formatTime(message.serverTime);
  const initials = isSelf
    ? currentUser
      ? getInitials(currentUser.username)
      : "ME"
    : getInitials(message.senderId);

  return (
    <div
      id={`msg-${message.id || message.clientMsgId}`}
      className={cn(
        "group relative my-3 flex items-start gap-2.5 px-3 sm:px-5 transition-colors duration-200",
        isSelf ? "flex-row-reverse" : "flex-row",
      )}
    >
      {/* Avatar */}
      {showAvatar && (
        <Avatar className="ring-border/50 size-9 shrink-0 shadow-2xs ring-1">
          <AvatarFallback
            className={cn(
              "text-xs font-semibold select-none",
              isSelf
                ? "bg-[#0099ff]/15 text-[#0099ff] dark:bg-primary/20 dark:text-primary"
                : "bg-muted text-muted-foreground",
            )}
          >
            {initials}
          </AvatarFallback>
        </Avatar>
      )}

      {/* Main message column */}
      <div
        className={cn(
          "flex flex-col min-w-0 max-w-[75%] sm:max-w-[70%]",
          isSelf ? "items-end" : "items-start",
        )}
      >
        {/* Nickname (shown only for other users in group chat) */}
        {!isSelf && isGroup && (
          <span className="text-muted-foreground mb-1 px-1 text-[12px] font-medium select-none">
            {message.senderId.slice(0, 8)}
          </span>
        )}

        {renderQuotedMessage()}

        {/* Bubble container with hover action toolbar */}
        <div className="group/bubble relative flex items-center">
          {/* Action Toolbar on hover */}
          <div
            className={cn(
              "absolute top-1/2 -translate-y-1/2 z-20 transition-all duration-150",
              isSelf ? "right-[calc(100%+8px)]" : "left-[calc(100%+8px)]",
              actionsOpen
                ? "opacity-100 pointer-events-auto"
                : "opacity-0 group-hover/bubble:opacity-100 pointer-events-none group-hover/bubble:pointer-events-auto",
            )}
          >
            <div
              className="bg-background/95 flex items-center rounded-full border px-1 py-0.5 shadow-md backdrop-blur-xs"
              data-message-actions=""
            >
              <MessageActions
                message={message}
                isSelf={isSelf}
                onDismiss={() => setActionsOpen(false)}
              />
            </div>
          </div>

          {/* Interactive bubble */}
          <button
            type="button"
            aria-label="Message actions"
            className="block cursor-pointer text-left transition-transform focus-visible:outline-none active:scale-[0.99]"
            onClick={() => {
              setActionsOpen((prev) => !prev);
            }}
            onDoubleClick={() => {
              setReplyingToMessage(message);
            }}
            onContextMenu={(event) => {
              event.preventDefault();
              setMenuAnchor({ x: event.clientX, y: event.clientY });
            }}
          >
            {renderBody()}
          </button>
        </div>

        {/* Right-click Context Menu */}
        {menuAnchor &&
          createPortal(
            <MessageContextMenu
              message={message}
              isSelf={isSelf}
              anchor={menuAnchor}
              onClose={() => {
                setMenuAnchor(null);
              }}
            />,
            document.body,
          )}

        <MessageReactions message={message} />

        {/* Footer timestamp & status */}
        <div
          className={cn(
            "flex items-center gap-1.5 mt-1 px-1 text-[11px] text-muted-foreground select-none",
            isSelf ? "justify-end" : "justify-start",
          )}
        >
          {formattedTime && <span>{formattedTime}</span>}
          {message.roomSeq > 0 && <span>#{message.roomSeq}</span>}
          {message.pinned && <span className="font-medium text-[#0099ff]">pinned</span>}
          {renderStatus()}
        </div>
      </div>
    </div>
  );
}
