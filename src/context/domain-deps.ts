/**
 * Shared dependency contract for the chat domain hooks.
 *
 * The domain hooks (conversations, rooms, contacts, messages, account) each
 * own one slice of ChatContext state but need to call into each other — for
 * example dropping a room must refresh the room list. They receive this
 * contract instead of importing one another, which keeps the modules free of
 * cycles and makes the wiring in ChatContext explicit.
 */
export interface ChatDomainDeps {
  /** Active room id, or null when no room is selected. */
  readonly activeRoomId: string | null;
  /** Set the active room id. */
  readonly setActiveRoomId: (id: string | null) => void;
  /** Report an error through the shared banner. */
  readonly setError: (message: string | null) => void;
  /** True once a user is authenticated, so SDK calls are safe. */
  readonly isAuthenticated: () => boolean;
}
