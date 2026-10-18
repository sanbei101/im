package store

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type RoomHead struct {
	LastSeq     uint64
	LastMsg     *Message
	LastMsgTime int64
	UpdatedAt   time.Time
}

func (db *DB) RoomHead(ctx context.Context, roomID uuid.UUID) (RoomHead, error) {
	var head RoomHead
	var seq int64
	var lastMsg []byte
	err := db.pool.QueryRow(ctx, `
		SELECT last_seq, last_msg, last_msg_time, updated_at
		FROM room_heads WHERE room_id = $1::uuid`,
		roomID.String()).Scan(&seq, &lastMsg, &head.LastMsgTime, &head.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoomHead{}, ErrNotFound
	}
	if err != nil {
		return RoomHead{}, fmt.Errorf("load room head: %w", err)
	}
	if seq > 0 {
		head.LastSeq = uint64(seq)
	}
	if len(lastMsg) > 0 {
		var message Message
		if err := json.Unmarshal(lastMsg, &message); err != nil {
			return RoomHead{}, fmt.Errorf("decode room head message: %w", err)
		}
		head.LastMsg = &message
	}
	return head, nil
}

type RoomHeadUpdate struct {
	RoomID  uuid.UUID
	Seq     uint64
	Message *Message
}

func (db *DB) UpdateRoomHead(ctx context.Context, roomID uuid.UUID, seq uint64, message *Message) error {
	return db.UpdateRoomHeads(ctx, []RoomHeadUpdate{{RoomID: roomID, Seq: seq, Message: message}})
}

// UpdateRoomHeads projects several room heads in one statement. Rooms that
// were dissolved after the message commit are filtered by the JOIN.
func (db *DB) UpdateRoomHeads(ctx context.Context, updates []RoomHeadUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	byRoom := make(map[uuid.UUID]RoomHeadUpdate, len(updates))
	for _, update := range updates {
		if old, ok := byRoom[update.RoomID]; !ok || update.Seq >= old.Seq {
			byRoom[update.RoomID] = update
		}
	}
	roomIDs := make([]string, 0, len(byRoom))
	seqs := make([]int64, 0, len(byRoom))
	messages := make([]string, 0, len(byRoom))
	times := make([]int64, 0, len(byRoom))
	for _, update := range byRoom {
		roomIDs = append(roomIDs, update.RoomID.String())
		seqs = append(seqs, int64(update.Seq))
		if update.Message == nil {
			messages = append(messages, "null")
			times = append(times, 0)
			continue
		}
		encoded, err := json.Marshal(update.Message)
		if err != nil {
			return fmt.Errorf("encode room head message: %w", err)
		}
		messages = append(messages, string(encoded))
		times = append(times, update.Message.ServerTime)
	}
	if _, err := db.pool.Exec(ctx, `
		INSERT INTO room_heads (room_id, last_seq, last_msg, last_msg_time, updated_at)
		SELECT u.room_id, u.last_seq, NULLIF(u.last_msg, 'null'::jsonb), u.last_msg_time, now()
		FROM unnest($1::uuid[], $2::bigint[], $3::jsonb[], $4::bigint[])
		     AS u(room_id, last_seq, last_msg, last_msg_time)
		JOIN rooms r ON r.room_id = u.room_id
		ON CONFLICT (room_id) DO UPDATE
		SET last_seq = GREATEST(room_heads.last_seq, excluded.last_seq),
		    last_msg = CASE
		        WHEN excluded.last_seq >= room_heads.last_seq THEN excluded.last_msg
		        ELSE room_heads.last_msg
		    END,
		    last_msg_time = CASE
		        WHEN excluded.last_seq >= room_heads.last_seq THEN excluded.last_msg_time
		        ELSE room_heads.last_msg_time
		    END,
		    updated_at = now()`,
		roomIDs, seqs, messages, times); err != nil {
		return fmt.Errorf("update room heads: %w", err)
	}
	return nil
}

func (db *DB) MarkRead(ctx context.Context, userID, roomID uuid.UUID, readSeq uint64) error {
	if _, err := db.pool.Exec(ctx, `
		INSERT INTO room_reads (user_id, room_id, read_seq, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, now())
		ON CONFLICT (user_id, room_id) DO UPDATE
		SET read_seq = GREATEST(room_reads.read_seq, excluded.read_seq),
		    updated_at = now()`,
		userID.String(), roomID.String(), int64(readSeq)); err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}
	return nil
}

func (db *DB) ReadUsers(
	ctx context.Context,
	roomID uuid.UUID,
	messageSeq uint64,
	members []Member,
) ([]uuid.UUID, error) {
	if len(members) == 0 {
		return nil, nil
	}
	raw := make([]string, len(members))
	for i, member := range members {
		raw[i] = member.UserID.String()
	}
	rows, err := db.pool.Query(ctx, `
		SELECT user_id::text FROM room_reads
		WHERE room_id = $1::uuid AND read_seq >= $2 AND user_id = ANY($3::uuid[])`,
		roomID.String(), int64(messageSeq), raw)
	if err != nil {
		return nil, fmt.Errorf("load read users: %w", err)
	}
	defer rows.Close()
	var result []uuid.UUID
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			return nil, err
		}
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
