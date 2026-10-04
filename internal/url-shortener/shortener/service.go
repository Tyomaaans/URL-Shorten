package shortener

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

	"url-shorten/internal/url-shortener/domain"
	"url-shorten/pkg"

	user "url-shorten/internal/auth-session/domain"
	jsonValidator "url-shorten/pkg"
)

// Interface

type ShortenService interface {
	// Shorten CRUD
	CreateShortenPublic(ctx context.Context, req CreateShortenPublicRequest) (*ShortenResponse, error)
	CreateShortenAuthorized(ctx context.Context, req *CreateShortenAuthorizedRequest) (*ShortenResponse, error)
	UpdateShorten(ctx context.Context, update *UpdateShortenAuthorizedRequest, role string) (*ShortenResponse, error)
	GetShortens(ctx context.Context, page, limit int) (*PaginatedShortenResponse, error)
	GetShortenByID(ctx context.Context, rawShortenID, userID, role string) (*ShortenResponse, error)
	GetShortenByOwner(ctx context.Context, rawOwnerID string, page, limit int) (*PaginatedShortenResponse, error)
	SetURLStatus(ctx context.Context, userID, rawShortenID, role, status string) (*ShortenResponse, error)
	DeleteShorten(ctx context.Context, rawShortenID, rawUserID, role string) error

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
	expiryQueueKey     = "shorten:expiry_queue"
)

type shortenService struct {
	shortenRepo domain.ShortenRepository
	redisClient *redis.Client
	validate    *validator.Validate
}

func NewShortenService(
	shortenRepo domain.ShortenRepository,
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

	if err := s.scheduleExpiry(ctx, result.ShortCode, result.ExpiresAt); err != nil {
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

	if err := s.scheduleExpiry(ctx, result.ShortCode, result.ExpiresAt); err != nil {
		return nil, err
	}

	res, err := ToShortenResponse(result)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) UpdateShorten(ctx context.Context, update *UpdateShortenAuthorizedRequest, role string) (*ShortenResponse, error) {
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

	update.ID = shortenID
	update.Owner = userID

	existing, err := s.shortenRepo.GetShortenByID(ctx, update.ID)
	if err != nil {
		return nil, err
	}

	if role == string(user.User) {
		if existing.Owner == nil || *existing.Owner != update.Owner {
			return nil, pkg.ErrForbidden
		}
	}

	_, err = s.shortenRepo.UpdateShorten(ctx, ToUpdateShortenAuthorizedEntity(update))
	if err != nil {
		return nil, err
	}

	result, err := s.shortenRepo.GetShortenByID(ctx, update.ID)
	if err != nil {
		return nil, err
	}

	if existing.ShortCode != result.ShortCode {
		s.redisClient.Del(ctx, shortenCachePrefix+existing.ShortCode)
		s.redisClient.ZRem(ctx, expiryQueueKey, existing.ShortCode)
	}

	if err := s.cacheShorten(ctx, result); err != nil {
		return nil, err
	}

	if err := s.scheduleExpiry(ctx, result.ShortCode, result.ExpiresAt); err != nil {
		return nil, err
	}

	res, err := ToShortenResponse(result)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) GetShortens(ctx context.Context, page, limit int) (*PaginatedShortenResponse, error) {
	shortens, pages, err := s.shortenRepo.GetShortens(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	res, err := ToPaginatedShortenResponse(shortens, page, limit, pages)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) GetShortenByID(ctx context.Context, rawShortenID, userID, role string) (*ShortenResponse, error) {
	shortenID, err := parseOrDecodeUUID(rawShortenID)
	if err != nil {
		return nil, err
	}

	shorten, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
	if err != nil {
		return nil, err
	}

	if role == string(user.User) {
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

func (s *shortenService) GetShortenByOwner(ctx context.Context, rawOwnerID string, page, limit int) (*PaginatedShortenResponse, error) {
	ownerID, err := parseOrDecodeUUID(rawOwnerID)
	if err != nil {
		return nil, err
	}

	shortens, pages, err := s.shortenRepo.GetShortenByOwner(ctx, ownerID, page, limit)
	if err != nil {
		return nil, err
	}

	res, err := ToPaginatedShortenResponse(shortens, page, limit, pages)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) SetURLStatus(ctx context.Context, userID, rawShortenID, role, status string) (*ShortenResponse, error) {
	var active bool
	if status == "active" {
		active = true
	} else {
		active = false
	}

	shortenID, err := parseOrDecodeUUID(rawShortenID)
	if err != nil {
		return nil, err
	}

	existing, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
	if err != nil {
		return nil, err
	}

	if role == string(user.User) {
		if existing.Owner == nil || *existing.Owner != userID {
			return nil, pkg.ErrForbidden
		}
	}

	_, err = s.shortenRepo.UpdateShorten(ctx, &domain.UpdateShortenEntity{
		ID:       shortenID,
		IsActive: &active,
	})
	if err != nil {
		return nil, err
	}

	updatedShorten, err := s.shortenRepo.GetShortenByID(ctx, shortenID)
	if err != nil {
		return nil, err
	}

	if updatedShorten.IsActive != nil && *updatedShorten.IsActive {
		if err := s.cacheShorten(ctx, updatedShorten); err != nil {
			return nil, err
		}
		if err := s.scheduleExpiry(ctx, updatedShorten.ShortCode, updatedShorten.ExpiresAt); err != nil {
			return nil, err
		}
	} else {
		s.redisClient.Del(ctx, shortenCachePrefix+updatedShorten.ShortCode)
		s.redisClient.ZRem(ctx, expiryQueueKey, existing.ShortCode)
	}

	res, err := ToShortenResponse(updatedShorten)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *shortenService) DeleteShorten(ctx context.Context, rawShortenID, rawUserID, role string) error {
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

	if role == string(user.User) {
		if existing.Owner == nil || *existing.Owner != userID {
			return pkg.ErrForbidden
		}
	}

	s.redisClient.Del(ctx, shortenCachePrefix+existing.ShortCode)
	s.redisClient.ZRem(ctx, expiryQueueKey, existing.ShortCode)

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
		var cached domain.ShortenCacheEntity
		if err := json.Unmarshal(data, &cached); err == nil {
			if !shouldServe(*cached.IsActive, cached.ExpiresAt) {
				s.redisClient.Del(ctx, cacheKey)
				return "", pkg.ErrNotFound
			}
			return cached.OriginalURL, nil
		}
	}

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
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.processExpiredQueue(ctx); err != nil {
				log.Printf("failed to process expired queue: %v", err)
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

func (s *shortenService) cacheShorten(ctx context.Context, shorten *domain.ShortenEntity) error {
	cacheKey := shortenCachePrefix + shorten.ShortCode
	ttl := resolveTTL(shorten.ExpiresAt)

	cache := ToShortenCacheEntity(shorten)

	data, err := json.Marshal(cache)
	if err != nil {
		return pkg.ErrRedisMarshal
	}

	if err := s.redisClient.Set(ctx, cacheKey, data, ttl).Err(); err != nil {
		return pkg.HandleRedisError(err)
	}

	return nil
}

func (s *shortenService) scheduleExpiry(ctx context.Context, shortCode string, expiresAt *time.Time) error {
	if expiresAt == nil {
		return s.redisClient.ZRem(ctx, expiryQueueKey, shortCode).Err()
	}

	return s.redisClient.ZAdd(ctx, expiryQueueKey, redis.Z{
		Score:  float64(expiresAt.Unix()),
		Member: shortCode,
	}).Err()
}

func (s *shortenService) processExpiredQueue(ctx context.Context) error {
	now := float64(time.Now().Unix())

	shortCodes, err := s.redisClient.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     expiryQueueKey,
		Start:   "-inf",
		Stop:    fmt.Sprintf("%f", now),
		ByScore: true,
		Offset:  0,
		Count:   100,
	}).Result()

	if err != nil {
		return err
	}

	if len(shortCodes) == 0 {
		return nil
	}

	isActiveFalse := false

	for _, code := range shortCodes {
		removed, err := s.redisClient.ZRem(ctx, expiryQueueKey, code).Result()
		if err != nil || removed == 0 {
			continue
		}

		err = s.shortenRepo.DeactivateByShortCode(ctx, code, &isActiveFalse)
		if err != nil {
			log.Printf("failed to deactivate %s in DB: %v", code, err)
		}

		s.redisClient.Del(ctx, shortenCachePrefix+code)
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
