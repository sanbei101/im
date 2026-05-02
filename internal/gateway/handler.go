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

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("gateway accept connection failed")
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	c, session := gateway.setupClient(userID, conn)
	defer gateway.cleanupClient(userID, c, session)

	go c.writePump(context.Background())

	senderUUID, err := uuid.Parse(userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gateway parse user_id to uuid failed")
		return
	}

	gateway.readPump(r.Context(), conn, c, userID, senderUUID)
}

func (gateway *Gateway) authenticate(r *http.Request) (string, error) {
	jwtToken := r.Header.Get("Authorization")
	if jwtToken == "" {
		log.Error().Str("remote_addr", r.RemoteAddr).Msg("gateway missing Authorization header")
		return "", errors.New("missing Authorization header")
	}
	if len(jwtToken) > 7 && jwtToken[:7] == "Bearer " {
		jwtToken = jwtToken[7:]
	}
	userID, err := jwt.ParseToken(jwtToken)
	if err != nil {
		log.Error().Err(err).Msg("gateway parse token failed")
		return "", err
	}
	return userID, nil
}

func (gateway *Gateway) setupClient(userID string, conn *websocket.Conn) (*Client, *UserSession) {
	c := &Client{
		Conn: conn,
		Send: make(chan []byte, 100),
	}
	session := gateway.sessions.LoadOrCreate(userID, NewUserSession)
	session.Add(c)

	return c, session
}

func (gateway *Gateway) cleanupClient(userID string, c *Client, session *UserSession) {
	if session.Remove(c) {
		gateway.sessions.Delete(userID)
	}
	close(c.Send)
}

func (gateway *Gateway) readPump(ctx context.Context, conn *websocket.Conn, c *Client, userID string, senderUUID uuid.UUID) {
	for {
		_, payload, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", userID).Msg("gateway read message failed")
			}
			return
		}
		gateway.handleIncomingMessage(ctx, payload, c, userID, senderUUID)
	}
}

func (gateway *Gateway) handleIncomingMessage(ctx context.Context, payload []byte, c *Client, userID string, senderUUID uuid.UUID) {
	var message db.Message
	if err := json.Unmarshal(payload, &message); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gateway unmarshal message failed")
		gateway.sendError(c, "invalid message format")
		return
	}

	if message.ClientMsgID == uuid.Nil {
		log.Error().Str("user_id", userID).Msg("gateway missing client_msg_id")
		gateway.sendError(c, "missing client_msg_id")
		return
	}

	var err error
	message.MsgID, err = uuid.NewV7()
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gateway generate msg_id failed")
		gateway.sendError(c, "failed to generate msg_id")
		return
	}
	message.SenderID = senderUUID
	message.ServerTime = time.Now().UnixMicro()

	if err := gateway.redis.GatewayPushMessage(ctx, []*db.Message{&message}); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gateway push message failed")
	}
}

func (gateway *Gateway) sendError(c *Client, errMsg string) {
	select {
	case c.Send <- []byte(errMsg):
	default:
		log.Warn().
			Str("error_msg", errMsg).
			Msg("gateway send error message failed, send channel is full")
	}
}
