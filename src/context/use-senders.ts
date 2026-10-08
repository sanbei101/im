import { useCallback } from "react";
import type { MessageType as SdkMessageType } from "go-chat-sdk";

import type { ChatDomainDeps } from "./domain-deps";
import { updateMessageAck, updateMessageStatus, type UIMessage } from "@/types/chat";

/** What a sender needs: the SDK call plus the type/payload the bubble renders. */
export interface OutgoingSend {
  (clientMsgId: string): Promise<{ msg_id: string; room_seq: number; server_time: number }>;
}

export interface UseSendersOptions extends ChatDomainDeps {
  readonly sdk: {
    generateMessageId: () => string;
    sendTextMessage: (req: {
      room_id: string;
      client_msg_id: string;
      text: string;
      reply_to_msg_id?: string;
    }) => Promise<{ msg_id: string; room_seq: number; server_time: number }>;
    sendImageMessage: (req: {
      room_id: string;
      client_msg_id: string;
      url: string;
      width?: number;
      height?: number;
      size?: number;
      reply_to_msg_id?: string;
    }) => Promise<{ msg_id: string; room_seq: number; server_time: number }>;
    sendVideoMessage: (req: {
      room_id: string;
      client_msg_id: string;
      url: string;
      duration?: number;
      width?: number;
      height?: number;
      size?: number;
      thumbnail_url?: string;
      reply_to_msg_id?: string;
    }) => Promise<{ msg_id: string; room_seq: number; server_time: number }>;
    sendFileMessage: (req: {
      room_id: string;
      client_msg_id: string;
      url: string;
      name: string;
      size: number;
      mime_type?: string;
      reply_to_msg_id?: string;
    }) => Promise<{ msg_id: string; room_seq: number; server_time: number }>;
  };
  readonly currentUser: { readonly user_id: string } | null;
  readonly replyingToMessage: UIMessage | null;
  readonly setReplyingToMessage: (value: UIMessage | null) => void;
  readonly setMessagesByRoom: (
    fn: (
      prev: Readonly<Record<string, readonly UIMessage[]>>,
    ) => Record<string, readonly UIMessage[]>,
  ) => void;
  /** Called after a successful send so unread/preview stay current. */
  readonly onSent: () => void;
}

/** Result of one send: the shared pipeline plus the typed wrappers. */
export interface UseSendersResult {
  readonly sendMessage: (
    msgType: SdkMessageType | string,
    payload: unknown,
    send: OutgoingSend,
    fallbackError: string,
  ) => Promise<void>;
  readonly sendTextMessage: (text: string, explicitReplyToId?: string) => Promise<void>;
  readonly sendImageMessage: (
    url: string,
    width?: number,
    height?: number,
    size?: number,
    explicitReplyToId?: string,
  ) => Promise<void>;
  readonly sendVideoMessage: (
    url: string,
    duration?: number,
    width?: number,
    height?: number,
    size?: number,
    thumbnailUrl?: string,
    explicitReplyToId?: string,
  ) => Promise<void>;
  readonly sendFileMessage: (
    url: string,
    name: string,
    size: number,
    mimeType?: string,
    explicitReplyToId?: string,
  ) => Promise<void>;
}

/**
 * Outgoing messages all share one optimistic/ack pipeline: append a local
 * echo, send, then swap in the server fields on success or flag the error.
 */
export function useSenders(options: UseSendersOptions): UseSendersResult {
  const {
    sdk,
    activeRoomId,
    currentUser,
    replyingToMessage,
    setReplyingToMessage,
    setMessagesByRoom,
    onSent,
  } = options;

  const sendMessage = useCallback(
    async (msgType: SdkMessageType | string, payload: unknown, send: OutgoingSend, fallbackError: string) => {
      if (!activeRoomId || !currentUser) {
        return;
      }
      const targetReplyId = replyingToMessage?.id ?? replyingToMessage?.clientMsgId;
      const clientMsgId = sdk.generateMessageId();
      const optimisticMsg: UIMessage = {
        id: clientMsgId,
        clientMsgId,
        senderId: currentUser.user_id,
        roomId: activeRoomId,
        roomSeq: 0,
        serverTime: Date.now(),
        msgType,
        payload,
        replyToMsgId: targetReplyId,
        status: "sending",
      };

      setMessagesByRoom((prev) => ({
        ...prev,
        [activeRoomId]: [...(prev[activeRoomId] ?? []), optimisticMsg],
      }));
      setReplyingToMessage(null);

      try {
        const ack = await send(clientMsgId);
        setMessagesByRoom((prev) => ({
          ...prev,
          [activeRoomId]: (prev[activeRoomId] ?? []).map((m) =>
            m.clientMsgId === clientMsgId
              ? updateMessageAck(m, ack.msg_id, ack.room_seq, ack.server_time)
              : m,
          ),
        }));
        onSent();
      } catch (err) {
        const msg = err instanceof Error ? err.message : fallbackError;
        setMessagesByRoom((prev) => ({
          ...prev,
          [activeRoomId]: (prev[activeRoomId] ?? []).map((m) =>
            m.clientMsgId === clientMsgId ? updateMessageStatus(m, "error", msg) : m,
          ),
        }));
      }
    },
    [
      activeRoomId,
      currentUser,
      replyingToMessage,
      sdk,
      setMessagesByRoom,
      setReplyingToMessage,
      onSent,
    ],
  );

  const sendTextMessage = useCallback(
    async (text: string, explicitReplyToId?: string) => {
      const replyTo = explicitReplyToId ?? replyingToMessage?.id ?? replyingToMessage?.clientMsgId;
      await sendMessage(
        "text",
        { text },
        (clientMsgId) =>
          sdk.sendTextMessage({
            room_id: activeRoomId as string,
            client_msg_id: clientMsgId,
            text,
            reply_to_msg_id: replyTo,
          }),
        "Failed to send message",
      );
    },
    [sendMessage, sdk, activeRoomId, replyingToMessage],
  );

  const sendImageMessage = useCallback(
    async (url: string, width?: number, height?: number, size?: number, explicitReplyToId?: string) => {
      const replyTo = explicitReplyToId ?? replyingToMessage?.id ?? replyingToMessage?.clientMsgId;
      await sendMessage(
        "image",
        { url, width, height, size },
        (clientMsgId) =>
          sdk.sendImageMessage({
            room_id: activeRoomId as string,
            client_msg_id: clientMsgId,
            url,
            width,
            height,
            size,
            reply_to_msg_id: replyTo,
          }),
        "Failed to send image",
      );
    },
    [sendMessage, sdk, activeRoomId, replyingToMessage],
  );

  const sendVideoMessage = useCallback(
    async (
      url: string,
      duration?: number,
      width?: number,
      height?: number,
      size?: number,
      thumbnailUrl?: string,
      explicitReplyToId?: string,
    ) => {
      const replyTo = explicitReplyToId ?? replyingToMessage?.id ?? replyingToMessage?.clientMsgId;
      await sendMessage(
        "video",
        { url, duration, width, height, size, thumbnail_url: thumbnailUrl },
        (clientMsgId) =>
          sdk.sendVideoMessage({
            room_id: activeRoomId as string,
            client_msg_id: clientMsgId,
            url,
            duration,
            width,
            height,
            size,
            thumbnail_url: thumbnailUrl,
            reply_to_msg_id: replyTo,
          }),
        "Failed to send video",
      );
    },
    [sendMessage, sdk, activeRoomId, replyingToMessage],
  );

  const sendFileMessage = useCallback(
    async (
      url: string,
      name: string,
      size: number,
      mimeType?: string,
      explicitReplyToId?: string,
    ) => {
      const replyTo = explicitReplyToId ?? replyingToMessage?.id ?? replyingToMessage?.clientMsgId;
      await sendMessage(
        "file",
        { url, name, size, mime_type: mimeType },
        (clientMsgId) =>
          sdk.sendFileMessage({
            room_id: activeRoomId as string,
            client_msg_id: clientMsgId,
            url,
            name,
            size,
            mime_type: mimeType,
            reply_to_msg_id: replyTo,
          }),
        "Failed to send file",
      );
    },
    [sendMessage, sdk, activeRoomId, replyingToMessage],
  );

  return { sendMessage, sendTextMessage, sendImageMessage, sendVideoMessage, sendFileMessage };
}
