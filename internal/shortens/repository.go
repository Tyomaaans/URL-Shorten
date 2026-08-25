package shortens

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"url-shorten/internal/domains"
	"url-shorten/internal/infrastructure/postgres"
	"url-shorten/pkg"
)

type shortenRepository struct {
	db *gorm.DB
}

func NewShortenRepository(db *gorm.DB) domains.ShortenRepository {
	return &shortenRepository{
		db: db,
	}
}

func (r *shortenRepository) CreateShorten(ctx context.Context, shorten *domains.ShortenEntity) (*domains.ShortenEntity, error) {
	storage := ToShortenStoreage(shorten)

	if err := r.db.WithContext(ctx).
		Create(storage).
		Clauses(clause.Returning{}).
		Error; err != nil {
			return nil, pkg.HandleDBError(err)
		}

	return ToShortenEntity(storage), nil
}

func (r *shortenRepository) UpdateShorten(ctx context.Context, update *domains.UpdateShortenEntity) (*domains.ShortenEntity, error) {
	fields := make(map[string]any)

	if update.OriginalURL != nil && *update.OriginalURL != "" {
		fields["original_url"] = *update.OriginalURL
	}
	if update.ShortCode != nil && *update.ShortCode != "" {
		fields["short_code"] = *update.ShortCode
	}
	if update.IsActive != nil {
		fields["is_active"] = *update.IsActive
	}
	if update.ExpiresAt != nil {
		fields["expires_at"] = *update.ExpiresAt
	}

	var storage postgres.ShortenStorage

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

func (r *shortenRepository) GetShortens(ctx context.Context) ([]domains.ShortenEntity, error) {
	var storages []postgres.ShortenStorage

	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&storages).Error; err != nil {
			return nil, pkg.HandleDBError(err)
		}

	return ToShortenListEntity(storages), nil
}

func (r *shortenRepository) GetShortenByID(ctx context.Context, id string) (*domains.ShortenEntity, error) {
	var storage postgres.ShortenStorage

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToShortenEntity(&storage), nil
}

func (r *shortenRepository) GetShortenByOwner(ctx context.Context, owner string) ([]domains.ShortenEntity, error) {
	var storages []postgres.ShortenStorage

	if err := r.db.WithContext(ctx).
		Where("owner = ?", owner).
		Order("created_at ASC").
		Find(&storages).Error; err != nil {
			return nil, pkg.HandleDBError(err)
		}

	return ToShortenListEntity(storages), nil
}

func (r *shortenRepository) GetShortenByShortCode(ctx context.Context, shortCode string) (*domains.ShortenEntity, error) {
	var storage postgres.ShortenStorage

	err := r.db.WithContext(ctx).
		Where("short_code = ?", shortCode).
		First(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToShortenEntity(&storage), nil
}

func (r *shortenRepository) GetActiveWithExpiry(ctx context.Context) ([]domains.ShortenEntity, error) {
	var storage []postgres.ShortenStorage

	err := r.db.WithContext(ctx).
		Where("is_active = ? AND expires_at IS NOT NULL", true).
		Find(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToShortenListEntity(storage), nil
}

func (r *shortenRepository) DeleteShorten(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&postgres.ShortenStorage{})
	
	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return nil
}