package api

import (
	"context"
	"encoding/json/v2"
	"uuid"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/store"
)

// Notifier delivers an event locally and publishes it to every API node
// through PostgreSQL. Local delivery is synchronous so the request handler
// cannot race its own response.
type Notifier struct {
	db      *store.DB
	members *Membership
	stream  *StreamHandler
	origin  string
}

func NewNotifier(db *store.DB, members *Membership, stream *StreamHandler, origin string) *Notifier {
	return &Notifier{db: db, members: members, stream: stream, origin: origin}
}

func (n *Notifier) Notify(
	ctx context.Context,
	event string,
	userIDs []uuid.UUID,
	payload any,
) {
	if n == nil || len(userIDs) == 0 {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Str("event", event).Msg("marshal notification failed")
		return
	}
	if n.stream != nil {
		n.stream.NotifyRaw(event, userIDs, data)
	}
	if n.db != nil {
		if err := n.db.PublishNotification(ctx, store.Notification{
			Origin: n.origin, Event: event, UserIDs: userIDs, Data: data,
		}); err != nil {
			log.Error().Err(err).Str("event", event).Msg("publish notification failed")
		}
	}
}

func (n *Notifier) NotifyRoom(
	ctx context.Context,
	roomID, exclude uuid.UUID,
	event string,
	payload any,
) {
	if n == nil {
		return
	}
	if n.members == nil {
		return
	}
	members, err := n.members.Members(ctx, roomID)
	if err != nil {
		log.Error().Err(err).Str("room_id", roomID.String()).Msg("load room members for notification failed")
		return
	}
	userIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		if member.UserID != exclude {
			userIDs = append(userIDs, member.UserID)
		}
	}
	n.Notify(ctx, event, userIDs, payload)
}
