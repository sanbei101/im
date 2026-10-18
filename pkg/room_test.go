package pkg

import (
	"testing"
	"uuid"
)

func TestSingleRoomID(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()

	first := SingleRoomID(a, b)
	if again := SingleRoomID(a, b); again != first {
		t.Fatalf("single room id is not deterministic: %s != %s", first, again)
	}
	if reversed := SingleRoomID(b, a); reversed != first {
		t.Fatalf("single room id depends on user order: %s != %s", reversed, first)
	}
	if first == uuid.Nil() {
		t.Fatal("single room id is nil")
	}
	if first == SingleRoomID(a, uuid.NewV7()) {
		t.Fatal("different user pairs produced the same room id")
	}

	hash := SingleChatHash(a, b)
	if hash != SingleChatHash(b, a) {
		t.Fatal("single chat hash depends on user order")
	}
	var derived uuid.UUID
	copy(derived[:], hash[:16])
	derived[6] = (derived[6] & 0x0f) | 0x80
	derived[8] = (derived[8] & 0x3f) | 0x80
	if derived != first {
		t.Fatalf("room id does not match the chat hash: %s != %s", first, derived)
	}
}
