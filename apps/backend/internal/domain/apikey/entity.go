// Package apikey is the API Key bounded context. Only the hash is persisted;
// the plaintext is generated once at creation. Imports only stdlib.
package apikey

import (
	"errors"
	"time"
)

// ErrNotFound is returned when an api_keys row is absent.
var ErrNotFound = errors.New("apikey: not found")

type APIKey struct {
	ID         string
	ProjectID  string
	Name       string
	KeyHash    string
	KeyPrefix  string
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}
