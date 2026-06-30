// Package user is the auth bounded context: the User identity, the Session that
// authenticates a dashboard request, and their repository ports. Imports only
// stdlib (domain purity).
package user

import (
	"errors"
	"time"
)

// ErrNotFound is returned by repositories when a row is absent.
var ErrNotFound = errors.New("user: not found")

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	UserAgent string
	IP        string
}

// IsExpired reports whether the session is no longer valid at now.
func (s Session) IsExpired(now time.Time) bool { return !now.Before(s.ExpiresAt) }
