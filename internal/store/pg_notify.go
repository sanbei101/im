package store

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/phuslu/log"
)

const notifyChannel = "im_events"

type Notification struct {
	Origin  string         `json:"origin"`
	Event   string         `json:"event"`
	UserIDs []uuid.UUID    `json:"user_ids"`
	Data    jsontext.Value `json:"data"`
}

func (db *DB) PublishNotification(ctx context.Context, notification Notification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("encode notification: %w", err)
	}
	if _, err := db.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, notifyChannel, string(payload)); err != nil {
		return fmt.Errorf("publish notification: %w", err)
	}
	return nil
}

// ListenNotifications delivers events published by every API node. The
// listener owns a dedicated connection because LISTEN is connection-scoped.
func (db *DB) ListenNotifications(ctx context.Context, handler func(Notification)) {
	for {
		if ctx.Err() != nil {
			return
		}
		conn, err := pgx.Connect(ctx, db.url)
		if err == nil {
			_, err = conn.Exec(ctx, "LISTEN "+notifyChannel)
		}
		if err != nil {
			log.Error().Err(err).Msg("listen notifications failed")
		} else {
			for {
				raw, waitErr := conn.WaitForNotification(ctx)
				if waitErr != nil {
					if !errors.Is(waitErr, context.Canceled) {
						log.Error().Err(waitErr).Msg("notification listener stopped")
					}
					break
				}
				var notification Notification
				if err := json.Unmarshal([]byte(raw.Payload), &notification); err != nil {
					log.Error().Err(err).Msg("decode notification failed")
					continue
				}
				handler(notification)
			}
		}
		if conn != nil {
			_ = conn.Close(context.Background())
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
