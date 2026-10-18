package api

import (
	"errors"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type HealthResp struct {
	Status string `json:"status"`
}

func newBaseRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	// JWT travels in the Authorization header rather than cookies, so the
	// policy stays non-credentialed: wildcard origins and credentials are
	// mutually exclusive per the fetch spec.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"*"},
		MaxAge:         86400,
	}))
	return r
}

// NewRouter wires every handler into one process. Tests and single-process
// deployments use it; the split services use NewMetaRouter/NewWorkRouter.
func NewRouter(
	s *store.Store,
	streamHandler *StreamHandler,
	objects *store.MinioObjectStore,
	publicBaseURL string,
) http.Handler {
	r := newBaseRouter()
	userAPI := &UserAPI{store: s, streamHandler: streamHandler}
	roomAPI := &RoomAPI{store: s, streamHandler: streamHandler, users: s}
	messageAPI := &MessageAPI{store: s, streamHandler: streamHandler}
	friendAPI := &FriendAPI{store: s, streamHandler: streamHandler, users: s}
	conversationAPI := &ConversationAPI{store: s}
	fileAPI := NewFileAPI(objects, publicBaseURL)

	registerHealth(r, s)
	r.Route("/api/v1", func(r chi.Router) {
		registerUsers(r, userAPI, true, true)
		registerDevices(r, userAPI)
		registerFriends(r, friendAPI)
		registerWork(r, roomAPI, messageAPI, conversationAPI, fileAPI)
	})
	return r
}

// NewMetaRouter serves the authoritative identity/social control plane.
func NewMetaRouter(s *store.Store, internalSecret string) http.Handler {
	r := newBaseRouter()
	userAPI := &UserAPI{store: s}

	registerHealth(r, s)
	r.Route("/api/v1", func(r chi.Router) {
		registerUsers(r, userAPI, true, false)
		registerDevices(r, userAPI)
	})
	r.Route("/internal", func(r chi.Router) {
		r.Use(internalAuth(internalSecret))
		r.Get("/users", userAPI.InternalUsers)
	})
	return r
}

// NewWorkRouter serves rooms, messages, files, friends, and presence. User
// profiles are resolved through users (the Meta service in production).
func NewWorkRouter(
	s *store.Store,
	streamHandler *StreamHandler,
	users UserDirectory,
	objects *store.MinioObjectStore,
	publicBaseURL string,
) http.Handler {
	r := newBaseRouter()
	presenceAPI := &UserAPI{streamHandler: streamHandler}
	roomAPI := &RoomAPI{store: s, streamHandler: streamHandler, users: users}
	messageAPI := &MessageAPI{store: s, streamHandler: streamHandler}
	friendAPI := &FriendAPI{store: s, streamHandler: streamHandler, users: users}
	conversationAPI := &ConversationAPI{store: s}
	fileAPI := NewFileAPI(objects, publicBaseURL)

	registerHealth(r, s)
	r.Route("/api/v1", func(r chi.Router) {
		registerUsers(r, presenceAPI, false, true)
		registerFriends(r, friendAPI)
		registerWork(r, roomAPI, messageAPI, conversationAPI, fileAPI)
	})
	return r
}

func registerHealth(r chi.Router, s *store.Store) {
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Ping(r.Context()); err != nil {
			render.Error(w, http.StatusServiceUnavailable, "database unavailable: "+err.Error())
			return
		}
		render.Success(w, "ok", HealthResp{Status: "healthy"})
	})
}

func registerUsers(r chi.Router, api *UserAPI, meta, presence bool) {
	r.Route("/users", func(r chi.Router) {
		if meta {
			r.Post("/register", api.Register)
			r.Post("/login", api.Login)
		}
		r.Group(func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			if meta {
				r.Post("/logout", api.Logout)
				r.Get("/search", api.Search)
				r.Get("/profile", api.GetProfile)
				r.Get("/{id}", api.GetProfile)
				r.Put("/profile", api.UpdateProfile)
				r.Put("/password", api.UpdatePassword)
			}
			if presence {
				r.Post("/presence", api.Presence)
			}
		})
	})
}

func registerDevices(r chi.Router, api *UserAPI) {
	r.Route("/devices", func(r chi.Router) {
		r.Use(jwt.AuthMiddleware)
		r.Post("/token", api.SaveDeviceToken)
	})
}

func registerFriends(r chi.Router, api *FriendAPI) {
	r.Route("/friends", func(r chi.Router) {
		r.Use(jwt.AuthMiddleware)
		r.Get("/", api.ListFriends)
		r.Post("/apply", api.Apply)
		r.Post("/audit", api.Audit)
		r.Get("/applications", api.ListApplications)
		r.Delete("/{id}", api.DeleteFriend)
		r.Put("/{id}/remark", api.UpdateRemark)
		r.Get("/blacklist", api.ListBlacklist)
		r.Post("/blacklist", api.AddBlacklist)
		r.Delete("/blacklist/{id}", api.RemoveBlacklist)
	})
}

func registerWork(
	r chi.Router,
	roomAPI *RoomAPI,
	messageAPI *MessageAPI,
	conversationAPI *ConversationAPI,
	fileAPI *FileAPI,
) {
	r.Route("/messages", func(r chi.Router) {
		r.Use(jwt.AuthMiddleware)
		r.Get("/history", messageAPI.GetHistory)
		r.Get("/search", messageAPI.Search)
		r.Post("/recall", messageAPI.Recall)
		r.Post("/{id}/reactions", messageAPI.AddReaction)
		r.Delete("/{id}/reactions", messageAPI.RemoveReaction)
		r.Get("/{id}/reactions", messageAPI.GetReactions)
		r.Get("/{id}/read_users", messageAPI.GetReadUsers)
	})

	r.Route("/rooms", func(r chi.Router) {
		r.Use(jwt.AuthMiddleware)
		r.Post("/single", roomAPI.CreateOrGetSingleChatRoom)
		r.Post("/group", roomAPI.CreateGroupRoom)
		r.Post("/list", roomAPI.ListRooms)
		r.Get("/{id}", roomAPI.GetRoom)
		r.Put("/{id}", roomAPI.UpdateRoom)
		r.Delete("/{id}", roomAPI.DissolveRoom)
		r.Get("/{id}/members", roomAPI.ListMembers)
		r.Post("/{id}/members", roomAPI.AddMembers)
		r.Put("/{id}/members/{user_id}/role", roomAPI.UpdateMemberRole)
		r.Delete("/{id}/members/{user_id}", roomAPI.RemoveMember)
		r.Get("/{id}/messages/search", messageAPI.Search)
		r.Post("/{id}/leave", roomAPI.LeaveRoom)
		r.Post("/{id}/transfer", roomAPI.TransferOwner)
		r.Post("/{id}/read", conversationAPI.MarkRead)
		r.Put("/{id}/clear_unread", conversationAPI.ClearUnread)
		r.Post("/{id}/pins", roomAPI.PinMessage)
		r.Delete("/{id}/pins/{msg_id}", roomAPI.UnpinMessage)
		r.Get("/{id}/pins", roomAPI.GetPinnedMessages)
	})

	r.Route("/conversations", func(r chi.Router) {
		r.Use(jwt.AuthMiddleware)
		r.Get("/", conversationAPI.ListConversations)
		r.Put("/{id}/pin", conversationAPI.PinConversation)
		r.Put("/{id}/mute", conversationAPI.MuteConversation)
	})

	r.Route("/files", func(r chi.Router) {
		r.Use(jwt.AuthMiddleware)
		r.Post("/presign", fileAPI.PresignUpload)
	})
}

func getContextUserID(r *http.Request) (uuid.UUID, error) {
	raw := jwt.GetUserIDFromContext(r)
	if raw == "" {
		return uuid.Nil(), errors.New("user not authenticated")
	}
	return uuid.Parse(raw)
}

// requireUser resolves the authenticated caller; on failure the response is
// already written and it returns false.
func requireUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return uuid.Nil(), false
	}
	return userID, true
}

// roomIDFromPath parses the {id} URL parameter as a room UUID; on failure the
// response is already written and it returns false.
func roomIDFromPath(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return uuid.Nil(), false
	}
	return roomID, true
}

// isManager reports whether the role may manage room-wide settings.
func isManager(role string) bool {
	return role == store.RoleOwner || role == store.RoleAdmin
}

// requireRoomMember authenticates the caller and verifies membership of
// roomID; on failure the response is already written and it returns false.
func requireRoomMember(s *store.Store, w http.ResponseWriter, r *http.Request, roomID uuid.UUID) bool {
	userID, ok := requireUser(w, r)
	if !ok {
		return false
	}
	_, err := s.Member(r.Context(), roomID, userID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return false
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, "load room membership: "+err.Error())
		return false
	}
	return true
}
