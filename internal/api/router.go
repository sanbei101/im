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

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Post("/register", userAPI.Register)
			r.Post("/login", userAPI.Login)
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
		})
	})

	return r
}
