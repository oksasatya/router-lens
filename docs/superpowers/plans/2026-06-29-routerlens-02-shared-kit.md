# RouterLens — Plan 02: Shared Kit, Security & Cost Calculator

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the reusable, well-tested cross-cutting kit every later plan depends on — security primitives (argon2id, opaque session tokens, API keys, session cookie), validation, pagination (offset + keyset), date-range parsing, streaming CSV, and the pure cost calculator.

**Architecture:** Pure, dependency-light packages under `internal/shared/*` plus the domain cost calculator under `internal/domain/pricing`. No HTTP routing, no DB. Everything here is unit-testable in isolation and is TDD-first (security + money + parsers — exactly the code where "looks right" is dangerous).

**Tech Stack:** Go 1.26 stdlib, `golang.org/x/crypto/argon2`, `github.com/shopspring/decimal`, `crypto/rand`, `crypto/sha256`, `crypto/subtle`.

## Global Constraints

Inherits Plan 01's Global Constraints (module `router-lens`, hexagonal layering, Sonar block, golang-expert hub, verification routine). Plan-specific:

- **TDD = YES for every task in this plan.** Red test first, then minimal implementation. This is the money/security/parser layer.
- **Money is `decimal.Decimal`** (`shopspring/decimal`), never `float64`. Compute `tokens * price / 1_000_000` (multiply before divide).
- **Secrets** are generated with `crypto/rand` and stored only as `sha256`.
- `internal/domain/pricing` imports **nothing** outside stdlib + `shopspring/decimal` (domain purity).

---

### Task 1: Dependencies

**TDD:** no — dependency fetch.

**Files:** Modify `go.mod` / `go.sum`.

- [ ] **Step 1: Add deps**

```bash
go get golang.org/x/crypto@latest
go get github.com/shopspring/decimal@latest
go get github.com/go-playground/validator/v10@latest
go get github.com/go-playground/universal-translator@latest
go get github.com/go-playground/locales@latest
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add x/crypto, shopspring/decimal, go-playground validator+translator"
```

---

### Task 2: Password hashing (argon2id)

**TDD:** yes — security primitive with a clear contract.

**Files:**
- Create: `internal/shared/security/password.go`
- Test: `internal/shared/security/password_test.go`

**Interfaces:**
- Produces:
  ```go
  func HashPassword(plain string) (string, error)       // returns PHC-style argon2id string
  func VerifyPassword(plain, encoded string) (bool, error)
  ```

- [ ] **Step 1: Write the failing test**

```go
package security

import "testing"

func TestPassword(t *testing.T) {
	t.Run("hash then verify round-trips", func(t *testing.T) {
		enc, err := HashPassword("s3cret-pw")
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		ok, err := VerifyPassword("s3cret-pw", enc)
		if err != nil || !ok {
			t.Fatalf("verify correct: ok=%v err=%v", ok, err)
		}
	})

	t.Run("wrong password fails", func(t *testing.T) {
		enc, _ := HashPassword("s3cret-pw")
		ok, _ := VerifyPassword("wrong", enc)
		if ok {
			t.Fatal("expected verify to fail for wrong password")
		}
	})

	t.Run("salts differ per hash", func(t *testing.T) {
		a, _ := HashPassword("same")
		b, _ := HashPassword("same")
		if a == b {
			t.Fatal("expected unique salts to produce different encodings")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/security/ -run TestPassword -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
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
	var mem, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &time, &threads); err != nil {
		return false, fmt.Errorf("security: bad params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("security: bad salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("security: bad key: %w", err)
	}
	got := argon2.IDKey([]byte(plain), salt, time, mem, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/security/ -run TestPassword -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/security/password.go internal/shared/security/password_test.go
git commit -m "feat: argon2id password hashing"
```

---

### Task 3: Session token + API key generation

**TDD:** yes — security primitives.

**Files:**
- Create: `internal/shared/security/token.go`
- Test: `internal/shared/security/token_test.go`

**Interfaces:**
- Produces:
  ```go
  func GenerateSessionToken() (token string, err error)   // 32 random bytes, base64url
  func HashToken(token string) string                      // sha256 hex
  const APIKeyPrefix = "rl_live_"
  func GenerateAPIKey() (plaintext, prefix, hash string, err error)
  func HashAPIKey(plaintext string) string                 // sha256 hex; == hash from GenerateAPIKey
  ```

- [ ] **Step 1: Write the failing test**

```go
package security

import (
	"strings"
	"testing"
)

func TestTokens(t *testing.T) {
	t.Run("session token unique and hashable", func(t *testing.T) {
		a, err := GenerateSessionToken()
		if err != nil || a == "" {
			t.Fatalf("gen: %v", err)
		}
		b, _ := GenerateSessionToken()
		if a == b {
			t.Fatal("tokens should be unique")
		}
		if h := HashToken(a); h != HashToken(a) {
			t.Fatal("hash should be deterministic")
		}
		if HashToken(a) == a {
			t.Fatal("hash must differ from token")
		}
	})

	t.Run("api key has prefix and stable hash", func(t *testing.T) {
		plain, prefix, hash, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("gen: %v", err)
		}
		if !strings.HasPrefix(plain, APIKeyPrefix) {
			t.Fatalf("missing prefix: %q", plain)
		}
		if !strings.HasPrefix(plain, prefix) || len(prefix) == 0 {
			t.Fatalf("prefix mismatch: %q vs %q", prefix, plain)
		}
		if HashAPIKey(plain) != hash {
			t.Fatal("HashAPIKey must match the generated hash")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/security/ -run TestTokens -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	APIKeyPrefix     = "rl_live_"
	tokenBytes       = 32
	apiKeyRandBytes  = 24
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

// GenerateSessionToken returns an opaque, URL-safe random token.
func GenerateSessionToken() (string, error) { return randomBase64(tokenBytes) }

// HashToken returns the hex sha256 of a token (what gets stored server-side).
func HashToken(token string) string { return sha256Hex(token) }

// GenerateAPIKey returns the plaintext key (shown once), a display prefix, and
// the sha256 hash to persist.
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
func HashAPIKey(plaintext string) string { return sha256Hex(plaintext) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/security/ -run TestTokens -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/security/token.go internal/shared/security/token_test.go
git commit -m "feat: session token and API key generation/hashing"
```

---

### Task 4: Session cookie builder

**TDD:** yes — cookie attributes are security-critical.

**Files:**
- Create: `internal/shared/security/cookie.go`
- Test: `internal/shared/security/cookie_test.go`

**Interfaces:**
- Produces:
  ```go
  const SessionCookieName = "rl_session"
  type CookieOpts struct { Secure bool; CrossSite bool; MaxAge time.Duration }
  func BuildSessionCookie(token string, o CookieOpts) *http.Cookie
  func ClearSessionCookie(o CookieOpts) *http.Cookie
  ```
  No CSRF token in v0.1: the default same-origin deployment relies on `SameSite=Lax` plus the rule
  that no GET mutates state (CLAUDE.md decision 13). `CrossSite` flips the cookie to
  `SameSite=None; Secure` for a reverse-proxied split-origin setup (a documented escape hatch).

- [ ] **Step 1: Write the failing test**

```go
package security

import (
	"net/http"
	"testing"
	"time"
)

func TestSessionCookie(t *testing.T) {
	t.Run("same-origin uses Lax + HttpOnly + Secure", func(t *testing.T) {
		c := BuildSessionCookie("tok", CookieOpts{Secure: true, CrossSite: false, MaxAge: time.Hour})
		if c.Name != SessionCookieName || c.Value != "tok" || !c.HttpOnly {
			t.Fatalf("bad cookie: %+v", c)
		}
		if c.SameSite != http.SameSiteLaxMode || !c.Secure {
			t.Fatalf("expected Lax+Secure: %+v", c)
		}
	})

	t.Run("cross-site forces None + Secure", func(t *testing.T) {
		c := BuildSessionCookie("tok", CookieOpts{Secure: false, CrossSite: true, MaxAge: time.Hour})
		if c.SameSite != http.SameSiteNoneMode || !c.Secure {
			t.Fatalf("cross-site must be None+Secure: %+v", c)
		}
	})

	t.Run("clear cookie expires", func(t *testing.T) {
		c := ClearSessionCookie(CookieOpts{})
		if c.MaxAge >= 0 || c.Value != "" {
			t.Fatalf("expected cleared cookie: %+v", c)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/security/ -run TestSessionCookie -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package security

import (
	"net/http"
	"time"
)

const SessionCookieName = "rl_session"

type CookieOpts struct {
	Secure    bool
	CrossSite bool
	MaxAge    time.Duration
}

func sameSite(o CookieOpts) http.SameSite {
	if o.CrossSite {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func BuildSessionCookie(token string, o CookieOpts) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   o.Secure || o.CrossSite, // None requires Secure
		SameSite: sameSite(o),
		MaxAge:   int(o.MaxAge.Seconds()),
	}
}

func ClearSessionCookie(o CookieOpts) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   o.Secure || o.CrossSite,
		SameSite: sameSite(o),
		MaxAge:   -1,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/security/ -run TestSessionCookie -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/security/cookie.go internal/shared/security/cookie_test.go
git commit -m "feat: session cookie builder (httpOnly, SameSite)"
```

---

### Task 5: Validator (go-playground/validator/v10 + EN/ID translations)

**TDD:** yes — validation + localized translation have a clear contract.

**Files:**
- Create: `internal/shared/validator/validator.go`
- Test: `internal/shared/validator/validator_test.go`

**Interfaces:**
- Consumes: `shared/errors`, `shared/i18n`.
- Produces:
  ```go
  type Validator struct { /* wraps *govalidator.Validate + *ut.UniversalTranslator */ }
  func New() (*Validator, error)                          // registers EN + ID translators; constructed ONCE at wiring
  func (v *Validator) Struct(s any, lang i18n.Lang) error // nil, or *errors.AppError(KindValidation) with localized field→message details
  ```
  `details` keys use each field's `json` tag. The handler passes `response.LangOf(c)` as `lang`.
  Construct once in `cmd/server/main.go` and inject into handlers/use cases.
  **Note:** if the pinned validator version lacks `translations/id`, register custom Indonesian
  translations for the tags actually used (`required,email,max,min,gte,lte`) via `RegisterTranslation`
  — same shape, ~10 lines per tag.

- [ ] **Step 1: Write the failing test**

```go
package validator

import (
	"testing"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

type sample struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,max=5"`
}

func TestValidator(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	t.Run("valid passes", func(t *testing.T) {
		if err := v.Struct(sample{Email: "a@b.com", Name: "ok"}, i18n.EN); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("invalid produces localized validation AppError", func(t *testing.T) {
		err := v.Struct(sample{Email: "bad", Name: "toolong"}, i18n.ID)
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindValidation {
			t.Fatalf("expected validation AppError, got %v", err)
		}
		details, ok := ae.Details.(map[string]string)
		if !ok || details["email"] == "" || details["name"] == "" {
			t.Fatalf("expected localized field details, got %+v", ae.Details)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/validator/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// Package validator wraps go-playground/validator with EN + ID translators and
// returns a localized validation AppError.
package validator

import (
	"reflect"
	"strings"

	en_locale "github.com/go-playground/locales/en"
	id_locale "github.com/go-playground/locales/id"
	ut "github.com/go-playground/universal-translator"
	govalidator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	id_translations "github.com/go-playground/validator/v10/translations/id"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

const codeValidation = "validation_failed"

type Validator struct {
	validate *govalidator.Validate
	uni      *ut.UniversalTranslator
}

// New builds a Validator with EN (default) and ID translators registered.
func New() (*Validator, error) {
	enLoc := en_locale.New()
	uni := ut.New(enLoc, enLoc, id_locale.New())
	validate := govalidator.New()
	validate.RegisterTagNameFunc(jsonFieldName)

	enT, _ := uni.GetTranslator(string(i18n.EN))
	if err := en_translations.RegisterDefaultTranslations(validate, enT); err != nil {
		return nil, err
	}
	idT, _ := uni.GetTranslator(string(i18n.ID))
	if err := id_translations.RegisterDefaultTranslations(validate, idT); err != nil {
		return nil, err
	}
	return &Validator{validate: validate, uni: uni}, nil
}

// Struct validates s by its `validate` tags, returning a localized validation
// AppError (details = field → translated message) or nil.
func (v *Validator) Struct(s any, lang i18n.Lang) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}
	verrs, ok := err.(govalidator.ValidationErrors)
	if !ok {
		return err
	}
	trans, _ := v.uni.GetTranslator(string(lang))
	details := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		details[fe.Field()] = fe.Translate(trans)
	}
	return apperrors.New(apperrors.KindValidation, codeValidation, "validation failed").WithDetails(details)
}

// jsonFieldName makes validation errors report the JSON field name.
func jsonFieldName(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	if name == "-" {
		return ""
	}
	if name == "" {
		return fld.Name
	}
	return name
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/validator/ -v`
Expected: PASS. (If `translations/id` is missing in the pinned version, apply the custom-translation note above, then re-run.)

- [ ] **Step 5: Commit**

```bash
git add internal/shared/validator/
git commit -m "feat: go-playground validator with EN/ID localized validation errors"
```

---

### Task 6: Pagination (offset + keyset cursor)

**TDD:** yes — cursor encode/decode must round-trip; offset clamps.

**Files:**
- Create: `internal/shared/pagination/offset.go`
- Create: `internal/shared/pagination/keyset.go`
- Test: `internal/shared/pagination/pagination_test.go`

**Interfaces:**
- Produces:
  ```go
  // offset
  type Offset struct { Page, Limit int }
  func ParseOffset(page, limit string) Offset       // defaults page=1 limit=20, limit capped 1..100
  func (o Offset) SQLOffset() int                    // (Page-1)*Limit
  // keyset
  type Cursor struct { Time time.Time; ID string }
  func EncodeCursor(c Cursor) string                 // base64url
  func DecodeCursor(s string) (Cursor, error)        // empty string => zero Cursor, nil error
  ```

- [ ] **Step 1: Write the failing test**

```go
package pagination

import (
	"testing"
	"time"
)

func TestOffset(t *testing.T) {
	t.Run("defaults and clamps", func(t *testing.T) {
		if o := ParseOffset("", ""); o.Page != 1 || o.Limit != 20 {
			t.Fatalf("defaults wrong: %+v", o)
		}
		if o := ParseOffset("3", "500"); o.Limit != 100 || o.SQLOffset() != 200 {
			t.Fatalf("clamp/offset wrong: %+v off=%d", o, o.SQLOffset())
		}
		if o := ParseOffset("0", "-5"); o.Page != 1 || o.Limit != 20 {
			t.Fatalf("invalid should fall back: %+v", o)
		}
	})
}

func TestCursor(t *testing.T) {
	t.Run("round-trips", func(t *testing.T) {
		now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
		c := Cursor{Time: now, ID: "abc-123"}
		got, err := DecodeCursor(EncodeCursor(c))
		if err != nil || !got.Time.Equal(now) || got.ID != "abc-123" {
			t.Fatalf("round-trip failed: %+v err=%v", got, err)
		}
	})

	t.Run("empty cursor is zero value", func(t *testing.T) {
		got, err := DecodeCursor("")
		if err != nil || !got.Time.IsZero() || got.ID != "" {
			t.Fatalf("empty cursor wrong: %+v err=%v", got, err)
		}
	})

	t.Run("garbage errors", func(t *testing.T) {
		if _, err := DecodeCursor("!!!not-base64!!!"); err == nil {
			t.Fatal("expected error for garbage cursor")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/pagination/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write `offset.go`**

```go
// Package pagination provides offset pagination for CRUD lists and keyset
// (cursor) pagination for the high-volume events list.
package pagination

import "strconv"

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Offset struct {
	Page  int
	Limit int
}

func ParseOffset(page, limit string) Offset {
	p, err := strconv.Atoi(page)
	if err != nil || p < 1 {
		p = 1
	}
	l, err := strconv.Atoi(limit)
	if err != nil || l < 1 {
		l = defaultLimit
	}
	if l > maxLimit {
		l = maxLimit
	}
	return Offset{Page: p, Limit: l}
}

func (o Offset) SQLOffset() int { return (o.Page - 1) * o.Limit }
```

- [ ] **Step 4: Write `keyset.go`**

```go
package pagination

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type Cursor struct {
	Time time.Time
	ID   string
}

// EncodeCursor packs the cursor as base64url("<rfc3339nano>|<id>").
func EncodeCursor(c Cursor) string {
	raw := c.Time.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor reverses EncodeCursor. An empty string yields the zero Cursor.
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
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/shared/pagination/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/shared/pagination/
git commit -m "feat: offset and keyset cursor pagination helpers"
```

---

### Task 7: Date-range parser

**TDD:** yes — defaulting, presets, and the 90-day max window are easy to get wrong.

**Files:**
- Create: `internal/shared/datetime/range.go`
- Test: `internal/shared/datetime/range_test.go`

**Interfaces:**
- Consumes: `shared/errors`.
- Produces:
  ```go
  type Range struct { From, To time.Time }
  // ParseRange resolves a bounded range. Precedence: explicit from/to > preset > default 24h.
  // Presets: "24h","7d","30d". Enforces From<To and (To-From) <= 90d. `now` is injected for tests.
  func ParseRange(from, to, preset string, now time.Time) (Range, error)
  ```

- [ ] **Step 1: Write the failing test**

```go
package datetime

import (
	"testing"
	"time"
)

func TestParseRange(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

	t.Run("default is last 24h", func(t *testing.T) {
		r, err := ParseRange("", "", "", now)
		if err != nil || !r.To.Equal(now) || !r.From.Equal(now.Add(-24*time.Hour)) {
			t.Fatalf("default wrong: %+v err=%v", r, err)
		}
	})

	t.Run("preset 7d", func(t *testing.T) {
		r, _ := ParseRange("", "", "7d", now)
		if !r.From.Equal(now.AddDate(0, 0, -7)) {
			t.Fatalf("7d wrong: %+v", r)
		}
	})

	t.Run("explicit range", func(t *testing.T) {
		from := "2026-06-01T00:00:00Z"
		to := "2026-06-10T00:00:00Z"
		r, err := ParseRange(from, to, "", now)
		if err != nil || r.From.Day() != 1 || r.To.Day() != 10 {
			t.Fatalf("explicit wrong: %+v err=%v", r, err)
		}
	})

	t.Run("rejects inverted and over-wide ranges", func(t *testing.T) {
		if _, err := ParseRange("2026-06-10T00:00:00Z", "2026-06-01T00:00:00Z", "", now); err == nil {
			t.Fatal("expected error for from>to")
		}
		if _, err := ParseRange("2026-01-01T00:00:00Z", "2026-06-01T00:00:00Z", "", now); err == nil {
			t.Fatal("expected error for >90d window")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/datetime/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// Package datetime parses the dashboard's bounded date-range filter.
package datetime

import (
	"time"

	apperrors "router-lens/internal/shared/errors"
)

const (
	defaultWindow = 24 * time.Hour
	maxWindow     = 90 * 24 * time.Hour
	codeBadRange  = "bad_date_range"
)

type Range struct {
	From time.Time
	To   time.Time
}

var presets = map[string]time.Duration{
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

func ParseRange(from, to, preset string, now time.Time) (Range, error) {
	if from != "" || to != "" {
		return parseExplicit(from, to, now)
	}
	window := defaultWindow
	if d, ok := presets[preset]; ok {
		window = d
	}
	return Range{From: now.Add(-window), To: now}, nil
}

func parseExplicit(from, to string, now time.Time) (Range, error) {
	f, err := parseOr(from, now.Add(-defaultWindow))
	if err != nil {
		return Range{}, err
	}
	t, err := parseOr(to, now)
	if err != nil {
		return Range{}, err
	}
	if !f.Before(t) {
		return Range{}, apperrors.New(apperrors.KindValidation, codeBadRange, "from must be before to")
	}
	if t.Sub(f) > maxWindow {
		return Range{}, apperrors.New(apperrors.KindValidation, codeBadRange, "date range exceeds the 90-day maximum")
	}
	return Range{From: f, To: t}, nil
}

func parseOr(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, apperrors.New(apperrors.KindValidation, codeBadRange, "invalid timestamp; use RFC3339")
	}
	return ts.UTC(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/datetime/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/datetime/
git commit -m "feat: bounded date-range parser with presets and 90d cap"
```

---

### Task 8: Streaming CSV exporter (formula-injection safe)

**TDD:** yes — escaping logic is a security rule.

**Files:**
- Create: `internal/shared/csv/exporter.go`
- Test: `internal/shared/csv/exporter_test.go`

**Interfaces:**
- Produces:
  ```go
  type Writer struct { /* wraps encoding/csv over an io.Writer */ }
  func NewWriter(w io.Writer) *Writer
  func (w *Writer) Write(record []string) error   // escapes formula-injection-prone cells
  func (w *Writer) Flush() error
  ```

- [ ] **Step 1: Write the failing test**

```go
package csv

import (
	"bytes"
	"strings"
	"testing"
)

func TestExporter(t *testing.T) {
	t.Run("escapes formula-injection cells", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewWriter(&buf)
		if err := w.Write([]string{"=cmd()", "+1", "-2", "@x", "safe"}); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := w.Flush(); err != nil {
			t.Fatalf("flush: %v", err)
		}
		out := buf.String()
		for _, dangerous := range []string{"'=cmd()", "'+1", "'-2", "'@x"} {
			if !strings.Contains(out, dangerous) {
				t.Fatalf("expected %q escaped in output: %q", dangerous, out)
			}
		}
		if !strings.Contains(out, "safe") || strings.Contains(out, "'safe") {
			t.Fatalf("safe cell must not be escaped: %q", out)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shared/csv/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// Package csv writes streaming, spreadsheet-injection-safe CSV.
package csv

import (
	stdcsv "encoding/csv"
	"io"
	"strings"
)

const injectionPrefixes = "=+-@"

type Writer struct {
	w *stdcsv.Writer
}

func NewWriter(w io.Writer) *Writer { return &Writer{w: stdcsv.NewWriter(w)} }

func (w *Writer) Write(record []string) error {
	safe := make([]string, len(record))
	for i, cell := range record {
		safe[i] = escapeCell(cell)
	}
	return w.w.Write(safe)
}

func (w *Writer) Flush() error {
	w.w.Flush()
	return w.w.Error()
}

// escapeCell prefixes a single quote to any cell that starts with a character
// a spreadsheet would interpret as a formula.
func escapeCell(s string) string {
	if s != "" && strings.ContainsRune(injectionPrefixes, rune(s[0])) {
		return "'" + s
	}
	return s
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shared/csv/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/csv/
git commit -m "feat: streaming CSV writer with formula-injection escaping"
```

---

### Task 9: Cost calculator (domain/pricing)

**TDD:** yes — money math; the canonical "looks right is dangerous" case. (§8 complexity: O(1), trivial — no loop.)

**Files:**
- Create: `internal/domain/pricing/calculator.go`
- Test: `internal/domain/pricing/calculator_test.go`

**Interfaces:**
- Produces:
  ```go
  type TokenUsage struct { InputTokens, OutputTokens int64 }
  type Rule struct { InputPricePer1M, OutputPricePer1M decimal.Decimal }
  type Cost struct { USD, InputPrice1M, OutputPrice1M decimal.Decimal }
  // CalculateCost returns nil when rule is nil (unpriced). cost = tokens*price/1_000_000.
  func CalculateCost(usage TokenUsage, rule *Rule) *Cost
  ```

- [ ] **Step 1: Write the failing test**

```go
package pricing

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculateCost(t *testing.T) {
	t.Run("priced computes tokens*price/1e6", func(t *testing.T) {
		rule := &Rule{
			InputPricePer1M:  decimal.RequireFromString("3.00"),
			OutputPricePer1M: decimal.RequireFromString("15.00"),
		}
		c := CalculateCost(TokenUsage{InputTokens: 12000, OutputTokens: 1800}, rule)
		if c == nil {
			t.Fatal("expected a cost")
		}
		// 12000*3/1e6 = 0.036 ; 1800*15/1e6 = 0.027 ; total = 0.063
		if !c.USD.Equal(decimal.RequireFromString("0.063")) {
			t.Fatalf("cost: got %s want 0.063", c.USD.String())
		}
		if !c.InputPrice1M.Equal(rule.InputPricePer1M) {
			t.Fatalf("snapshot not captured: %s", c.InputPrice1M)
		}
	})

	t.Run("nil rule => unpriced", func(t *testing.T) {
		if c := CalculateCost(TokenUsage{InputTokens: 100, OutputTokens: 100}, nil); c != nil {
			t.Fatalf("expected nil cost for unpriced, got %+v", c)
		}
	})

	t.Run("zero tokens => zero cost", func(t *testing.T) {
		rule := &Rule{InputPricePer1M: decimal.NewFromInt(3), OutputPricePer1M: decimal.NewFromInt(15)}
		c := CalculateCost(TokenUsage{}, rule)
		if c == nil || !c.USD.Equal(decimal.Zero) {
			t.Fatalf("expected zero cost, got %+v", c)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/pricing/ -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// Package pricing holds pricing rules and the pure cost calculator. It depends
// on nothing outside stdlib + shopspring/decimal (domain purity).
package pricing

import "github.com/shopspring/decimal"

var oneMillion = decimal.NewFromInt(1_000_000)

type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
}

type Rule struct {
	InputPricePer1M  decimal.Decimal
	OutputPricePer1M decimal.Decimal
}

type Cost struct {
	USD           decimal.Decimal
	InputPrice1M  decimal.Decimal
	OutputPrice1M decimal.Decimal
}

// CalculateCost returns nil when rule is nil (the model is unpriced).
// Multiplies before dividing to avoid precision loss.
func CalculateCost(usage TokenUsage, rule *Rule) *Cost {
	if rule == nil {
		return nil
	}
	in := decimal.NewFromInt(usage.InputTokens).Mul(rule.InputPricePer1M).Div(oneMillion)
	out := decimal.NewFromInt(usage.OutputTokens).Mul(rule.OutputPricePer1M).Div(oneMillion)
	return &Cost{
		USD:           in.Add(out),
		InputPrice1M:  rule.InputPricePer1M,
		OutputPrice1M: rule.OutputPricePer1M,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/pricing/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/pricing/
git commit -m "feat: pure cost calculator with price snapshot and unpriced handling"
```

---

## Self-Review

- **Spec coverage (Plan 02 scope):** argon2id ✓, session token + hash ✓, API key gen+hash+prefix ✓, session cookie builder (Lax, Secure-prod) ✓, validator via **go-playground/validator/v10 + EN/ID translator** → localized validation AppError ✓, offset + keyset pagination ✓, bounded date-range parser ✓, injection-safe streaming CSV ✓, pure cost calculator with snapshot + unpriced ✓. (CSRF token intentionally omitted — same-origin + SameSite=Lax, per CLAUDE.md decision 13.)
- **Placeholder scan:** none.
- **Type consistency:** `security.CookieOpts` + `security.SessionCookieName` used by Plan 03's auth handlers; `pagination.Cursor` consumed by Plan 05 event list; `datetime.Range` consumed by Plans 05/06 analytics; `pricing.TokenUsage`/`Rule`/`Cost` + `CalculateCost` consumed by Plan 05 ingestion; `validator.New() (*Validator, error)` + `Struct(s, lang)` consumed by Plan 05 ingestion validation (constructed once in `main.go`, lang from `response.LangOf(c)`). All public names fixed here.

## Next

Plan 03 — Auth + first-run setup: `domain/user`, postgres user/session repositories, auth use cases (race-safe Setup, Login, Logout, Me), session + error middleware, handlers. Sets the httpOnly session cookie on login (`SameSite=Lax`, `Secure` in production); no CSRF token (decision 13).
