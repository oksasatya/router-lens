package user

import "context"

// UserRepository is the port for persisting and querying User aggregates.
type UserRepository interface {
	CreateInitialAdmin(ctx context.Context, u *User) (created bool, err error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	AnyExists(ctx context.Context) (bool, error)
}

// SessionRepository is the port for persisting and querying Session aggregates.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}
