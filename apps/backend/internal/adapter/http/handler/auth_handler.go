package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"router-lens/internal/adapter/http/dto"
	mw "router-lens/internal/adapter/http/middleware"
	"router-lens/internal/platform/config"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/security"
	"router-lens/internal/shared/validator"
	"router-lens/internal/usecase/auth"
)

type AuthHandler struct {
	svc *auth.Service
	v   *validator.Validator
	cfg config.Config
}

func NewAuthHandler(svc *auth.Service, v *validator.Validator, cfg config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, v: v, cfg: cfg}
}

func (h *AuthHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/setup/status", h.setupStatus)
	api.POST("/setup", h.setup)
	api.POST("/auth/login", h.login)
	api.POST("/auth/logout", h.logout, session)
	api.GET("/auth/me", h.me, session)
	api.PUT("/auth/me", h.updateProfile, session)
	api.POST("/auth/change-password", h.changePassword, session)
}

func (h *AuthHandler) setupStatus(c echo.Context) error {
	needs, err := h.svc.NeedsSetup(c.Request().Context())
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, map[string]bool{"needs_setup": needs})
}

func (h *AuthHandler) setup(c echo.Context) error {
	var req dto.SetupRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	if err := h.svc.Setup(c.Request().Context(), req.Email, req.Password, req.Name); err != nil {
		return err
	}
	return response.Data(c, http.StatusCreated, map[string]bool{"created": true})
}

func (h *AuthHandler) login(c echo.Context) error {
	var req dto.LoginRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	token, err := h.svc.Login(c.Request().Context(), req.Email, req.Password,
		c.Request().UserAgent(), c.RealIP())
	if err != nil {
		return err
	}
	c.SetCookie(security.BuildSessionCookie(token, h.cookieOpts()))
	return response.Data(c, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) logout(c echo.Context) error {
	if s := mw.CurrentSession(c); s != nil {
		if err := h.svc.Logout(c.Request().Context(), s.TokenHash); err != nil {
			return err
		}
	}
	c.SetCookie(security.ClearSessionCookie(h.cookieOpts()))
	return response.NoContent(c)
}

func (h *AuthHandler) me(c echo.Context) error {
	u := mw.CurrentUser(c)
	return response.Data(c, http.StatusOK, dto.FromUser(u))
}

func (h *AuthHandler) updateProfile(c echo.Context) error {
	var req dto.UpdateProfileRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	u := mw.CurrentUser(c)
	updated, err := h.svc.UpdateProfile(c.Request().Context(), u.ID, req.Name)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromUser(updated))
}

// ponytail: no rate-limiting on this route — full per-endpoint rate-limiting is deferred
// to v0.2 (CLAUDE.md decision 14); the surviving "leaked session guesses the password"
// risk is closed by ChangePassword revoking every other session on success.
func (h *AuthHandler) changePassword(c echo.Context) error {
	var req dto.ChangePasswordRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	u := mw.CurrentUser(c)
	sess := mw.CurrentSession(c)
	err := h.svc.ChangePassword(c.Request().Context(), u.ID, sess.TokenHash, req.CurrentPassword, req.NewPassword)
	if err != nil {
		return err
	}
	return response.NoContent(c)
}

func (h *AuthHandler) cookieOpts() security.CookieOpts {
	return security.CookieOpts{
		Secure:    h.cfg.IsProduction(),
		CrossSite: h.cfg.CookieCrossSite,
		MaxAge:    auth.SessionTTL,
	}
}
