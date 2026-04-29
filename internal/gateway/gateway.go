package gateway

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/jwt"
)

type Gateway struct {
	sessions sync.Map
	redis    *db.Redis
	config   *config.Config
}

func New(config *config.Config) *Gateway {
	return &Gateway{
		redis:  db.NewRedis(config),
		config: config,
	}
}
func (g *Gateway) HandleUserMessage(w http.ResponseWriter, r *http.Request) {
	jwtToken := r.Header.Get("Authorization")
	if jwtToken == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		log.Error().Str("remote_addr", r.RemoteAddr).Msg("gateway missing Authorization header")
		return
	}
	if len(jwtToken) > 7 && jwtToken[:7] == "Bearer " {
		jwtToken = jwtToken[7:]
	}
	userID, err := jwt.ParseToken(jwtToken)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		log.Error().Err(err).Msg("gateway parse token failed")
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("gateway accept connection failed")
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	c := &client{
		conn: conn,
		send: make(chan []byte, 100),
	}
	sessionIface, _ := g.sessions.LoadOrStore(userID, NewUserSession())
	session := sessionIface.(*UserSession)
	session.Add(c)

	// 注册退出清理逻辑
	defer func() {
		// 如果 Remove 返回 true,说明该用户的所有设备都已断开,清理 sync.Map 中的记录
		if session.Remove(c) {
			g.sessions.Delete(userID)
		}
		close(c.send)
	}()

	go c.writePump(context.Background())
	senderUUID, err := uuid.Parse(userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gateway parse user_id to uuid failed")
		return
	}
	for {
		_, payload, err := conn.Read(r.Context())
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", userID).Msg("gateway read message failed")
			}
			return
		}

		var message db.Message
		if err := json.Unmarshal(payload, &message); err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("gateway unmarshal message failed")
			select {
			case c.send <- []byte("invalid message format"):
			default:
			}
			continue
		}

		if message.ClientMsgID == uuid.Nil {
			log.Error().Str("user_id", userID).Msg("gateway missing client_msg_id")
			select {
			case c.send <- []byte("missing client_msg_id"):
			default:
			}
			continue
		}

		message.MsgID, err = uuid.NewV7()
		if err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("gateway generate msg_id failed")
			select {
			case c.send <- []byte("failed to generate msg_id"):
			default:
			}
			continue
		}
		message.SenderID = senderUUID
		message.ServerTime = time.Now().UnixMicro()

		if err := g.redis.GatewayPushMessage(r.Context(), []*db.Message{&message}); err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("gateway push message failed")
		}
	}
}

func (g *Gateway) HandleWorkerMessages(ctx context.Context) {
	if err := g.redis.InitStreamGroups(ctx); err != nil {
		log.Warn().Err(err).Msg("gateway consumer group mkstream failed")
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			messages, err := g.redis.GatewayPullMessage(ctx, 1000)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error().Err(err).Msg("gateway pull message failed")
				time.Sleep(time.Second)
				continue
			}

			if len(messages) == 0 {
				continue
			}

			var msgIDs []string
			for _, msg := range messages {
				msgIDs = append(msgIDs, msg.ID)

				bin, marshalErr := json.Marshal(msg.Data)
				if marshalErr != nil {
					log.Error().Err(marshalErr).Msg("gateway marshal message failed")
					continue
				}
				roomID := msg.Data.RoomID.String()
				g.deliverToClient(roomID, bin)
			}
			err = g.redis.GatewayAckMessage(ctx, msgIDs...)
			if err != nil {
				log.Error().Err(err).Msg("gateway ack messages failed")
			}
		}
	}
}

func (g *Gateway) deliverToClient(userID string, payload []byte) {
	if sessionIface, ok := g.sessions.Load(userID); ok {
		session := sessionIface.(*UserSession)
		session.Broadcast(payload)
	} else {
		log.Debug().Str("user_id", userID).Msg("user not connected to this gateway instance")
	}
}
