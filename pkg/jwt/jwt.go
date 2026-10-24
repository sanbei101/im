package jwt

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unsafe"

	"github.com/cristalhq/jwt/v5"

	"github.com/sanbei101/im/pkg/render"
)

const (
	jwtExpiration = 7 * 24 * time.Hour
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
)

var (
	jwtSigner   jwt.Signer
	jwtVerifier jwt.Verifier
)

type userClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Init sets the HMAC secret and builds the signer/verifier. It must be called
// once at startup before any token is generated or parsed.
func Init(secret string) error {
	if strings.TrimSpace(secret) == "" {
		return errors.New("jwt secret is empty")
	}
	signer, err := jwt.NewSignerHS(jwt.HS256, []byte(secret))
	if err != nil {
		return fmt.Errorf("create jwt signer: %w", err)
	}
	verifier, err := jwt.NewVerifierHS(jwt.HS256, []byte(secret))
	if err != nil {
		return fmt.Errorf("create jwt verifier: %w", err)
	}
	jwtSigner = signer
	jwtVerifier = verifier
	return nil
}

func GenerateToken(userID string) (string, error) {
	c := userClaims{
		UserID:    userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
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

	// cristalhq only reads the slice, so the token string can be viewed as
	// bytes without copying it.
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
	id, ok := r.Context().Value(userIDKey).(string)
	if !ok {
		return ""
	}
	return id
}
