package jwt

import (
	"context"
	"doing_now/be/biz/util/random"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJwt(t *testing.T) {

	secret := "secret"
	tokenId := random.RandStr(10)
	sessId := random.RandStr(10)

	jwtStr, err := generateToken(Payload{
		UserID: random.RandStr(32),
	}, time.Second, tokenId, sessId, secret, "go test")
	assert.Nil(t, err)
	t.Log(jwtStr)

	t.Run("success", func(t *testing.T) {
		claims, err := validateToken(jwtStr, secret)
		assert.Nil(t, err)
		assert.True(t, claims.CheckSum(sessId))
	})

	t.Run("secret key invalid", func(t *testing.T) {
		_, err := validateToken(jwtStr, secret+"123")
		assert.ErrorIs(t, ErrJwtInvalid, err)
	})

	t.Run("expired", func(t *testing.T) {
		time.Sleep(time.Second * 2)
		_, err := validateToken(jwtStr, secret)
		assert.ErrorIs(t, ErrJwtExpired, err)
	})

}

func TestGetPayloadAndRemoveToken(t *testing.T) {
	t.Run("GetPayload default empty", func(t *testing.T) {
		p := GetPayload(context.Background())
		assert.Equal(t, Payload{}, p)
	})

	t.Run("GetPayload from context", func(t *testing.T) {
		claims := &Claims{Payload: Payload{UserID: "u1", Account: "a"}}
		ctx := context.WithValue(context.Background(), Payload{}, claims)
		p := GetPayload(ctx)
		assert.Equal(t, "u1", p.UserID)
		assert.Equal(t, "a", p.Account)
	})

	t.Run("RemoveToken without claims is noop", func(t *testing.T) {
		err := RemoveToken(context.Background(), "")
		assert.Nil(t, err)
	})
}
