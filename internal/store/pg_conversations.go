package store

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgtype"
)

// Conversations returns every conversation for a user in one query. It joins
// the room projection with the user's read cursor so callers do not issue
// per-room head and read-cursor lookups.
func (db *DB) Conversations(ctx context.Context, userID uuid.UUID) ([]ConversationInfo, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT r.room_id::text, r.chat_type, r.name, r.avatar_url, r.notice,
		       r.single_chat_hash, r.created_at, r.updated_at,
		       m.room_id::text, m.user_id::text, m.role, m.is_hidden, m.is_muted, m.is_pinned,
		       COALESCE(h.last_seq, 0), h.last_msg, h.updated_at,
		       COALESCE(rs.read_seq, 0)
		FROM room_members m
		JOIN rooms r ON r.room_id = m.room_id
		LEFT JOIN room_heads h ON h.room_id = m.room_id
		LEFT JOIN room_reads rs ON rs.room_id = m.room_id AND rs.user_id = m.user_id
		WHERE m.user_id = $1::uuid
		ORDER BY COALESCE(h.last_msg_time, 0) DESC,
		         COALESCE(h.updated_at, r.updated_at) DESC,
		         r.room_id`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	result := make([]ConversationInfo, 0, 16)
	for rows.Next() {
		var (
			room          Room
			member        Member
			roomID        string
			memberRoomID  string
			memberUserID  string
			headSeq       int64
			lastMsg       []byte
			headUpdatedAt pgtype.Timestamptz
			readSeq       int64
		)
		if err := rows.Scan(
			&roomID, &room.ChatType, &room.Name, &room.AvatarURL, &room.Notice,
			&room.SingleChatHash, &room.CreatedAt, &room.UpdatedAt,
			&memberRoomID, &memberUserID, &member.Role,
			&member.IsHidden, &member.IsMuted, &member.IsPinned,
			&headSeq, &lastMsg, &headUpdatedAt, &readSeq,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		if room.RoomID, err = uuid.Parse(roomID); err != nil {
			return nil, fmt.Errorf("parse conversation room_id: %w", err)
		}
		if member.RoomID, err = uuid.Parse(memberRoomID); err != nil {
			return nil, fmt.Errorf("parse conversation member room_id: %w", err)
		}
		if member.UserID, err = uuid.Parse(memberUserID); err != nil {
			return nil, fmt.Errorf("parse conversation user_id: %w", err)
		}

		var lastMessage *Message
		if len(lastMsg) > 0 && string(lastMsg) != "null" {
			var message Message
			if err := json.Unmarshal(lastMsg, &message); err != nil {
				return nil, fmt.Errorf("decode conversation message: %w", err)
			}
			lastMessage = &message
		}
		room.LastSeq = uint64(headSeq)
		if headUpdatedAt.Valid {
			room.UpdatedAt = headUpdatedAt.Time
		}
		var unread uint64
		if room.LastSeq > uint64(readSeq) {
			unread = room.LastSeq - uint64(readSeq)
		}
		result = append(result, ConversationInfo{
			Room:        room,
			Member:      member,
			UnreadCount: unread,
			LastMessage: lastMessage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return result, nil
}
