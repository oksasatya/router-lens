package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"router-lens/internal/domain/user"
)

func testPool(t *testing.T) context.Context {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	return context.Background()
}

func TestAuthRepositories(t *testing.T) {
	ctx := testPool(t)
	pool, err := NewPool(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE users, sessions CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	users := NewUserRepository(pool)
	sessions := NewSessionRepository(pool)

	admin := &user.User{Email: "admin@example.com", PasswordHash: "hash", Name: "Admin"}
	created, err := users.CreateInitialAdmin(ctx, admin)
	if err != nil || !created || admin.ID == "" {
		t.Fatalf("first admin should be created: created=%v err=%v", created, err)
	}
	again, _ := users.CreateInitialAdmin(ctx, &user.User{Email: "x@y.com", PasswordHash: "h"})
	if again {
		t.Fatal("second CreateInitialAdmin must be locked (false)")
	}

	got, err := users.FindByEmail(ctx, "admin@example.com")
	if err != nil || got.ID != admin.ID {
		t.Fatalf("FindByEmail: %v %+v", err, got)
	}
	if _, err := users.FindByEmail(ctx, "nobody@x.com"); err != user.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	s := &user.Session{UserID: admin.ID, TokenHash: "abc", ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessions.Create(ctx, s); err != nil {
		t.Fatalf("session create: %v", err)
	}
	if found, err := sessions.FindByTokenHash(ctx, "abc"); err != nil || found.UserID != admin.ID {
		t.Fatalf("session find: %v %+v", err, found)
	}
	if err := sessions.DeleteByTokenHash(ctx, "abc"); err != nil {
		t.Fatalf("session delete: %v", err)
	}
	if _, err := sessions.FindByTokenHash(ctx, "abc"); err != user.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
