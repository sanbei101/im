package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/jwt"
)

type UserService struct {
	query *db.Queries
}

func NewUserService(query *db.Queries) *UserService {
	return &UserService{query: query}
}

type RegisterReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserResp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
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
	ErrCountOutOfRange = errors.New("count must be between 1 and 1000")
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

	user, err := s.query.CreateUser(ctx, db.CreateUserParams{
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

	user, err := s.query.GetUserByUsername(ctx, req.Username)
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
