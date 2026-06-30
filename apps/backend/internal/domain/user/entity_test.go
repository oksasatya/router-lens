package user

import (
	"testing"
	"time"
)

func TestSessionIsExpired(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	if (Session{ExpiresAt: now.Add(time.Hour)}).IsExpired(now) {
		t.Fatal("future expiry should not be expired")
	}
	if !(Session{ExpiresAt: now.Add(-time.Hour)}).IsExpired(now) {
		t.Fatal("past expiry should be expired")
	}
}
