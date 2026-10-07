package jwt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cristalhq/jwt/v5"
)

func TestJWT(t *testing.T) {
	t.Run("generate and parse round trip", func(t *testing.T) {
		token, err := GenerateToken("01999999-9999-7999-9999-999999999999")
		if err != nil {
			t.Fatal(err)
		}
		userID, err := ParseToken(token)
		if err != nil {
			t.Fatalf("parse own token: %v", err)
		}
		if userID != "01999999-9999-7999-9999-999999999999" {
			t.Fatalf("user id = %q", userID)
		}
	})

	t.Run("garbage and tampered tokens rejected", func(t *testing.T) {
		for name, token := range map[string]string{
			"garbage":  "not-a-token",
			"tampered": tamper(t, "user-123"),
			"empty":    "",
		} {
			if _, err := ParseToken(token); err == nil {
				t.Fatalf("%s token must be rejected", name)
			}
		}
	})

	t.Run("expired token rejected", func(t *testing.T) {
		claims := userClaims{
			UserID:    "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		}
		token, err := jwt.NewBuilder(jwtSigner).Build(claims)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseToken(token.String()); err == nil {
			t.Fatal("expired token must be rejected")
		}
	})

	t.Run("middleware authenticates and rejects", func(t *testing.T) {
		handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if id := GetUserIDFromContext(r); id != "user-123" {
				t.Errorf("handler got user id %q", id)
			}
			w.WriteHeader(http.StatusOK)
		}))

		token, err := GenerateToken("user-123")
		if err != nil {
			t.Fatal(err)
		}
		for name, tc := range map[string]struct {
			header string
			want   int
		}{
			"missing header": {"", http.StatusUnauthorized},
			"wrong scheme":   {"Basic " + token, http.StatusUnauthorized},
			"invalid token":  {"Bearer nope", http.StatusUnauthorized},
			"valid token":    {"Bearer " + token, http.StatusOK},
		} {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				request.Header.Set("Authorization", tc.header)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tc.want {
				t.Fatalf("%s: status = %d, want %d", name, recorder.Code, tc.want)
			}
		}
	})
}

// tamper flips a character inside the payload segment so signature
// verification must fail.
func tamper(t *testing.T, userID string) string {
	t.Helper()
	token, err := GenerateToken(userID)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token layout: %s", token)
	}
	payload := []byte(parts[1])
	if payload[0] == 'e' {
		payload[0] = 'f'
	} else {
		payload[0] = 'e'
	}
	return parts[0] + "." + string(payload) + "." + parts[2]
}
