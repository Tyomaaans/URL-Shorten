package user

import (
	"encoding/base64"
	"url-shorten/internal/auth-session/domain"

	"github.com/google/uuid"
)

// User to Storage <-> Entity

func ToUserEntity(s *UserStorage) *domain.UserEntity {
	return &domain.UserEntity{
		ID:         s.ID,
		Name:       s.Name,
		Email:      s.Email,
		Role:       s.Role,
		RememberMe: s.RememberMe,
	}
}

func ToUserStorage(e *domain.UserEntity) *UserStorage {
	return &UserStorage{
		ID:         e.ID,
		Name:       e.Name,
		Email:      e.Email,
		Role:       domain.Role(e.Role),
		RememberMe: e.RememberMe,
	}
}

func ToUserListEntity(list []UserStorage) []domain.UserEntity {
	result := make([]domain.UserEntity, len(list))

	for i := range list {
		result[i] = *ToUserEntity(&list[i])
	}

	return result
}

// Request to Entity

func ToRegisterUserEntity(id string, req RegisterUserRequest) *domain.CreateUserEntity {
	return &domain.CreateUserEntity{
		ID:       id,
		Name:     req.Name,
		Email:    req.Email,
		Role:     req.Role,
		Password: req.Password,
	}
}

func ToRegisterUserStorage(e *domain.CreateUserEntity) *UserStorage {
	return &UserStorage{
		ID:       e.ID,
		Name:     e.Name,
		Email:    e.Email,
		Role:     e.Role,
		Password: e.Password,
	}
}

func ToUpdateUserEntity(req *UpdateUserRequest) *domain.UpdateUserEntity {
	if req == nil {
		return nil
	}

	return &domain.UpdateUserEntity{
		ID:         req.ID,
		Name:       req.Name,
		Email:      req.Email,
		RememberMe: req.RememberMe,
	}
}

// Entity to Response

func ToUserResponse(e domain.UserEntity) (*UserResponse, error) {
	id, err := uuidToBase64(e.ID)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:         id,
		Name:       e.Name,
		Email:      e.Email,
		Role:       e.Role,
		RememberMe: e.RememberMe,
	}, nil
}

func ToUserListResponse(list []domain.UserEntity) ([]UserResponse, error) {
	result := make([]UserResponse, len(list))

	for i := range list {
		res, err := ToUserResponse(list[i])
		if err != nil {
			return nil, err
		}
		result[i] = *res
	}

	return result, nil
}

func ToPaginatedUserResponse(list []domain.UserEntity, page, limit int, totalData int64) (*PaginatedUserResponse, error) {
	listResponse, err := ToUserListResponse(list)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	totalPages := int((totalData + int64(limit) - 1) / int64(limit))

	return &PaginatedUserResponse{
		Data: listResponse,
		Meta: PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			PageSize:    limit,
			TotalData:   totalData,
		},
	}, nil
}

// Internal Helper

func uuidToBase64(uuidStr string) (string, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(parsed[:]), nil
}
