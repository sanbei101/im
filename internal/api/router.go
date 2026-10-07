package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
)

// NewRouter wires the HTTP handlers into a chi router.
func NewRouter(s *store.Store) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)

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
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))
	userAPI := &UserAPI{store: s}
	roomAPI := &RoomAPI{store: s}
	messageAPI := &MessageAPI{store: s}
	friendAPI := &FriendAPI{store: s}
	conversationAPI := &ConversationAPI{store: s}
	fileAPI := NewFileAPI("")

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Post("/register", userAPI.Register)
			r.Post("/login", userAPI.Login)

			r.Group(func(r chi.Router) {
				r.Use(jwt.AuthMiddleware)
				r.Get("/search", userAPI.Search)
				r.Get("/profile", userAPI.GetProfile)
				r.Get("/{id}", userAPI.GetProfile)
				r.Put("/profile", userAPI.UpdateProfile)
			})
		})

		r.Route("/friends", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/", friendAPI.ListFriends)
			r.Post("/apply", friendAPI.Apply)
			r.Post("/audit", friendAPI.Audit)
			r.Get("/applications", friendAPI.ListApplications)
			r.Delete("/{id}", friendAPI.DeleteFriend)
			r.Put("/{id}/remark", friendAPI.UpdateRemark)
			r.Get("/blacklist", friendAPI.ListBlacklist)
			r.Post("/blacklist", friendAPI.AddBlacklist)
			r.Delete("/blacklist/{id}", friendAPI.RemoveBlacklist)
		})

		r.Route("/messages", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/history", messageAPI.GetHistory)
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
			r.Delete("/{id}/members/{user_id}", roomAPI.RemoveMember)
			r.Post("/{id}/leave", roomAPI.LeaveRoom)
			r.Post("/{id}/transfer", roomAPI.TransferOwner)
			r.Post("/{id}/read", conversationAPI.MarkRead)
			r.Put("/{id}/clear_unread", conversationAPI.ClearUnread)
		})

		r.Route("/conversations", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/", conversationAPI.ListConversations)
			r.Put("/{id}/pin", conversationAPI.PinConversation)
			r.Put("/{id}/mute", conversationAPI.MuteConversation)
		})

		r.Route("/files", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(jwt.AuthMiddleware)
				r.Post("/upload", fileAPI.Upload)
			})
			r.Get("/{name}", fileAPI.Serve)
		})
	})

	return r
}
