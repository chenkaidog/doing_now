package security

import (
	"context"
	"testing"

	"doing_now/be/biz/model/errs"

	"github.com/stretchr/testify/assert"
)

func TestCheckCredentialValid(t *testing.T) {
	ctx := context.Background()

	t.Run("empty user id", func(t *testing.T) {
		bizErr := CheckCredentialValid(ctx, "", 0, 0, nil)
		assert.NotNil(t, bizErr)
		assert.Equal(t, errs.Unauthorized.Code(), bizErr.Code())
		assert.Equal(t, msgUserNotLoggedIn, bizErr.Msg())
	})

	t.Run("user not exist", func(t *testing.T) {
		bizErr := CheckCredentialValid(ctx, "u1", 1, 1, errs.UserNotExist)
		assert.NotNil(t, bizErr)
		assert.Equal(t, errs.SessionExpired.Code(), bizErr.Code())
		assert.Equal(t, msgUserNotFound, bizErr.Msg())
	})

	t.Run("db error fail open", func(t *testing.T) {
		bizErr := CheckCredentialValid(ctx, "u1", 1, 1, errs.ServerError)
		assert.Nil(t, bizErr)
	})

	t.Run("credential mismatch", func(t *testing.T) {
		bizErr := CheckCredentialValid(ctx, "u1", 1, 2, nil)
		assert.NotNil(t, bizErr)
		assert.Equal(t, errs.SessionExpired.Code(), bizErr.Code())
		assert.Equal(t, msgCredentialChanged, bizErr.Msg())
	})

	t.Run("credential match", func(t *testing.T) {
		bizErr := CheckCredentialValid(ctx, "u1", 2, 2, nil)
		assert.Nil(t, bizErr)
	})
}
