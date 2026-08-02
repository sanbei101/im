package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unsafe"

	"github.com/phuslu/log"
	"github.com/sanbei101/im/pkg/render"

	"github.com/cristalhq/jwt/v5"
)

const (
	jwtExpiration        = 7 * 24 * time.Hour
	userIDKey     string = "user_id"
)

var (
	jwtSigner   jwt.Signer
	jwtVerifier jwt.Verifier
	jwtSecret   = []byte("blue-book-secret-key")
)

type userClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func init() {
	var err error
	jwtSigner, err = jwt.NewSignerHS(jwt.HS256, jwtSecret)
	if err != nil {
		log.Fatal().Err(err).Msg("初始化 JWT 签名器失败")
	}
	jwtVerifier, err = jwt.NewVerifierHS(jwt.HS256, jwtSecret)
	if err != nil {
		log.Fatal().Err(err).Msg("初始化 JWT 验证器失败")
	}
}

func GenerateToken(userID string) (string, error) {
	c := userClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	builder := jwt.NewBuilder(jwtSigner)
	token, err := builder.Build(c)
	if err != nil {
		return "", err
	}
	return token.String(), nil
}

func ParseToken(tokenStr string) (string, error) {
	var c userClaims

	tokenBytes := unsafe.Slice(unsafe.StringData(tokenStr), len(tokenStr))

	if err := jwt.ParseClaims(tokenBytes, jwtVerifier, &c); err != nil {
		return "", err
	}

	if !c.IsValidAt(time.Now()) {
		return "", errors.New("jwt token has expired")
	}

	return c.UserID, nil
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			render.Error(w, http.StatusUnauthorized, "未登录")
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			render.Error(w, http.StatusUnauthorized, "无效的认证格式")
			return
		}

		token, err := jwt.Parse([]byte(parts[1]), jwtVerifier)
		if err != nil {
			render.Error(w, http.StatusUnauthorized, "无效的登录凭证")
			return
		}

		var c userClaims
		if err := json.Unmarshal(token.Claims(), &c); err != nil {
			render.Error(w, http.StatusUnauthorized, "凭证数据解析失败")
			return
		}

		if !c.IsValidAt(time.Now()) {
			render.Error(w, http.StatusUnauthorized, "登录已过期")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, c.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(r *http.Request) string {
	id, _ := r.Context().Value(userIDKey).(string)
	return id
}
