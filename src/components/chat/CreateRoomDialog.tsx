import { UserPlus, Users, MessageSquarePlus, AlertCircle } from "lucide-react";
import { useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useChat } from "@/context/ChatContext";
import { isErrorWithMessage } from "@/types/chat";

interface CreateRoomDialogProps {
  readonly trigger?: ReactNode;
}

export function CreateRoomDialog({ trigger }: CreateRoomDialogProps) {
  const { createSingleRoom, createGroupRoom, currentUser } = useChat();
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<string>("single");

  // Single chat form
  const [targetUserId, setTargetUserId] = useState("");

  // Group chat form
  const [groupName, setGroupName] = useState("");
  const [membersText, setMembersText] = useState("");

  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const resetForm = () => {
    setTargetUserId("");
    setGroupName("");
    setMembersText("");
    setErrorMessage(null);
    setLoading(false);
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      resetForm();
    }
    setOpen(nextOpen);
  };

  const handleCreateSingle = async () => {
    const trimmedId = targetUserId.trim();
    if (!trimmedId) {
      setErrorMessage("Please enter a target User ID.");
      return;
    }
    if (currentUser && trimmedId === currentUser.user_id) {
      setErrorMessage("You cannot create a direct chat with yourself.");
      return;
    }

    setLoading(true);
    setErrorMessage(null);
    try {
      await createSingleRoom(trimmedId);
      setOpen(false);
      resetForm();
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to create direct conversation";
      setErrorMessage(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateGroup = async () => {
    const rawMembers = membersText
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter((s) => s.length > 0);

    // Dedup and remove current user if present (current user is added by backend)
    const uniqueMembers = Array.from(new Set(rawMembers)).filter(
      (id) => !currentUser || id !== currentUser.user_id,
    );

    if (uniqueMembers.length < 2) {
      setErrorMessage("Please provide at least 2 other member User IDs for a group room.");
      return;
    }

    setLoading(true);
    setErrorMessage(null);
    try {
      await createGroupRoom(groupName.trim(), uniqueMembers);
      setOpen(false);
      resetForm();
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Failed to create group conversation";
      setErrorMessage(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger>
        {trigger ?? (
          <Button size="sm" className="w-full gap-1.5">
            <MessageSquarePlus className="size-4" />
            <span>New Conversation</span>
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Start Conversation</DialogTitle>
          <DialogDescription>
            Create a one-on-one direct chat or start a group conversation.
          </DialogDescription>
        </DialogHeader>

        <Tabs value={tab} onValueChange={setTab} className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="single" className="gap-1.5">
              <UserPlus className="size-3.5" />
              <span>Direct Chat</span>
            </TabsTrigger>
            <TabsTrigger value="group" className="gap-1.5">
              <Users className="size-3.5" />
              <span>Group Chat</span>
            </TabsTrigger>
          </TabsList>

          {errorMessage && (
            <div className="bg-destructive/10 text-destructive mt-3 flex items-center gap-2 rounded-lg p-2.5 text-xs">
              <AlertCircle className="size-4 shrink-0" />
              <span>{errorMessage}</span>
            </div>
          )}

          <TabsContent value="single" className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium">Target User ID</label>
              <Input
                value={targetUserId}
                onChange={(e) => {
                  setTargetUserId(e.target.value);
                }}
                placeholder="e.g. 018f3a5c-7890-..."
              />
              <span className="text-muted-foreground text-[11px]">
                Enter the user ID of the person you want to message.
              </span>
            </div>

            <DialogFooter className="mt-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => {
                  setOpen(false);
                }}
                disabled={loading}
              >
                Cancel
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={handleCreateSingle}
                disabled={loading || !targetUserId.trim()}
              >
                {loading ? "Creating..." : "Start Chat"}
              </Button>
            </DialogFooter>
          </TabsContent>

          <TabsContent value="group" className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium">
                Group Name (Optional)
              </label>
              <Input
                value={groupName}
                onChange={(e) => {
                  setGroupName(e.target.value);
                }}
                placeholder="e.g. Frontend Team"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-muted-foreground text-xs font-medium">
                Member User IDs (At least 2 members)
              </label>
              <Textarea
                value={membersText}
                onChange={(e) => {
                  setMembersText(e.target.value);
                }}
                placeholder="Enter member user IDs separated by commas or newlines"
                className="h-20 resize-none font-mono text-xs"
              />
              <span className="text-muted-foreground text-[11px]">
                Separate multiple user IDs with commas or newlines.
              </span>
            </div>

            <DialogFooter className="mt-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => {
                  setOpen(false);
                }}
                disabled={loading}
              >
                Cancel
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={handleCreateGroup}
                disabled={loading || !membersText.trim()}
              >
                {loading ? "Creating..." : "Create Group"}
              </Button>
            </DialogFooter>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
