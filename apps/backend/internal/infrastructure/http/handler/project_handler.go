package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	projectapp "router-lens/internal/application/project"
	projectdomain "router-lens/internal/domain/project"
	mw "router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/shared/pagination"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

type ProjectHandler struct {
	svc *projectapp.Service
	v   *validator.Validator
}

func NewProjectHandler(svc *projectapp.Service, v *validator.Validator) *ProjectHandler {
	return &ProjectHandler{svc: svc, v: v}
}

func (h *ProjectHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	const idPath = "/projects/:id"
	api.POST("/projects", h.create, session)
	api.GET("/projects", h.list, session)
	api.GET(idPath, h.get, session)
	api.PUT(idPath, h.update, session)
	api.DELETE(idPath, h.delete, session)
}

type projectRequest struct {
	Name        string `json:"name" validate:"required,max=120"`
	Description string `json:"description" validate:"max=500"`
}

type projectDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toProjectDTO(p *projectdomain.Project) projectDTO {
	return projectDTO{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:   p.UpdatedAt.UTC().Format(timeLayout),
	}
}

func (h *ProjectHandler) create(c echo.Context) error {
	var req projectRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Create(c.Request().Context(), mw.CurrentUser(c).ID, req.Name, req.Description)
	if err != nil {
		return err
	}
	return response.Created(c, toProjectDTO(p))
}

func (h *ProjectHandler) list(c echo.Context) error {
	off := pagination.ParseOffset(c.QueryParam("page"), c.QueryParam("limit"))
	items, total, err := h.svc.List(c.Request().Context(), off.Limit, off.SQLOffset())
	if err != nil {
		return err
	}
	dtos := make([]projectDTO, 0, len(items))
	for _, p := range items {
		dtos = append(dtos, toProjectDTO(p))
	}
	return response.Paginated(c, http.StatusOK, dtos, off.Page, off.Limit, total)
}

func (h *ProjectHandler) get(c echo.Context) error {
	p, err := h.svc.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, toProjectDTO(p))
}

func (h *ProjectHandler) update(c echo.Context) error {
	var req projectRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Update(c.Request().Context(), c.Param("id"), req.Name, req.Description)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, toProjectDTO(p))
}

func (h *ProjectHandler) delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
