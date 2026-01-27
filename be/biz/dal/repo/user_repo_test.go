package repo

import (
	"context"
	"testing"

	"doing_now/be/biz/model/storage"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&storage.UserRecord{}, &storage.UserCredentialRecord{})
	assert.NoError(t, err)
	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)
	ctx := context.Background()

	u := &storage.UserRecord{
		Account: "test_account",
		Name:    "test_name",
	}

	created, err := r.Create(ctx, u)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.UserId)
	assert.Equal(t, u.Account, created.Account)

	// Verify in DB
	var m storage.UserRecord
	err = db.First(&m, "user_id = ?", created.UserId).Error
	assert.NoError(t, err)
	assert.Equal(t, u.Account, m.Account)
}

func TestUserRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)
	ctx := context.Background()

	u := &storage.UserRecord{
		UserId:  "test_user_id",
		Account: "test_account",
		Name:    "test_name",
	}
	db.Create(u)

	// Test found
	found, err := r.FindByUserID(ctx, "test_user_id")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, u.UserId, found.UserId)

	// Test not found
	found, err = r.FindByUserID(ctx, "non_existent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestUserRepository_FindByAccount(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)
	ctx := context.Background()

	u := &storage.UserRecord{
		UserId:  "test_user_id",
		Account: "test_account",
		Name:    "test_name",
	}
	db.Create(u)

	// Test found
	found, err := r.FindByAccount(ctx, "test_account")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, u.Account, found.Account)

	// Test not found
	found, err = r.FindByAccount(ctx, "non_existent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)
	ctx := context.Background()

	u := &storage.UserRecord{
		UserId:  "test_user_id",
		Account: "test_account",
		Name:    "test_name",
	}
	db.Create(u)

	u.Name = "updated_name"
	err := r.Update(ctx, u)
	assert.NoError(t, err)

	// Verify in DB
	var m storage.UserRecord
	err = db.First(&m, "user_id = ?", u.UserId).Error
	assert.NoError(t, err)
	assert.Equal(t, "updated_name", m.Name)
}

func TestUserCredentialRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserCredentialRepository(db)
	ctx := context.Background()

	c := &storage.UserCredentialRecord{
		UserId:            "test_user_id",
		PasswordSalt:      "salt",
		PasswordHash:      "hash",
		CredentialVersion: 1,
	}

	err := r.Create(ctx, c)
	assert.NoError(t, err)

	// Verify in DB
	var m storage.UserCredentialRecord
	err = db.First(&m, "user_id = ?", c.UserId).Error
	assert.NoError(t, err)
	assert.Equal(t, c.PasswordHash, m.PasswordHash)
}

func TestUserCredentialRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserCredentialRepository(db)
	ctx := context.Background()

	c := &storage.UserCredentialRecord{
		UserId:       "test_user_id",
		PasswordSalt: "salt",
		PasswordHash: "hash",
	}
	db.Create(c)

	// Test found
	found, err := r.FindByUserID(ctx, "test_user_id")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, c.UserId, found.UserId)

	// Test not found
	found, err = r.FindByUserID(ctx, "non_existent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}
