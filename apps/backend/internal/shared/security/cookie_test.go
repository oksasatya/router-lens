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
