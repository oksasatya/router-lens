package security

import (
	"strings"
	"testing"
)

func TestTokens(t *testing.T) {
	t.Run("session token unique and hashable", func(t *testing.T) {
		a, err := GenerateSessionToken()
		if err != nil || a == "" {
			t.Fatalf("gen: %v", err)
		}
		b, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("gen b: %v", err)
		}
		if a == b {
			t.Fatal("tokens should be unique")
		}
		if h := HashToken(a); h != HashToken(a) {
			t.Fatal("hash should be deterministic")
		}
		if HashToken(a) == a {
			t.Fatal("hash must differ from token")
		}
	})

	t.Run("api key has prefix and stable hash", func(t *testing.T) {
		plain, prefix, hash, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("gen: %v", err)
		}
		if !strings.HasPrefix(plain, APIKeyPrefix) {
			t.Fatalf("missing prefix: %q", plain)
		}
		if !strings.HasPrefix(plain, prefix) || len(prefix) == 0 {
			t.Fatalf("prefix mismatch: %q vs %q", prefix, plain)
		}
		if HashAPIKey(plain) != hash {
			t.Fatal("HashAPIKey must match the generated hash")
		}
	})
}
