package users

import (
	"url-shorten/internal/domains"
	"url-shorten/internal/infrastructure/postgres"
	"url-shorten/internal/shortens"
)

// User to Storage <-> Entitu

func ToUserEntity(s *postgres.UserStorage) *domains.UserEntity {
	user := &domains.UserEntity{
		ID:         s.ID,
		Name:       s.Name,
		Email:      s.Email,
		RememberMe: s.RememberMe,
	}

	if len(s.Shortens) > 0 {
		user.Shortens = shortens.ToShortenListEntity(s.Shortens)
	}

	return user
}

func ToUserStorage(e *domains.UserEntity) *postgres.UserStorage {
	return &postgres.UserStorage{
		ID:         e.ID,
		Name:       e.Name,
		Email:      e.Email,
		RememberMe: e.RememberMe,
	}
}

func ToUserListEntity(list []postgres.UserStorage) []domains.UserEntity {
	result := make([]domains.UserEntity, len(list))

	for i := range list {
		result[i] = *ToUserEntity(&list[i])
	}

	return result
}

// User Request to Entity

func ToRegisterUserEntity(id string, req RegisterUserRequest) *domains.CreateUserEntity {
	return &domains.CreateUserEntity{
		ID:       id,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}
}

func ToRegisterUserStorage(e *domains.CreateUserEntity) *postgres.UserStorage {
	return &postgres.UserStorage{
		ID:       e.ID,
		Name:     e.Name,
		Email:    e.Email,
		Password: e.Password,
	}
}

func ToUpdateUserEntity(req *UpdateUserRequest) *domains.UpdateUserEntity {
	if req == nil {
		return nil
	}

	return &domains.UpdateUserEntity{
		ID:         req.ID,
		Name:       req.Name,
		Email:      req.Email,
		RememberMe: req.RememberMe,
	}
}

func ToUpdatePasswordEntity(req UpdatePasswordRequest) *domains.UpdatePasswordEntity {
	return &domains.UpdatePasswordEntity{
		ID:       req.ID,
		Password: req.NewPassword,
	}
}

// User Response

func ToUserResponse(e *domains.UserEntity) *UserResponse {
	return &UserResponse{
		ID:         e.ID,
		Name:       e.Name,
		Email:      e.Email,
		RememberMe: e.RememberMe,
	}
}

func ToUserListResponse(list []domains.UserEntity) []UserResponse {
	result := make([]UserResponse, len(list))

	for i := range list {
		result[i] = *ToUserResponse(&list[i])
	}

	return result
}