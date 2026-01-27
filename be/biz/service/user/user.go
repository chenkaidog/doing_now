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
)

type Service struct {
	users *repo.UserRepository
}

func New(users *repo.UserRepository) *Service {
	return &Service{users: users}
}

func NewDefault() *Service {
	return New(repo.NewUserRepository(mysql.GetDbConn()))
}

func (s *Service) Register(ctx context.Context, account, name, password string) (*domain.User, errs.Error) {
	existing, err := s.users.FindByAccount(ctx, account)
	if err != nil {
		return nil, errs.ServerError
	}
	if existing != nil {
		return nil, errs.UserNameDuplicatedErr
	}

	salt := random.RandStr(16)
	hash := encode.EncodePassword(salt, password)
	u := &storage.UserRecord{
		Account:      account,
		Name:         name,
		PasswordSalt: salt,
		PasswordHash: hash,
	}
	userRecord, err := s.users.Create(ctx, u)
	if err != nil {
		if errs.IsDuplicatedErr(err) {
			return nil, errs.UserNameDuplicatedErr
		}
		return nil, errs.ServerError
	}
	return convert.UserRecordToDomain(userRecord), nil
}

func (s *Service) Login(ctx context.Context, account, password string) (*domain.User, errs.Error) {
	u, err := s.users.FindByAccount(ctx, account)
	if err != nil {
		return nil, errs.ServerError
	}
	if u == nil {
		return nil, errs.UserNotExist
	}
	if encode.EncodePassword(u.PasswordSalt, password) != u.PasswordHash {
		return nil, errs.PasswordIncorrect
	}
	return convert.UserRecordToDomain(u), nil
}

func (s *Service) GetByUserID(ctx context.Context, userID string) (*domain.User, errs.Error) {
	u, err := s.users.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errs.ServerError
	}
	if u == nil {
		return nil, errs.UserNotExist
	}
	return convert.UserRecordToDomain(u), nil
}

func (s *Service) UpdateInfo(ctx context.Context, userID, name string) errs.Error {
	u, err := s.users.FindByUserID(ctx, userID)
	if err != nil {
		return errs.ServerError
	}
	if u == nil {
		return errs.UserNotExist
	}
	u.Name = name
	if err := s.users.Update(ctx, u); err != nil {
		return errs.ServerError
	}
	return nil
}

func (s *Service) UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) errs.Error {
	u, err := s.users.FindByUserID(ctx, userID)
	if err != nil {
		return errs.ServerError
	}
	if u == nil {
		return errs.UserNotExist
	}

	if encode.EncodePassword(u.PasswordSalt, oldPassword) != u.PasswordHash {
		return errs.PasswordIncorrect
	}

	salt := random.RandStr(16)
	hash := encode.EncodePassword(salt, newPassword)
	u.PasswordSalt = salt
	u.PasswordHash = hash

	if err := s.users.Update(ctx, u); err != nil {
		return errs.ServerError
	}
	return nil
}
