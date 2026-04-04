package user

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"doing_now/be/biz/dal/repo"
	"doing_now/be/biz/db/mysql"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/model/storage"
	"doing_now/be/biz/service/security"

	"github.com/bytedance/mockey"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var (
	patchOnce sync.Once
	currentDB *gorm.DB
)

const testClientIP = "127.0.0.1"

func ensurePatches() {
	patchOnce.Do(func() {
		mockey.Mock(mysql.GetDbConn).To(func() *gorm.DB {
			return currentDB
		}).Build()

		mockey.Mock((*repo.UserRepository).FindByAccountLock).To(func(r *repo.UserRepository, ctx context.Context, account string) (*storage.UserRecord, error) {
			return r.FindByAccount(ctx, account)
		}).Build()
		mockey.Mock((*repo.UserRepository).FindByUserIDLock).To(func(r *repo.UserRepository, ctx context.Context, userID string) (*storage.UserRecord, error) {
			return r.FindByUserID(ctx, userID)
		}).Build()
		mockey.Mock((*repo.UserCredentialRepository).FindByUserIDLock).To(func(r *repo.UserCredentialRepository, ctx context.Context, userID string) (*storage.UserCredentialRecord, error) {
			return r.FindByUserID(ctx, userID)
		}).Build()
		mockey.Mock(security.CheckRegisterAllowed).Return(nil).Build()
		mockey.Mock(security.MarkRegisterSuccess).To(func(ctx context.Context, ip string) {}).Build()
		mockey.Mock(security.CheckLoginAllowed).Return(nil).Build()
		mockey.Mock(security.HandleLoginFailure).To(func(ctx context.Context, ip string) {}).Build()
	})
}

func setupSQLite(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(&storage.UserRecord{}, &storage.UserCredentialRecord{})
	assert.NoError(t, err)
	return db
}

func TestService_Register(t *testing.T) {
	ensurePatches()
	currentDB = setupSQLite(t)

	svc := New()

	u, bizErr := svc.Register(context.Background(), RegisterParam{
		Account:  "account01",
		Name:     "name0001",
		Password: "password01",
		ClientIP: testClientIP,
	})
	assert.Nil(t, bizErr)
	assert.NotEmpty(t, u.UserID)

	_, bizErr = svc.Register(context.Background(), RegisterParam{
		Account:  "account01",
		Name:     "name0001",
		Password: "password01",
		ClientIP: testClientIP,
	})
	assert.True(t, errs.ErrorEqual(errs.UserNameDuplicatedErr, bizErr))
}

func TestService_Login(t *testing.T) {
	ensurePatches()
	currentDB = setupSQLite(t)

	svc := New()
	_, bizErr := svc.Register(context.Background(), RegisterParam{
		Account:  "account01",
		Name:     "name0001",
		Password: "password01",
		ClientIP: testClientIP,
	})
	assert.Nil(t, bizErr)

	_, _, bizErr = svc.Login(context.Background(), LoginParam{
		Account:  "not_exist",
		Password: "password01",
		ClientIP: testClientIP,
	})
	assert.True(t, errs.ErrorEqual(errs.UserNotExist, bizErr))

	_, _, bizErr = svc.Login(context.Background(), LoginParam{
		Account:  "account01",
		Password: "badpassword",
		ClientIP: testClientIP,
	})
	assert.True(t, errs.ErrorEqual(errs.PasswordIncorrect, bizErr))

	u, cv, bizErr := svc.Login(context.Background(), LoginParam{
		Account:  "account01",
		Password: "password01",
		ClientIP: testClientIP,
	})
	assert.Nil(t, bizErr)
	assert.Equal(t, uint(0), cv)
	assert.NotEmpty(t, u.UserID)
}

func TestService_GetByUserID(t *testing.T) {
	ensurePatches()
	currentDB = setupSQLite(t)

	svc := New()
	_, bizErr := svc.GetByUserID(context.Background(), GetByUserIDParam{
		UserID: "u1",
	})
	assert.True(t, errs.ErrorEqual(errs.UserNotExist, bizErr))

	u, bizErr := svc.Register(context.Background(), RegisterParam{
		Account:  "account01",
		Name:     "name0001",
		Password: "password01",
		ClientIP: testClientIP,
	})
	assert.Nil(t, bizErr)

	out, bizErr := svc.GetByUserID(context.Background(), GetByUserIDParam{
		UserID: u.UserID,
	})
	assert.Nil(t, bizErr)
	assert.Equal(t, u.UserID, out.UserID)
}
