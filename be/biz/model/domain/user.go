package domain

import "time"

type User struct {
	UserID    string
	Account   string
	Name      string
	CredentialVersion uint
	CreatedAt time.Time
	UpdatedAt time.Time
}
