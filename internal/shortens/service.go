package shortens

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"url-shorten/internal/domains"
	"url-shorten/internal/middleware"
	"url-shorten/pkg"

	jsonValidator "url-shorten/pkg"
)

// Interface

type ShortenService interface {
	// Shorten CRUD
	CreateShortenPublic(ctx context.Context, req CreateShortenPublicRequest) (*ShortenResponse, error)
	CreateShortenAuthorized(ctx context.Context, req *CreateShortenAuthorizedRequest) (*ShortenResponse, error)
	UpdateShorten(ctx context.Context, update *UpdateShortenAuthorizedRequest) (*ShortenResponse, error)
	GetShortens(ctx context.Context) ([]ShortenResponse, error)
	GetShortenByID(ctx context.Context, rawShortenID, userID string) (*ShortenResponse, error)
	GetShortenByOwner(ctx context.Context, rawOwnerID string) ([]ShortenResponse, error)
	SetURLStatus(ctx context.Context, userID, rawShortenID string, active bool) (*ShortenResponse, error)
	DeleteShorten(ctx context.Context, rawShortenID, rawUserID string) error

	// Shorten Redirect
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)

	// Shorten Worker
	StartExpiryWorker(ctx context.Context)
}

// Implementation

const (
	shortenCachePrefix = "shorten:"
	shortCodeLength    = 7
	shortCodeChars     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type shortenService struct {
	shortenRepo domains.ShortenRepository
	redisClient *redis.Client
	validate    *validator.Validate
}

func NewShortenService(
	shortenRepo domains.ShortenRepository,
	redisClient *redis.Client,
	validate    *validator.Validate,
) ShortenService {
	return &shortenService{
		shortenRepo: shortenRepo,
		redisClient: redisClient,
		validate:    validate,
	}
}

// Shorten CRUD

func (s *shortenService) CreateShortenPublic(ctx context.Context, req CreateShortenPublicRequest) (*ShortenResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("%w%v", pkg.ErrInvalidInput, err)
	}

	req.ShortCode = generateShortCode()

	payload := ToCreateShortenPublicEntity(uuid.NewString(), req)

	result, err := s.shortenRepo.CreateShorten(ctx, payload)
	if err != nil {
		return nil, err
	}

	if err := s.cacheShorten(ctx, result); err != nil {
		return nil, err
	}

	res, err := ToShortenResponse(result)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) CreateShortenAuthorized(ctx context.Context, req *CreateShortenAuthorizedRequest) (*ShortenResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return nil, fmt.Errorf("%w %v", pkg.ErrInvalidInput, err)
	}

	req.ShortCode = generateShortCode()

	payload := ToCreateShortenAuthorizedEntity(uuid.NewString(), req)

	result, err := s.shortenRepo.CreateShorten(ctx, payload)
	if err != nil {
		return nil, err
	}

	if err := s.cacheShorten(ctx, result); err != nil {
		return nil, err
	}

	res, err := ToShortenResponse(result)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) UpdateShorten(ctx context.Context, update *UpdateShortenAuthorizedRequest) (*ShortenResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, update); err != nil {
		return nil, fmt.Errorf("%w%v", pkg.ErrInvalidInput, err)
	}

	shortenID, err := parseOrDecodeUUID(update.ID)
	if err != nil {
		return nil, err
	}

	userID, err := parseOrDecodeUUID(update.Owner)
    if err != nil {
        return nil, err
    }

	update.ID    = shortenID
	update.Owner = userID

	existing, err := s.shortenRepo.GetShortenByID(ctx, update.ID)
	if err != nil {
		return nil, err
	}

	isAdmin, _ := ctx.Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
        if existing.Owner == nil || *existing.Owner != update.Owner{
            return nil, pkg.ErrForbidden
        }
    }

	if update.ShortCode != nil {
		s.redisClient.Del(ctx, shortenCachePrefix+existing.ShortCode)
	}

	_, err = s.shortenRepo.UpdateShorten(ctx, ToUpdateShortenAuthorizedEntity(update))
	if err != nil {
		return nil, err
	}

	result, err := s.shortenRepo.GetShortenByID(ctx, update.ID)
	if err != nil {
		return nil, err
	}

	if err := s.cacheShorten(ctx, result); err != nil {
		return nil, err
	}

	res, err := ToShortenResponse(result)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) GetShortens(ctx context.Context) ([]ShortenResponse, error) {
	shortens, err := s.shortenRepo.GetShortens(ctx)
	if err != nil {
		return nil, err
	}

	res, err := ToShortenListResponse(shortens)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) GetShortenByID(ctx context.Context, rawShortenID, userID string) (*ShortenResponse, error) {
	shortenID, err := parseOrDecodeUUID(rawShortenID)
	if err != nil {
		return nil, err
	}

	shorten, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
	if err != nil {
		return nil, err
	}

	isAdmin, _ := ctx.Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
        if shorten.Owner == nil || *shorten.Owner != userID {
            return nil, pkg.ErrForbidden
        }
    }

	res, err := ToShortenResponse(shorten)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) GetShortenByOwner(ctx context.Context, rawOwnerID string) ([]ShortenResponse, error) {
	ownerID, err := parseOrDecodeUUID(rawOwnerID)
	if err != nil {
		return nil, err
	}

	shortens, err := s.shortenRepo.GetShortenByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	res, err := ToShortenListResponse(shortens)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) SetURLStatus(ctx context.Context, userID, rawShortenID string, active bool) (*ShortenResponse, error) {
    shortenID, err := parseOrDecodeUUID(rawShortenID)
    if err != nil {
        return nil, err
    }

    existing, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
    if err != nil {
        return nil, err
    }

    isAdmin, _ := ctx.Value(middleware.IsAdminKey).(bool)
    if !isAdmin {
        if existing.Owner == nil || *existing.Owner != userID {
            return nil, pkg.ErrForbidden
        }
    }

	_, err = s.shortenRepo.UpdateShorten(ctx, &domains.UpdateShortenEntity{
		ID:        shortenID,
		IsActive:  &active,
	})
	if err != nil {
		return nil, err
	}

    updatedShorten, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
    if err != nil {
        return nil, err
    }

    if updatedShorten.IsActive != nil && *updatedShorten.IsActive {
        s.redisClient.Del(ctx, shortenCachePrefix+updatedShorten.ShortCode)

        if err := s.cacheShorten(ctx, updatedShorten); err != nil {
            return nil, err
        }
    } else {
        s.redisClient.Del(ctx, shortenCachePrefix+updatedShorten.ShortCode)
    }

    res, err := ToShortenResponse(updatedShorten)
    if err != nil {
        return nil, err
    }

    return res, nil
}

func (s *shortenService) DeleteShorten(ctx context.Context, rawShortenID, rawUserID string) error {
	shortenID, err := parseOrDecodeUUID(rawShortenID)
	if err != nil {
		return err
	}

	userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return err
    }

	existing, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
	if err != nil {
		return err
	}

	isAdmin, _ := ctx.Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
        if existing.Owner == nil || *existing.Owner != userID {
            return pkg.ErrForbidden
        }
    }

	s.redisClient.Del(ctx, shortenCachePrefix+existing.ShortCode)

	if err := s.shortenRepo.DeleteShorten(ctx, shortenID); err != nil {
		return err
	}

	return nil
}

// Shorten Redirect

func (s *shortenService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	cacheKey := shortenCachePrefix + shortCode

	data, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var cached domains.ShortenCacheEntity
		if err := json.Unmarshal(data, &cached); err == nil {
			if !shouldServe(*cached.IsActive, cached.ExpiresAt) {
				s.redisClient.Del(ctx, cacheKey)
				return "", pkg.ErrNotFound
			}
			return cached.OriginalURL, nil
		}
	}

	log.Printf("waktu: %s", time.Now())

	if !errors.Is(err, redis.Nil) {
		return "", pkg.HandleRedisError(err)
	}

	shorten, err := s.shortenRepo.GetShortenByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}

	if !shouldServe(*shorten.IsActive, shorten.ExpiresAt) {
		return "", pkg.ErrNotFound
	}

	if err := s.cacheShorten(ctx, shorten); err != nil {
		return "", err
	}

	return shorten.OriginalURL, nil
}

func (s *shortenService) StartExpiryWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.deactivateExpired(ctx); err != nil {
				// log error
			}
		case <-ctx.Done():
			return
		}
	}
}

// Internal Helper

func generateShortCode() string {
	b := make([]byte, shortCodeLength)
	for i := range b {
		b[i] = shortCodeChars[rand.Intn(len(shortCodeChars))]
	}
	return string(b)
}

func parseOrDecodeUUID(input string) (string, error) {
	if input == "" {
		return "", pkg.ErrInvalidInput
	}

	if _, err := uuid.Parse(input); err == nil {
		return input, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return "", pkg.ErrInvalidInput
	}

	parsed, err := uuid.FromBytes(decoded)
	if err != nil {
		return "", pkg.ErrInvalidInput
	}

	return parsed.String(), nil
}

func (s *shortenService) cacheShorten(ctx context.Context, shorten *domains.ShortenEntity) error {
    cacheKey := shortenCachePrefix + shorten.ShortCode
    ttl      := resolveTTL(shorten.ExpiresAt)

    cache := ToShortenCacheEntity(shorten)

    data, err := json.Marshal(cache)
    if err != nil {
        return pkg.ErrRedisMarshal
    }

    created, err := s.redisClient.SetNX(ctx, cacheKey, data, ttl).Result()
    if err := pkg.HandleRedisSetNX(created, err, true); err != nil {
        return err
    }

    return nil
}

func (s *shortenService) deactivateExpired(ctx context.Context) error {
    shortens, err := s.shortenRepo.GetActiveWithExpiry(ctx)
    if err != nil {
        return err
    }

    isActiveFalse := false

    for _, sh := range shortens {
        if isExpired(sh.ExpiresAt) {
            _, err := s.shortenRepo.UpdateShorten(ctx, &domains.UpdateShortenEntity{
                ID:       sh.ID,
                IsActive: &isActiveFalse,
            })
            if err != nil {
                log.Printf("failed to update is_active: %v", err)
            }

            s.redisClient.Del(ctx, shortenCachePrefix+sh.ShortCode)
        }
    }

    return nil
}

func resolveTTL(expiresAt *time.Time) time.Duration {
    if expiresAt == nil {
        return 0
    }

    ttl := time.Until(*expiresAt)

    if ttl <= 0 {
        return 1 * time.Second
    }

    return ttl
}

func isExpired(expiresAt *time.Time) bool {
    if expiresAt == nil {
        return false
    }
    return time.Now().After(*expiresAt)
}

func shouldServe(isActive bool, expiresAt *time.Time) bool {
    if !isActive {
        return false
    }
    return !isExpired(expiresAt)
}