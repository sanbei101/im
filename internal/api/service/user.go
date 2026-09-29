package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
)

type UserService struct{ store *store.Store }

type RegisterReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserResp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

var (
	ErrUserExists      = errors.New("username already exists")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidInput    = errors.New("invalid input")
)

func NewUserService(s *store.Store) *UserService { return &UserService{store: s} }

func (s *UserService) Register(ctx context.Context, req RegisterReq) (*UserResp, error) {
	if strings.TrimSpace(req.Username) == "" || len(req.Password) < 6 {
		return nil, ErrInvalidInput
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := s.store.CreateUser(ctx, req.Username, string(hashed))
	if errors.Is(err, store.ErrAlreadyExists) {
		return nil, ErrUserExists
	}
	if err != nil {
		return nil, err
	}
	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		return nil, err
	}
	return &UserResp{UserID: user.UserID.String(), Username: user.Username, Token: token}, nil
}

func (s *UserService) Login(ctx context.Context, req RegisterReq) (*UserResp, error) {
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}
	user, err := s.store.UserByUsername(ctx, req.Username)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidPassword
	}
	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		return nil, err
	}
	return &UserResp{UserID: user.UserID.String(), Username: user.Username, Token: token}, nil
}
