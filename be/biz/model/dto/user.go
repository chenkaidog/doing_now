package dto

type RegisterReq struct {
	Account  string `json:"account" validate:"required,max=64"`
	Name     string `json:"name" validate:"required,max=64"`
	Password string `json:"password" validate:"required,max=128"`
}

type RegisterResp struct {
	UserID string `json:"user_id"`
}

type LoginReq struct {
	Account  string `json:"account" validate:"max=64"`
	Password string `json:"password" validate:"required,max=128"`
}

type LoginResp struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type RefreshTokenReq struct {
}

type RefreshTokenResp struct {
	AccessToken      string `json:"access_token"`
	ExpiresAt        int64  `json:"expires_at"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresAt int64  `json:"refresh_expires_at"`
}

type LogoutReq struct{}

type LogoutResp struct{}

type GetUserInfoReq struct{}

type GetUserInfoResp struct {
	UserID    string `json:"user_id"`
	Account   string `json:"account"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
