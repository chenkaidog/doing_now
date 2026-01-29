package user

import (
	"context"

	"doing_now/be/biz/dal/repo"
	"doing_now/be/biz/db/mysql"
	"doing_now/be/biz/model/convert"
	"doing_now/be/biz/model/domain"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/model/storage"
	"doing_now/be/biz/util/encode"
	"doing_now/be/biz/util/random"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
}

func New() *Service {
	return &Service{}
}

func NewDefault() *Service {
	return New()
}

func (s *Service) Register(ctx context.Context, account, name, password string) (*domain.User, errs.Error) {
	var userRecord *storage.UserRecord
	err := mysql.GetDbConn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := repo.NewUserRepository(tx)
		credentials := repo.NewUserCredentialRepository(tx)

		existing, err := users.FindByAccount(ctx, account)
		if err != nil {
			return err
		}
		if existing != nil {
			return errs.UserNameDuplicatedErr
		}

		userRecord, err = users.Create(ctx, &storage.UserRecord{
			UserId:  uuid.New().String(),
			Account: account,
			Name:    name,
		})
		if err != nil {
			return err
		}

		salt := random.RandStr(32)
		hash := encode.EncodePassword(salt, password)
		if err := credentials.Create(ctx, &storage.UserCredentialRecord{
			UserId:            userRecord.UserId,
			PasswordSalt:      salt,
			PasswordHash:      hash,
			CredentialVersion: 0,
		}); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errs.IsDuplicatedErr(err) {
			hlog.CtxNoticef(ctx, "user name duplicated: %s", account)
			return nil, errs.UserNameDuplicatedErr
		}
		if bizErr, ok := err.(errs.Error); ok {
			return nil, bizErr
		}
		hlog.CtxErrorf(ctx, "register user err: %v", err)
		return nil, errs.ServerError.SetErr(err)
	}
	userDomain := convert.UserRecordToDomain(userRecord)
	return userDomain, nil
}

func (s *Service) Login(ctx context.Context, account, password string) (*domain.User, uint, errs.Error) {
	var userRecord *storage.UserRecord
	var credentialVersion uint
	err := mysql.GetDbConn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := repo.NewUserRepository(tx)
		credentials := repo.NewUserCredentialRepository(tx)

		// 1. Lock user record
		u, err := users.FindByAccountLock(ctx, account)
		if err != nil {
			hlog.CtxErrorf(ctx, "find user by account lock err: %v", err)
			return err
		}
		if u == nil {
			hlog.CtxNoticef(ctx, "user not exist: %s", account)
			return errs.UserNotExist
		}
		userRecord = u

		// 2. Get credential
		c, err := credentials.FindByUserID(ctx, u.UserId)
		if err != nil {
			hlog.CtxErrorf(ctx, "find credential by user id err: %v", err)
			return err
		}
		if c == nil {
			// Inconsistent data state
			hlog.CtxErrorf(ctx, "credential not found for user id: %s", userRecord.UserId)
			return errs.ServerError.SetMsg("credential not found")
		}

		// 3. Verify password
		if encode.EncodePassword(c.PasswordSalt, password) != c.PasswordHash {
			hlog.CtxNoticef(ctx, "password incorrect for user id: %s", userRecord.UserId)
			return errs.PasswordIncorrect
		}

		credentialVersion = c.CredentialVersion
		return nil
	})

	if err != nil {
		if bizErr, ok := err.(errs.Error); ok {
			hlog.CtxNoticef(ctx, "login user err: %v", bizErr)
			return nil, 0, bizErr
		}
		hlog.CtxErrorf(ctx, "login user err: %v", err)
		return nil, 0, errs.ServerError.SetErr(err)
	}
	userDomain := convert.UserRecordToDomain(userRecord)
	return userDomain, credentialVersion, nil
}

func (s *Service) GetByUserID(ctx context.Context, userID string) (*domain.User, errs.Error) {
	users := repo.NewUserRepository(mysql.GetDbConn().WithContext(ctx))
	u, err := users.FindByUserID(ctx, userID)
	if err != nil {
		if bizErr, ok := err.(errs.Error); ok {
			return nil, bizErr
		}
		hlog.CtxErrorf(ctx, "get user by id err: %v", err)
		return nil, errs.ServerError.SetErr(err)
	}
	if u == nil {
		return nil, errs.UserNotExist
	}
	return convert.UserRecordToDomain(u), nil
}

func (s *Service) UpdateInfo(ctx context.Context, userID, name string) errs.Error {
	err := mysql.GetDbConn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := repo.NewUserRepository(tx)
		u, err := users.FindByUserIDLock(ctx, userID)
		if err != nil {
			return err
		}
		if u == nil {
			return errs.UserNotExist
		}
		u.Name = name
		if err := users.Update(ctx, u); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if bizErr, ok := err.(errs.Error); ok {
			hlog.CtxNoticef(ctx, "update user info err: %v", bizErr)
			return bizErr
		}
		hlog.CtxErrorf(ctx, "update user info err: %v", err)
		return errs.ServerError.SetErr(err)
	}
	return nil
}

func (s *Service) GetCredentialVersion(ctx context.Context, userID string) (uint, errs.Error) {
	credentials := repo.NewUserCredentialRepository(mysql.GetDbConn().WithContext(ctx))
	c, err := credentials.FindByUserID(ctx, userID)
	if err != nil {
		hlog.CtxErrorf(ctx, "find credential by user id err: %v", err)
		return 0, errs.ServerError.SetErr(err)
	}
	if c == nil {
		return 0, errs.UserNotExist
	}
	return c.CredentialVersion, nil
}

func (s *Service) UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) errs.Error {
	err := mysql.GetDbConn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := repo.NewUserRepository(tx)
		credentials := repo.NewUserCredentialRepository(tx)

		// 1. Lock user
		u, err := users.FindByUserIDLock(ctx, userID)
		if err != nil {
			return err
		}
		if u == nil {
			return errs.UserNotExist
		}

		// 2. Get credential
		c, err := credentials.FindByUserIDLock(ctx, userID)
		if err != nil {
			return err
		}
		if c == nil {
			return errs.ServerError.SetMsg("credential not found")
		}

		// 3. Verify old password
		if encode.EncodePassword(c.PasswordSalt, oldPassword) != c.PasswordHash {
			return errs.PasswordIncorrect
		}

		// 4. Update password
		salt := random.RandStr(32)
		hash := encode.EncodePassword(salt, newPassword)
		c.PasswordSalt = salt
		c.PasswordHash = hash
		c.CredentialVersion += 1 // Increment version

		if err := credentials.Update(ctx, c); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if bizErr, ok := err.(errs.Error); ok {
			hlog.CtxNoticef(ctx, "update password err: %v", bizErr)
			return bizErr
		}
		hlog.CtxErrorf(ctx, "update password err: %v", err)
		return errs.ServerError.SetErr(err)
	}
	return nil
}
