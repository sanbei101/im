package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/phuslu/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/jwt"
)

type UserService struct {
	q *db.Queries
}

func NewUserService(q *db.Queries) *UserService {
	return &UserService{q: q}
}

type RegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type BatchGenerateReq struct {
	Count int `json:"count"`
}

type BatchGenerateResp struct {
	Users []BatchUserResp `json:"users"`
}

type BatchUserResp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

var (
	ErrUserExists      = errors.New("username already exists")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrCountOutOfRange = errors.New("count must be between 1 and 100")
)

func (s *UserService) Register(ctx context.Context, req RegisterReq) (*UserResp, error) {
	if req.Username == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}
	if len(req.Password) < 6 {
		return nil, ErrInvalidInput
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Username: req.Username,
		Password: string(hashed),
	})
	if err != nil {
		return nil, err
	}

	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		return nil, err
	}

	return &UserResp{
		UserID:   user.UserID.String(),
		Username: user.Username,
		Token:    token,
	}, nil
}

func (s *UserService) Login(ctx context.Context, req RegisterReq) (*UserResp, error) {
	if req.Username == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.q.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		return nil, err
	}

	return &UserResp{
		UserID:   user.UserID.String(),
		Username: user.Username,
		Token:    token,
	}, nil
}

func (s *UserService) BatchGenerate(ctx context.Context, req BatchGenerateReq) (*BatchGenerateResp, error) {
	if req.Count < 1 || req.Count > 100 {
		return nil, ErrCountOutOfRange
	}

	users := make([]BatchUserResp, req.Count)
	batchCreatedUser := make([]db.BatchCreateUsersParams, req.Count)
	for i := 0; i < req.Count; i++ {
		username := "user_" + uuid.New().String()[:8]
		password := randomPassword(12)
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Error().Err(err).Msg("failed to hash password")
			return nil, err
		}
		batchCreatedUser[i] = db.BatchCreateUsersParams{
			Username: username,
			Password: string(hashed),
		}
		users[i] = BatchUserResp{
			Username: username,
			Password: password,
		}
	}

	result := s.q.BatchCreateUsers(ctx, batchCreatedUser)
	defer result.Close()
	var batchErr error
	result.QueryRow(func(i int, returnedID uuid.UUID, err error) {
		if err != nil {
			if batchErr == nil {
				batchErr = err
				log.Error().Err(err).
					Int("index", i).
					Str("username", batchCreatedUser[i].Username).
					Msg("failed to execute batch insert for user")
			}
			return
		}
		users[i].UserID = returnedID.String()
	})
	if batchErr != nil {
		return nil, batchErr
	}
	for i := 0; i < req.Count; i++ {
		token, err := jwt.GenerateToken(users[i].UserID)
		if err != nil {
			log.Error().Err(err).Str("user_id", users[i].UserID).Msg("failed to generate token for user")
			return nil, err
		}
		users[i].Token = token
	}

	return &BatchGenerateResp{Users: users}, nil
}

func randomPassword(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[int(uuid.New().ID())%len(letters)]
	}
	return string(b)
}
