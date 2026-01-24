package convert

import (
	"doing_now/be/biz/model/domain"
	"doing_now/be/biz/model/storage"
)

func UserDomainToRecord(u *domain.User) *storage.UserRecord {
	if u == nil {
		return nil
	}
	return &storage.UserRecord{
		GormModel: storage.GormModel{
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		},
		UserId:       u.UserID,
		Account:      u.Account,
		Name:         u.Name,
		PasswordSalt: u.PasswordSalt,
		PasswordHash: u.PasswordHash,
	}
}

func UserRecordToDomain(m *storage.UserRecord) *domain.User {
	if m == nil {
		return nil
	}
	return &domain.User{
		UserID:       m.UserId,
		Account:      m.Account,
		Name:         m.Name,
		PasswordSalt: m.PasswordSalt,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
