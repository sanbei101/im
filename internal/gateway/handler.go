package gateway

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
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

	userClient, userSession := gateway.setupClient(userID, conn)
	defer gateway.cleanupClient(userID, userClient, userSession)

	go userClient.writePump(r.Context())

	gateway.readPump(r.Context(), userClient)
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

func (gateway *Gateway) setupClient(userID uuid.UUID, conn *websocket.Conn) (*Client, *UserSession) {
	c := &Client{
		Conn:   conn,
		Send:   make(chan [][]byte, 100),
		UserID: userID,
	}
	session := gateway.sessions.LoadOrCreate(userID.String(), NewUserSession)
	session.Add(c)

	return c, session
}

func (gateway *Gateway) cleanupClient(userID uuid.UUID, c *Client, session *UserSession) {
	if session.Remove(c) {
		gateway.sessions.Delete(userID.String())
	}
	close(c.Send)
}

func (gateway *Gateway) readPump(ctx context.Context, c *Client) {
	for {
		_, payload, err := c.Conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("gateway read message failed")
			}
			return
		}
		gateway.handleIncomingMessage(ctx, payload, c)
	}
}

func (gateway *Gateway) handleIncomingMessage(
	ctx context.Context,
	payload []byte,
	c *Client,
) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil && envelope.Type == "ping" {
		return
	}

	var message db.Message
	if err := json.Unmarshal(payload, &message); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("gateway unmarshal message failed")
		gateway.sendError(c, "invalid message format")
		return
	}

	if message.ClientMsgID == uuid.Nil {
		log.Error().Str("user_id", c.UserID.String()).Msg("gateway missing client_msg_id")
		gateway.sendError(c, "missing client_msg_id")
		return
	}

	var err error
	message.MsgID, err = uuid.NewV7()
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("gateway generate msg_id failed")
		gateway.sendError(c, "failed to generate msg_id")
		return
	}
	message.SenderID = c.UserID
	message.ServerTime = time.Now().UnixMicro()

	if err := gateway.redis.GatewayPushMessage(ctx, []*db.Message{&message}); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("gateway push message failed")
	}
}

func (gateway *Gateway) sendError(c *Client, errMsg string) {
	bin, _ := json.Marshal(map[string]string{"error": errMsg})
	select {
	case c.Send <- [][]byte{bin}:
	default:
		log.Warn().
			Str("error_msg", errMsg).
			Msg("gateway send error message failed, send channel is full")
	}
}
