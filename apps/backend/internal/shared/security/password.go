// Package security holds password hashing, token/API-key generation, and
// session cookie construction.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword returns a PHC-formatted argon2id hash.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether plain matches the encoded argon2id hash.
func VerifyPassword(plain, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("security: bad hash format")
	}
	// Fix 1+2: parse version and reject untrusted params before calling argon2.IDKey.
	// ponytail: we generate all hashes; require exact match to our consts — simpler than a range check.
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("security: bad version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("security: unsupported argon2 version %d", version)
	}
	var mem, tim uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &tim, &threads); err != nil {
		return false, fmt.Errorf("security: bad params: %w", err)
	}
	// Fix 2: reject params that deviate from our consts; t=0/p=0 would panic inside argon2.
	if mem != argonMemory || tim != argonTime || threads != argonThreads {
		return false, fmt.Errorf("security: hash params do not match expected (m=%d,t=%d,p=%d)", argonMemory, argonTime, argonThreads)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("security: bad salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("security: bad key: %w", err)
	}
	// Fix 1: reject wrong-length salt or key — subtle.ConstantTimeCompare([], []) == 1 → auth bypass.
	if len(want) != argonKeyLen || len(salt) != argonSaltLen {
		return false, fmt.Errorf("security: hash has unexpected key/salt length")
	}
	got := argon2.IDKey([]byte(plain), salt, tim, mem, threads, argonKeyLen)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
