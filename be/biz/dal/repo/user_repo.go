package repo

import (
	"context"

	"doing_now/be/biz/model/storage"
	"doing_now/be/biz/util/id_gen"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *storage.UserRecord) (*storage.UserRecord, error) {
	if u.UserId == "" {
		u.UserId = id_gen.NewID()
	}
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
