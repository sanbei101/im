package jwt

import (
	"testing"
)

func BenchmarkJWT(b *testing.B) {
	b.Run("generate", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := GenerateToken("01999999-9999-7999-9999-999999999999"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("parse", func(b *testing.B) {
		token, err := GenerateToken("01999999-9999-7999-9999-999999999999")
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for b.Loop() {
			if _, err := ParseToken(token); err != nil {
				b.Fatal(err)
			}
		}
	})
}
