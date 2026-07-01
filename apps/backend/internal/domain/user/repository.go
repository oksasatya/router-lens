package user

import "context"

// UserRepository is the port for persisting and querying User aggregates.
type UserRepository interface {
	CreateInitialAdmin(ctx context.Context, u *User) (created bool, err error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	AnyExists(ctx context.Context) (bool, error)
	UpdateName(ctx context.Context, id, name string) error
	UpdatePasswordHash(ctx context.Context, id, hash string) error
}

// SessionRepository is the port for persisting and querying Session aggregates.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	// DeleteByUserIDExceptTokenHash revokes every session belonging to userID except the
	// one identified by keepTokenHash (the session making the current request). Used after
	// a password change so a leaked session cookie stops working immediately.
	DeleteByUserIDExceptTokenHash(ctx context.Context, userID, keepTokenHash string) error
}
