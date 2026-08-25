package domains

import (
	"context"
)

// User Entity

type UserEntity struct {
	ID         string
	Name       string
	Email      string
	RememberMe bool

	Shortens []ShortenEntity
}

type CreateUserEntity struct {
	ID       string
	Name     string
	Email    string
	Password string
}

type UpdateUserEntity struct {
	ID         string
	Name       *string
	Email      *string
	RememberMe *bool
}

type UpdatePasswordEntity struct {
	ID       string
	Password string
}

// User Repository Interface

type UserRepository interface {
	CreateUser(ctx context.Context, user *CreateUserEntity) error
	UpdateUser(ctx context.Context, update *UpdateUserEntity) (*UserEntity, error)
	UpdatePassword(ctx context.Context, update *UpdatePasswordEntity) error
	GetUsers(ctx context.Context) ([]UserEntity, error)
	GetUserByID(ctx context.Context, id string) (*UserEntity, error)
	GetPasswordByEmail(ctx context.Context, email string) (string, string, error)
	DeleteUser(ctx context.Context, id string) error
}