import { useState } from "react";
import {
  BellOff,
  BellRing,
  CheckCheck,
  Crown,
  LogOut,
  Pin,
  PinOff,
  Shield,
  Trash2,
  UserPlus,
  UserX,
  X,
} from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useChat } from "@/context/ChatContext";
import { formatTime, getInitials, isTextPayload } from "@/types/chat";

interface RoomDetailsPanelProps {
  readonly onClose: () => void;
}

type RoomTab = "info" | "members" | "pins";

/** Room info, member management and pinned messages for the active room. */
export function RoomDetailsPanel({ onClose }: RoomDetailsPanelProps) {
  const {
    activeRoom,
    activeRoomId,
    activeRoomDetail,
    members,
    isLoadingMembers,
    pinnedMessages,
    refreshRoomDetail,
    updateActiveRoom,
    leaveActiveRoom,
    addMembers,
    updateMemberRole,
    removeMember,
    transferOwnership,
    unpinMessage,
    conversations,
    muteConversation,
    pinConversation,
    clearUnread,
    deleteRoom,
  } = useChat();
  const [tab, setTab] = useState<RoomTab>("info");
  const [nameDraft, setNameDraft] = useState(activeRoom?.name ?? "");
  const [noticeDraft, setNoticeDraft] = useState(activeRoom?.notice ?? "");
  const [memberInput, setMemberInput] = useState("");
  const [busy, setBusy] = useState(false);

  if (!activeRoom || !activeRoomId) {
    return null;
  }

  const isGroup = activeRoom.chat_type === "group";
  const myRole = activeRoomDetail?.my_role ?? "member";
  const isOwner = myRole === "owner";
  const isAdmin = myRole === "admin" || isOwner;
  const ownerId = members.find((m) => m.role === "owner")?.user_id;
  const displayName = activeRoom.name || (isGroup ? "Group Conversation" : "Direct Message");
  const conversation = conversations.find((c) => c.room.room_id === activeRoomId);
  const isMuted = conversation?.member.is_muted ?? false;
  const isConvPinned = conversation?.member.is_pinned ?? false;
  const unread = conversation?.unread_count ?? 0;

  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await action();
    } catch {
      // Error banner is already set by the context action.
    } finally {
      setBusy(false);
    }
  };

  const handleAddMembers = () => {
    const ids = memberInput
      .split(/[\s,]+/)
      .map((s) => s.trim())
      .filter((s) => s !== "");
    if (ids.length === 0) {
      return;
    }
    void run(async () => {
      await addMembers(ids);
      setMemberInput("");
    });
  };

  return (
    <aside className="bg-background/50 flex h-full w-80 shrink-0 flex-col border-l">
      <div className="flex h-14 shrink-0 items-center justify-between border-b px-4">
        <span className="text-sm font-semibold">Conversation Info</span>
        <Button variant="ghost" size="icon-sm" aria-label="Close details" onClick={onClose}>
          <X className="size-4" />
        </Button>
      </div>

      <div className="flex flex-col items-center p-4 text-center">
        <Avatar className="mb-2 size-16">
          <AvatarFallback className="bg-secondary text-lg">{getInitials(displayName)}</AvatarFallback>
        </Avatar>
        <h3 className="max-w-full truncate text-base font-semibold">{displayName}</h3>
        <div className="mt-1 flex items-center gap-1.5">
          <Badge variant="outline">{isGroup ? "Group" : "Direct"}</Badge>
          {isGroup && (
            <Badge variant={isOwner ? "default" : "secondary"}>
              {isOwner ? (
                <>
                  <Crown className="size-3" />
                  Owner
                </>
              ) : myRole === "admin" ? (
                <>
                  <Shield className="size-3" />
                  Admin
                </>
              ) : (
                "Member"
              )}
            </Badge>
          )}
        </div>
      </div>

      <Tabs
        value={tab}
        onValueChange={(next) => setTab(next as RoomTab)}
        className="flex min-h-0 flex-1 flex-col"
      >
        <div className="px-3">
          <TabsList className="w-full">
            <TabsTrigger value="info" className="flex-1">
              Info
            </TabsTrigger>
            <TabsTrigger value="members" className="flex-1">
              Members
            </TabsTrigger>
            <TabsTrigger value="pins" className="flex-1">
              Pins
              {pinnedMessages.length > 0 && <Badge className="ml-1">{pinnedMessages.length}</Badge>}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="info" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-3 p-3 text-xs">
              <div className="flex flex-col gap-1.5">
                <label className="text-muted-foreground font-medium" htmlFor="room-id">
                  Room ID
                </label>
                <code className="bg-muted rounded px-2 py-1 text-[11px] break-all">{activeRoomId}</code>
              </div>

              {isGroup && isAdmin && (
                <>
                  <Separator />
                  <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground font-medium" htmlFor="room-name">
                      Group name
                    </label>
                    <Input
                      id="room-name"
                      value={nameDraft}
                      onChange={(e) => setNameDraft(e.target.value)}
                      placeholder="Group name"
                      className="h-8 text-xs"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground font-medium" htmlFor="room-notice">
                      Notice
                    </label>
                    <Textarea
                      id="room-notice"
                      value={noticeDraft}
                      onChange={(e) => setNoticeDraft(e.target.value)}
                      placeholder="Announcement shown to members"
                      className="min-h-16 text-xs"
                    />
                  </div>
                  <Button
                    size="sm"
                    disabled={busy}
                    onClick={() =>
                      void run(async () => {
                        await updateActiveRoom({
                          name: nameDraft.trim(),
                          notice: noticeDraft.trim(),
                        });
                        await refreshRoomDetail();
                      })
                    }
                  >
                    Save changes
                  </Button>
                </>
              )}

              {!isGroup && activeRoom.notice !== "" && (
                <div className="flex flex-col gap-1.5">
                  <span className="text-muted-foreground font-medium">Notice</span>
                  <p className="bg-muted rounded px-2 py-1 text-[11px]">{activeRoom.notice}</p>
                </div>
              )}

              <Separator />

              {/* Conversation-level settings */}
              <div className="flex flex-col gap-1.5">
                <span className="text-muted-foreground font-medium">Conversation</span>
                <div className="flex flex-wrap gap-1.5">
                  <Button
                    variant={isMuted ? "default" : "outline"}
                    size="sm"
                    disabled={busy}
                    onClick={() => void run(() => muteConversation(activeRoomId, !isMuted))}
                  >
                    {isMuted ? <BellRing className="size-3.5" /> : <BellOff className="size-3.5" />}
                    {isMuted ? "Muted" : "Mute"}
                  </Button>
                  <Button
                    variant={isConvPinned ? "default" : "outline"}
                    size="sm"
                    disabled={busy}
                    onClick={() => void run(() => pinConversation(activeRoomId, !isConvPinned))}
                  >
                    <Pin className="size-3.5" />
                    {isConvPinned ? "Pinned" : "Pin"}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onClick={() => void run(() => clearUnread(activeRoomId))}
                  >
                    <CheckCheck className="size-3.5" />
                    {unread > 0 ? `Mark read (${unread})` : "Mark read"}
                  </Button>
                </div>
              </div>

              <Separator />

              <div className="flex flex-col gap-1.5">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={busy}
                  onClick={() =>
                    void run(async () => {
                      await leaveActiveRoom();
                      onClose();
                    })
                  }
                >
                  <LogOut className="size-3.5" />
                  {isGroup ? "Leave group" : "Leave conversation"}
                </Button>

                {isOwner && (
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={busy}
                    onClick={() =>
                      void run(async () => {
                        await deleteRoom(activeRoomId);
                        onClose();
                      })
                    }
                  >
                    <Trash2 className="size-3.5" />
                    Dissolve room for everyone
                  </Button>
                )}
              </div>
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="members" className="min-h-0 flex-1">
          <div className="flex h-full flex-col">
            {isGroup && isAdmin && (
              <div className="flex shrink-0 items-center gap-1 border-b p-2">
                <div className="relative flex-1">
                  <UserPlus className="text-muted-foreground pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2" />
                  <Input
                    value={memberInput}
                    onChange={(e) => setMemberInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        handleAddMembers();
                      }
                    }}
                    placeholder="User IDs, comma separated"
                    aria-label="Add members"
                    className="h-8 pl-7 text-xs"
                  />
                </div>
                <Button size="sm" disabled={busy} onClick={handleAddMembers}>
                  Add
                </Button>
              </div>
            )}

            <ScrollArea className="min-h-0 flex-1">
              <div className="flex flex-col gap-1 p-2">
                {isLoadingMembers && members.length === 0 && (
                  <div className="flex flex-col gap-2">
                    {[0, 1, 2].map((i) => (
                      <div key={i} className="bg-muted/50 h-12 animate-pulse rounded-md" />
                    ))}
                  </div>
                )}
                {members.map((member) => {
                  const isMe = ownerId !== undefined && member.user_id === ownerId;
                  return (
                    <div
                      key={member.user_id}
                      className="flex items-center gap-2 rounded-md p-2 hover:bg-muted/60"
                    >
                      <Avatar className="size-8">
                        <AvatarFallback className="text-xs">
                          {(member.nickname || member.username).slice(0, 2)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">
                          {member.nickname || member.username}
                        </p>
                        <div className="flex items-center gap-1">
                          <span className="text-muted-foreground truncate text-[11px]">
                            @{member.username}
                          </span>
                          {member.role !== "member" && (
                            <Badge variant="secondary" className="h-4 px-1 text-[10px]">
                              {member.role}
                            </Badge>
                          )}
                        </div>
                      </div>
                      {isAdmin && !isMe && (
                        <div className="flex items-center gap-0.5">
                          {isOwner && (
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label={`Transfer ownership to ${member.username}`}
                              disabled={busy}
                              onClick={() =>
                                void run(() => transferOwnership(member.user_id))
                              }
                            >
                              <Crown className="size-3.5" />
                            </Button>
                          )}
                          {isOwner && member.role === "member" && (
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label={`Promote ${member.username}`}
                              disabled={busy}
                              onClick={() => void run(() => updateMemberRole(member.user_id, "admin"))}
                            >
                              <Shield className="size-3.5" />
                            </Button>
                          )}
                          {isOwner && member.role === "admin" && (
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label={`Demote ${member.username}`}
                              disabled={busy}
                              onClick={() => void run(() => updateMemberRole(member.user_id, "member"))}
                            >
                              <BellOff className="size-3.5" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Remove ${member.username}`}
                            disabled={busy}
                            onClick={() => void run(() => removeMember(member.user_id))}
                          >
                            <UserX className="size-3.5" />
                          </Button>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </div>
        </TabsContent>

        <TabsContent value="pins" className="min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-1 p-2">
              {pinnedMessages.length === 0 && (
                <div className="text-muted-foreground flex flex-col items-center gap-2 py-10 text-center">
                  <div className="bg-muted flex size-10 items-center justify-center rounded-full">
                    <Pin className="size-5" />
                  </div>
                  <p className="text-foreground text-sm font-medium">No pinned messages</p>
                  <p className="text-xs">Pin a message from its hover menu.</p>
                </div>
              )}
              {pinnedMessages.map((msg) => (
                <div
                  key={msg.msg_id}
                  className="group/pin flex flex-col gap-1 rounded-md border p-2 text-xs"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-muted-foreground font-mono text-[10px]">
                      {msg.sender_id.slice(0, 8)} · {formatTime(msg.server_time)}
                    </span>
                    <div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover/pin:opacity-100">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label="Unpin message"
                        disabled={busy}
                        onClick={() => void run(() => unpinMessage(msg.msg_id))}
                      >
                        <PinOff className="size-3.5" />
                      </Button>
                    </div>
                  </div>
                  <p className="line-clamp-3 break-words">
                    {isTextPayload(msg.payload) ? msg.payload.text : JSON.stringify(msg.payload)}
                  </p>
                </div>
              ))}
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>
    </aside>
  );
}
