package repo

import (
	"context"

	"doing_now/be/biz/model/convert"
	"doing_now/be/biz/model/domain"
	"doing_now/be/biz/model/storage"
	"doing_now/be/biz/util/id_gen"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	FindByUserID(ctx context.Context, userID string) (*domain.User, error)
	FindByAccount(ctx context.Context, account string) (*domain.User, error)
	FindByID(ctx context.Context, id uint64) (*domain.User, error)
}

type UserRepositoryGorm struct {
	db *gorm.DB
}

func NewUserRepositoryGorm(db *gorm.DB) *UserRepositoryGorm {
	return &UserRepositoryGorm{db: db}
}

func (r *UserRepositoryGorm) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	m := convert.UserDomainToRecord(u)
	if m.UserId == "" {
		m.UserId = id_gen.NewID()
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return convert.UserRecordToDomain(m), nil
}

func (r *UserRepositoryGorm) FindByUserID(ctx context.Context, userID string) (*domain.User, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return convert.UserRecordToDomain(&m), nil
}

func (r *UserRepositoryGorm) FindByAccount(ctx context.Context, account string) (*domain.User, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Where("account = ?", account).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return convert.UserRecordToDomain(&m), nil
}

func (r *UserRepositoryGorm) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var m storage.UserRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return convert.UserRecordToDomain(&m), nil
}
