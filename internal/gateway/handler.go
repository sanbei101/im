package gateway

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/coder/websocket"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

func (gateway *Gateway) HandleUserMessage(w http.ResponseWriter, r *http.Request) {
	userID, err := gateway.authenticate(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
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
	gateway.registerUser(r.Context(), userID, true)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		userClient.writePump(r.Context())
	}()

	userClient.readPump(r.Context())
	// readPump 返回即连接已断开：关闭 Send 唤醒 writePump，避免协程泄漏。
	close(userClient.Send)
	<-writeDone
}

func (gateway *Gateway) authenticate(r *http.Request) (uuid.UUID, error) {
	jwtToken := r.URL.Query().Get("token")
	if jwtToken == "" {
		log.Error().Str("remote_addr", r.RemoteAddr).Msg("gateway missing token query parameter")
		return uuid.Nil(), errors.New("missing token query parameter")
	}
	userIDStr, err := jwt.ParseToken(jwtToken)
	if err != nil {
		log.Error().Err(err).Msg("gateway parse token failed")
		return uuid.Nil(), err
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("gateway parse user_id to uuid failed")
		return uuid.Nil(), err
	}
	return userID, nil
}

func (gateway *Gateway) setupUserClient(userID uuid.UUID, conn *websocket.Conn) (*UserClient, *UserSession) {
	userClient := &UserClient{
		gateway: gateway,
		Conn:    conn,
		Send:    make(chan []byte, 100),
		UserID:  userID,
		frames:  render.NewFrameWriter(),
	}
	userSession := gateway.UserSessionManager.LoadOrCreate(userID.String(), NewUserSession)
	userSession.Add(userClient)

	return userClient, userSession
}

func (gateway *Gateway) cleanUserClient(userID uuid.UUID, c *UserClient, session *UserSession) {
	gateway.registerUser(context.Background(), userID, false)
	if session.Remove(c) {
		gateway.UserSessionManager.Delete(userID.String())
	}
	c.closed.Store(true)
}
