package shortens

import (
	"encoding/base64"
	"time"
	"url-shorten/internal/domains"
	"url-shorten/internal/infrastructure/postgres"

	"github.com/google/uuid"
)

// Shorten to Storage <-> Entity

func ToShortenEntity(s *postgres.ShortenStorage) *domains.ShortenEntity {
	if s == nil {
		return nil
	}

	shorten := &domains.ShortenEntity{
		ID:          s.ID,
		OriginalURL: s.OriginalURL,
		ShortCode:   s.ShortCode,
		Owner:       s.Owner,
		IsActive:    s.IsActive,
		ExpiresAt:   s.ExpiresAt,
	}

	return shorten
}

func ToShortenStoreage(e *domains.ShortenEntity) *postgres.ShortenStorage {
	if e == nil {
		return nil
	}

	shorten := &postgres.ShortenStorage{
		ID:          e.ID,
		OriginalURL: e.OriginalURL,
		ShortCode:   e.ShortCode,
		Owner:       e.Owner,
		IsActive:    e.IsActive,
		ExpiresAt:   e.ExpiresAt,
	}

	return shorten
}

func ToShortenListEntity(list []postgres.ShortenStorage) []domains.ShortenEntity {
	result := make([]domains.ShortenEntity, len(list))

	for i := range list {
		result[i] = *ToShortenEntity(&list[i])
	}

	return result
}

// Shorten Request to Entity

func ToCreateShortenPublicEntity(id string, req CreateShortenPublicRequest) *domains.ShortenEntity {
	now := time.Now().AddDate(0, 0, 3)
	return &domains.ShortenEntity{
		ID:          id,
		OriginalURL: req.OriginalURL,
		ShortCode:   req.ShortCode,
		ExpiresAt:   &now,
	}
}

func ToCreateShortenAuthorizedEntity(id string, req *CreateShortenAuthorizedRequest) *domains.ShortenEntity {
	if req == nil {
		return nil
	}

	return &domains.ShortenEntity{
		ID:          id,
		OriginalURL: req.OriginalURL,
		ShortCode:   req.ShortCode,
		Owner:       &req.Owner,
		ExpiresAt:   req.ExpiresAt,
	}
}

func ToUpdateShortenAuthorizedEntity(req *UpdateShortenAuthorizedRequest) *domains.UpdateShortenEntity {
	if req == nil {
		return nil
	}

	return &domains.UpdateShortenEntity{
		ID:          req.ID,
		OriginalURL: req.OriginalURL,
		ShortCode:   req.ShortCode,
		Owner:       req.Owner,
		ExpiresAt:   req.ExpiresAt,
	}
}

// Shorten Cache Entity

func ToShortenCacheEntity(e *domains.ShortenEntity) *domains.ShortenCacheEntity {
	if e == nil {
		return nil
	}

	return &domains.ShortenCacheEntity{
		ID:          e.ID,
		OriginalURL: e.OriginalURL,
		Owner:       e.Owner,
		IsActive:    e.IsActive,
		ExpiresAt:   e.ExpiresAt,
	}
}

// Shorten Response

func ToShortenResponse(e *domains.ShortenEntity) (*ShortenResponse, error) {
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
    }, nil
}

func ToShortenListResponse(list []domains.ShortenEntity) ([]ShortenResponse, error) {
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

func uuidToBase64(uuidStr string) (string, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(parsed[:]), nil
}