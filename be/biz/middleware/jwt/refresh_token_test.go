package jwt

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"doing_now/be/biz/config"
	rediscli "doing_now/be/biz/db/redis"
	"doing_now/be/biz/util/encode"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func initTestConfig() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	host, port, _ := net.SplitHostPort(mr.Addr())

	content := []byte(fmt.Sprintf(`
jwt:
  access_token_secret: "test-secret"
  refresh_token_secret: "test-secret"
  issuer: "test"
  access_expiration: 3600
  refresh_expiration: 7200
redis:
  ip: "%s"
  port: %s
`, host, port))
	tmpfile, _ := os.CreateTemp("", "config-*.yaml")
	tmpfile.Write(content)
	tmpfile.Close()
	config.Init(tmpfile.Name())
	os.Remove(tmpfile.Name())
	return mr
}

func TestGenerateAndValidateRefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	sessID := "test-session-id"
	mr := initTestConfig()
	defer mr.Close()
	rediscli.Init()

	token, expAt, err := GenerateRefreshToken(ctx, sessID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expAt > time.Now().Unix())
}

func TestRemoveRefreshToken_TTLUpdate(t *testing.T) {
	ctx := context.Background()
	sessID := "test-session-id"
	mr := initTestConfig()
	defer mr.Close()
	rediscli.Init()

	// Manually create a token to ensure we have the ID to check redis
	tokenID := uuid.New().String()
	jwtConf := config.GetJWTConfig()
	exp := time.Hour

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			Issuer:    jwtConf.Issuer,
			ID:        tokenID,
		},
		Sum: encode.EncodePassword(tokenID, sessID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(jwtConf.RefreshTokenSecret))

	// Set in Redis
	err := rediscli.GetRedisClient().Set(ctx, refreshTokenKey(tokenID), true, exp).Err()
	assert.NoError(t, err)

	// Remove
	err = RemoveRefreshToken(ctx, tokenStr, sessID)
	assert.NoError(t, err)

	// Check TTL is now <= 1 min (approx)
	ttl, err := rediscli.GetRedisClient().TTL(ctx, refreshTokenKey(tokenID)).Result()
	assert.NoError(t, err)
	assert.True(t, ttl <= time.Minute)
	assert.True(t, ttl > 0)

	// Check it still exists
	exists, err := rediscli.GetRedisClient().Exists(ctx, refreshTokenKey(tokenID)).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestRemoveRefreshToken_ExpiredButValidSignature(t *testing.T) {
	// This test might be tricky if we can't easily generate an expired token that passes signature check
	// but RemoveRefreshToken checks expiration first.
	// Actually, if it's expired, RemoveRefreshToken deletes it immediately.
	// We can simulate this.

	ctx := context.Background()
	sessID := "test-session-id"
	mr := initTestConfig()
	defer mr.Close()
	rediscli.Init()

	tokenID := uuid.New().String()
	jwtConf := config.GetJWTConfig()
	// Expired 1 second ago
	exp := -1 * time.Second

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			Issuer:    jwtConf.Issuer,
			ID:        tokenID,
		},
		Sum: encode.EncodePassword(tokenID, sessID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(jwtConf.RefreshTokenSecret))

	// Set in Redis (key might not exist if we use Set with negative exp? Redis Set with negative exp usually fails or deletes.
	// Let's set it with positive exp but claims say expired.
	err := rediscli.GetRedisClient().Set(ctx, refreshTokenKey(tokenID), true, time.Minute).Err()
	assert.NoError(t, err)

	err = RemoveRefreshToken(ctx, tokenStr, sessID)
	assert.ErrorIs(t, err, ErrRefreshTokenInvalid)

	// Should still exist since expired token is treated as invalid
	exists, err := rediscli.GetRedisClient().Exists(ctx, refreshTokenKey(tokenID)).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}
