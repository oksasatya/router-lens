package security

import (
	"net/http"
	"time"
)

const SessionCookieName = "rl_session"

// CookieOpts controls session cookie behaviour.
// CrossSite forces SameSite=None; Secure (required for split-origin proxies).
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

// BuildSessionCookie returns a cookie carrying the given token.
// CrossSite=true forces SameSite=None + Secure; otherwise Lax.
func BuildSessionCookie(token string, o CookieOpts) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   o.Secure || o.CrossSite, // ponytail: None requires Secure; keep the rule co-located
		SameSite: sameSite(o),
		MaxAge:   int(o.MaxAge.Seconds()),
	}
}

// ClearSessionCookie returns an expired cookie that instructs the browser to delete the session.
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
