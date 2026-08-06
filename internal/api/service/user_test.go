package service

import "testing"

func TestRefreshTokenIsRandomAndHashStable(t *testing.T) {
	first, err := newRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("generated identical refresh tokens")
	}
	if got, want := string(hashRefreshToken(first)), string(hashRefreshToken(first)); got != want {
		t.Fatal("refresh token hash is not stable")
	}
	if string(hashRefreshToken(first)) == string(hashRefreshToken(second)) {
		t.Fatal("distinct refresh tokens have the same hash")
	}
}

func TestUpdateProfileReqRequiresAField(t *testing.T) {
	if (&UpdateProfileReq{}).DisplayName != nil {
		t.Fatal("zero-value profile request should omit fields")
	}
}
