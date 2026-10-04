package shortener

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"url-shorten/internal/url-shortener/domain"
	"url-shorten/pkg"
)

type shortenRepository struct {
	db *gorm.DB
}

func NewShortenRepository(db *gorm.DB) domain.ShortenRepository {
	return &shortenRepository{
		db: db,
	}
}

func (r *shortenRepository) CreateShorten(ctx context.Context, shorten *domain.ShortenEntity) (*domain.ShortenEntity, error) {
	storage := ToShortenStoreage(shorten)

	if err := r.db.WithContext(ctx).
		Create(storage).
		Clauses(clause.Returning{}).
		Error; err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToShortenEntity(storage), nil
}

func (r *shortenRepository) UpdateShorten(ctx context.Context, update *domain.UpdateShortenEntity) (*domain.ShortenEntity, error) {
	fields := make(map[string]any)

	if update.OriginalURL != nil && *update.OriginalURL != "" {
		fields["original_url"] = *update.OriginalURL
	}
	if update.IsActive != nil {
		fields["is_active"] = *update.IsActive
	}
	if update.ExpiresAt.Set {
		fields["expires_at"] = update.ExpiresAt.Value
	}

	var storage ShortenStorage

	result := r.db.WithContext(ctx).
		Model(&storage).
		Clauses(clause.Returning{}).
		Where("id = ?", update.ID).
		Updates(fields)

	if result.Error != nil {
		return nil, pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return ToShortenEntity(&storage), nil
}

func (r *shortenRepository) GetShortens(ctx context.Context, page, limit int) ([]domain.ShortenEntity, int64, error) {
	var storages []ShortenStorage
	var totalData int64

	if err := r.db.WithContext(ctx).Model(&ShortenStorage{}).Count(&totalData).Error; err != nil {
		return nil, 0, pkg.HandleDBError(err)
	}

	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&storages).Error; err != nil {
		return nil, 0, pkg.HandleDBError(err)
	}

	if len(storages) == 0 {
		return nil, 0, pkg.ErrNotFound
	}

	return ToShortenListEntity(storages), totalData, nil
}

func (r *shortenRepository) GetShortenByID(ctx context.Context, id string) (*domain.ShortenEntity, error) {
	var storage ShortenStorage

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToShortenEntity(&storage), nil
}

func (r *shortenRepository) GetShortenByOwner(ctx context.Context, owner string, page, limit int) ([]domain.ShortenEntity, int64, error) {
	var storages []ShortenStorage
	var totalData int64

	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	if err := r.db.WithContext(ctx).
		Model(&ShortenStorage{}).
		Where("owner = ?", owner).
		Count(&totalData).Error; err != nil {
		return nil, 0, pkg.HandleDBError(err)
	}

	if err := r.db.WithContext(ctx).
		Where("owner = ?", owner).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&storages).Error; err != nil {
		return nil, 0, pkg.HandleDBError(err)
	}

	if len(storages) == 0 {
		return nil, 0, pkg.ErrNotFound
	}

	return ToShortenListEntity(storages), totalData, nil
}

func (r *shortenRepository) GetShortenByShortCode(ctx context.Context, shortCode string) (*domain.ShortenEntity, error) {
	var storage ShortenStorage

	err := r.db.WithContext(ctx).
		Where("short_code = ?", shortCode).
		First(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToShortenEntity(&storage), nil
}

func (r *shortenRepository) DeactivateByShortCode(ctx context.Context, shortCode string, isActive *bool) error {
	result := r.db.WithContext(ctx).
		Model(&ShortenStorage{}).
		Where("short_code = ?", shortCode).
		Update("is_active", isActive)

	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *shortenRepository) DeleteShorten(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&ShortenStorage{})

	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return nil
}
