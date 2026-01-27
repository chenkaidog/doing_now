package jwt

import (
	"context"
	"doing_now/be/biz/config"
	rediscli "doing_now/be/biz/db/redis"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const TokenRemovalTTL = time.Minute
const refreshTokenCookieName = "refresh_token"

func GenerateRefreshToken(ctx context.Context, sessID string) (string, int64, error) {
	tokenID := uuid.New().String()

	jwtConf := config.GetJWTConfig()
	exp := refreshExpiration(jwtConf)
	expAt := time.Now().Add(exp).Unix()

	refreshToken, err := generateToken(Payload{}, exp, tokenID, sessID, jwtConf.RefreshTokenSecret, jwtConf.Issuer)
	if err != nil {
		hlog.CtxErrorf(ctx, "generate refresh token err: %v", err)
		return "", 0, err
	}

	if err := rediscli.GetRedisClient().Set(ctx, refreshTokenKey(tokenID), true, exp).Err(); err != nil {
		hlog.CtxErrorf(ctx, "set refresh token to redis err: %v", err)
		return "", 0, err
	}

	return refreshToken, expAt, nil
}

func RemoveRefreshToken(ctx context.Context, refreshToken, sessID string) error {
	// 先校验token的有效性
	jwtConf := config.GetJWTConfig()
	claims, err := validateToken(refreshToken, jwtConf.RefreshTokenSecret)
	if err != nil {
		hlog.CtxErrorf(ctx, "validate refresh token err: %v", err)
		return ErrRefreshTokenInvalid
	}
	if !claims.CheckSum(sessID) {
		return ErrRefreshTokenInvalid
	}

	exist, err := rediscli.GetRedisClient().
		Get(ctx, refreshTokenKey(claims.ID)).Bool()
	if err != nil && !errors.Is(err, redis.Nil) {
		hlog.CtxErrorf(ctx, "get refresh token from redis err: %v", err)
		return err
	}
	if !exist {
		return ErrRefreshTokenInvalid
	}

	// 校验通过后，删除redis中的token
	timeLeft := time.Until(claims.ExpiresAt.Time)
	if timeLeft < 0 {
		return rediscli.GetRedisClient().Del(ctx, refreshTokenKey(claims.ID)).Err()
	}

	newTTL := TokenRemovalTTL
	if timeLeft < newTTL {
		newTTL = timeLeft
	}

	return rediscli.GetRedisClient().Expire(ctx, refreshTokenKey(claims.ID), newTTL).Err()
}

func GetRefreshTokenFromCookie(c *app.RequestContext) string {
	return string(c.Cookie(refreshTokenCookieName))
}

func SetRefreshTokenCookie(c *app.RequestContext, refreshToken string, expireAt int64) {
	conf := config.GetSessionConf()
	maxAge := int(expireAt - time.Now().Unix())

	c.SetCookie(
		refreshTokenCookieName,
		refreshToken,
		maxAge,
		defaultString(conf.Path, "/"),
		conf.Domain,
		parseCookieSameSite(conf.SameSite),
		conf.Secure,
		conf.HTTPOnly,
	)
}

func ClearRefreshTokenCookie(c *app.RequestContext) {
	conf := config.GetSessionConf()
	c.SetCookie(
		refreshTokenCookieName,
		"",
		-1,
		defaultString(conf.Path, "/"),
		conf.Domain,
		parseCookieSameSite(conf.SameSite),
		conf.Secure,
		conf.HTTPOnly,
	)
}

func refreshTokenKey(t string) string {
	return fmt.Sprintf("refresh_token:%s", t)
}

func refreshExpiration(conf config.JWTConf) time.Duration {
	if conf.RefreshExpiration > 0 {
		return time.Duration(conf.RefreshExpiration) * time.Second
	}
	return 30 * 24 * time.Hour
}

func defaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func parseCookieSameSite(v string) protocol.CookieSameSite {
	switch v {
	case "Lax":
		return protocol.CookieSameSiteLaxMode
	case "None":
		return protocol.CookieSameSiteNoneMode
	case "Strict":
		fallthrough
	default:
		return protocol.CookieSameSiteStrictMode
	}
}
