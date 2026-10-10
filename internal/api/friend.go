package api

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

type FriendAPI struct {
	store         *store.Store
	streamHandler *StreamHandler
}

type ApplyFriendReq struct {
	TargetID string `json:"target_id" validate:"required,uuid"`
	Greeting string `json:"greeting"`
}

type AuditFriendReq struct {
	FromUserID string `json:"from_user_id" validate:"required,uuid"`
	Action     string `json:"action"       validate:"required"` // accept or reject
}

type UpdateRemarkReq struct {
	Remark string `json:"remark"`
}

type BlacklistReq struct {
	TargetID string `json:"target_id" validate:"required,uuid"`
}

type FriendItemResp struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *FriendAPI) Apply(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	req, err := render.ReadBody[ApplyFriendReq](w, r)
	if err != nil {
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid target_id")
		return
	}

	if targetID == myID {
		render.Error(w, http.StatusBadRequest, "cannot add yourself as friend")
		return
	}

	// 检查目标用户是否存在
	if _, err := a.store.UserByID(r.Context(), targetID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "target user not found")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.store.ApplyFriend(r.Context(), myID, targetID, req.Greeting); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			render.Error(w, http.StatusBadRequest, "already friends")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if a.streamHandler != nil {
		data := map[string]any{
			"from_user_id": myID.String(),
			"greeting":     req.Greeting,
		}
		if myUser, err := a.store.UserByID(r.Context(), myID); err == nil {
			data["from_username"] = myUser.Username
			data["from_nickname"] = myUser.Nickname
			data["from_avatar_url"] = myUser.AvatarURL
		}
		if payload, err := json.Marshal(data); err == nil {
			if pushErr := a.streamHandler.PushNotification(
				r.Context(),
				targetID,
				"friend_application",
				payload,
			); pushErr != nil {
				log.Error().Err(pushErr).Msg("push friend_application notification failed")
			}
		}
	}

	render.SuccessNoData(w, http.StatusOK, "好友申请已发送")
}

func (a *FriendAPI) Audit(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	req, err := render.ReadBody[AuditFriendReq](w, r)
	if err != nil {
		return
	}

	fromID, err := uuid.Parse(req.FromUserID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid from_user_id")
		return
	}

	accept := strings.ToLower(req.Action) == "accept"
	if err := a.store.AuditFriend(r.Context(), myID, fromID, accept); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "application not found")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if accept && a.streamHandler != nil {
		data := map[string]any{
			"user_id": myID.String(),
		}
		if myUser, err := a.store.UserByID(r.Context(), myID); err == nil {
			data["username"] = myUser.Username
			data["nickname"] = myUser.Nickname
			data["avatar_url"] = myUser.AvatarURL
		}
		if payload, err := json.Marshal(data); err == nil {
			if pushErr := a.streamHandler.PushNotification(
				r.Context(),
				fromID,
				"friend_accepted",
				payload,
			); pushErr != nil {
				log.Error().Err(pushErr).Msg("push friend_accepted notification failed")
			}
		}
	}

	msg := "已同意好友申请"
	if !accept {
		msg = "已拒绝好友申请"
	}
	render.SuccessNoData(w, http.StatusOK, msg)
}

func (a *FriendAPI) ListApplications(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	apps, err := a.store.Applications(r.Context(), myID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取好友申请列表成功", apps)
}

func (a *FriendAPI) ListFriends(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	friends, err := a.store.Friends(r.Context(), myID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	ids := make([]uuid.UUID, len(friends))
	for i, f := range friends {
		ids[i] = f.FriendID
	}
	users, err := a.store.UsersByIDs(r.Context(), ids)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]FriendItemResp, 0, len(friends))
	for _, f := range friends {
		u, ok := users[f.FriendID]
		if !ok {
			continue
		}
		result = append(result, FriendItemResp{
			UserID:    u.UserID.String(),
			Username:  u.Username,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
			Remark:    f.Remark,
			CreatedAt: f.CreatedAt,
		})
	}

	render.Success(w, "获取好友列表成功", result)
}

func (a *FriendAPI) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	friendID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid friend id")
		return
	}

	if err := a.store.DeleteFriend(r.Context(), myID, friendID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "删除好友成功")
}

func (a *FriendAPI) UpdateRemark(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	friendID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid friend id")
		return
	}

	req, err := render.ReadBody[UpdateRemarkReq](w, r)
	if err != nil {
		return
	}

	if err := a.store.UpdateFriendRemark(r.Context(), myID, friendID, req.Remark); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "friend not found")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "更新备注成功")
}

func (a *FriendAPI) AddBlacklist(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	req, err := render.ReadBody[BlacklistReq](w, r)
	if err != nil {
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid target_id")
		return
	}

	if err := a.store.AddBlacklist(r.Context(), myID, targetID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "拉黑成功")
}

func (a *FriendAPI) RemoveBlacklist(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid target id")
		return
	}

	if err := a.store.RemoveBlacklist(r.Context(), myID, targetID); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "解除拉黑成功")
}

func (a *FriendAPI) ListBlacklist(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	targetIDs, err := a.store.Blacklist(r.Context(), myID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	users, err := a.store.UsersByIDs(r.Context(), targetIDs)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]UserProfileResp, 0, len(targetIDs))
	for _, id := range targetIDs {
		u, ok := users[id]
		if !ok {
			continue
		}
		result = append(result, UserProfileResp{
			UserID:    u.UserID.String(),
			Username:  u.Username,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
		})
	}

	render.Success(w, "获取黑名单成功", result)
}
