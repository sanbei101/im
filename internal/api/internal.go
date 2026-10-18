package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"uuid"

	"github.com/sanbei101/im/pkg/render"
)

const internalTokenHeader = "X-Internal-Token"

func internalAuth(secret string) func(http.Handler) http.Handler {
	expected := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(expected) == 0 ||
				subtle.ConstantTimeCompare([]byte(r.Header.Get(internalTokenHeader)), expected) != 1 {
				render.Error(w, http.StatusUnauthorized, "invalid internal token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// InternalUsers resolves profiles for Work nodes. It is internal-only and
// deliberately excludes passwords, which the User JSON never serializes.
func (a *UserAPI) InternalUsers(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("ids")
	if raw == "" {
		render.Success(w, "ok", map[uuid.UUID]struct{}{})
		return
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 100 {
		render.Error(w, http.StatusBadRequest, "too many ids")
		return
	}
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil {
			render.Error(w, http.StatusBadRequest, "invalid user id: "+part)
			return
		}
		ids = append(ids, id)
	}
	users, err := a.store.UsersByIDs(r.Context(), ids)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(w, "ok", users)
}
