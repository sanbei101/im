import { AuthScreen } from "@/components/auth/AuthScreen";
import { ChatLayout } from "@/components/chat/ChatLayout";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ChatProvider, useChat } from "@/context/ChatContext";

function MainContent() {
  const { currentUser } = useChat();

  if (currentUser === null) {
    return <AuthScreen />;
  }

  return <ChatLayout />;
}

export default function App() {
  return (
    <TooltipProvider>
      <ChatProvider>
        <MainContent />
      </ChatProvider>
    </TooltipProvider>
  );
}
