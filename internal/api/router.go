package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/sanbei101/im/internal/api/handler"
	"github.com/sanbei101/im/pkg/jwt"
)

// SetupRouter wires the HTTP handlers into a chi router. Returned handler
// implements http.Handler so the caller can mount it on any net/http server.
func SetupRouter(
	userHandler *handler.UserHandler,
	messageHandler *handler.MessageHandler,
	roomHandler *handler.RoomHandler,
	friendHandler *handler.FriendHandler,
	benchHandler *handler.BenchMockHandler,
	allowedOrigins ...[]string,
) http.Handler {
	origins := []string{"http://localhost:3000"}
	if len(allowedOrigins) > 0 && len(allowedOrigins[0]) > 0 {
		origins = allowedOrigins[0]
	}
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		r.Route("/users", func(r chi.Router) {
			r.Post("/register", userHandler.Register)
			r.Post("/login", userHandler.Login)
			r.Post("/refresh", userHandler.Refresh)
			r.Group(func(r chi.Router) {
				r.Use(jwt.AuthMiddleware)
				r.Post("/logout", userHandler.Logout)
				r.Get("/me", userHandler.GetProfile)
				r.Patch("/me", userHandler.UpdateProfile)
				r.Patch("/me/password", userHandler.ChangePassword)
			})
		})

		r.Route("/friends", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/", friendHandler.ListFriends)
			r.Post("/requests", friendHandler.SendRequest)
			r.Get("/requests", friendHandler.ListReceivedRequests)
			r.Post("/requests/{requestID}/accept", friendHandler.AcceptRequest)
			r.Post("/requests/{requestID}/reject", friendHandler.RejectRequest)
			r.Delete("/{userID}", friendHandler.DeleteFriend)
		})

		r.Route("/blocks", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/", friendHandler.ListBlocks)
			r.Post("/", friendHandler.Block)
			r.Delete("/{userID}", friendHandler.Unblock)
		})

		r.Route("/messages", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/history", messageHandler.GetHistory)
			r.Get("/sync", messageHandler.Sync)
			r.Post("/read", messageHandler.MarkRead)
			r.Post("/{msgID}/recall", messageHandler.Recall)
		})

		r.Route("/rooms", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Post("/single", roomHandler.CreateOrGetSingleChatRoom)
			r.Post("/group", roomHandler.CreateGroupRoom)
			r.Post("/list", roomHandler.ListRooms)
			r.Get("/{roomID}", roomHandler.GetRoom)
			r.Get("/{roomID}/members", roomHandler.ListMembers)
			r.Post("/{roomID}/members", roomHandler.AddMember)
			r.Delete("/{roomID}/members/{userID}", roomHandler.KickMember)
			r.Post("/{roomID}/leave", roomHandler.Leave)
			r.Delete("/{roomID}", roomHandler.Dissolve)
			r.Post("/{roomID}/transfer", roomHandler.TransferOwnership)
			r.Patch("/{roomID}/members/{userID}/role", roomHandler.SetRole)
			r.Patch("/{roomID}/settings", roomHandler.UpdateSettings)
		})

		r.Route("/bench", func(r chi.Router) {
			r.Post("/mock", benchHandler.CreateMock)
		})
	})

	return r
}
