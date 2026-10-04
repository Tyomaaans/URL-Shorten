package user

import "url-shorten/internal/auth-session/domain"

// User Requests

type RegisterUserRequest struct {
	Name            string      `json:"name"     validate:"required,alphaspaceunicode"`
	Email           string      `json:"email"    validate:"required,email"`
	Password        string      `json:"password"         validate:"required,password"`
	ConfirmPassword string      `json:"confirm_password" validate:"required,eqfield=Password"`
	Role            domain.Role `json:"-"`
}

type UpdateUserRequest struct {
	ID         string  `json:"-"`
	Name       *string `json:"name"  validate:"omitempty,alphaspaceunicode"`
	Email      *string `json:"email" validate:"omitempty,email"`
	RememberMe *bool   `json:"-"`
}

type LoginRequest struct {
	Email      string `json:"email"       validate:"required,email"`
	Password   string `json:"password"    validate:"required"`
	RememberMe bool   `json:"remember_me" validate:"omitempty"`
}

type HeartbeatRequest struct {
	State string `json:"state" validate:"required,oneof=active inactive"`
}

// User Response

type UserResponse struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Email      string      `json:"email"`
	Role       domain.Role `json:"role"`
	RememberMe bool        `json:"remember_me"`
}

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	PageSize    int   `json:"page_size"`
	TotalData   int64 `json:"total_data"`
}

type PaginatedUserResponse struct {
	Data []UserResponse `json:"users"`
	Meta PaginationMeta `json:"meta"`
}
