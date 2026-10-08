import { ConnectionState } from "go-chat-sdk";
import { Send, Image as ImageIcon, Paperclip, Reply, X, Smile, Loader2 } from "lucide-react";
import { useState, useRef, useEffect, type KeyboardEvent, type ClipboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useChat } from "@/context/ChatContext";
import { getMessagePreviewText } from "@/types/chat";

const COMMON_EMOJIS = [
  "😀",
  "😁",
  "😂",
  "🤣",
  "😃",
  "😄",
  "😅",
  "😆",
  "😉",
  "😊",
  "😋",
  "😎",
  "😍",
  "😘",
  "🥰",
  "😗",
  "😚",
  "☺️",
  "🙂",
  "🤗",
  "🤩",
  "🤔",
  "🫡",
  "🤨",
  "😐",
  "😑",
  "😶",
  "🙄",
  "😏",
  "😣",
  "😥",
  "😮",
  "🤐",
  "😯",
  "😪",
  "😫",
  "🥱",
  "😴",
  "😌",
  "😛",
  "😜",
  "😝",
  "🤤",
  "😒",
  "😓",
  "😔",
  "😕",
  "🙃",
  "🫠",
  "🤑",
  "😲",
  "☹️",
  "🙁",
  "😖",
  "😞",
  "😟",
  "😤",
  "😢",
  "😭",
  "😦",
  "😩",
  "🤯",
  "😬",
  "😮‍💨",
  "😰",
  "😱",
  "🥵",
  "🥶",
  "😳",
  "🤪",
  "😵",
  "🥴",
  "😠",
  "😡",
  "🤬",
  "😷",
  "🤒",
  "🤕",
  "🤢",
  "🤮",
  "😇",
  "🥳",
  "🥺",
  "🥹",
  "🤠",
  "🤡",
  "💩",
  "👻",
  "💀",
  "🤖",
  "👋",
  "👌",
  "🤌",
  "✌️",
  "🤞",
  "🫰",
  "🤟",
  "🤘",
  "🤙",
  "👈",
  "👉",
  "👆",
  "👇",
  "👍",
  "👎",
  "✊",
  "👊",
  "👏",
  "🙌",
  "🫶",
  "🤝",
  "🙏",
  "💪",
  "❤️",
  "🧡",
  "💛",
  "💚",
  "💙",
  "💜",
  "🖤",
  "💔",
  "❤️‍🔥",
  "✨",
  "🔥",
  "💯",
  "🎉",
  "🎊",
  "⭐",
  "🌟",
  "☕",
] as const;

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
  const [isUploading, setIsUploading] = useState(false);

  // Popovers
  const [emojiOpen, setEmojiOpen] = useState(false);
  const [imagePopoverOpen, setImagePopoverOpen] = useState(false);
  const [imageUrl, setImageUrl] = useState("");

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Auto focus when replyingToMessage is set
  useEffect(() => {
    if (replyingToMessage && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [replyingToMessage]);

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

  // Support pasting images directly from clipboard (e.g. screenshots)
  const handlePaste = (e: ClipboardEvent<HTMLTextAreaElement>) => {
    const items = e.clipboardData?.items;
    if (!items) return;

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.type.startsWith("image/")) {
        const file = item.getAsFile();
        if (file) {
          e.preventDefault();
          void handleUpload(file, "image");
          return;
        }
      }
    }
  };

  const handleInsertEmoji = (emoji: string) => {
    const textarea = textareaRef.current;
    if (!textarea) {
      setText((prev) => prev + emoji);
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const nextText = text.substring(0, start) + emoji + text.substring(end);
    setText(nextText);
    setEmojiOpen(false);

    // Restore focus and cursor position after insertion
    setTimeout(() => {
      textarea.focus();
      textarea.selectionStart = textarea.selectionEnd = start + emoji.length;
    }, 0);
  };

  const handleSendImageUrl = async () => {
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
    setImageUrl("");
    try {
      const uploaded = await uploadFile(file);
      if (kind === "image") {
        await sendImageMessage(uploaded.url, undefined, undefined, uploaded.size);
      } else {
        await sendFileMessage(uploaded.url, uploaded.name, uploaded.size, file.type);
      }
    } catch {
      // Error is handled in context
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <footer className="bg-background shrink-0 border-t transition-colors select-none">
      {/* Hidden file inputs for toolbar buttons */}
      <input
        ref={imageInputRef}
        type="file"
        accept="image/*"
        aria-label="Upload image"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          e.target.value = "";
          if (file) {
            void handleUpload(file, "image");
          }
        }}
      />
      <input
        ref={fileInputRef}
        type="file"
        aria-label="Upload file"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          e.target.value = "";
          if (file) {
            void handleUpload(file, "file");
          }
        }}
      />

      <div className="flex flex-col">
        {/* Replying banner */}
        {replyingToMessage && (
          <div className="bg-muted/50 flex items-center justify-between border-b px-4 py-1.5 text-xs">
            <div className="flex min-w-0 items-center gap-2">
              <div className="bg-primary h-3.5 w-1 shrink-0 rounded-full" />
              <Reply className="text-primary size-3 shrink-0" />
              <div className="flex min-w-0 items-center gap-1.5">
                <span className="text-foreground shrink-0 text-[11px] font-semibold">
                  回复{" "}
                  {replyingToMessage.senderId === currentUser?.user_id
                    ? "自己"
                    : replyingToMessage.senderId.slice(0, 8)}
                  :
                </span>
                <span className="text-muted-foreground truncate text-[11px]">
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
              className="text-muted-foreground hover:text-foreground size-5 shrink-0"
              title="取消回复"
            >
              <X className="size-3" />
            </Button>
          </div>
        )}

        {/* Top Toolbar (QQ style) */}
        <div className="border-b/30 flex items-center justify-between px-3 pt-2 pb-1">
          <div className="flex items-center gap-0.5">
            {/* Emoji Popover */}
            <Popover open={emojiOpen} onOpenChange={setEmojiOpen}>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground hover:bg-muted size-7"
                  title="表情"
                >
                  <Smile className="size-4" />
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-72 p-2" align="start">
                <div className="grid max-h-52 grid-cols-8 gap-1 overflow-y-auto p-1">
                  {COMMON_EMOJIS.map((emoji) => (
                    <button
                      key={emoji}
                      type="button"
                      onClick={() => handleInsertEmoji(emoji)}
                      className="hover:bg-muted flex size-7 items-center justify-center rounded text-base transition-colors"
                    >
                      {emoji}
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>

            {/* Image Upload Button & URL Popover */}
            <Popover open={imagePopoverOpen} onOpenChange={setImagePopoverOpen}>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground hover:bg-muted size-7"
                  title="发送图片"
                >
                  <ImageIcon className="size-4" />
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-72 p-3" align="start">
                <div className="flex flex-col gap-2.5">
                  <span className="text-xs font-semibold">发送图片</span>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="w-full text-xs"
                    onClick={() => imageInputRef.current?.click()}
                  >
                    从本地选择图片
                  </Button>
                  <div className="relative my-1">
                    <div className="absolute inset-0 flex items-center">
                      <div className="border-border w-full border-t" />
                    </div>
                    <div className="relative flex justify-center text-[10px] uppercase">
                      <span className="bg-popover text-muted-foreground px-2">或输入 URL</span>
                    </div>
                  </div>
                  <Input
                    value={imageUrl}
                    onChange={(e) => setImageUrl(e.target.value)}
                    placeholder="https://..."
                    className="h-8 text-xs"
                  />
                  <Button
                    type="button"
                    size="xs"
                    onClick={() => void handleSendImageUrl()}
                    disabled={!imageUrl.trim() || isUploading}
                    className="w-full"
                  >
                    发送图片链接
                  </Button>
                </div>
              </PopoverContent>
            </Popover>

            {/* File Upload Button */}
            <Tooltip>
              <TooltipTrigger>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isUploading}
                  className="text-muted-foreground hover:text-foreground hover:bg-muted size-7"
                >
                  {isUploading ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Paperclip className="size-4" />
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent>发送文件</TooltipContent>
            </Tooltip>
          </div>

          {/* Quick upload indicator */}
          {isUploading && (
            <span className="text-muted-foreground flex items-center gap-1 text-[11px]">
              <Loader2 className="text-primary size-3 animate-spin" />
              正在上传...
            </span>
          )}
        </div>

        {/* Text input area */}
        <div className="px-3 py-1">
          <Textarea
            ref={textareaRef}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
            placeholder={
              isConnected
                ? "输入消息... (Enter 发送，Shift+Enter 换行，支持截图粘贴)"
                : "网络离线中..."
            }
            className="placeholder:text-muted-foreground/60 h-20 max-h-36 min-h-[60px] resize-none border-0 bg-transparent p-1 text-sm leading-relaxed shadow-none focus-visible:ring-0"
          />
        </div>

        {/* Bottom bar with shortcuts and Send button */}
        <div className="flex items-center justify-between px-3 pt-0.5 pb-2.5">
          <span className="text-muted-foreground/70 hidden text-[11px] sm:inline-block">
            按 Enter 发送，Shift+Enter 换行
          </span>

          <div className="ml-auto flex items-center gap-2">
            <Button
              type="button"
              size="sm"
              onClick={() => void handleSendText()}
              disabled={!text.trim() || isSending}
              className="h-8 gap-1.5 rounded-lg bg-[#0099ff] px-4 text-xs font-medium text-white shadow-xs transition-colors hover:bg-[#008de6] disabled:opacity-40"
            >
              <span>发送</span>
              <Send className="size-3" />
            </Button>
          </div>
        </div>
      </div>
    </footer>
  );
}
