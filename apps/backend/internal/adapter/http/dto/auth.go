package dto

import "router-lens/internal/domain/user"

// SetupRequest is the first-run admin creation payload.
type SetupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	Name     string `json:"name" validate:"max=100"`
}

// LoginRequest is the credentials payload.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UpdateProfileRequest is the PUT /auth/me payload.
type UpdateProfileRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

// ChangePasswordRequest is the POST /auth/change-password payload.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// UserResponse is the public shape of an authenticated user.
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// FromUser maps a domain user to its response shape.
func FromUser(u *user.User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, Name: u.Name}
}
