package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cristalhq/jwt/v5"

	"github.com/sanbei101/im/pkg/render"
)

const jwtExpiration = 15 * time.Minute

var ErrNotConfigured = errors.New("jwt is not configured")

type contextKey string

const userIDKey contextKey = "user_id"

var (
	jwtSigner   jwt.Signer
	jwtVerifier jwt.Verifier
)

type userClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Configure installs the signing key once during service startup.
func Configure(secret string) error {
	if secret == "" {
		return errors.New("jwt secret is required")
	}

	signer, err := jwt.NewSignerHS(jwt.HS256, []byte(secret))
	if err != nil {
		return err
	}
	verifier, err := jwt.NewVerifierHS(jwt.HS256, []byte(secret))
	if err != nil {
		return err
	}
	jwtSigner = signer
	jwtVerifier = verifier
	return nil
}

func GenerateToken(userID string) (string, error) {
	if jwtSigner == nil {
		return "", ErrNotConfigured
	}
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
	if jwtVerifier == nil {
		return "", ErrNotConfigured
	}
	var c userClaims
	if err := jwt.ParseClaims([]byte(tokenStr), jwtVerifier, &c); err != nil {
		return "", err
	}
	if !c.IsValidAt(time.Now()) {
		return "", errors.New("jwt token has expired")
	}
	return c.UserID, nil
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if jwtVerifier == nil {
			render.Error(w, http.StatusServiceUnavailable, "认证服务未配置")
			return
		}
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
