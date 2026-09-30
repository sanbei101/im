import { MessageType } from "go-chat-sdk";
import { Check, CheckCheck, Clock, AlertCircle, FileText, Download } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
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
} from "@/types/chat";
import { useChat } from "@/context/ChatContext";

interface ChatMessageItemProps {
  readonly message: UIMessage;
  readonly isSelf: boolean;
  readonly showAvatar?: boolean;
}

export function ChatMessageItem({ message, isSelf, showAvatar = true }: ChatMessageItemProps) {
  const { activeRoom } = useChat();
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

  const formattedTime = formatTime(message.serverTime);
  const initials = getInitials(message.senderId);

  return (
    <Message align={isSelf ? "end" : "start"} className="my-1.5">
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

        {renderBody()}

        <MessageFooter className="gap-1.5 px-1 text-[10px] text-muted-foreground select-none">
          {formattedTime && <span>{formattedTime}</span>}
          {message.roomSeq > 0 && <span>#{message.roomSeq}</span>}
          {renderStatus()}
        </MessageFooter>
      </MessageContent>
    </Message>
  );
}
