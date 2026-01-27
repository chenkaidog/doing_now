package repo

import (
	"context"

	"doing_now/be/biz/model/storage"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserCredentialRepository struct {
	db *gorm.DB
}

func NewUserCredentialRepository(db *gorm.DB) *UserCredentialRepository {
	return &UserCredentialRepository{db: db}
}

func (r *UserCredentialRepository) Create(ctx context.Context, c *storage.UserCredentialRecord) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *UserCredentialRepository) FindByUserID(ctx context.Context, userID string) (*storage.UserCredentialRecord, error) {
	var m storage.UserCredentialRecord
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserCredentialRepository) FindByUserIDLock(ctx context.Context, userID string) (*storage.UserCredentialRecord, error) {
	var m storage.UserCredentialRecord
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserCredentialRepository) Update(ctx context.Context, c *storage.UserCredentialRecord) error {
	return r.db.WithContext(ctx).Save(c).Error
}
