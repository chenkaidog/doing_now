package domain

import "time"

type User struct {
	UserID       string
	Account      string
	Name         string
	PasswordSalt string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
