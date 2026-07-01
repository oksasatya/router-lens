package event

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func baseInput() IngestInput {
	return IngestInput{
		Provider:         "anthropic",
		Model:            "claude-sonnet-4-5",
		InputTokens:      12000,
		OutputTokens:     1800,
		RequestStartedAt: time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
	}
}

func TestValidate(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	maxBackdate := 7 * 24 * time.Hour

	t.Run("accepts a sane event", func(t *testing.T) {
		if err := Validate(baseInput(), now, maxBackdate); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("rejects negative tokens", func(t *testing.T) {
		in := baseInput()
		in.InputTokens = -1
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for negative tokens")
		}
	})
	t.Run("rejects future timestamp", func(t *testing.T) {
		in := baseInput()
		in.RequestStartedAt = now.Add(time.Hour)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for future timestamp")
		}
	})
	t.Run("rejects timestamp older than max backdate", func(t *testing.T) {
		in := baseInput()
		in.RequestStartedAt = now.Add(-8 * 24 * time.Hour)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for stale timestamp")
		}
	})
	t.Run("rejects out-of-range status code", func(t *testing.T) {
		in := baseInput()
		bad := 700
		in.StatusCode = &bad
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for status 700")
		}
	})
	t.Run("rejects oversized metadata", func(t *testing.T) {
		in := baseInput()
		in.Metadata = []byte(`{"x":"` + strings.Repeat("a", maxMetadataBytes) + `"}`)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for oversized metadata")
		}
	})
	t.Run("rejects negative latency", func(t *testing.T) {
		in := baseInput()
		neg := -5
		in.LatencyMs = &neg
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for negative latency")
		}
	})
	t.Run("validation error exposes an i18n code", func(t *testing.T) {
		in := baseInput()
		in.InputTokens = -1
		err := Validate(in, now, maxBackdate)
		var ve interface{ Code() string }
		if !errors.As(err, &ve) || ve.Code() == "" {
			t.Fatalf("want a coded validation error, got %v", err)
		}
	})
	t.Run("token count at the max boundary is accepted, one over is rejected", func(t *testing.T) {
		in := baseInput()
		in.InputTokens = maxTokens
		if err := Validate(in, now, maxBackdate); err != nil {
			t.Fatalf("want max tokens accepted, got %v", err)
		}
		in.InputTokens = maxTokens + 1
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for tokens one over the max")
		}
	})
	t.Run("status code just below the valid range is rejected", func(t *testing.T) {
		in := baseInput()
		bad := 99
		in.StatusCode = &bad
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for status 99")
		}
	})
	t.Run("status code just above the valid range is rejected", func(t *testing.T) {
		in := baseInput()
		bad := 600
		in.StatusCode = &bad
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatal("want error for status 600")
		}
	})
	t.Run("string field at maxStringLen is accepted, one over is rejected", func(t *testing.T) {
		in := baseInput()
		in.Provider = strings.Repeat("a", maxStringLen)
		if err := Validate(in, now, maxBackdate); err != nil {
			t.Fatalf("want %d-char provider accepted, got %v", maxStringLen, err)
		}
		in.Provider = strings.Repeat("a", maxStringLen+1)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatalf("want error for provider longer than %d chars", maxStringLen)
		}
	})
	t.Run("metadata at maxMetadataBytes is accepted, one byte over is rejected", func(t *testing.T) {
		in := baseInput()
		in.Metadata = make([]byte, maxMetadataBytes)
		if err := Validate(in, now, maxBackdate); err != nil {
			t.Fatalf("want %d-byte metadata accepted, got %v", maxMetadataBytes, err)
		}
		in.Metadata = make([]byte, maxMetadataBytes+1)
		if err := Validate(in, now, maxBackdate); err == nil {
			t.Fatalf("want error for metadata over %d bytes", maxMetadataBytes)
		}
	})
}
