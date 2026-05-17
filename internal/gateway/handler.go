package gateway

import (
	"errors"
	"net/http"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/pkg/jwt"
)

func (gateway *Gateway) HandleUserMessage(w http.ResponseWriter, r *http.Request) {
	userID, err := gateway.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Msg("gateway accept connection failed")
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	userClient, userSession := gateway.setupUserClient(userID, conn)
	defer gateway.cleanUserClient(userID, userClient, userSession)

	go userClient.writePump(r.Context())

	userClient.readPump(r.Context())
}

func (gateway *Gateway) authenticate(r *http.Request) (uuid.UUID, error) {
	jwtToken := r.URL.Query().Get("token")
	if jwtToken == "" {
		log.Error().Str("remote_addr", r.RemoteAddr).Msg("gateway missing token query parameter")
		return uuid.Nil, errors.New("missing token query parameter")
	}
	userIDStr, err := jwt.ParseToken(jwtToken)
	if err != nil {
		log.Error().Err(err).Msg("gateway parse token failed")
		return uuid.Nil, err
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("gateway parse user_id to uuid failed")
		return uuid.Nil, err
	}
	return userID, nil
}

func (gateway *Gateway) setupUserClient(userID uuid.UUID, conn *websocket.Conn) (*UserClient, *UserSession) {
	c := &UserClient{
		Conn:   conn,
		Send:   make(chan [][]byte, 100),
		UserID: userID,
	}
	session := gateway.sessions.LoadOrCreate(userID.String(), NewUserSession)
	session.Add(c)

	return c, session
}

func (gateway *Gateway) cleanUserClient(userID uuid.UUID, c *UserClient, session *UserSession) {
	if session.Remove(c) {
		gateway.sessions.Delete(userID.String())
	}
	close(c.Send)
}
