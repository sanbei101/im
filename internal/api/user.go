package api

import (
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type UserAPI struct {
	store *store.Store
}

type UserAuthReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserResp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (a *UserAPI) Register(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[UserAuthReq](w, r)
	if err != nil {
		return
	}
	if strings.TrimSpace(req.Username) == "" || len(req.Password) < 6 {
		render.Error(w, http.StatusBadRequest, "invalid username or password")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	user, err := a.store.CreateUser(r.Context(), req.Username, string(hashed))
	if errors.Is(err, store.ErrAlreadyExists) {
		render.Error(w, http.StatusBadRequest, "username already exists")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "注册成功", UserResp{
		UserID:   user.UserID.String(),
		Username: user.Username,
		Token:    token,
	})
}

func (a *UserAPI) Login(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[UserAuthReq](w, r)
	if err != nil {
		return
	}
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		render.Error(w, http.StatusBadRequest, "invalid username or password")
		return
	}

	user, err := a.store.UserByUsername(r.Context(), req.Username)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusUnauthorized, "user not found")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		render.Error(w, http.StatusUnauthorized, "invalid password")
		return
	}

	token, err := jwt.GenerateToken(user.UserID.String())
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "登录成功", UserResp{
		UserID:   user.UserID.String(),
		Username: user.Username,
		Token:    token,
	})
}
