package api

import (
	"net/http"
	"testing"
)

func TestMetaAndWorkRouters(t *testing.T) {
	env := newTestEnv(t)
	metaHandler := NewMetaRouter(env.store)
	workHandler := NewWorkRouter(env.store, env.stream, env.store, nil, "https://cdn.example.test")

	alice, token := env.addUser(t, "alice")
	_ = alice

	t.Run("meta serves user routes", func(t *testing.T) {
		recorder := doRequest(t, metaHandler, http.MethodGet, "/api/v1/users/profile", token, nil)
		requireStatus(t, recorder, http.StatusOK)
		profile := decodeResponse[UserProfileResp](t, recorder)
		if profile.Data.Username != "alice" {
			t.Fatalf("profile = %+v", profile.Data)
		}
	})

	t.Run("meta rejects work routes", func(t *testing.T) {
		recorder := doRequest(t, metaHandler, http.MethodGet, "/api/v1/friends/", token, nil)
		requireStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("work rejects user auth routes", func(t *testing.T) {
		recorder := doRequest(t, workHandler, http.MethodPost, "/api/v1/users/register", "",
			jsonBody(t, UserAuthReq{Username: "bob", Password: "secret1"}))
		requireStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("work serves presence", func(t *testing.T) {
		recorder := doRequest(t, workHandler, http.MethodPost, "/api/v1/users/presence", token,
			jsonBody(t, PresenceReq{UserIDs: []string{alice.UserID.String()}}))
		requireStatus(t, recorder, http.StatusOK)
		presence := decodeResponse[PresenceResp](t, recorder)
		if presence.Data.Presence[alice.UserID.String()] {
			t.Fatal("user without a session reported online")
		}
	})
}
