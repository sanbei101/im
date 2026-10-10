package api

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/phuslu/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type UserAPI struct {
	store         *store.Store
	streamHandler *StreamHandler
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

type UpdateProfileReq struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type UserProfileResp struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

func (a *UserAPI) GetProfile(w http.ResponseWriter, r *http.Request) {
	var id uuid.UUID
	var err error
	if idStr := chi.URLParam(r, "id"); idStr == "" || idStr == "me" {
		id, err = getContextUserID(r)
	} else {
		id, err = uuid.Parse(idStr)
	}
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := a.store.UserByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取用户信息成功", UserProfileResp{
		UserID:    user.UserID.String(),
		Username:  user.Username,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
	})
}

func (a *UserAPI) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	req, err := render.ReadBody[UpdateProfileReq](w, r)
	if err != nil {
		return
	}

	user, err := a.store.UpdateUserProfile(r.Context(), id, req.Nickname, req.AvatarURL)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "更新用户资料成功", UserProfileResp{
		UserID:    user.UserID.String(),
		Username:  user.Username,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
	})
}

func (a *UserAPI) Search(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if strings.TrimSpace(keyword) == "" {
		render.Success(w, "搜索用户成功", []UserProfileResp{})
		return
	}

	users, err := a.store.SearchUsers(r.Context(), keyword, 20)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]UserProfileResp, len(users))
	for i, u := range users {
		result[i] = UserProfileResp{
			UserID:    u.UserID.String(),
			Username:  u.Username,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
		}
	}
	render.Success(w, "搜索用户成功", result)
}

type PresenceReq struct {
	UserIDs []string `json:"user_ids" validate:"required"`
}

type PresenceResp struct {
	Presence map[string]bool `json:"presence"`
}

func (a *UserAPI) Presence(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[PresenceReq](w, r)
	if err != nil {
		return
	}
	if len(req.UserIDs) > 100 {
		render.Error(w, http.StatusBadRequest, "too many user_ids (max 100)")
		return
	}
	result := make(map[string]bool, len(req.UserIDs))
	for _, uid := range req.UserIDs {
		online := false
		if a.streamHandler != nil {
			online = a.streamHandler.IsOnline(uid)
		}
		result[uid] = online
	}
	render.Success(w, "获取在线状态成功", PresenceResp{Presence: result})
}

type DeviceTokenReq struct {
	Token    string `json:"token"    validate:"required"`
	Platform string `json:"platform" validate:"required"`
}

func (a *UserAPI) SaveDeviceToken(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[DeviceTokenReq](w, r)
	if err != nil {
		return
	}
	userID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := a.store.SaveDeviceToken(r.Context(), userID, store.DeviceInfo{
		Token:     req.Token,
		Platform:  req.Platform,
		UpdatedAt: time.Now(),
	}); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.SuccessNoData(w, http.StatusOK, "保存设备Token成功")
}

func (a *UserAPI) Logout(w http.ResponseWriter, r *http.Request) {
	// Logout must succeed even with a bad token; the device token is best-effort.
	if userID, err := getContextUserID(r); err == nil {
		if delErr := a.store.DeleteDeviceToken(
			r.Context(),
			userID,
		); delErr != nil &&
			!errors.Is(delErr, store.ErrNotFound) {
			log.Error().Err(delErr).Msg("delete device token failed")
		}
	}
	render.SuccessNoData(w, http.StatusOK, "登出成功")
}

type UpdatePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

func (a *UserAPI) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	req, err := render.ReadBody[UpdatePasswordReq](w, r)
	if err != nil {
		return
	}

	if req.OldPassword == "" || len(req.NewPassword) < 6 {
		render.Error(w, http.StatusBadRequest, "invalid password format or new password too short (min 6 characters)")
		return
	}

	user, err := a.store.UserByID(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		render.Error(w, http.StatusBadRequest, "incorrect old password")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.store.UpdateUserPassword(r.Context(), userID, string(hashed)); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "修改密码成功")
}
