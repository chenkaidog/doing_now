package main_test

import (
	"bytes"
	"context"
	be "doing_now/be"
	"doing_now/be/biz/config"
	"doing_now/be/biz/dal/repo"
	"doing_now/be/biz/db/mysql"
	redisdb "doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/model/storage"
	usersvc "doing_now/be/biz/service/user"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const (
	testSessionCookieName = "auth_session_id"
	testRefreshCookieName = "refresh_token"
)

var (
	testServerOnce    sync.Once
	testServer        *server.Hertz
	testServerCleanup func()
	testServerErr     error
)

func TestMain(m *testing.M) {
	testServerOnce.Do(func() {
		testServer, testServerCleanup, testServerErr = startTestServer()
	})
	code := m.Run()
	if testServerCleanup != nil {
		testServerCleanup()
	}
	os.Exit(code)
}

func writeTestConfig(t *testing.T, redisPort string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "deploy.yml")
	content := fmt.Sprintf(`mysql:
  db_name: ""
  ip: "127.0.0.1"
  port: 3306
  username: ""
  password: ""
  slow_threshold: 0
  log_level: 0
redis:
  ip: "127.0.0.1"
  port: %s
  password: ""
  db: 0
jwt:
  issuer: "test"
  access_token_secret: "accesstoken-secret"
  refresh_token_secret: "refreshtoken-secret"
  access_expiration: 3600
  refresh_expiration: 7200
cors:
  allow_origins: ["*"]
  allow_methods: ["GET","POST"]
  allow_headers: ["Origin","Content-Type","Authorization","Cookie"]
  allow_credentials: true
  max_age: 600
session:
  store_prefix: "auth_session:"
  name: "auth_session_id"
  path: "/"
  domain: ""
  max_age: 604800
  secure: false
  http_only: true
  same_site: "Strict"
rate_limit:
  - path: "/api/v1/user/register"
    window_seconds: 1
    limit: 100
    has_session: false
  - path: "/api/v1/user/login"
    window_seconds: 1
    limit: 100
    has_session: false
  - path: "/api/v1/user/info"
    window_seconds: 1
    limit: 100
    has_session: true
  - path: "/api/v1/user/logout"
    window_seconds: 1
    limit: 100
    has_session: true
  - path: "/api/v1/user/refresh_token"
    window_seconds: 1
    limit: 100
    has_session: false
  - path: "/api/v1/user/update_info"
    window_seconds: 1
    limit: 100
    has_session: true
  - path: "/api/v1/user/update_password"
    window_seconds: 1
    limit: 100
    has_session: true
logger:
  level: "debug"
  dir: ""
  file_name: ""
  max_size: 10
  max_backups: 1
  max_age: 1
login_protection:
  window_seconds: 300
  limit: 3
  block_min_duration: 5
  block_hour_duration: 24
  level_duration: 1800
  success_window_seconds: 60
  success_limit: 10
register_protection:
  block_minutes: 10
`, redisPort)
	err := os.WriteFile(p, []byte(content), 0600)
	assert.NoError(t, err)
	return p
}

func writeTestConfigForMainTest(redisPort string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "doing-now-config-*")
	if err != nil {
		return "", nil, err
	}
	p := filepath.Join(dir, "deploy.yml")
	content := fmt.Sprintf(`mysql:
  db_name: ""
  ip: "127.0.0.1"
  port: 3306
  username: ""
  password: ""
  slow_threshold: 0
  log_level: 0
redis:
  ip: "127.0.0.1"
  port: %s
  password: ""
  db: 0
jwt:
  issuer: "test"
  access_token_secret: "accesstoken-secret"
  refresh_token_secret: "refreshtoken-secret"
  access_expiration: 3600
  refresh_expiration: 7200
cors:
  allow_origins: ["*"]
  allow_methods: ["GET","POST"]
  allow_headers: ["Origin","Content-Type","Authorization","Cookie"]
  allow_credentials: true
  max_age: 600
session:
  store_prefix: "auth_session:"
  name: "auth_session_id"
  path: "/"
  domain: ""
  max_age: 604800
  secure: false
  http_only: true
  same_site: "Strict"
rate_limit:
  - path: "/api/v1/user/register"
    window_seconds: 1
    limit: 100
    has_session: false
  - path: "/api/v1/user/login"
    window_seconds: 1
    limit: 100
    has_session: false
  - path: "/api/v1/user/info"
    window_seconds: 1
    limit: 100
    has_session: true
  - path: "/api/v1/user/logout"
    window_seconds: 1
    limit: 100
    has_session: true
  - path: "/api/v1/user/refresh_token"
    window_seconds: 1
    limit: 100
    has_session: false
  - path: "/api/v1/user/update_info"
    window_seconds: 1
    limit: 100
    has_session: true
  - path: "/api/v1/user/update_password"
    window_seconds: 1
    limit: 100
    has_session: true
logger:
  level: "debug"
  dir: ""
  file_name: ""
  max_size: 10
  max_backups: 1
  max_age: 1
login_protection:
  window_seconds: 300
  limit: 3
  block_min_duration: 5
  block_hour_duration: 24
  level_duration: 1800
  success_window_seconds: 60
  success_limit: 10
register_protection:
  block_minutes: 10
`, redisPort)
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	return p, cleanup, nil
}

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&storage.UserRecord{}, &storage.UserCredentialRecord{})
	assert.NoError(t, err)
	return db
}

func newSQLiteDBForMainTest() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&storage.UserRecord{}, &storage.UserCredentialRecord{}); err != nil {
		return nil, err
	}
	return db, nil
}

func startTestServer() (*server.Hertz, func(), error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, nil, err
	}
	configPath, cleanupConfig, err := writeTestConfigForMainTest(mr.Port())
	if err != nil {
		mr.Close()
		return nil, nil, err
	}
	config.Init(configPath)

	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:" + mr.Port(),
		Password: "",
		DB:       0,
	})
	db, err := newSQLiteDBForMainTest()
	if err != nil {
		_ = rdb.Close()
		cleanupConfig()
		mr.Close()
		return nil, nil, err
	}

	mockey.Mock(redisdb.GetRedisClient).To(func() *redis.Client {
		return rdb
	}).Build()
	mockey.Mock(mysql.GetDbConn).To(func() *gorm.DB {
		return db
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

	h := be.NewEngine()
	cleanup := func() {
		mockey.UnPatchAll()
		_ = rdb.Close()
		cleanupConfig()
		mr.Close()
	}
	return h, cleanup, nil
}

func newTestServer(t *testing.T) (*server.Hertz, func()) {
	t.Helper()
	if testServerErr != nil {
		t.Fatalf("start test server err: %v", testServerErr)
	}
	if testServer == nil {
		t.Fatalf("test server not initialized")
	}
	return testServer, func() {}
}

func perform(h *server.Hertz, method, path, body string, headers ...ut.Header) *ut.ResponseRecorder {
	var reqBody *ut.Body
	if body != "" {
		reqBody = &ut.Body{
			Body: bytes.NewBufferString(body),
			Len:  len(body),
		}
	}
	return ut.PerformRequest(h.Engine, method, path, reqBody, headers...)
}

func decodeCommonResp(t *testing.T, body []byte) dto.CommonResp {
	t.Helper()
	var resp dto.CommonResp
	err := json.Unmarshal(body, &resp)
	assert.NoError(t, err)
	return resp
}

func extractCookies(rr *ut.ResponseRecorder) map[string]string {
	cookies := make(map[string]string)
	rr.Header().VisitAllCookie(func(key, value []byte) {
		cookies[string(key)] = string(value)
	})
	return cookies
}

func mustCookieHeader(t *testing.T, cookies map[string]string, names ...string) string {
	t.Helper()
	parts := make([]string, 0, len(names))
	for _, name := range names {
		v, ok := cookies[name]
		assert.True(t, ok)
		assert.NotEmpty(t, v)
		if strings.HasPrefix(v, name+"=") {
			v = strings.TrimPrefix(v, name+"=")
		}
		parts = append(parts, name+"="+v)
	}
	return strings.Join(parts, "; ")
}

func cookieValue(cookieHeader string, name string) string {
	parts := strings.Split(cookieHeader, ";")
	prefix := name + "="
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func mustCreateUserViaService(t *testing.T, account, name, password string) {
	t.Helper()
	_, bizErr := usersvc.NewDefault().Register(context.Background(), usersvc.RegisterParam{
		Account:  account,
		Name:     name,
		Password: password,
		ClientIP: "svc_" + account,
	})
	assert.Nil(t, bizErr)
}

func mustLogin(t *testing.T, h *server.Hertz, account, password, ip string) (string, string, string) {
	t.Helper()
	rr := perform(
		h,
		http.MethodPost,
		"/api/v1/user/login",
		fmt.Sprintf(`{"account":"%s","password":"%s"}`, account, password),
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-Forwarded-For", Value: ip},
	)
	assert.Equal(t, http.StatusOK, rr.Code)
	resp := decodeCommonResp(t, rr.Body.Bytes())
	assert.True(t, resp.Success)
	data, ok := resp.Data.(map[string]any)
	assert.True(t, ok)
	accessToken, ok := data["access_token"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, accessToken)
	cookies := extractCookies(rr)
	sessionCookieHeader := mustCookieHeader(t, cookies, testSessionCookieName)
	allCookieHeader := mustCookieHeader(t, cookies, testSessionCookieName, testRefreshCookieName)
	return accessToken, sessionCookieHeader, allCookieHeader
}

func TestUserRegister(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()

	t.Run("param error", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/register",
			`{"account":"a","name":"b","password":"c"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
		)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.ParamError.Code()), resp.Code)
	})

	t.Run("success and block by ip", func(t *testing.T) {
		ip := "127.0.0.1"
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/register",
			`{"account":"account_reg_01","name":"name_reg_01","password":"password01"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "X-Forwarded-For", Value: ip},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)

		rr2 := perform(
			h,
			http.MethodPost,
			"/api/v1/user/register",
			`{"account":"account_reg_02","name":"name_reg_02","password":"password02"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "X-Forwarded-For", Value: ip},
		)
		assert.Equal(t, http.StatusOK, rr2.Code)
		resp2 := decodeCommonResp(t, rr2.Body.Bytes())
		assert.False(t, resp2.Success)
		assert.Equal(t, int(errs.RequestBlocked.Code()), resp2.Code)
	})

	t.Run("duplicate account", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/register",
			`{"account":"account_dup_01","name":"name_dup_01","password":"password01"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "X-Forwarded-For", Value: "10.0.0.2"},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)

		rr2 := perform(
			h,
			http.MethodPost,
			"/api/v1/user/register",
			`{"account":"account_dup_01","name":"name_dup_02","password":"password01"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "X-Forwarded-For", Value: "10.0.0.3"},
		)
		assert.Equal(t, http.StatusOK, rr2.Code)
		resp2 := decodeCommonResp(t, rr2.Body.Bytes())
		assert.False(t, resp2.Success)
		assert.Equal(t, int(errs.UserNameDuplicatedErr.Code()), resp2.Code)
	})
}

func TestUserLogin(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_login_01", "name_login_01", "password01")

	t.Run("param error", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/login",
			`{"account":"a","password":"b"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
		)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.ParamError.Code()), resp.Code)
	})

	t.Run("wrong password", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/login",
			`{"account":"account_login_01","password":"badpassword"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "X-Forwarded-For", Value: "10.0.1.1"},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.PasswordIncorrect.Code()), resp.Code)
	})

	t.Run("success", func(t *testing.T) {
		accessToken, sessionCookieHeader, allCookieHeader := mustLogin(t, h, "account_login_01", "password01", "10.0.1.2")
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, sessionCookieHeader)
		assert.NotEmpty(t, allCookieHeader)
	})

	t.Run("block after repeated failures", func(t *testing.T) {
		ip := "10.0.1.3"
		for i := 0; i < 3; i++ {
			_ = perform(
				h,
				http.MethodPost,
				"/api/v1/user/login",
				`{"account":"account_login_01","password":"badpassword"}`,
				ut.Header{Key: "Content-Type", Value: "application/json"},
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
		}
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/login",
			`{"account":"account_login_01","password":"password01"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "X-Forwarded-For", Value: ip},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.RequestBlocked.Code()), resp.Code)
	})
}

func TestRefreshToken(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_refresh_01", "name_refresh_01", "password01")
	accessToken, sessionCookieHeader, allCookieHeader := mustLogin(t, h, "account_refresh_01", "password01", "10.0.2.1")
	assert.NotEmpty(t, accessToken)

	t.Run("missing refresh cookie", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/refresh_token",
			`{}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("invalid refresh cookie", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/refresh_token",
			`{}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader + "; " + testRefreshCookieName + "=invalid"},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("success", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/refresh_token",
			`{}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Cookie", Value: allCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)
		data, ok := resp.Data.(map[string]any)
		assert.True(t, ok)
		_, ok = data["access_token"].(string)
		assert.True(t, ok)
	})
}

func TestGetUserInfo(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_info_01", "name_info_01", "password01")
	accessToken, sessionCookieHeader, _ := mustLogin(t, h, "account_info_01", "password01", "10.0.3.1")

	t.Run("unauthorized without token", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("success", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)
	})
}

func TestUpdateInfo(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_update_info_01", "name_info_old", "password01")
	accessToken, sessionCookieHeader, _ := mustLogin(t, h, "account_update_info_01", "password01", "10.0.4.1")

	t.Run("unauthorized", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_info",
			`{"name":"name_info_new"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("param error", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_info",
			`{"name":"a"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("success and verify info", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_info",
			`{"name":"name_info_new"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)

		infoRR := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, infoRR.Code)
		infoResp := decodeCommonResp(t, infoRR.Body.Bytes())
		assert.True(t, infoResp.Success)
		data, ok := infoResp.Data.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "name_info_new", data["name"])
	})
}

func TestUpdatePassword(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_update_pwd_01", "name_pwd_01", "password01")
	accessToken, sessionCookieHeader, _ := mustLogin(t, h, "account_update_pwd_01", "password01", "10.0.5.1")

	t.Run("param error", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_password",
			`{"old_password":"a","new_password":"b"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("wrong old password", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_password",
			`{"old_password":"badpassword","new_password":"password02"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.PasswordIncorrect.Code()), resp.Code)
	})

	t.Run("success then old session expired", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_password",
			`{"old_password":"password01","new_password":"password02"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)

		infoRR := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusForbidden, infoRR.Code)
		infoResp := decodeCommonResp(t, infoRR.Body.Bytes())
		assert.False(t, infoResp.Success)
		assert.Equal(t, int(errs.SessionExpired.Code()), infoResp.Code)
	})

	t.Run("relogin with new password", func(t *testing.T) {
		newAccessToken, newSessionCookieHeader, _ := mustLogin(t, h, "account_update_pwd_01", "password02", "10.0.5.2")
		infoRR := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: newAccessToken},
			ut.Header{Key: "Cookie", Value: newSessionCookieHeader},
		)
		assert.Equal(t, http.StatusOK, infoRR.Code)
		infoResp := decodeCommonResp(t, infoRR.Body.Bytes())
		assert.True(t, infoResp.Success)
	})
}

func TestUserCrossConsistency(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_cross_01", "name_cross_01", "password01")
	mustCreateUserViaService(t, "account_cross_02", "name_cross_02", "password01")

	accessToken1, sessionCookieHeader1, allCookieHeader1 := mustLogin(t, h, "account_cross_01", "password01", "10.0.7.1")
	_, sessionCookieHeader2, allCookieHeader2 := mustLogin(t, h, "account_cross_02", "password01", "10.0.7.2")

	t.Run("user2 session with user1 token get info unauthorized", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken1},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader2},
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("user2 session with user1 token update password unauthorized", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/update_password",
			`{"old_password":"password01","new_password":"password02"}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken1},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader2},
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("user2 session with user1 token logout unauthorized", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/logout",
			`{}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken1},
			ut.Header{Key: "Cookie", Value: allCookieHeader2},
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("user2 session with user1 refresh token refresh unauthorized", func(t *testing.T) {
		refreshToken1 := cookieValue(allCookieHeader1, testRefreshCookieName)
		assert.NotEmpty(t, refreshToken1)
		mixedCookie := sessionCookieHeader2 + "; " + testRefreshCookieName + "=" + refreshToken1
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/refresh_token",
			`{}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Cookie", Value: mixedCookie},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("user1 token without session cookie unauthorized", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken1},
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})

	t.Run("user1 token with user1 session still valid", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken1},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader1},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)
	})
}

func TestLogout(t *testing.T) {
	h, cleanup := newTestServer(t)
	defer cleanup()
	mustCreateUserViaService(t, "account_logout_01", "name_logout_01", "password01")
	accessToken, sessionCookieHeader, allCookieHeader := mustLogin(t, h, "account_logout_01", "password01", "10.0.6.1")

	t.Run("success", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodPost,
			"/api/v1/user/logout",
			`{}`,
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: allCookieHeader},
		)
		assert.Equal(t, http.StatusOK, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.True(t, resp.Success)
	})

	t.Run("old token becomes unauthorized", func(t *testing.T) {
		rr := perform(
			h,
			http.MethodGet,
			"/api/v1/user/info",
			"",
			ut.Header{Key: "Authorization", Value: accessToken},
			ut.Header{Key: "Cookie", Value: sessionCookieHeader},
		)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		resp := decodeCommonResp(t, rr.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, int(errs.Unauthorized.Code()), resp.Code)
	})
}
