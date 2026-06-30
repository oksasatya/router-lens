package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apikeyapp "router-lens/internal/application/apikey"
	apikeydomain "router-lens/internal/domain/apikey"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type APIKeyHandler struct {
	svc *apikeyapp.Service
	v   *validator.Validator
}

func NewAPIKeyHandler(svc *apikeyapp.Service, v *validator.Validator) *APIKeyHandler {
	return &APIKeyHandler{svc: svc, v: v}
}

func (h *APIKeyHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.POST("/projects/:projectId/api-keys", h.create, session)
	api.GET("/projects/:projectId/api-keys", h.list, session)
	api.DELETE("/api-keys/:id", h.revoke, session)
}

type apiKeyRequest struct {
	Name string `json:"name" validate:"required,max=120"`
}

type apiKeyDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	LastUsedAt *string `json:"last_used_at"`
	RevokedAt  *string `json:"revoked_at"`
	CreatedAt  string  `json:"created_at"`
}

// apiKeyCreatedDTO is returned ONCE on creation — it carries the plaintext key,
// which is never persisted and never returned again.
type apiKeyCreatedDTO struct {
	apiKeyDTO
	Key string `json:"key"`
}

func toAPIKeyDTO(k *apikeydomain.APIKey) apiKeyDTO {
	return apiKeyDTO{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		LastUsedAt: formatNullableTime(k.LastUsedAt),
		RevokedAt:  formatNullableTime(k.RevokedAt),
		CreatedAt:  k.CreatedAt.UTC().Format(timeLayout),
	}
}

func (h *APIKeyHandler) create(c echo.Context) error {
	var req apiKeyRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	plaintext, k, err := h.svc.Create(c.Request().Context(), c.Param("projectId"), req.Name)
	if err != nil {
		return err
	}
	return response.Created(c, apiKeyCreatedDTO{apiKeyDTO: toAPIKeyDTO(k), Key: plaintext})
}

func (h *APIKeyHandler) list(c echo.Context) error {
	keys, err := h.svc.List(c.Request().Context(), c.Param("projectId"))
	if err != nil {
		return err
	}
	dtos := make([]apiKeyDTO, 0, len(keys))
	for _, k := range keys {
		dtos = append(dtos, toAPIKeyDTO(k))
	}
	return response.Data(c, http.StatusOK, dtos)
}

func (h *APIKeyHandler) revoke(c echo.Context) error {
	if err := h.svc.Revoke(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
