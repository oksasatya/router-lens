package pagination

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// Cursor is an opaque keyset pagination cursor.
//
// Complexity note: keyset pagination is O(log n + limit) via the index seek,
// versus offset pagination's O(n) skip through sorted rows.
// ponytail: cursor is intentionally a flat struct; add compression if payload grows.
type Cursor struct {
	Time time.Time
	ID   string
}

// EncodeCursor packs the cursor as base64url("<rfc3339nano>|<id>").
func EncodeCursor(c Cursor) string {
	raw := c.Time.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor reverses EncodeCursor. An empty string yields the zero Cursor
// (nil error), allowing callers to treat the first page uniformly.
func DecodeCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("pagination: decode cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return Cursor{}, fmt.Errorf("pagination: malformed cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, fmt.Errorf("pagination: cursor time: %w", err)
	}
	return Cursor{Time: ts, ID: parts[1]}, nil
}
