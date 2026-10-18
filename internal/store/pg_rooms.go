package store

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/phuslu/log"
)

const roomColumns = `room_id::text, chat_type, name, avatar_url, notice,
	single_chat_hash, created_at, updated_at`

func scanRoom(row pgx.Row) (Room, error) {
	var room Room
	var id string
	err := row.Scan(&id, &room.ChatType, &room.Name, &room.AvatarURL, &room.Notice,
		&room.SingleChatHash, &room.CreatedAt, &room.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, ErrNotFound
	}
	if err != nil {
		return Room{}, err
	}
	room.RoomID, err = uuid.Parse(id)
	if err != nil {
		return Room{}, fmt.Errorf("parse room_id %q: %w", id, err)
	}
	return room, nil
}

func scanMember(row pgx.Row) (Member, error) {
	var member Member
	var roomID, userID string
	err := row.Scan(&roomID, &userID, &member.Role, &member.IsHidden, &member.IsMuted, &member.IsPinned)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	if err != nil {
		return Member{}, err
	}
	member.RoomID, err = uuid.Parse(roomID)
	if err != nil {
		return Member{}, fmt.Errorf("parse member room_id %q: %w", roomID, err)
	}
	member.UserID, err = uuid.Parse(userID)
	if err != nil {
		return Member{}, fmt.Errorf("parse member user_id %q: %w", userID, err)
	}
	return member, nil
}

func (db *DB) CreateRoom(ctx context.Context, room Room, members []Member) error {
	if room.RoomID == uuid.Nil() {
		return errors.New("room id is required")
	}
	if room.CreatedAt.IsZero() {
		room.CreatedAt = time.Now()
	}
	if room.UpdatedAt.IsZero() {
		room.UpdatedAt = room.CreatedAt
	}
	var singleHash []byte
	if len(room.SingleChatHash) > 0 {
		singleHash = room.SingleChatHash
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create room: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("rollback room transaction failed")
		}
	}()
	if _, err := tx.Exec(ctx, `
		INSERT INTO rooms (room_id, chat_type, name, avatar_url, notice, single_chat_hash, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`,
		room.RoomID.String(), room.ChatType, room.Name, room.AvatarURL, room.Notice,
		singleHash, room.CreatedAt, room.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert room: %w", err)
	}
	for _, member := range members {
		if _, err := tx.Exec(ctx, `
			INSERT INTO room_members (room_id, user_id, role, is_hidden, is_muted, is_pinned)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
			ON CONFLICT DO NOTHING`,
			room.RoomID.String(), member.UserID.String(), member.Role,
			member.IsHidden, member.IsMuted, member.IsPinned); err != nil {
			if isForeignKeyViolation(err) {
				return ErrNotFound
			}
			return fmt.Errorf("insert room member: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create room: %w", err)
	}
	return nil
}

func (db *DB) Room(ctx context.Context, roomID uuid.UUID) (Room, error) {
	room, err := scanRoom(db.pool.QueryRow(ctx,
		`SELECT `+roomColumns+` FROM rooms WHERE room_id = $1::uuid`, roomID.String()))
	if err != nil {
		return Room{}, fmt.Errorf("load room: %w", err)
	}
	return room, nil
}

func (db *DB) RoomBySingleHash(ctx context.Context, hash []byte) (Room, error) {
	if len(hash) == 0 {
		return Room{}, ErrNotFound
	}
	room, err := scanRoom(db.pool.QueryRow(ctx,
		`SELECT `+roomColumns+` FROM rooms WHERE single_chat_hash = $1`, hash))
	if err != nil {
		return Room{}, fmt.Errorf("load single room: %w", err)
	}
	return room, nil
}

func (db *DB) Members(ctx context.Context, roomID uuid.UUID) ([]Member, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT room_id::text, user_id::text, role, is_hidden, is_muted, is_pinned
		FROM room_members WHERE room_id = $1::uuid
		ORDER BY user_id`, roomID.String())
	if err != nil {
		return nil, fmt.Errorf("list room members: %w", err)
	}
	defer rows.Close()
	var result []Member
	for rows.Next() {
		member, err := scanMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan room member: %w", err)
		}
		result = append(result, member)
	}
	return result, rows.Err()
}

func (db *DB) Member(ctx context.Context, roomID, userID uuid.UUID) (Member, error) {
	member, err := scanMember(db.pool.QueryRow(ctx, `
		SELECT room_id::text, user_id::text, role, is_hidden, is_muted, is_pinned
		FROM room_members WHERE room_id = $1::uuid AND user_id = $2::uuid`,
		roomID.String(), userID.String()))
	if err != nil {
		return Member{}, fmt.Errorf("load room member: %w", err)
	}
	return member, nil
}

func (db *DB) RoomsByUser(ctx context.Context, userID uuid.UUID) ([]RoomInfo, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT r.room_id::text, r.chat_type, r.name, r.avatar_url, r.notice,
		       r.single_chat_hash, r.created_at, r.updated_at,
		       m.room_id::text, m.user_id::text, m.role, m.is_hidden, m.is_muted, m.is_pinned
		FROM room_members m
		JOIN rooms r ON r.room_id = m.room_id
		WHERE m.user_id = $1::uuid
		ORDER BY r.updated_at DESC`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list user rooms: %w", err)
	}
	defer rows.Close()
	var result []RoomInfo
	for rows.Next() {
		var room Room
		var member Member
		var roomID, memberRoomID, memberUserID string
		if err := rows.Scan(
			&roomID, &room.ChatType, &room.Name, &room.AvatarURL, &room.Notice,
			&room.SingleChatHash, &room.CreatedAt, &room.UpdatedAt,
			&memberRoomID, &memberUserID, &member.Role, &member.IsHidden, &member.IsMuted, &member.IsPinned,
		); err != nil {
			return nil, fmt.Errorf("scan user room: %w", err)
		}
		if room.RoomID, err = uuid.Parse(roomID); err != nil {
			return nil, err
		}
		if member.RoomID, err = uuid.Parse(memberRoomID); err != nil {
			return nil, err
		}
		if member.UserID, err = uuid.Parse(memberUserID); err != nil {
			return nil, err
		}
		result = append(result, RoomInfo{Room: room, Member: member})
	}
	return result, rows.Err()
}

func (db *DB) UpdateRoom(
	ctx context.Context,
	roomID uuid.UUID,
	name, avatarURL, notice string,
) (Room, error) {
	room, err := scanRoom(db.pool.QueryRow(ctx, `
		UPDATE rooms
		SET name = CASE WHEN $2 <> '' THEN $2 ELSE name END,
		    avatar_url = CASE WHEN $3 <> '' THEN $3 ELSE avatar_url END,
		    notice = CASE WHEN $4 <> '' THEN $4 ELSE notice END,
		    updated_at = now()
		WHERE room_id = $1::uuid
		RETURNING `+roomColumns,
		roomID.String(), name, avatarURL, notice))
	if err != nil {
		return Room{}, fmt.Errorf("update room: %w", err)
	}
	return room, nil
}

func (db *DB) AddMembers(
	ctx context.Context,
	roomID uuid.UUID,
	newMembers []Member,
) ([]Member, error) {
	added := make([]Member, 0, len(newMembers))
	for _, member := range newMembers {
		var userID string
		err := db.pool.QueryRow(ctx, `
			INSERT INTO room_members (room_id, user_id, role, is_hidden, is_muted, is_pinned)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
			ON CONFLICT DO NOTHING
			RETURNING user_id::text`,
			roomID.String(), member.UserID.String(), member.Role,
			member.IsHidden, member.IsMuted, member.IsPinned).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("add room member: %w", err)
		}
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return nil, err
		}
		added = append(added, Member{RoomID: roomID, UserID: parsed, Role: member.Role})
	}
	return added, nil
}

func (db *DB) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, `
		DELETE FROM room_members WHERE room_id = $1::uuid AND user_id = $2::uuid`,
		roomID.String(), userID.String())
	if err != nil {
		return fmt.Errorf("remove room member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) UpdateMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string) error {
	tag, err := db.pool.Exec(ctx, `
		UPDATE room_members SET role = $3
		WHERE room_id = $1::uuid AND user_id = $2::uuid`,
		roomID.String(), userID.String(), role)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) TransferOwnership(ctx context.Context, roomID, from, to uuid.UUID) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transfer ownership: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("rollback room transaction failed")
		}
	}()
	tag, err := tx.Exec(ctx, `
		UPDATE room_members SET role = $3
		WHERE room_id = $1::uuid AND user_id = $2::uuid AND role = $4`,
		roomID.String(), from.String(), RoleMember, RoleOwner)
	if err != nil {
		return fmt.Errorf("demote owner: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	tag, err = tx.Exec(ctx, `
		UPDATE room_members SET role = $3
		WHERE room_id = $1::uuid AND user_id = $2::uuid`,
		roomID.String(), to.String(), RoleOwner)
	if err != nil {
		return fmt.Errorf("promote owner: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transfer ownership: %w", err)
	}
	return nil
}

func (db *DB) UpdateMemberSettings(
	ctx context.Context,
	roomID, userID uuid.UUID,
	isPinned, isMuted *bool,
) error {
	tag, err := db.pool.Exec(ctx, `
		UPDATE room_members
		SET is_pinned = COALESCE($3, is_pinned),
		    is_muted = COALESCE($4, is_muted)
		WHERE room_id = $1::uuid AND user_id = $2::uuid`,
		roomID.String(), userID.String(), isPinned, isMuted)
	if err != nil {
		return fmt.Errorf("update member settings: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) DissolveRoom(ctx context.Context, roomID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, `DELETE FROM rooms WHERE room_id = $1::uuid`, roomID.String())
	if err != nil {
		return fmt.Errorf("dissolve room: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
