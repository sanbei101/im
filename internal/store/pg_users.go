package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const userColumns = `user_id::text, username, password, nickname, avatar_url, created_at`

func scanUser(row pgx.Row) (User, error) {
	var user User
	var id string
	err := row.Scan(&id, &user.Username, &user.Password, &user.Nickname, &user.AvatarURL, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	user.UserID, err = uuid.Parse(id)
	if err != nil {
		return User{}, fmt.Errorf("parse user_id %q: %w", id, err)
	}
	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (db *DB) CreateUser(ctx context.Context, username, password string) (User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return User{}, errors.New("username is required")
	}
	userID := uuid.NewV7()
	row := db.pool.QueryRow(ctx, `
		INSERT INTO users (user_id, username, password, nickname, avatar_url, created_at)
		VALUES ($1::uuid, $2, $3, $2, '', $4)
		RETURNING `+userColumns,
		userID.String(), username, password, time.Now(),
	)
	user, err := scanUser(row)
	if isUniqueViolation(err) {
		return User{}, ErrAlreadyExists
	}
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (db *DB) UserByID(ctx context.Context, id uuid.UUID) (User, error) {
	user, err := scanUser(db.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE user_id = $1::uuid`, id.String()))
	if err != nil {
		return User{}, fmt.Errorf("load user: %w", err)
	}
	return user, nil
}

func (db *DB) UserByUsername(ctx context.Context, username string) (User, error) {
	user, err := scanUser(db.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE lower(username) = lower($1)`, strings.TrimSpace(username)))
	if err != nil {
		return User{}, fmt.Errorf("load user by username: %w", err)
	}
	return user, nil
}

func (db *DB) UsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]User, error) {
	result := make(map[uuid.UUID]User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	raw := make([]string, len(ids))
	for i, id := range ids {
		raw[i] = id.String()
	}
	rows, err := db.pool.Query(ctx,
		`SELECT `+userColumns+` FROM users WHERE user_id = ANY($1::uuid[])`, raw)
	if err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		result[user.UserID] = user
	}
	return result, rows.Err()
}

func (db *DB) UpdateUserProfile(
	ctx context.Context,
	id uuid.UUID,
	nickname, avatarURL string,
) (User, error) {
	row := db.pool.QueryRow(ctx, `
		UPDATE users
		SET nickname = CASE WHEN $2 <> '' THEN $2 ELSE nickname END,
		    avatar_url = CASE WHEN $3 <> '' THEN $3 ELSE avatar_url END
		WHERE user_id = $1::uuid
		RETURNING `+userColumns,
		id.String(), strings.TrimSpace(nickname), avatarURL,
	)
	user, err := scanUser(row)
	if err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}
	return user, nil
}

func (db *DB) UpdateUserPassword(ctx context.Context, id uuid.UUID, password string) error {
	tag, err := db.pool.Exec(ctx,
		`UPDATE users SET password = $2 WHERE user_id = $1::uuid`, id.String(), password)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) SearchUsers(ctx context.Context, keyword string, limit int) ([]User, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	pattern := "%" + strings.ToLower(keyword) + "%"
	rows, err := db.pool.Query(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE lower(username) LIKE $1 OR lower(nickname) LIKE $1
		ORDER BY username
		LIMIT $2`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()
	users := make([]User, 0, limit)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan searched user: %w", err)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (db *DB) SaveDeviceToken(ctx context.Context, userID uuid.UUID, device DeviceInfo) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO devices (user_id, token, platform, updated_at)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET token = excluded.token, platform = excluded.platform, updated_at = excluded.updated_at`,
		userID.String(), device.Token, device.Platform, device.UpdatedAt)
	if isForeignKeyViolation(err) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("save device token: %w", err)
	}
	return nil
}

func (db *DB) DeviceToken(ctx context.Context, userID uuid.UUID) (DeviceInfo, error) {
	var device DeviceInfo
	err := db.pool.QueryRow(ctx,
		`SELECT token, platform, updated_at FROM devices WHERE user_id = $1::uuid`,
		userID.String()).Scan(&device.Token, &device.Platform, &device.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceInfo{}, ErrNotFound
	}
	if err != nil {
		return DeviceInfo{}, fmt.Errorf("load device token: %w", err)
	}
	return device, nil
}

func (db *DB) DeleteDeviceToken(ctx context.Context, userID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, `DELETE FROM devices WHERE user_id = $1::uuid`, userID.String())
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
