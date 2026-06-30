package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	APIKeyPrefix    = "rl_live_"
	tokenBytes      = 32
	apiKeyRandBytes = 24
	// apiKeyPrefixSize is the display-safe prefix length returned from GenerateAPIKey.
	apiKeyPrefixSize = 12
)

func randomBase64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("security: read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// GenerateSessionToken returns an opaque, URL-safe random token (32 random bytes, base64url).
func GenerateSessionToken() (string, error) { return randomBase64(tokenBytes) }

// HashToken returns the hex sha256 of a token (stored server-side; never the raw token).
func HashToken(token string) string { return sha256Hex(token) }

// GenerateAPIKey returns the plaintext key (shown once to the user), a display prefix,
// and the sha256 hash to persist in the database.
func GenerateAPIKey() (plaintext, prefix, hash string, err error) {
	body, err := randomBase64(apiKeyRandBytes)
	if err != nil {
		return "", "", "", err
	}
	plaintext = APIKeyPrefix + body
	if len(plaintext) < apiKeyPrefixSize {
		prefix = plaintext
	} else {
		prefix = plaintext[:apiKeyPrefixSize]
	}
	return plaintext, prefix, sha256Hex(plaintext), nil
}

// HashAPIKey returns the hex sha256 of an API key plaintext.
// The result equals the hash field returned by GenerateAPIKey.
func HashAPIKey(plaintext string) string { return sha256Hex(plaintext) }
