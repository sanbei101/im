package api

import (
	"errors"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/engine"
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
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
			http.MethodDelete, http.MethodHead, http.MethodOptions,
		},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"*"},
		MaxAge:         86400,
	}))
	return r
}

// NewRouter wires the control plane (PostgreSQL) and the message plane
// (Pebble) into one API process. Every API node serves every HTTP route.
func NewRouter(
	db *store.DB,
	data *engine.Engine,
	streamHandler *StreamHandler,
	members *Membership,
	objects *MinioObjectStore,
	publicBaseURL string,
	nodeID string,
) http.Handler {
	r := newBaseRouter()
	notifier := NewNotifier(db, members, streamHandler, nodeID)
	userAPI := &UserAPI{store: db, streamHandler: streamHandler}
	roomAPI := &RoomAPI{db: db, data: data, streamHandler: streamHandler, notifier: notifier, members: members}
	messageAPI := &MessageAPI{store: data, members: members, streamHandler: streamHandler}
	friendAPI := &FriendAPI{store: db, notifier: notifier}
	conversationAPI := &ConversationAPI{db: db, data: data, members: members}
	fileAPI := NewFileAPI(objects, publicBaseURL)

	registerHealth(r, db, data)
	r.Route("/api/v1", func(r chi.Router) {
		registerUsers(r, userAPI)
		registerDevices(r, userAPI)
		registerFriends(r, friendAPI)
		registerWork(r, roomAPI, messageAPI, conversationAPI, fileAPI)
	})
	return r
}

func registerHealth(r chi.Router, db *store.DB, data *engine.Engine) {
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			render.Error(w, http.StatusServiceUnavailable, "postgres unavailable: "+err.Error())
			return
		}
		if err := data.Ping(r.Context()); err != nil {
			render.Error(w, http.StatusServiceUnavailable, "pebble unavailable: "+err.Error())
			return
		}
		render.Success(w, "ok", HealthResp{Status: "healthy"})
	})
}

func registerUsers(r chi.Router, api *UserAPI) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/register", api.Register)
		r.Post("/login", api.Login)
		r.Group(func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Post("/logout", api.Logout)
			r.Get("/search", api.Search)
			r.Get("/profile", api.GetProfile)
			r.Get("/{id}", api.GetProfile)
			r.Put("/profile", api.UpdateProfile)
			r.Put("/password", api.UpdatePassword)
			r.Post("/presence", api.Presence)
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
		r.Post("/{seq}/reactions", messageAPI.AddReaction)
		r.Delete("/{seq}/reactions", messageAPI.RemoveReaction)
		r.Get("/{seq}/reactions", messageAPI.GetReactions)
		r.Get("/{seq}/read_users", messageAPI.GetReadUsers)
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
		r.Delete("/{id}/pins/{seq}", roomAPI.UnpinMessage)
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

func requireUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return uuid.Nil(), false
	}
	return userID, true
}

func roomIDFromPath(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return uuid.Nil(), false
	}
	return roomID, true
}

func isManager(role string) bool {
	return role == store.RoleOwner || role == store.RoleAdmin
}

func requireRoomMember(m *Membership, w http.ResponseWriter, r *http.Request, roomID uuid.UUID) bool {
	userID, ok := requireUser(w, r)
	if !ok {
		return false
	}
	_, err := m.Member(r.Context(), roomID, userID)
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
