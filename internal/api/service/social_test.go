package service

import (
	"testing"

	"github.com/google/uuid"
)

func TestRelationUsers(t *testing.T) {
	id := uuid.New()
	if _, _, err := relationUsers(id.String(), id.String()); err != ErrCannotRelateSelf {
		t.Fatalf("same user error = %v", err)
	}
	other := uuid.New()
	left, right, err := relationUsers(id.String(), other.String())
	if err != nil || left != id || right != other {
		t.Fatalf("relationUsers = %s, %s, %v", left, right, err)
	}
}

func TestOrderedUsersIsStable(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	low1, high1 := orderedUsers(first, second)
	low2, high2 := orderedUsers(second, first)
	if low1 != low2 || high1 != high2 || low1 == high1 {
		t.Fatalf("ordered pairs differ: %s/%s vs %s/%s", low1, high1, low2, high2)
	}
}
