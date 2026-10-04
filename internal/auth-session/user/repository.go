package user

import (
	"context"

	"gorm.io/gorm"

	"url-shorten/internal/auth-session/domain"
	"url-shorten/pkg"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// User Repository

func (r *userRepository) CreateUser(ctx context.Context, user *domain.CreateUserEntity) error {
	storage := ToRegisterUserStorage(user)

	if err := r.db.WithContext(ctx).Create(storage).Error; err != nil {
		return pkg.HandleDBError(err)
	}

	return nil
}

func (r *userRepository) UpdateUser(ctx context.Context, update *domain.UpdateUserEntity) error {
	fields := make(map[string]any)

	if update.Name != nil && *update.Name != "" {
		fields["name"] = *update.Name
	}
	if update.Email != nil && *update.Email != "" {
		fields["email"] = *update.Email
	}
	if update.RememberMe != nil {
		fields["remember_me"] = *update.RememberMe
	}

	result := r.db.WithContext(ctx).
		Model(&UserStorage{}).
		Where("id = ?", update.ID).
		Updates(fields)

	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return nil
}

func (r *userRepository) GetUsers(ctx context.Context, page, limit int) ([]domain.UserEntity, int64, error) {
	var storages []UserStorage
	var totalData int64

	if err := r.db.WithContext(ctx).Model(&UserStorage{}).Count(&totalData).Error; err != nil {
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
		return nil, 0, err
	}

	if len(storages) == 0 {
		return nil, 0, pkg.ErrNotFound
	}

	return ToUserListEntity(storages), totalData, nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*domain.UserEntity, error) {
	var storage UserStorage

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToUserEntity(&storage), nil
}

func (r *userRepository) GetPasswordByEmail(ctx context.Context, email string) (string, string, error) {
	var storage UserStorage

	err := r.db.WithContext(ctx).
		Select("id", "password").
		Where("email = ?", email).
		First(&storage).Error

	if err != nil {
		return "", "", pkg.HandleDBError(err)
	}

	return storage.Password, storage.ID, nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&UserStorage{})

	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return nil
}
