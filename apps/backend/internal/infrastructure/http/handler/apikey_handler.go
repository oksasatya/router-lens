package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apikeyapp "router-lens/internal/application/apikey"
	"router-lens/internal/infrastructure/http/dto"
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

func (h *APIKeyHandler) create(c echo.Context) error {
	var req dto.APIKeyRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	plaintext, k, err := h.svc.Create(c.Request().Context(), c.Param("projectId"), req.Name)
	if err != nil {
		return err
	}
	return response.Created(c, dto.APIKeyCreatedResponse{APIKeyResponse: dto.FromAPIKey(k), Key: plaintext})
}

func (h *APIKeyHandler) list(c echo.Context) error {
	keys, err := h.svc.List(c.Request().Context(), c.Param("projectId"))
	if err != nil {
		return err
	}
	dtos := make([]dto.APIKeyResponse, 0, len(keys))
	for _, k := range keys {
		dtos = append(dtos, dto.FromAPIKey(k))
	}
	return response.Data(c, http.StatusOK, dtos)
}

func (h *APIKeyHandler) revoke(c echo.Context) error {
	if err := h.svc.Revoke(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
