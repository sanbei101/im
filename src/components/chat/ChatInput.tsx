import { ConnectionState } from "go-chat-sdk";
import { Send, Image as ImageIcon, Paperclip, Reply, X } from "lucide-react";
import { useState, useRef, useEffect, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { getMessagePreviewText } from "@/types/chat";

export function ChatInput() {
  const {
    sendTextMessage,
    sendImageMessage,
    sendFileMessage,
    uploadFile,
    connectionState,
    replyingToMessage,
    setReplyingToMessage,
    currentUser,
  } = useChat();
  const [text, setText] = useState("");
  const [isSending, setIsSending] = useState(false);

  // Image popover
  const [imagePopoverOpen, setImagePopoverOpen] = useState(false);
  const [imageUrl, setImageUrl] = useState("");

  // File popover
  const [filePopoverOpen, setFilePopoverOpen] = useState(false);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto focus when replyingToMessage is set
  useEffect(() => {
    if (replyingToMessage && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [replyingToMessage]);

  const isConnected = connectionState === ConnectionState.Connected;

  const [isUploading, setIsUploading] = useState(false);

  const handleSendText = async () => {
    const trimmed = text.trim();
    if (!trimmed || isSending) {
      return;
    }
    setIsSending(true);
    setText("");
    try {
      await sendTextMessage(trimmed);
      if (textareaRef.current) {
        textareaRef.current.focus();
      }
    } finally {
      setIsSending(false);
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void handleSendText();
    }
  };

  // Images and files go through the backend presign endpoint, then a direct
  // PUT to object storage; only the resulting URL is sent over the wire.
  const handleSendImage = async () => {
    const trimmed = imageUrl.trim();
    if (!trimmed) {
      return;
    }
    setImagePopoverOpen(false);
    setImageUrl("");
    await sendImageMessage(trimmed, 400, 300);
  };

  const handleUpload = async (file: File, kind: "image" | "file") => {
    if (isUploading) {
      return;
    }
    setIsUploading(true);
    setImagePopoverOpen(false);
    setFilePopoverOpen(false);
    setImageUrl("");
    try {
      const uploaded = await uploadFile(file);
      if (kind === "image") {
        await sendImageMessage(uploaded.url, undefined, undefined, uploaded.size);
      } else {
        await sendFileMessage(uploaded.url, uploaded.name, uploaded.size, file.type);
      }
    } catch {
      // Error banner is set by the context action.
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <footer className="bg-background/95 shrink-0 border-t p-3 backdrop-blur-xs">
      <div className="flex flex-col gap-2">
        <div className="bg-muted/30 focus-within:border-ring focus-within:ring-ring/20 relative rounded-lg border transition-all focus-within:ring-2">
          {replyingToMessage && (
            <div className="bg-muted/70 border-b flex items-center justify-between px-3 py-1.5 text-xs rounded-t-lg select-none">
              <div className="flex items-center gap-2 min-w-0">
                <div className="w-0.5 h-6 bg-primary rounded-full shrink-0" />
                <Reply className="size-3.5 text-primary shrink-0" />
                <div className="flex flex-col min-w-0">
                  <span className="font-semibold text-[11px] leading-tight text-foreground truncate">
                    Replying to {replyingToMessage.senderId === currentUser?.user_id ? "yourself" : replyingToMessage.senderId.slice(0, 8)}
                  </span>
                  <span className="text-[11px] text-muted-foreground truncate leading-tight">
                    {getMessagePreviewText(replyingToMessage)}
                  </span>
                </div>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                onClick={() => {
                  setReplyingToMessage(null);
                }}
                className="text-muted-foreground hover:text-foreground shrink-0 size-6"
                title="Cancel reply"
              >
                <X className="size-3.5" />
              </Button>
            </div>
          )}

          <Textarea
            ref={textareaRef}
            value={text}
            onChange={(e) => {
              setText(e.target.value);
            }}
            onKeyDown={handleKeyDown}
            placeholder={
              isConnected
                ? "Type a message... (Press Enter to send, Shift+Enter for new line)"
                : "WebSocket offline - messages will queue or reconnect..."
            }
            className="max-h-36 min-h-[60px] resize-none border-0 bg-transparent p-2.5 text-sm shadow-none focus-visible:ring-0"
          />

          <div className="flex items-center justify-between px-2 pb-2">
            <div className="flex items-center gap-1">
              {/* Image message popover */}
              <Popover open={imagePopoverOpen} onOpenChange={setImagePopoverOpen}>
                <PopoverTrigger>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    title="Send Image"
                    className="text-muted-foreground hover:text-foreground"
                  >
                    <ImageIcon className="size-4" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-80 p-3" align="start">
                  <div className="flex flex-col gap-2.5">
                    <span className="text-xs font-semibold">Send Image Message</span>
                    <input
                      type="file"
                      accept="image/*"
                      aria-label="Choose image"
                      className="text-xs"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        e.target.value = "";
                        if (file) {
                          void handleUpload(file, "image");
                        }
                      }}
                    />
                    <span className="text-muted-foreground text-[11px]">
                      Uploaded through the server presign endpoint.
                    </span>
                    <Input
                      value={imageUrl}
                      onChange={(e) => {
                        setImageUrl(e.target.value);
                      }}
                      placeholder="…or paste an image URL"
                      className="h-8 text-xs"
                    />
                    <div className="flex items-center justify-between">
                      <Button
                        type="button"
                        size="xs"
                        onClick={() => {
                          void handleSendImage();
                        }}
                        disabled={!imageUrl.trim() || isUploading}
                      >
                        {isUploading ? "Uploading…" : "Send URL"}
                      </Button>
                    </div>
                  </div>
                </PopoverContent>
              </Popover>

              {/* File message popover */}
              <Popover open={filePopoverOpen} onOpenChange={setFilePopoverOpen}>
                <PopoverTrigger>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    title="Send File"
                    className="text-muted-foreground hover:text-foreground"
                  >
                    <Paperclip className="size-4" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-80 p-3" align="start">
                  <div className="flex flex-col gap-2.5">
                    <span className="text-xs font-semibold">Send File Message</span>
                    <input
                      type="file"
                      aria-label="Choose file"
                      className="text-xs"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        e.target.value = "";
                        if (file) {
                          void handleUpload(file, "file");
                        }
                      }}
                    />
                    <span className="text-muted-foreground text-[11px]">
                      The file name and size are sent with the uploaded URL.
                    </span>
                  </div>
                </PopoverContent>
              </Popover>
            </div>

            <div className="flex items-center gap-2">
              <Tooltip>
                <TooltipTrigger>
                  <Button
                    type="button"
                    size="icon-sm"
                    onClick={() => {
                      void handleSendText();
                    }}
                    disabled={!text.trim() || isSending}
                    className="rounded-lg shadow-xs"
                  >
                    <Send className="size-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <span>Send (Enter)</span>
                </TooltipContent>
              </Tooltip>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}
