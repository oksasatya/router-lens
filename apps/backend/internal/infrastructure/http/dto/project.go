package dto

import projectdomain "router-lens/internal/domain/project"

// ProjectRequest is the create/update payload for a project.
type ProjectRequest struct {
	Name        string `json:"name" validate:"required,max=120"`
	Description string `json:"description" validate:"max=500"`
}

// ProjectResponse is the wire shape of a project.
type ProjectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// FromProject maps a domain project to its response shape.
func FromProject(p *projectdomain.Project) ProjectResponse {
	return ProjectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:   p.UpdatedAt.UTC().Format(timeLayout),
	}
}
