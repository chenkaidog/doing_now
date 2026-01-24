package storage

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type GormModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt soft_delete.DeletedAt
}

type UserRecord struct {
	GormModel
	UserId       string `gorm:"size:64;not null;uniqueIndex"` // 用户唯一索引
	Account      string `gorm:"size:64;not null;uniqueIndex"` // 用户唯一登录账号
	Name         string `gorm:"size:64;not null"`             // 用户姓名
	PasswordSalt string `gorm:"size:64;not null"`
	PasswordHash string `gorm:"size:128;not null"`
}

func (UserRecord) TableName() string {
	return "users"
}
