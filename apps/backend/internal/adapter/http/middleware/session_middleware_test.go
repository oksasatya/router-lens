package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"router-lens/internal/domain/user"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/security"
)

type stubSessions struct{ s *user.Session }

func (x stubSessions) Create(context.Context, *user.Session) error { return nil }
func (x stubSessions) FindByTokenHash(_ context.Context, h string) (*user.Session, error) {
	if x.s != nil && x.s.TokenHash == h {
		return x.s, nil
	}
	return nil, user.ErrNotFound
}
func (x stubSessions) DeleteByTokenHash(context.Context, string) error { return nil }
func (x stubSessions) DeleteByUserIDExceptTokenHash(context.Context, string, string) error {
	return nil
}

type stubUsers struct{ u *user.User }

func (x stubUsers) CreateInitialAdmin(context.Context, *user.User) (bool, error) { return false, nil }
func (x stubUsers) FindByEmail(context.Context, string) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (x stubUsers) FindByID(_ context.Context, id string) (*user.User, error) {
	if x.u != nil && x.u.ID == id {
		return x.u, nil
	}
	return nil, user.ErrNotFound
}
func (x stubUsers) AnyExists(context.Context) (bool, error)                  { return true, nil }
func (x stubUsers) UpdateName(context.Context, string, string) error         { return nil }
func (x stubUsers) UpdatePasswordHash(context.Context, string, string) error { return nil }

// errUsers returns the configured error from FindByID regardless of input.
type errUsers struct{ err error }

func (x errUsers) CreateInitialAdmin(context.Context, *user.User) (bool, error) { return false, nil }
func (x errUsers) FindByEmail(context.Context, string) (*user.User, error)      { return nil, nil }
func (x errUsers) FindByID(context.Context, string) (*user.User, error)         { return nil, x.err }
func (x errUsers) AnyExists(context.Context) (bool, error)                      { return true, nil }
func (x errUsers) UpdateName(context.Context, string, string) error             { return nil }
func (x errUsers) UpdatePasswordHash(context.Context, string, string) error     { return nil }

func TestSessionMiddleware(t *testing.T) {
	e := echo.New()
	ok := func(c echo.Context) error { return c.String(http.StatusOK, "ok") }
	token := "tok"
	hash := security.HashToken(token)

	run := func(cookie *http.Cookie, sessions user.SessionRepository, users user.UserRepository) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		c := e.NewContext(req, rec)
		_ = Session(sessions, users)(ok)(c)
		return rec.Code
	}

	t.Run("no cookie -> handled as error (nil body, error returned)", func(t *testing.T) {
		// Session returns an AppError; without the central handler the recorder stays 200,
		// so assert the error directly instead.
		c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
		if err := Session(stubSessions{}, stubUsers{})(ok)(c); err == nil {
			t.Fatal("missing cookie must return an error")
		}
	})

	valid := &user.Session{ID: "s1", UserID: "u1", TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}
	if code := run(&http.Cookie{Name: security.SessionCookieName, Value: token},
		stubSessions{s: valid}, stubUsers{u: &user.User{ID: "u1"}}); code != http.StatusOK {
		t.Fatalf("valid session should pass, got %d", code)
	}

	expired := &user.Session{TokenHash: hash, ExpiresAt: time.Now().Add(-time.Hour)}
	c := e.NewContext(func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: token})
		return r
	}(), httptest.NewRecorder())
	if err := Session(stubSessions{s: expired}, stubUsers{})(ok)(c); err == nil {
		t.Fatal("expired session must return an error")
	}

	// DB error from FindByID must propagate as-is, NOT be masked as a 401.
	t.Run("FindByID DB error propagates (not masked as 401)", func(t *testing.T) {
		dbErr := errors.New("connection reset by peer")
		validSess := &user.Session{ID: "s2", UserID: "u2", TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: token})
		c := e.NewContext(req, httptest.NewRecorder())
		err := Session(stubSessions{s: validSess}, errUsers{err: dbErr})(ok)(c)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if ae, ok := apperrors.As(err); ok && ae.Kind == apperrors.KindUnauthorized {
			t.Fatalf("DB error must not be masked as 401; got %v", err)
		}
		if !errors.Is(err, dbErr) {
			t.Fatalf("expected underlying DB error to be propagated, got %v", err)
		}
	})
}
