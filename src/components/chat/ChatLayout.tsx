import { AlertCircle, X } from "lucide-react";
import { useState } from "react";

import { useChat } from "@/context/ChatContext";

import { ChatArea } from "./ChatArea";
import { ChatSidebar } from "./ChatSidebar";
import { ContactPanel } from "./ContactPanel";
import { SearchPanel } from "./SearchPanel";
import { ProfileDialog } from "./ProfileDialog";

export function ChatLayout() {
  const { error, clearError } = useChat();
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [showContacts, setShowContacts] = useState(false);
  const [showSearch, setShowSearch] = useState(false);
  const [showProfile, setShowProfile] = useState(false);

  const handleToggleSidebar = () => {
    setIsSidebarCollapsed((prev) => !prev);
  };

  return (
    <div className="bg-background flex h-screen w-screen flex-col overflow-hidden">
      {error && (
        <div className="bg-destructive/15 border-destructive/20 text-destructive flex shrink-0 items-center justify-between border-b px-4 py-2 text-xs">
          <div className="flex items-center gap-2">
            <AlertCircle className="size-4 shrink-0" />
            <span>{error}</span>
          </div>
          <button
            type="button"
            onClick={clearError}
            className="hover:bg-destructive/10 rounded-sm p-1"
          >
            <X className="size-3.5" />
          </button>
        </div>
      )}

      <div className="flex flex-1 overflow-hidden">
        <ChatSidebar
          isCollapsed={isSidebarCollapsed}
          onToggleCollapse={handleToggleSidebar}
          onShowContacts={() => {
            setShowSearch(false);
            setShowContacts(true);
          }}
          onShowSearch={() => {
            setShowContacts(false);
            setShowSearch(true);
          }}
          onShowProfile={() => setShowProfile(true)}
          contactsOpen={showContacts}
          searchOpen={showSearch}
        />
        <ChatArea
          isSidebarCollapsed={isSidebarCollapsed}
          onToggleSidebar={handleToggleSidebar}
        />

        {showContacts && (
          <ContactPanel
            onClose={() => {
              setShowContacts(false);
            }}
          />
        )}
        {showSearch && (
          <SearchPanel
            onClose={() => {
              setShowSearch(false);
            }}
          />
        )}
      </div>

      <ProfileDialog
        open={showProfile}
        onOpenChange={setShowProfile}
      />
    </div>
  );
}
