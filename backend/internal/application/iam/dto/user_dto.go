package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest is the input for user registration.
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

// LoginRequest is the input for user login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ChangePasswordRequest is the input for POST /auth/change-password.
//
// The account comes from the token, never from the body: a user id here would
// turn a self-service endpoint into a password reset for anybody. CurrentPassword
// is required even though the caller is already authenticated — a stolen session
// should not be enough to lock the real owner out.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// LoginResponse is the output for successful login.
type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo is a public user representation.
type UserInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// AssignRoleRequest is the input for assigning a role to a user.
//
// No organization_id. The organization is the caller's, read from the token: a
// body field here let a caller with user:write in one tenant grant a role in
// another, and validating it against the token would only be a longer way of
// saying the token wins.
type AssignRoleRequest struct {
	UserID uuid.UUID  `json:"user_id" validate:"required"`
	RoleID uuid.UUID  `json:"role_id" validate:"required"`
	AppID  *uuid.UUID `json:"app_id,omitempty"`
}

// UpdateUserRequest is the input for updating a user.
type UpdateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"`
}
