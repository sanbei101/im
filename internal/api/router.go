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
	benchHandler *handler.BenchMockHandler,
) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"*"},
		AllowedHeaders: []string{"*"},
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Post("/register", userHandler.Register)
			r.Post("/login", userHandler.Login)
		})

		r.Route("/messages", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Get("/history", messageHandler.GetHistory)
		})

		r.Route("/rooms", func(r chi.Router) {
			r.Use(jwt.AuthMiddleware)
			r.Post("/single", roomHandler.CreateOrGetSingleChatRoom)
			r.Post("/group", roomHandler.CreateGroupRoom)
			r.Post("/list", roomHandler.ListRooms)
		})

		r.Route("/bench", func(r chi.Router) {
			r.Post("/mock", benchHandler.CreateMock)
		})
	})

	return r
}
