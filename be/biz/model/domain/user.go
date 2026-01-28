package domain

import "time"

type User struct {
	UserID    string
	Account   string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
