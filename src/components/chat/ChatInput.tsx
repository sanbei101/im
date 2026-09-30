import { ConnectionState } from "go-chat-sdk";
import { Send, Image as ImageIcon, Paperclip, Sparkles } from "lucide-react";
import { useState, useRef, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";

export function ChatInput() {
  const { sendTextMessage, sendImageMessage, sendFileMessage, connectionState } = useChat();
  const [text, setText] = useState("");
  const [isSending, setIsSending] = useState(false);

  // Image popover
  const [imagePopoverOpen, setImagePopoverOpen] = useState(false);
  const [imageUrl, setImageUrl] = useState("");

  // File popover
  const [filePopoverOpen, setFilePopoverOpen] = useState(false);
  const [fileUrl, setFileUrl] = useState("");
  const [fileName, setFileName] = useState("");

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const isConnected = connectionState === ConnectionState.Connected;

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

  const handleSendImage = async () => {
    const trimmed = imageUrl.trim();
    if (!trimmed) {
      return;
    }
    setImagePopoverOpen(false);
    setImageUrl("");
    await sendImageMessage(trimmed, 400, 300);
  };

  const handleSendFile = async () => {
    const trimmedUrl = fileUrl.trim();
    const trimmedName = fileName.trim() || "document.pdf";
    if (!trimmedUrl) {
      return;
    }
    setFilePopoverOpen(false);
    setFileUrl("");
    setFileName("");
    await sendFileMessage(trimmedUrl, trimmedName, 1024 * 512, "application/octet-stream");
  };

  return (
    <footer className="bg-background/95 shrink-0 border-t p-3 backdrop-blur-xs">
      <div className="flex flex-col gap-2">
        <div className="bg-muted/30 focus-within:border-ring focus-within:ring-ring/20 relative rounded-lg border transition-all focus-within:ring-2">
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
                    <Input
                      value={imageUrl}
                      onChange={(e) => {
                        setImageUrl(e.target.value);
                      }}
                      placeholder="Enter image URL"
                      className="h-8 text-xs"
                    />
                    <div className="flex items-center justify-between">
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        onClick={() => {
                          setImageUrl(
                            "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=500&auto=format&fit=crop",
                          );
                        }}
                        className="text-muted-foreground gap-1 text-[11px]"
                      >
                        <Sparkles className="size-3" /> Sample Photo
                      </Button>
                      <Button
                        type="button"
                        size="xs"
                        onClick={() => {
                          void handleSendImage();
                        }}
                        disabled={!imageUrl.trim()}
                      >
                        Send
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
                    <Input
                      value={fileName}
                      onChange={(e) => {
                        setFileName(e.target.value);
                      }}
                      placeholder="File Name (e.g. project_plan.pdf)"
                      className="h-8 text-xs"
                    />
                    <Input
                      value={fileUrl}
                      onChange={(e) => {
                        setFileUrl(e.target.value);
                      }}
                      placeholder="File URL"
                      className="h-8 text-xs"
                    />
                    <div className="flex items-center justify-between">
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        onClick={() => {
                          setFileName("specification_v1.pdf");
                          setFileUrl(
                            "https://raw.githubusercontent.com/sanbei101/im/dev/README.md",
                          );
                        }}
                        className="text-muted-foreground gap-1 text-[11px]"
                      >
                        <Sparkles className="size-3" /> Sample File
                      </Button>
                      <Button
                        type="button"
                        size="xs"
                        onClick={() => {
                          void handleSendFile();
                        }}
                        disabled={!fileUrl.trim()}
                      >
                        Send
                      </Button>
                    </div>
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
