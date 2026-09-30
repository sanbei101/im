import { Settings, RotateCcw } from "lucide-react";
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
import { useChat } from "@/context/ChatContext";

interface SettingsDialogProps {
  readonly trigger?: ReactNode;
}

export function SettingsDialog({ trigger }: SettingsDialogProps) {
  const { config, updateConfig } = useChat();
  const [open, setOpen] = useState(false);
  const [baseURL, setBaseURL] = useState(config.baseURL);
  const [gatewayURL, setGatewayURL] = useState(config.gatewayURL);

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setBaseURL(config.baseURL);
      setGatewayURL(config.gatewayURL);
    }
    setOpen(nextOpen);
  };

  const handleSave = () => {
    updateConfig({
      baseURL: baseURL.trim(),
      gatewayURL: gatewayURL.trim(),
    });
    setOpen(false);
  };

  const handleReset = () => {
    setBaseURL("http://127.0.0.1:8801");
    setGatewayURL("ws://127.0.0.1:8800/ws");
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger>
        {trigger ?? (
          <Button variant="ghost" size="icon" title="Gateway Settings">
            <Settings className="size-4" />
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Server Configuration</DialogTitle>
          <DialogDescription>
            Configure HTTP API and WebSocket gateway connection endpoints.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 py-2">
          <div className="flex flex-col gap-1.5">
            <label className="text-muted-foreground text-xs font-medium">HTTP API Base URL</label>
            <Input
              value={baseURL}
              onChange={(e) => {
                setBaseURL(e.target.value);
              }}
              placeholder="http://127.0.0.1:8801"
            />
            <span className="text-muted-foreground text-[11px]">
              Handles user registration, authentication, room management, and history.
            </span>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-muted-foreground text-xs font-medium">
              WebSocket Gateway URL
            </label>
            <Input
              value={gatewayURL}
              onChange={(e) => {
                setGatewayURL(e.target.value);
              }}
              placeholder="ws://127.0.0.1:8800/ws"
            />
            <span className="text-muted-foreground text-[11px]">
              Provides real-time bi-directional messaging with ACK delivery.
            </span>
          </div>
        </div>

        <DialogFooter className="flex items-center justify-between sm:justify-between">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleReset}
            className="text-muted-foreground gap-1.5"
          >
            <RotateCcw className="size-3.5" />
            <span>Reset Defaults</span>
          </Button>

          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                setOpen(false);
              }}
            >
              Cancel
            </Button>
            <Button type="button" size="sm" onClick={handleSave}>
              Save Changes
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
