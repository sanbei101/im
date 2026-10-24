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

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// insertMembers writes the whole roster in one statement. ON CONFLICT DO
// NOTHING plus RETURNING yields only members that were actually inserted.
func insertMembers(
	ctx context.Context,
	q rowQuerier,
	roomID uuid.UUID,
	members []Member,
) ([]Member, error) {
	if len(members) == 0 {
		return nil, nil
	}
	userIDs := make([]string, len(members))
	roles := make([]string, len(members))
	hidden := make([]bool, len(members))
	muted := make([]bool, len(members))
	pinned := make([]bool, len(members))
	for i, member := range members {
		userIDs[i] = member.UserID.String()
		roles[i] = member.Role
		hidden[i] = member.IsHidden
		muted[i] = member.IsMuted
		pinned[i] = member.IsPinned
	}
	rows, err := q.Query(ctx, `
		INSERT INTO room_members (room_id, user_id, role, is_hidden, is_muted, is_pinned)
		SELECT $1::uuid, u.user_id, u.role, u.is_hidden, u.is_muted, u.is_pinned
		FROM unnest($2::uuid[], $3::text[], $4::bool[], $5::bool[], $6::bool[])
		     AS u(user_id, role, is_hidden, is_muted, is_pinned)
		ON CONFLICT DO NOTHING
		RETURNING user_id::text, role, is_hidden, is_muted, is_pinned`,
		roomID.String(), userIDs, roles, hidden, muted, pinned)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("insert room members: %w", err)
	}
	defer rows.Close()
	added := make([]Member, 0, len(members))
	for rows.Next() {
		member := Member{RoomID: roomID}
		var userID string
		if err := rows.Scan(&userID, &member.Role, &member.IsHidden, &member.IsMuted, &member.IsPinned); err != nil {
			return nil, fmt.Errorf("scan inserted room member: %w", err)
		}
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("parse inserted member user_id: %w", err)
		}
		member.UserID = parsed
		added = append(added, member)
	}
	if err := rows.Err(); err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("iterate inserted room members: %w", err)
	}
	return added, nil
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
	if _, err := insertMembers(ctx, tx, room.RoomID, members); err != nil {
		return err
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
	return insertMembers(ctx, db.pool, roomID, newMembers)
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
