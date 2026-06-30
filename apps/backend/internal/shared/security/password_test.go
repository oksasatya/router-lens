package security

import "testing"

func TestPassword(t *testing.T) {
	t.Run("hash then verify round-trips", func(t *testing.T) {
		enc, err := HashPassword("s3cret-pw")
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		ok, err := VerifyPassword("s3cret-pw", enc)
		if err != nil || !ok {
			t.Fatalf("verify correct: ok=%v err=%v", ok, err)
		}
	})

	t.Run("wrong password fails", func(t *testing.T) {
		enc, err := HashPassword("s3cret-pw")
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		ok, _ := VerifyPassword("wrong", enc)
		if ok {
			t.Fatal("expected verify to fail for wrong password")
		}
	})

	t.Run("salts differ per hash", func(t *testing.T) {
		a, err := HashPassword("same")
		if err != nil {
			t.Fatalf("hash a: %v", err)
		}
		b, err := HashPassword("same")
		if err != nil {
			t.Fatalf("hash b: %v", err)
		}
		if a == b {
			t.Fatal("expected unique salts to produce different encodings")
		}
	})

	// Regression: auth bypass — malformed hash with empty key segment must never verify true.
	t.Run("malformed hash empty key must not verify", func(t *testing.T) {
		// PHC string with empty final segment (no key bytes after base64-decode of "").
		// argon2.IDKey with keyLen=0 produces []byte{}, and
		// subtle.ConstantTimeCompare([], []) == 1 on the unfixed code → auth bypass.
		malformed := "$argon2id$v=19$m=65536,t=1,p=4$c29tZXNhbHQ$"
		ok, err := VerifyPassword("anything", malformed)
		if ok {
			t.Fatal("SECURITY: malformed hash with empty key verified true — auth bypass")
		}
		if err == nil {
			t.Log("note: returned false without error; acceptable, but an error is preferred")
		}
	})

	// Regression: DoS — hash with t=0 (time=0) must not panic and must return false/error.
	t.Run("hash with t=0 must not panic and must return false", func(t *testing.T) {
		// argon2.IDKey with time=0 panics on the unfixed code.
		zeroTime := "$argon2id$v=19$m=65536,t=0,p=4$c29tZXNhbHQ$c29tZWtleXNvbWVrZXlzb21la2V5c29tZWtleQ"
		ok, _ := VerifyPassword("anything", zeroTime)
		if ok {
			t.Fatal("hash with t=0 should not verify true")
		}
	})
}
