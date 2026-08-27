package users

import (
	"encoding/base64"
	"url-shorten/internal/domains"
	"url-shorten/internal/infrastructure/postgres"
	"url-shorten/internal/shortens"

	"github.com/google/uuid"
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

func ToUserResponse(e *domains.UserEntity) (*UserResponse, error) {
	if e == nil {
		return nil, nil
	}

	sub, err := uuidToBase64(e.ID)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:         sub,
		Name:       e.Name,
		Email:      e.Email,
		RememberMe: e.RememberMe,
	}, nil
}

func ToUserListResponse(list []domains.UserEntity) ([]UserResponse, error) {
	result := make([]UserResponse, len(list))

	for i := range list {
		res, err := ToUserResponse(&list[i])
		if err != nil {
			return nil, err
		}
		result[i] = *res
	}

	return result, nil
}

func uuidToBase64(uuidStr string) (string, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(parsed[:]), nil
}