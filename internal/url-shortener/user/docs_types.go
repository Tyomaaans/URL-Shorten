package user

import (
	"url-shorten/internal/auth-session/session"
	authUser "url-shorten/internal/auth-session/user"
)

type UserProfileResponseData struct {
	User authUser.UserResponse `json:"user"`
}

type UserProfileSwaggerResponse struct {
	Success bool                    `json:"success"`
	Message string                  `json:"message,omitempty"`
	Data    UserProfileResponseData `json:"data,omitempty"`
}

type UsersListResponseData struct {
	Users authUser.PaginatedUserResponse `json:"users"`
}

type UsersListSwaggerResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message,omitempty"`
	Data    UsersListResponseData `json:"data,omitempty"`
}

type LoginResponseData struct {
	User        authUser.UserResponse `json:"user"`
	AccessToken string                `json:"access_token"`
}

type LoginSwaggerResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message,omitempty"`
	Data    LoginResponseData `json:"data,omitempty"`
}

type RefreshTokenResponseData struct {
	AccessToken string `json:"access_token"`
}

type RefreshTokenSwaggerResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message,omitempty"`
	Data    RefreshTokenResponseData `json:"data,omitempty"`
}

type SessionsListResponseData struct {
	Session []session.SessionResponse `json:"session"`
}

type SessionsListSwaggerResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message,omitempty"`
	Data    SessionsListResponseData `json:"data,omitempty"`
}
