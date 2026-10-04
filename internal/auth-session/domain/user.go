package domain

import (
	"context"
)

type Role string

const (
	User  Role = "user"
	Admin Role = "admin"
)

// User Entities

type UserEntity struct {
	ID         string
	Name       string
	Email      string
	Role       Role
	RememberMe bool
}

type CreateUserEntity struct {
	ID       string
	Name     string
	Email    string
	Role     Role
	Password string
}

type UpdateUserEntity struct {
	ID         string
	Name       *string
	Email      *string
	RememberMe *bool
}

// User Repository

type UserRepository interface {
	CreateUser(ctx context.Context, user *CreateUserEntity) error
	UpdateUser(ctx context.Context, update *UpdateUserEntity) error
	GetUsers(ctx context.Context, page, limit int) ([]UserEntity, int64, error)
	GetUserByID(ctx context.Context, id string) (*UserEntity, error)
	GetPasswordByEmail(ctx context.Context, email string) (string, string, error)
	DeleteUser(ctx context.Context, id string) error
}
