// Package project is the Project bounded context: the aggregate, its repository
// port, and the slug rule. Imports only stdlib (domain purity).
package project

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a project row is absent.
var ErrNotFound = errors.New("project: not found")

// ErrSlugTaken is returned when (owner_user_id, slug) already exists.
var ErrSlugTaken = errors.New("project: slug already taken for owner")

type Project struct {
	ID          string
	OwnerUserID string
	Name        string
	Slug        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
