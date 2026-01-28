package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"doing_now/be/biz/dal/repo"
	"doing_now/be/biz/db/mysql"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/model/storage"
	"doing_now/be/biz/util/encode"
	"doing_now/be/biz/util/random"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func mockDB() {
	mockey.Mock(mysql.GetDbConn).Return(&gorm.DB{}).Build()
	mockey.Mock((*gorm.DB).WithContext).To(func(db *gorm.DB, ctx context.Context) *gorm.DB {
		return db
	}).Build()
	mockey.Mock((*gorm.DB).Transaction).To(func(tx *gorm.DB, fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
		return fc(tx)
	}).Build()
}

func TestService_Register(t *testing.T) {
	t.Run("find error", func(t *testing.T) {
		mockey.PatchConvey("find error", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByAccount).Return(nil, errors.New("db error")).Build()

			svc := New()
			_, bizErr := svc.Register(context.Background(), "a", "n", "p")
			assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
		})
	})

	t.Run("account duplicated", func(t *testing.T) {
		mockey.PatchConvey("account duplicated", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByAccount).Return(&storage.UserRecord{UserId: "u1"}, nil).Build()

			svc := New()
			_, bizErr := svc.Register(context.Background(), "a", "n", "p")
			assert.True(t, errs.ErrorEqual(errs.UserNameDuplicatedErr, bizErr))
		})
	})

	t.Run("create error", func(t *testing.T) {
		mockey.PatchConvey("create error", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByAccount).Return(nil, nil).Build()
			mockey.Mock((*repo.UserRepository).Create).Return(nil, errors.New("insert error")).Build()
			mockey.Mock(random.RandStr).Return("salt").Build()
			mockey.Mock(encode.EncodePassword).Return("hash").Build()

			svc := New()
			_, bizErr := svc.Register(context.Background(), "a", "n", "p")
			assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
		})
	})

	t.Run("success", func(t *testing.T) {
		mockey.PatchConvey("success", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByAccount).Return(nil, nil).Build()
			mockey.Mock((*repo.UserRepository).Create).Return(&storage.UserRecord{
				UserId: "u1", Account: "a", Name: "n",
			}, nil).Build()
			mockey.Mock((*repo.UserCredentialRepository).Create).Return(nil).Build()
			mockey.Mock(random.RandStr).Return("salt").Build()
			mockey.Mock(encode.EncodePassword).Return("hash").Build()

			svc := New()
			u, bizErr := svc.Register(context.Background(), "a", "n", "p")
			assert.Nil(t, bizErr)
			assert.Equal(t, "u1", u.UserID)
		})
	})
}

func TestService_Login(t *testing.T) {
	t.Run("find error", func(t *testing.T) {
		mockey.PatchConvey("find error", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByAccountLock).Return(nil, errors.New("db error")).Build()

			svc := New()
			_, _, bizErr := svc.Login(context.Background(), "a", "p")
			assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
		})
	})

	t.Run("user not exist", func(t *testing.T) {
		mockey.PatchConvey("user not exist", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByAccountLock).Return(nil, nil).Build()

			svc := New()
			_, _, bizErr := svc.Login(context.Background(), "a", "p")
			assert.True(t, errs.ErrorEqual(errs.UserNotExist, bizErr))
		})
	})

	t.Run("password incorrect", func(t *testing.T) {
		mockey.PatchConvey("password incorrect", t, func() {
			mockDB()
			u := &storage.UserRecord{UserId: "u1"}
			c := &storage.UserCredentialRecord{UserId: "u1", PasswordSalt: "salt", PasswordHash: "right_hash"}
			mockey.Mock((*repo.UserRepository).FindByAccountLock).Return(u, nil).Build()
			mockey.Mock((*repo.UserCredentialRepository).FindByUserID).Return(c, nil).Build()
			mockey.Mock(encode.EncodePassword).Return("wrong_hash").Build()

			svc := New()
			_, _, bizErr := svc.Login(context.Background(), "a", "wrong")
			assert.True(t, errs.ErrorEqual(errs.PasswordIncorrect, bizErr))
		})
	})

	t.Run("success", func(t *testing.T) {
		mockey.PatchConvey("success", t, func() {
			mockDB()
			u := &storage.UserRecord{UserId: "u1"}
			c := &storage.UserCredentialRecord{UserId: "u1", PasswordSalt: "salt", PasswordHash: "right_hash"}
			mockey.Mock((*repo.UserRepository).FindByAccountLock).Return(u, nil).Build()
			mockey.Mock((*repo.UserCredentialRepository).FindByUserID).Return(c, nil).Build()
			mockey.Mock(encode.EncodePassword).Return("right_hash").Build()

			svc := New()
			out, _, bizErr := svc.Login(context.Background(), "a", "p")
			assert.Nil(t, bizErr)
			assert.Equal(t, "u1", out.UserID)
		})
	})
}

func TestService_GetByUserID(t *testing.T) {
	t.Run("find error", func(t *testing.T) {
		mockey.PatchConvey("find error", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByUserID).Return(nil, errors.New("db error")).Build()

			svc := New()
			_, bizErr := svc.GetByUserID(context.Background(), "u1")
			assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
		})
	})

	t.Run("user not exist", func(t *testing.T) {
		mockey.PatchConvey("user not exist", t, func() {
			mockDB()
			mockey.Mock((*repo.UserRepository).FindByUserID).Return(nil, nil).Build()

			svc := New()
			_, bizErr := svc.GetByUserID(context.Background(), "u1")
			assert.True(t, errs.ErrorEqual(errs.UserNotExist, bizErr))
		})
	})

	t.Run("success", func(t *testing.T) {
		mockey.PatchConvey("success", t, func() {
			mockDB()
			u := &storage.UserRecord{UserId: "u1"}
			mockey.Mock((*repo.UserRepository).FindByUserID).Return(u, nil).Build()

			svc := New()
			out, bizErr := svc.GetByUserID(context.Background(), "u1")
			assert.Nil(t, bizErr)
			assert.Equal(t, "u1", out.UserID)
		})
	})
}
