package user

import (
	"context"
	"errors"
	"testing"

	"doing_now/be/biz/model/domain"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/util/encode"

	"github.com/stretchr/testify/assert"
)

type fakeUserRepo struct {
	findByAccountUser *domain.User
	findByAccountErr  error

	findByUserIDUser *domain.User
	findByUserIDErr  error

	createRetUser *domain.User
	createRetErr  error
	createInput   *domain.User
}

func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) (*domain.User, error) {
	r.createInput = u
	return r.createRetUser, r.createRetErr
}

func (r *fakeUserRepo) FindByUserID(_ context.Context, _ string) (*domain.User, error) {
	return r.findByUserIDUser, r.findByUserIDErr
}

func (r *fakeUserRepo) FindByAccount(_ context.Context, _ string) (*domain.User, error) {
	return r.findByAccountUser, r.findByAccountErr
}

func (r *fakeUserRepo) FindByID(_ context.Context, _ uint64) (*domain.User, error) {
	return nil, nil
}

func TestService_Register(t *testing.T) {
	t.Run("find error", func(t *testing.T) {
		svc := New(&fakeUserRepo{findByAccountErr: errors.New("db error")})
		_, bizErr := svc.Register(context.Background(), "a", "n", "p")
		assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
	})

	t.Run("account duplicated", func(t *testing.T) {
		svc := New(&fakeUserRepo{findByAccountUser: &domain.User{UserID: "u1"}})
		_, bizErr := svc.Register(context.Background(), "a", "n", "p")
		assert.True(t, errs.ErrorEqual(errs.UserNameDuplicatedErr, bizErr))
	})

	t.Run("create error", func(t *testing.T) {
		svc := New(&fakeUserRepo{createRetErr: errors.New("insert error")})
		_, bizErr := svc.Register(context.Background(), "a", "n", "p")
		assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
	})

	t.Run("success sets salt and hash", func(t *testing.T) {
		repo := &fakeUserRepo{
			createRetUser: &domain.User{UserID: "u1", Account: "a", Name: "n"},
		}
		svc := New(repo)

		u, bizErr := svc.Register(context.Background(), "a", "n", "p")
		assert.Nil(t, bizErr)
		assert.Equal(t, "u1", u.UserID)

		if assert.NotNil(t, repo.createInput) {
			assert.Equal(t, "a", repo.createInput.Account)
			assert.Equal(t, "n", repo.createInput.Name)
			assert.Len(t, repo.createInput.PasswordSalt, 16)
			assert.NotEmpty(t, repo.createInput.PasswordHash)
			assert.Equal(t, encode.EncodePassword(repo.createInput.PasswordSalt, "p"), repo.createInput.PasswordHash)
		}
	})
}

func TestService_Login(t *testing.T) {
	t.Run("find error", func(t *testing.T) {
		svc := New(&fakeUserRepo{findByAccountErr: errors.New("db error")})
		_, bizErr := svc.Login(context.Background(), "a", "p")
		assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
	})

	t.Run("user not exist", func(t *testing.T) {
		svc := New(&fakeUserRepo{findByAccountUser: nil})
		_, bizErr := svc.Login(context.Background(), "a", "p")
		assert.True(t, errs.ErrorEqual(errs.UserNotExist, bizErr))
	})

	t.Run("password incorrect", func(t *testing.T) {
		u := &domain.User{UserID: "u1", PasswordSalt: "salt", PasswordHash: encode.EncodePassword("salt", "right")}
		svc := New(&fakeUserRepo{findByAccountUser: u})
		_, bizErr := svc.Login(context.Background(), "a", "wrong")
		assert.True(t, errs.ErrorEqual(errs.PasswordIncorrect, bizErr))
	})

	t.Run("success", func(t *testing.T) {
		u := &domain.User{UserID: "u1", PasswordSalt: "salt", PasswordHash: encode.EncodePassword("salt", "p")}
		svc := New(&fakeUserRepo{findByAccountUser: u})
		out, bizErr := svc.Login(context.Background(), "a", "p")
		assert.Nil(t, bizErr)
		assert.Equal(t, u, out)
	})
}

func TestService_GetByUserID(t *testing.T) {
	t.Run("find error", func(t *testing.T) {
		svc := New(&fakeUserRepo{findByUserIDErr: errors.New("db error")})
		_, bizErr := svc.GetByUserID(context.Background(), "u1")
		assert.True(t, errs.ErrorEqual(errs.ServerError, bizErr))
	})

	t.Run("user not exist", func(t *testing.T) {
		svc := New(&fakeUserRepo{findByUserIDUser: nil})
		_, bizErr := svc.GetByUserID(context.Background(), "u1")
		assert.True(t, errs.ErrorEqual(errs.UserNotExist, bizErr))
	})

	t.Run("success", func(t *testing.T) {
		u := &domain.User{UserID: "u1"}
		svc := New(&fakeUserRepo{findByUserIDUser: u})
		out, bizErr := svc.GetByUserID(context.Background(), "u1")
		assert.Nil(t, bizErr)
		assert.Equal(t, u, out)
	})
}
