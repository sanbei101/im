import { MessageType } from "go-chat-sdk";
import { Check, CheckCheck, Clock, AlertCircle, FileText, Download, Reply } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Message,
  MessageAvatar,
  MessageContent,
  MessageHeader,
  MessageFooter,
} from "@/components/ui/message";
import { Bubble, BubbleContent } from "@/components/ui/bubble";
import {
  Attachment,
  AttachmentMedia,
  AttachmentContent,
  AttachmentTitle,
  AttachmentDescription,
  AttachmentActions,
} from "@/components/ui/attachment";
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
import { useChat } from "@/context/ChatContext";

interface ChatMessageItemProps {
  readonly message: UIMessage;
  readonly isSelf: boolean;
  readonly showAvatar?: boolean;
}

export function ChatMessageItem({ message, isSelf, showAvatar = true }: ChatMessageItemProps) {
  const { activeRoom, messages, currentUser, setReplyingToMessage } = useChat();
  const isGroup = activeRoom ? activeRoom.chat_type === "group" : false;

  if (message.msgType === MessageType.System || message.msgType === "system") {
    let systemText = "System notice";
    if (isTextPayload(message.payload)) {
      systemText = message.payload.text;
    } else if (typeof message.payload === "string") {
      systemText = message.payload;
    }
    return (
      <div className="my-2 flex justify-center">
        <Badge
          variant="secondary"
          className="text-muted-foreground px-2.5 py-0.5 text-xs font-normal"
        >
          {systemText}
        </Badge>
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
          <Bubble variant={isSelf ? "default" : "muted"}>
            <BubbleContent className="p-1 rounded-xl overflow-hidden">
              <img
                src={message.payload.url}
                alt="Sent attachment"
                className="max-h-60 rounded-lg object-cover transition-opacity hover:opacity-90"
                loading="lazy"
              />
            </BubbleContent>
          </Bubble>
        );
      }
    }

    if (message.msgType === MessageType.File || message.msgType === "file") {
      if (isFilePayload(message.payload)) {
        return (
          <Attachment size="sm" className="max-w-xs">
            <AttachmentMedia variant="icon">
              <FileText className="size-4" />
            </AttachmentMedia>
            <AttachmentContent>
              <AttachmentTitle className="truncate max-w-44 text-xs">
                {message.payload.name}
              </AttachmentTitle>
              <AttachmentDescription className="text-[11px]">
                {formatFileSize(message.payload.size)}
              </AttachmentDescription>
            </AttachmentContent>
            <AttachmentActions>
              <a
                href={message.payload.url}
                target="_blank"
                rel="noopener noreferrer"
                className="hover:bg-muted text-muted-foreground hover:text-foreground rounded-md p-1.5"
                title="Download file"
              >
                <Download className="size-3.5" />
              </a>
            </AttachmentActions>
          </Attachment>
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
      <Bubble variant={isSelf ? "default" : "muted"}>
        <BubbleContent className="whitespace-pre-wrap break-words leading-relaxed text-sm">
          {text}
        </BubbleContent>
      </Bubble>
    );
  };

  const quotedMessage = message.replyToMsgId
    ? messages.find(
        (m) => m.id === message.replyToMsgId || m.clientMsgId === message.replyToMsgId,
      ) ?? null
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

    const quotedText = quotedMessage
      ? getMessagePreviewText(quotedMessage)
      : "Original message";
    const isQuotedSelf = currentUser !== null && quotedMessage?.senderId === currentUser.user_id;
    const quotedSender = quotedMessage
      ? (isQuotedSelf ? "You" : quotedMessage.senderId.slice(0, 8))
      : "Message";

    return (
      <button
        type="button"
        onClick={handleScrollToOriginal}
        title="Click to locate original message"
        className="mb-1 flex max-w-full items-center gap-1.5 rounded-r-md border-l-2 border-primary/70 bg-muted/50 hover:bg-muted/80 px-2 py-1 text-left text-xs transition-colors"
      >
        <Reply className="size-3 text-primary shrink-0 rotate-180" />
        <span className="font-semibold text-foreground/80 shrink-0 text-[11px]">
          {quotedSender}:
        </span>
        <span className="truncate text-muted-foreground text-[11px]">
          {quotedText}
        </span>
      </button>
    );
  };

  const formattedTime = formatTime(message.serverTime);
  const initials = getInitials(message.senderId);

  return (
    <Message
      id={`msg-${message.id || message.clientMsgId}`}
      align={isSelf ? "end" : "start"}
      className="group relative my-1.5 rounded-lg p-0.5 transition-colors duration-300"
    >
      {!isSelf && showAvatar && (
        <MessageAvatar className="size-8">
          <Avatar className="size-8">
            <AvatarFallback className="bg-secondary text-secondary-foreground text-xs font-medium">
              {initials}
            </AvatarFallback>
          </Avatar>
        </MessageAvatar>
      )}

      <MessageContent className="gap-0.5">
        {!isSelf && isGroup && (
          <MessageHeader className="text-[11px] text-muted-foreground px-1 mb-0.5 font-mono">
            {message.senderId.slice(0, 8)}
          </MessageHeader>
        )}

        <div className="relative">
          {renderQuotedMessage()}
          <div
            onDoubleClick={() => {
              setReplyingToMessage(message);
            }}
          >
            {renderBody()}
          </div>

          {/* Quick Action Bar on Hover */}
          <div
            className={
              isSelf
                ? "absolute top-1 left-0 -translate-x-full pr-1.5 opacity-0 group-hover:opacity-100 transition-opacity hidden sm:flex items-center z-10"
                : "absolute top-1 right-0 translate-x-full pl-1.5 opacity-0 group-hover:opacity-100 transition-opacity hidden sm:flex items-center z-10"
            }
          >
            <div className="bg-background/95 border shadow-2xs rounded-md p-0.5 flex items-center backdrop-blur-xs">
              <Tooltip>
                <TooltipTrigger>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    onClick={() => {
                      setReplyingToMessage(message);
                    }}
                    className="size-6 text-muted-foreground hover:text-foreground"
                    title="Reply"
                  >
                    <Reply className="size-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top">Reply</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </div>

        <MessageFooter className="gap-1.5 px-1 text-[10px] text-muted-foreground select-none">
          {formattedTime && <span>{formattedTime}</span>}
          {message.roomSeq > 0 && <span>#{message.roomSeq}</span>}
          {renderStatus()}
        </MessageFooter>
      </MessageContent>
    </Message>
  );
}
