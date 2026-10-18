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

const (
	FriendStatusPending  = "pending"
	FriendStatusAccepted = "accepted"
	FriendStatusRejected = "rejected"
)

func (db *DB) ApplyFriend(ctx context.Context, fromID, toID uuid.UUID, greeting string) error {
	if fromID == toID {
		return errors.New("cannot apply to add yourself as friend")
	}
	var friends bool
	if err := db.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM friendships WHERE user_id = $1::uuid AND friend_id = $2::uuid
		)`, fromID.String(), toID.String()).Scan(&friends); err != nil {
		return fmt.Errorf("check friendship: %w", err)
	}
	if friends {
		return ErrAlreadyExists
	}
	_, err := db.pool.Exec(ctx, `
		INSERT INTO friend_applications (from_user_id, to_user_id, greeting, status, created_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		ON CONFLICT (to_user_id, from_user_id) DO UPDATE
		SET greeting = excluded.greeting, status = excluded.status, created_at = excluded.created_at`,
		fromID.String(), toID.String(), greeting, FriendStatusPending, time.Now())
	if isForeignKeyViolation(err) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("apply friend: %w", err)
	}
	return nil
}

func (db *DB) AuditFriend(ctx context.Context, toID, fromID uuid.UUID, accept bool) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin friend audit: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Error().Err(err).Msg("rollback friend audit failed")
		}
	}()

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM friend_applications
		WHERE to_user_id = $1::uuid AND from_user_id = $2::uuid
		FOR UPDATE`, toID.String(), fromID.String()).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load application: %w", err)
	}
	if status != FriendStatusPending {
		return ErrConflict
	}
	next := FriendStatusRejected
	if accept {
		next = FriendStatusAccepted
	}
	if _, err := tx.Exec(ctx, `
		UPDATE friend_applications SET status = $3
		WHERE to_user_id = $1::uuid AND from_user_id = $2::uuid`,
		toID.String(), fromID.String(), next); err != nil {
		return fmt.Errorf("update application: %w", err)
	}
	if accept {
		now := time.Now()
		if _, err := tx.Exec(ctx, `
			INSERT INTO friendships (user_id, friend_id, remark, created_at)
			VALUES ($1::uuid, $2::uuid, '', $3), ($2::uuid, $1::uuid, '', $3)
			ON CONFLICT DO NOTHING`,
			toID.String(), fromID.String(), now); err != nil {
			return fmt.Errorf("create friendships: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit friend audit: %w", err)
	}
	return nil
}

func (db *DB) Applications(ctx context.Context, toID uuid.UUID) ([]FriendApplication, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT from_user_id::text, to_user_id::text, greeting, status, created_at
		FROM friend_applications WHERE to_user_id = $1::uuid
		ORDER BY created_at DESC`, toID.String())
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()
	var result []FriendApplication
	for rows.Next() {
		var app FriendApplication
		var fromID, toIDStr string
		if err := rows.Scan(&fromID, &toIDStr, &app.Greeting, &app.Status, &app.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		if app.FromUserID, err = uuid.Parse(fromID); err != nil {
			return nil, err
		}
		if app.ToUserID, err = uuid.Parse(toIDStr); err != nil {
			return nil, err
		}
		result = append(result, app)
	}
	return result, rows.Err()
}

func (db *DB) Friends(ctx context.Context, userID uuid.UUID) ([]Friend, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT friend_id::text, remark, created_at
		FROM friendships WHERE user_id = $1::uuid
		ORDER BY created_at`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	defer rows.Close()
	var result []Friend
	for rows.Next() {
		friend := Friend{UserID: userID}
		var friendID string
		if err := rows.Scan(&friendID, &friend.Remark, &friend.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan friend: %w", err)
		}
		if friend.FriendID, err = uuid.Parse(friendID); err != nil {
			return nil, err
		}
		result = append(result, friend)
	}
	return result, rows.Err()
}

func (db *DB) DeleteFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if _, err := db.pool.Exec(ctx, `
		DELETE FROM friendships
		WHERE (user_id = $1::uuid AND friend_id = $2::uuid)
		   OR (user_id = $2::uuid AND friend_id = $1::uuid)`,
		userID.String(), friendID.String()); err != nil {
		return fmt.Errorf("delete friend: %w", err)
	}
	return nil
}

func (db *DB) UpdateFriendRemark(
	ctx context.Context,
	userID, friendID uuid.UUID,
	remark string,
) error {
	tag, err := db.pool.Exec(ctx, `
		UPDATE friendships SET remark = $3
		WHERE user_id = $1::uuid AND friend_id = $2::uuid`,
		userID.String(), friendID.String(), remark)
	if err != nil {
		return fmt.Errorf("update friend remark: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) AddBlacklist(ctx context.Context, userID, targetID uuid.UUID) error {
	if userID == targetID {
		return errors.New("cannot blacklist yourself")
	}
	_, err := db.pool.Exec(ctx, `
		INSERT INTO blacklist (user_id, target_id, created_at)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT DO NOTHING`, userID.String(), targetID.String(), time.Now())
	if isForeignKeyViolation(err) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("add blacklist: %w", err)
	}
	return nil
}

func (db *DB) RemoveBlacklist(ctx context.Context, userID, targetID uuid.UUID) error {
	_, err := db.pool.Exec(ctx,
		`DELETE FROM blacklist WHERE user_id = $1::uuid AND target_id = $2::uuid`,
		userID.String(), targetID.String())
	if err != nil {
		return fmt.Errorf("remove blacklist: %w", err)
	}
	return nil
}

func (db *DB) Blacklist(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT target_id::text FROM blacklist
		WHERE user_id = $1::uuid ORDER BY created_at`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list blacklist: %w", err)
	}
	defer rows.Close()
	var result []uuid.UUID
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (db *DB) IsBlacklisted(ctx context.Context, userID, targetID uuid.UUID) (bool, error) {
	var blocked bool
	err := db.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM blacklist WHERE user_id = $1::uuid AND target_id = $2::uuid
		)`, userID.String(), targetID.String()).Scan(&blocked)
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	return blocked, nil
}
