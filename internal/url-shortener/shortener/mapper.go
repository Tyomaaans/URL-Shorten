package shortener

import (
	"encoding/base64"
	"time"

	"url-shorten/internal/url-shortener/domain"

	"github.com/google/uuid"
)

// Shorten to Storage <-> Entity

func ToShortenEntity(s *ShortenStorage) *domain.ShortenEntity {
	if s == nil {
		return nil
	}

	shorten := &domain.ShortenEntity{
		ID:          s.ID,
		OriginalURL: s.OriginalURL,
		ShortCode:   s.ShortCode,
		Owner:       s.Owner,
		IsActive:    s.IsActive,
		ExpiresAt:   s.ExpiresAt,
		CreatedAt:   s.CreatedAt,
	}

	return shorten
}

func ToShortenStoreage(e *domain.ShortenEntity) *ShortenStorage {
	if e == nil {
		return nil
	}

	shorten := &ShortenStorage{
		ID:          e.ID,
		OriginalURL: e.OriginalURL,
		ShortCode:   e.ShortCode,
		Owner:       e.Owner,
		IsActive:    e.IsActive,
		ExpiresAt:   e.ExpiresAt,
		CreatedAt:   e.CreatedAt,
	}

	return shorten
}

func ToShortenListEntity(list []ShortenStorage) []domain.ShortenEntity {
	result := make([]domain.ShortenEntity, len(list))

	for i := range list {
		result[i] = *ToShortenEntity(&list[i])
	}

	return result
}

// Shorten Request to Entity

func ToCreateShortenPublicEntity(id string, req CreateShortenPublicRequest) *domain.ShortenEntity {
	now := time.Now().AddDate(0, 0, 3)
	return &domain.ShortenEntity{
		ID:          id,
		OriginalURL: req.OriginalURL,
		ShortCode:   req.ShortCode,
		ExpiresAt:   &now,
	}
}

func ToCreateShortenAuthorizedEntity(id string, req *CreateShortenAuthorizedRequest) *domain.ShortenEntity {
	if req == nil {
		return nil
	}

	return &domain.ShortenEntity{
		ID:          id,
		OriginalURL: req.OriginalURL,
		ShortCode:   req.ShortCode,
		Owner:       &req.Owner,
		ExpiresAt:   req.ExpiresAt,
	}
}

func ToUpdateShortenAuthorizedEntity(req *UpdateShortenAuthorizedRequest) *domain.UpdateShortenEntity {
	if req == nil {
		return nil
	}

	return &domain.UpdateShortenEntity{
		ID:          req.ID,
		OriginalURL: req.OriginalURL,
		Owner:       req.Owner,
		ExpiresAt:   req.ExpiresAt,
	}
}

// Shorten Cache Entity

func ToShortenCacheEntity(e *domain.ShortenEntity) *domain.ShortenCacheEntity {
	if e == nil {
		return nil
	}

	return &domain.ShortenCacheEntity{
		ID:          e.ID,
		OriginalURL: e.OriginalURL,
		Owner:       e.Owner,
		IsActive:    e.IsActive,
		ExpiresAt:   e.ExpiresAt,
	}
}

// Shorten Response

func ToShortenResponse(e *domain.ShortenEntity) (*ShortenResponse, error) {
	if e == nil {
		return nil, nil
	}

	sid, err := uuidToBase64(e.ID)
	if err != nil {
		return nil, err
	}

	var owner *string
	if e.Owner != nil {
		sub, err := uuidToBase64(*e.Owner)
		if err != nil {
			return nil, err
		}
		owner = &sub
	}

	return &ShortenResponse{
		ID:          sid,
		OriginalURL: e.OriginalURL,
		ShortCode:   e.ShortCode,
		Owner:       owner,
		IsActive:    *e.IsActive,
		ExpiresAt:   e.ExpiresAt,
		CreatedAt:   e.CreatedAt,
	}, nil
}

func ToShortenListResponse(list []domain.ShortenEntity) ([]ShortenResponse, error) {
	result := make([]ShortenResponse, len(list))

	for i := range list {
		res, err := ToShortenResponse(&list[i])
		if err != nil {
			return nil, err
		}
		result[i] = *res
	}

	return result, nil
}

func ToPaginatedShortenResponse(list []domain.ShortenEntity, page, limit int, totalData int64) (*PaginatedShortenResponse, error) {
	listResponse, err := ToShortenListResponse(list)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	totalPages := int((totalData + int64(limit) - 1) / int64(limit))

	return &PaginatedShortenResponse{
		Data: listResponse,
		Meta: PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			PageSize:    limit,
			TotalData:   totalData,
		},
	}, nil
}

func uuidToBase64(uuidStr string) (string, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(parsed[:]), nil
}
