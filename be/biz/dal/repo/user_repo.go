package repo

import (
	"context"

	"doing_now/be/biz/model/storage"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *storage.UserRecord) (*storage.UserRecord, error) {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByUserID(ctx context.Context, userID string) (*storage.UserRecord, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserRepository) FindByUserIDLock(ctx context.Context, userID string) (*storage.UserRecord, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserRepository) FindByAccount(ctx context.Context, account string) (*storage.UserRecord, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Where("account = ?", account).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserRepository) FindByAccountLock(ctx context.Context, account string) (*storage.UserRecord, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("account = ?", account).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint64) (*storage.UserRecord, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *UserRepository) Update(ctx context.Context, u *storage.UserRecord) error {
	return r.db.WithContext(ctx).Save(u).Error
}
