import {
  MessageSquare,
  Lock,
  User,
  AlertCircle,
  Settings2,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { useState, type SyntheticEvent } from "react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useChat } from "@/context/ChatContext";
import { isErrorWithMessage } from "@/types/chat";

export function AuthScreen() {
  const { login, register, config, updateConfig } = useChat();

  const [mode, setMode] = useState<string>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Server settings accordion
  const [showServerConfig, setShowServerConfig] = useState(false);
  const [baseURL, setBaseURL] = useState(config.baseURL);
  const [gatewayURL, setGatewayURL] = useState(config.gatewayURL);

  const handleSubmit = async (e: SyntheticEvent) => {
    e.preventDefault();
    const trimmedUsername = username.trim();
    const trimmedPassword = password.trim();

    if (!trimmedUsername || !trimmedPassword) {
      setErrorMsg("Please enter both username and password.");
      return;
    }

    setIsLoading(true);
    setErrorMsg(null);

    // If server endpoints were modified, save them first
    if (baseURL !== config.baseURL || gatewayURL !== config.gatewayURL) {
      updateConfig({
        baseURL: baseURL.trim(),
        gatewayURL: gatewayURL.trim(),
      });
    }

    try {
      if (mode === "login") {
        await login({ username: trimmedUsername, password: trimmedPassword });
      } else {
        await register({ username: trimmedUsername, password: trimmedPassword });
      }
    } catch (err) {
      const msg = isErrorWithMessage(err) ? err.message : "Authentication failed";
      setErrorMsg(msg);
    } finally {
      setIsLoading(false);
    }
  };

  const handleQuickFill = (name: string) => {
    setUsername(name);
    setPassword("password123");
  };

  return (
    <div className="bg-muted/20 flex min-h-screen w-full items-center justify-center p-4">
      <div className="flex w-full max-w-md flex-col gap-4">
        {/* App Branding */}
        <div className="mb-2 flex flex-col items-center gap-1.5 text-center">
          <div className="bg-primary text-primary-foreground flex size-12 items-center justify-center rounded-xl shadow-md">
            <MessageSquare className="size-6" />
          </div>
          <h1 className="mt-1 text-xl font-bold tracking-tight">Go IM Instant Messenger</h1>
          <p className="text-muted-foreground text-xs">
            Type-Safe, Pebble-backed High-Performance IM Client
          </p>
        </div>

        <Card className="bg-card border shadow-lg">
          <CardHeader className="pb-3 text-center">
            <CardTitle className="text-base">
              {mode === "login" ? "Welcome Back" : "Create Account"}
            </CardTitle>
            <CardDescription className="text-xs">
              {mode === "login"
                ? "Enter your credentials to access your chats"
                : "Register a new user account on the server"}
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            <Tabs value={mode} onValueChange={setMode} className="w-full">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="login">Sign In</TabsTrigger>
                <TabsTrigger value="register">Register</TabsTrigger>
              </TabsList>
            </Tabs>

            {errorMsg && (
              <div className="bg-destructive/10 text-destructive flex items-center gap-2 rounded-lg p-3 text-xs">
                <AlertCircle className="size-4 shrink-0" />
                <span>{errorMsg}</span>
              </div>
            )}

            <form onSubmit={handleSubmit} className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <label className="text-muted-foreground flex items-center gap-1 text-xs font-medium">
                  <User className="size-3.5" /> Username
                </label>
                <Input
                  value={username}
                  onChange={(e) => {
                    setUsername(e.target.value);
                  }}
                  placeholder="Enter your username"
                  autoComplete="username"
                  disabled={isLoading}
                  className="h-9 text-xs"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-muted-foreground flex items-center gap-1 text-xs font-medium">
                  <Lock className="size-3.5" /> Password
                </label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value);
                  }}
                  placeholder="Enter your password"
                  autoComplete={mode === "login" ? "current-password" : "new-password"}
                  disabled={isLoading}
                  className="h-9 text-xs"
                />
              </div>

              {/* Quick test account buttons */}
              <div className="text-muted-foreground flex items-center justify-between pt-1 text-[11px]">
                <span>Quick demo:</span>
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    onClick={() => {
                      handleQuickFill("alice");
                    }}
                    className="hover:text-foreground hover:underline"
                  >
                    alice
                  </button>
                  <span>•</span>
                  <button
                    type="button"
                    onClick={() => {
                      handleQuickFill("bob");
                    }}
                    className="hover:text-foreground hover:underline"
                  >
                    bob
                  </button>
                  <span>•</span>
                  <button
                    type="button"
                    onClick={() => {
                      handleQuickFill("charlie");
                    }}
                    className="hover:text-foreground hover:underline"
                  >
                    charlie
                  </button>
                </div>
              </div>

              <Button
                type="submit"
                className="mt-2 h-9 w-full text-xs font-medium"
                disabled={isLoading}
              >
                {isLoading ? "Connecting..." : mode === "login" ? "Sign In" : "Register & Sign In"}
              </Button>
            </form>
          </CardContent>

          <CardFooter className="flex flex-col pt-0 pb-3">
            <button
              type="button"
              onClick={() => {
                setShowServerConfig((prev) => !prev);
              }}
              className="text-muted-foreground hover:text-foreground flex w-full items-center justify-between border-t py-2 text-[11px]"
            >
              <span className="flex items-center gap-1">
                <Settings2 className="size-3.5" /> Server Endpoints
              </span>
              {showServerConfig ? (
                <ChevronUp className="size-3.5" />
              ) : (
                <ChevronDown className="size-3.5" />
              )}
            </button>

            {showServerConfig && (
              <div className="flex w-full flex-col gap-2.5 pt-2 text-xs">
                <div className="flex flex-col gap-1">
                  <label className="text-muted-foreground text-[11px]">HTTP API URL</label>
                  <Input
                    value={baseURL}
                    onChange={(e) => {
                      setBaseURL(e.target.value);
                    }}
                    className="h-7 font-mono text-xs"
                    placeholder="http://127.0.0.1:8801"
                  />
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-muted-foreground text-[11px]">Gateway WS URL</label>
                  <Input
                    value={gatewayURL}
                    onChange={(e) => {
                      setGatewayURL(e.target.value);
                    }}
                    className="h-7 font-mono text-xs"
                    placeholder="ws://127.0.0.1:8800/ws"
                  />
                </div>
              </div>
            )}
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
