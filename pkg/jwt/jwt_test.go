package jwt

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseToken(t *testing.T) {
	userID := uuid.New().String()
	t.Log("userID:", userID)

	token, err := GenerateToken(userID)
	t.Log(token)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsedUserID, err := ParseToken(token)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if parsedUserID != userID {
		t.Errorf("expected userID %s, got %s", userID, parsedUserID)
	}
}
