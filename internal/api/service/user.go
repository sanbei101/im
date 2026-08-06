package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/jwt"
)

const refreshTokenLifetime = 30 * 24 * time.Hour

type UserService struct {
	query *db.Queries
	db    *pgxpool.Pool
}

func NewUserService(query *db.Queries, dbPool *pgxpool.Pool) *UserService {
	return &UserService{query: query, db: dbPool}
}

type RegisterReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UpdateProfileReq struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Bio         *string `json:"bio"`
}

type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=6"`
}

type UserResp struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
	Bio          string `json:"bio"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type BatchUserResp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

var (
	ErrUserExists       = errors.New("username already exists")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidSession   = errors.New("invalid refresh session")
	ErrSessionForbidden = errors.New("refresh session belongs to another user")
	ErrCountOutOfRange  = errors.New("count must be between 1 and 1000")
)

func (s *UserService) Register(ctx context.Context, req RegisterReq) (*UserResp, error) {
	if !validCredentials(req) {
		return nil, ErrInvalidInput
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin register transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msg("failed to rollback register transaction")
			}
		}
	}()
	q := s.query.WithTx(tx)
	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Username: req.Username,
		Password: string(hashed),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	resp, err := s.issueSessionWithQuery(ctx, q, user.UserID, user.Username, user.DisplayName, user.AvatarUrl, user.Bio)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit register transaction: %w", err)
	}
	committed = true
	return resp, nil
}

func (s *UserService) Login(ctx context.Context, req RegisterReq) (*UserResp, error) {
	if !validCredentials(req) {
		return nil, ErrInvalidInput
	}

	user, err := s.query.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	return s.issueSessionWithQuery(ctx, s.query, user.UserID, user.Username, user.DisplayName, user.AvatarUrl, user.Bio)
}

func (s *UserService) Refresh(ctx context.Context, refreshToken string) (*UserResp, error) {
	if refreshToken == "" {
		return nil, ErrInvalidSession
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin refresh transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msg("failed to rollback refresh transaction")
			}
		}
	}()

	q := s.query.WithTx(tx)
	session, err := q.GetRefreshSessionForUpdate(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidSession
		}
		return nil, fmt.Errorf("get refresh session: %w", err)
	}
	if session.RevokedAt.Valid || !session.ExpiresAt.After(time.Now()) {
		return nil, ErrInvalidSession
	}
	if err := q.RevokeRefreshSession(ctx, session.SessionID); err != nil {
		return nil, fmt.Errorf("revoke refresh session: %w", err)
	}

	user, err := q.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("get refresh user: %w", err)
	}
	resp, err := s.issueSessionWithQuery(ctx, q, user.UserID, user.Username, user.DisplayName, user.AvatarUrl, user.Bio)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh transaction: %w", err)
	}
	committed = true
	return resp, nil
}

func (s *UserService) Logout(ctx context.Context, userID, refreshToken string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	if refreshToken == "" {
		return ErrInvalidSession
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin logout transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msg("failed to rollback logout transaction")
			}
		}
	}()

	q := s.query.WithTx(tx)
	session, err := q.GetRefreshSessionForUpdate(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidSession
		}
		return fmt.Errorf("get refresh session for logout: %w", err)
	}
	if session.UserID != userUUID {
		return ErrSessionForbidden
	}
	if err := q.RevokeRefreshSession(ctx, session.SessionID); err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit logout transaction: %w", err)
	}
	committed = true
	return nil
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*UserResp, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	user, err := s.query.GetUserByID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return userResponse(user.UserID, user.Username, user.DisplayName, user.AvatarUrl, user.Bio), nil
}

func (s *UserService) SearchUsers(ctx context.Context, query string) ([]*UserResp, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 100 {
		return nil, ErrInvalidInput
	}
	rows, err := s.query.SearchPublicUsers(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	result := make([]*UserResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, userResponse(row.UserID, row.Username, row.DisplayName, row.AvatarUrl, row.Bio))
	}
	return result, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, req UpdateProfileReq) (*UserResp, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if req.DisplayName == nil && req.AvatarURL == nil && req.Bio == nil {
		return nil, ErrInvalidInput
	}
	current, err := s.query.GetUserByID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get profile for update: %w", err)
	}
	displayName, avatarURL, bio := current.DisplayName, current.AvatarUrl, current.Bio
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		avatarURL = *req.AvatarURL
	}
	if req.Bio != nil {
		bio = *req.Bio
	}
	if len(displayName) > 100 || len(avatarURL) > 1024 || len(bio) > 500 {
		return nil, ErrInvalidInput
	}
	user, err := s.query.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		UserID:      userUUID,
		DisplayName: displayName,
		AvatarUrl:   avatarURL,
		Bio:         bio,
	})
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return userResponse(user.UserID, user.Username, user.DisplayName, user.AvatarUrl, user.Bio), nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID string, req ChangePasswordReq) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	if req.CurrentPassword == "" || len(req.NewPassword) < 6 {
		return ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msg("failed to rollback password transaction")
			}
		}
	}()
	q := s.query.WithTx(tx)
	user, err := q.GetUserByID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get user for password change: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return ErrInvalidPassword
	}
	if err := q.UpdateUserPassword(
		ctx,
		db.UpdateUserPasswordParams{UserID: userUUID, Password: string(hash)},
	); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := q.RevokeUserRefreshSessions(ctx, userUUID); err != nil {
		return fmt.Errorf("revoke refresh sessions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit password transaction: %w", err)
	}
	committed = true
	return nil
}

func (s *UserService) issueSessionWithQuery(
	ctx context.Context,
	q *db.Queries,
	userID uuid.UUID,
	username, displayName, avatarURL, bio string,
) (*UserResp, error) {
	accessToken, err := jwt.GenerateToken(userID.String())
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	if _, err := q.CreateRefreshSession(ctx, db.CreateRefreshSessionParams{
		UserID:    userID,
		TokenHash: hashRefreshToken(refreshToken),
		ExpiresAt: time.Now().Add(refreshTokenLifetime),
	}); err != nil {
		return nil, fmt.Errorf("create refresh session: %w", err)
	}
	resp := userResponse(userID, username, displayName, avatarURL, bio)
	resp.Token = accessToken
	resp.RefreshToken = refreshToken
	return resp, nil
}

func validCredentials(req RegisterReq) bool {
	return req.Username != "" && len(req.Username) <= 100 && req.Password != "" && len(req.Password) >= 6
}

func userResponse(userID uuid.UUID, username, displayName, avatarURL, bio string) *UserResp {
	return &UserResp{
		UserID:      userID.String(),
		Username:    username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Bio:         bio,
	}
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
