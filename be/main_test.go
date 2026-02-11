package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	be "doing_now/be"
	"doing_now/be/biz/config"
	"doing_now/be/biz/dal/repo"
	"doing_now/be/biz/db/mysql"
	redisdb "doing_now/be/biz/db/redis"
	jwtmw "doing_now/be/biz/middleware/jwt"
	"doing_now/be/biz/model/domain"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/model/storage"
	usersvc "doing_now/be/biz/service/user"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/glebarez/sqlite"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var testEngine *server.Hertz
var baseConfPath string
var baseConfContent string

func TestMain(t *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	dir, err := os.MkdirTemp("", "doing_now_test_conf_*")
	if err != nil {
		panic(err)
	}
	confPath := filepath.Join(dir, "deploy.yml")
	confStr := `mysql:
  db_name: ""
  ip: "127.0.0.1"
  port: 3306
  username: ""
  password: ""

redis:
  ip: "` + mr.Host() + `"
  port: ` + mr.Port() + `
  password: ""
  db: 0

jwt:
  access_expiration: 3600
  refresh_expiration: 7200
  access_token_secret: "accesstoken-secret"
  refresh_token_secret: "refreshtoken-secret"
  issuer: "test"

cors:
  allow_origins:
    - "*"
  allow_methods:
    - "GET"
  allow_headers:
    - "Origin"
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
`
	conf := []byte(confStr)
	if err := os.WriteFile(confPath, conf, 0600); err != nil {
		panic(err)
	}
	baseConfPath = confPath
	baseConfContent = confStr
	config.Init(baseConfPath)
	redisdb.Init()

	testEngine = be.NewEngine()
	os.Exit(t.Run())
}

func newTestServer(t *testing.T) *server.Hertz {
	t.Helper()
	redisdb.GetRedisClient().FlushAll(context.Background())
	return testEngine
}

func perform(h *server.Hertz, method, url string, body string, headers ...ut.Header) *ut.ResponseRecorder {
	var b *ut.Body
	if body != "" {
		b = &ut.Body{Body: bytes.NewBufferString(body), Len: len(body)}
	}
	allHeaders := append([]ut.Header{{Key: "Content-Type", Value: "application/json"}}, headers...)
	return ut.PerformRequest(h.Engine, method, url, b, allHeaders...)
}

func decodeCommonResp(t *testing.T, respBody []byte) dto.CommonResp {
	t.Helper()
	var r dto.CommonResp
	err := json.Unmarshal(respBody, &r)
	assert.Nil(t, err)
	return r
}

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.Nil(t, err)

	sqlDB, err := db.DB()
	assert.Nil(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(&storage.UserRecord{}, &storage.UserCredentialRecord{})
	assert.Nil(t, err)
	return db
}

func patchSQLiteMySQLConn(t *testing.T, db *gorm.DB) {
	t.Helper()
	mockey.Mock(mysql.GetDbConn).Return(db).Build()

	mockey.Mock((*repo.UserRepository).FindByAccountLock).To(func(r *repo.UserRepository, ctx context.Context, account string) (*storage.UserRecord, error) {
		return r.FindByAccount(ctx, account)
	}).Build()
	mockey.Mock((*repo.UserRepository).FindByUserIDLock).To(func(r *repo.UserRepository, ctx context.Context, userID string) (*storage.UserRecord, error) {
		return r.FindByUserID(ctx, userID)
	}).Build()
	mockey.Mock((*repo.UserCredentialRepository).FindByUserIDLock).To(func(r *repo.UserCredentialRepository, ctx context.Context, userID string) (*storage.UserCredentialRecord, error) {
		return r.FindByUserID(ctx, userID)
	}).Build()
}

func cookiesFromRecorder(t *testing.T, rr *ut.ResponseRecorder) map[string]string {
	t.Helper()

	cookies := map[string]string{}

	getCookieValue := func(name string) (string, bool) {
		c := protocol.AcquireCookie()
		defer protocol.ReleaseCookie(c)
		c.SetKey(name)
		if !rr.Header().Cookie(c) {
			return "", false
		}
		return string(c.Value()), true
	}

	sessCookieName := config.GetSessionConf().Name
	if sessCookieName == "" {
		sessCookieName = "auth_session_id"
	}
	if v, ok := getCookieValue(sessCookieName); ok {
		cookies[sessCookieName] = v
	}
	if v, ok := getCookieValue("refresh_token"); ok {
		cookies["refresh_token"] = v
	}
	return cookies
}

func cookieHeaderFromRecorder(t *testing.T, rr *ut.ResponseRecorder) string {
	t.Helper()

	cookies := cookiesFromRecorder(t, rr)
	keys := make([]string, 0, len(cookies))
	for k := range cookies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+cookies[k])
	}
	return strings.Join(out, "; ")
}

func mergeCookieHeader(existing string, add string) string {
	if strings.TrimSpace(existing) == "" {
		return add
	}
	if strings.TrimSpace(add) == "" {
		return existing
	}
	return existing + "; " + add
}

func setCookie(existing, name, value string) string {
	cookies := map[string]string{}
	for _, part := range strings.Split(existing, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		cookies[kv[0]] = kv[1]
	}
	cookies[name] = value

	out := make([]string, 0, len(cookies))
	for k, v := range cookies {
		out = append(out, k+"="+v)
	}
	return strings.Join(out, "; ")
}

func parseAccessToken(t *testing.T, rr *ut.ResponseRecorder) string {
	t.Helper()
	resp := decodeCommonResp(t, rr.Body.Bytes())
	assert.True(t, resp.Success)
	data, ok := resp.Data.(map[string]any)
	assert.True(t, ok)
	at, ok := data["access_token"].(string)
	assert.True(t, ok)
	assert.True(t, at != "")
	return at
}

func parseAccessClaims(t *testing.T, accessToken string) *jwtmw.Claims {
	t.Helper()

	conf := config.GetJWTConfig()
	claims := &jwtmw.Claims{}
	_, err := jwtlib.ParseWithClaims(accessToken, claims, func(token *jwtlib.Token) (any, error) {
		return []byte(conf.AccessTokenSecret), nil
	})
	assert.Nil(t, err)
	return claims
}

func loginAndGetAuth(t *testing.T, h *server.Hertz, ip, account, name, password string) (string, string) {
	t.Helper()

	body := `{"account":"` + account + `","password":"` + password + `"}`
	rr := perform(h, http.MethodPost, "/api/v1/user/login", body, ut.Header{Key: "X-Forwarded-For", Value: ip})
	assert.DeepEqual(t, http.StatusOK, rr.Code)
	r := decodeCommonResp(t, rr.Body.Bytes())
	assert.True(t, r.Success)

	accessToken := parseAccessToken(t, rr)
	claims := parseAccessClaims(t, accessToken)

	cookieMap := cookiesFromRecorder(t, rr)
	sessCookieName := config.GetSessionConf().Name
	if sessCookieName == "" {
		sessCookieName = "auth_session_id"
	}
	sessID := cookieMap[sessCookieName]

	if sessID == "" || !claims.CheckSum(sessID) {
		storePrefix := config.GetSessionConf().StorePrefix
		if storePrefix == "" {
			storePrefix = "auth_session:"
		}
		keys, err := redisdb.GetRedisClient().Keys(context.Background(), storePrefix+"*").Result()
		assert.Nil(t, err)

		for _, k := range keys {
			candidate := strings.TrimPrefix(k, storePrefix)
			if candidate != "" && claims.CheckSum(candidate) {
				sessID = candidate
				break
			}
		}
	}

	assert.True(t, sessID != "")
	assert.True(t, claims.CheckSum(sessID))

	cookieHeader := sessCookieName + "=" + sessID
	if rt, ok := cookieMap["refresh_token"]; ok && rt != "" {
		cookieHeader = cookieHeader + "; refresh_token=" + rt
	}
	return accessToken, cookieHeader
}

func deleteAccessTokenExistKey(t *testing.T, accessToken string) {
	t.Helper()

	claims := parseAccessClaims(t, accessToken)
	assert.True(t, claims.ID != "")

	err := redisdb.GetRedisClient().Del(context.Background(), "jwt_id_exist:"+claims.ID).Err()
	assert.Nil(t, err)
}

func mustCreateUserViaService(t *testing.T, account, name, password string) *domain.User {
	t.Helper()
	u, bizErr := usersvc.NewDefault().Register(context.Background(), account, name, password)
	assert.Nil(t, bizErr)
	assert.True(t, u != nil)
	return u
}

func TestUserRegister(t *testing.T) {
	mockey.PatchConvey("POST /api/v1/user/register", t, func() {
		h := newTestServer(t)
		db := newSQLiteDB(t)
		patchSQLiteMySQLConn(t, db)

		ip := "127.0.0.1"

		t.Run("handler拦截: BindAndValidate失败返回400", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			rr := perform(h, http.MethodPost, "/api/v1/user/register", `{"account":"a"}`, ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusBadRequest, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.ParamError.Code()), resp.Code)

			exists, err := redisdb.GetRedisClient().Exists(context.Background(), "rate_limit:register_block:"+ip).Result()
			assert.Nil(t, err)
			assert.DeepEqual(t, int64(0), exists)
		})

		t.Run("正常: 注册成功后RegisterProtection按IP写入block key，随后请求被中间件拦截", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			account := "account01"
			name := "name0001"
			password := "password01"
			rr := perform(h, http.MethodPost, "/api/v1/user/register",
				`{"account":"`+account+`","name":"`+name+`","password":"`+password+`"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			exists, err := redisdb.GetRedisClient().Exists(context.Background(), "rate_limit:register_block:"+ip).Result()
			assert.Nil(t, err)
			assert.DeepEqual(t, int64(1), exists)

			rr2 := perform(h, http.MethodPost, "/api/v1/user/register",
				`{"account":"account02","name":"name0002","password":"password02"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusForbidden, rr2.Code)
			resp2 := decodeCommonResp(t, rr2.Body.Bytes())
			assert.False(t, resp2.Success)
			assert.DeepEqual(t, int(errs.RequestBlocked.Code()), resp2.Code)
		})

		t.Run("业务错误: 重复账号注册返回UserNameDuplicatedErr", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			ip2 := "127.0.0.2"
			ip3 := "127.0.0.3"
			account := "account_dup01"
			name := "name_dup01"
			password := "password01"

			rr := perform(h, http.MethodPost, "/api/v1/user/register",
				`{"account":"`+account+`","name":"`+name+`","password":"`+password+`"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip2},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			rr2 := perform(h, http.MethodPost, "/api/v1/user/register",
				`{"account":"`+account+`","name":"name_dup02","password":"password02"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip3},
			)
			assert.DeepEqual(t, http.StatusOK, rr2.Code)
			resp2 := decodeCommonResp(t, rr2.Body.Bytes())
			assert.False(t, resp2.Success)
			assert.DeepEqual(t, int(errs.UserNameDuplicatedErr.Code()), resp2.Code)
		})
	})
}

func TestUserLogin(t *testing.T) {
	mockey.PatchConvey("POST /api/v1/user/login", t, func() {
		h := newTestServer(t)
		db := newSQLiteDB(t)
		patchSQLiteMySQLConn(t, db)

		ip := "127.0.0.1"

		t.Run("LoginSuccessRecorder拦截: 参数校验失败返回400", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			rr := perform(h, http.MethodPost, "/api/v1/user/login", `{}`, ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusBadRequest, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.ParamError.Code()), resp.Code)
		})

		t.Run("LoginSuccessRecorder拦截: 达到成功次数限制返回403", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			account := "account_limit10"
			name := "name_limit10"
			password := "password10"
			mustCreateUserViaService(t, account, name, password)

			err := redisdb.GetRedisClient().Set(context.Background(), "rate_limit:login_success:"+account, "9", 0).Err()
			assert.Nil(t, err)

			rr := perform(h, http.MethodPost, "/api/v1/user/login",
				`{"account":"`+account+`","password":"`+password+`"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusForbidden, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.LoginReachLimit.Code()), resp.Code)
		})

		t.Run("LoginProtection后置逻辑: 连续密码错误触发block key，随后请求被中间件前置拦截", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			account := "account_fail10"
			name := "name_fail10"
			password := "password10"
			mustCreateUserViaService(t, account, name, password)

			for i := 0; i < 3; i++ {
				rr := perform(h, http.MethodPost, "/api/v1/user/login",
					`{"account":"`+account+`","password":"badpassword"}`,
					ut.Header{Key: "X-Forwarded-For", Value: ip},
				)
				assert.DeepEqual(t, http.StatusOK, rr.Code)
				resp := decodeCommonResp(t, rr.Body.Bytes())
				assert.False(t, resp.Success)
				assert.DeepEqual(t, int(errs.PasswordIncorrect.Code()), resp.Code)
			}

			rr := perform(h, http.MethodPost, "/api/v1/user/login",
				`{"account":"`+account+`","password":"badpassword"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusForbidden, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.RequestBlocked.Code()), resp.Code)
		})

		t.Run("LoginProtection前置逻辑: 小时block key存在返回403", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			err := redisdb.GetRedisClient().Set(context.Background(), "rate_limit:login_block_h:"+ip, "1", 0).Err()
			assert.Nil(t, err)

			rr := perform(h, http.MethodPost, "/api/v1/user/login", `{}`, ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusForbidden, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.RequestBlocked.Code()), resp.Code)
		})

		t.Run("LoginProtection前置逻辑: 分钟block key存在返回403", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			err := redisdb.GetRedisClient().Set(context.Background(), "rate_limit:login_block_m:"+ip, "1", 0).Err()
			assert.Nil(t, err)

			rr := perform(h, http.MethodPost, "/api/v1/user/login", `{}`, ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusForbidden, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.RequestBlocked.Code()), resp.Code)
		})

		t.Run("LoginProtection后置逻辑: 非账户类失败不计入login_fail", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			patch := mockey.Mock((*usersvc.Service).Login).
				Return((*domain.User)(nil), uint(0), errs.ServerError.SetMsg("mock server error")).
				Build()
			defer patch.UnPatch()

			rr := perform(h, http.MethodPost, "/api/v1/user/login",
				`{"account":"account_server","password":"password11"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.ServerError.Code()), resp.Code)

			exists, err := redisdb.GetRedisClient().Exists(context.Background(), "rate_limit:login_fail:"+ip).Result()
			assert.Nil(t, err)
			assert.DeepEqual(t, int64(0), exists)
		})

		t.Run("LoginProtection后置逻辑: Level2触发小时block key", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			account := "account_lvl2_10"
			name := "name_lvl2_10"
			password := "password10"
			mustCreateUserViaService(t, account, name, password)

			err := redisdb.GetRedisClient().Set(context.Background(), "rate_limit:login_fail:"+ip, "2", 0).Err()
			assert.Nil(t, err)
			err = redisdb.GetRedisClient().Set(context.Background(), "login_fail_level:"+ip, "1", 0).Err()
			assert.Nil(t, err)

			rr := perform(h, http.MethodPost, "/api/v1/user/login",
				`{"account":"`+account+`","password":"badpassword"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.PasswordIncorrect.Code()), resp.Code)

			exists, err := redisdb.GetRedisClient().Exists(context.Background(), "rate_limit:login_block_h:"+ip).Result()
			assert.Nil(t, err)
			assert.DeepEqual(t, int64(1), exists)

			rr2 := perform(h, http.MethodPost, "/api/v1/user/login",
				`{"account":"`+account+`","password":"badpassword"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusForbidden, rr2.Code)
			resp2 := decodeCommonResp(t, rr2.Body.Bytes())
			assert.False(t, resp2.Success)
			assert.DeepEqual(t, int(errs.RequestBlocked.Code()), resp2.Code)
		})

		t.Run("正常: 登录成功返回access_token并下发session与refresh_token cookie", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			account := "account_ok10"
			name := "name_ok10"
			password := "password10"
			mustCreateUserViaService(t, account, name, password)

			rr := perform(h, http.MethodPost, "/api/v1/user/login",
				`{"account":"`+account+`","password":"`+password+`"}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			at := parseAccessToken(t, rr)
			assert.True(t, at != "")

			cookies := cookiesFromRecorder(t, rr)
			assert.True(t, cookies["auth_session_id"] != "")
			assert.True(t, cookies["refresh_token"] != "")

			count, err := redisdb.GetRedisClient().Get(context.Background(), "rate_limit:login_success:"+account).Int64()
			assert.Nil(t, err)
			assert.True(t, count >= 1)
		})
	})
}

func TestPing(t *testing.T) {
	h := newTestServer(t)
	rr := perform(h, http.MethodGet, "/ping", "")
	assert.DeepEqual(t, http.StatusOK, rr.Code)

	var out map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &out)
	assert.Nil(t, err)
	assert.DeepEqual(t, "pong", out["message"])
}

func TestRefreshToken(t *testing.T) {
	mockey.PatchConvey("POST /api/v1/user/refresh_token", t, func() {
		h := newTestServer(t)
		db := newSQLiteDB(t)
		patchSQLiteMySQLConn(t, db)

		ip := "127.0.0.1"
		account := "account20"
		name := "name0020"
		password := "password20"
		mustCreateUserViaService(t, account, name, password)

		t.Run("handler拦截: 非法JSON返回400", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			rr := perform(h, http.MethodPost, "/api/v1/user/refresh_token", `{"x":`, ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusBadRequest, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.ParamError.Code()), resp.Code)
		})

		t.Run("handler拦截: 缺少refresh_token cookie返回Unauthorized", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			rr := perform(h, http.MethodPost, "/api/v1/user/refresh_token", `{}`, ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.Unauthorized.Code()), resp.Code)
		})

		t.Run("handler拦截: refresh_token无效返回Unauthorized", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			_, cookieHeader := loginAndGetAuth(t, h, ip, account, name, password)
			cookieHeader = setCookie(cookieHeader, "refresh_token", "bad")

			rr := perform(h, http.MethodPost, "/api/v1/user/refresh_token", `{}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
				ut.Header{Key: "Cookie", Value: cookieHeader},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.Unauthorized.Code()), resp.Code)
		})

		t.Run("正常: 使用refresh_token刷新并下发新token与refresh cookie", func(t *testing.T) {
			redisdb.GetRedisClient().FlushAll(context.Background())
			_, cookieHeader := loginAndGetAuth(t, h, ip, account, name, password)
			rr := perform(h, http.MethodPost, "/api/v1/user/refresh_token", `{}`,
				ut.Header{Key: "X-Forwarded-For", Value: ip},
				ut.Header{Key: "Cookie", Value: cookieHeader},
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			cookies := cookiesFromRecorder(t, rr)
			assert.True(t, cookies["refresh_token"] != "")
		})
	})
}

func TestAuthorizedEndpoints(t *testing.T) {
	mockey.PatchConvey("JWT + CredentialCheck保护的接口", t, func() {
		h := newTestServer(t)
		db := newSQLiteDB(t)
		patchSQLiteMySQLConn(t, db)

		ip := "127.0.0.1"
		account := "account30"
		name := "name0030"
		password := "password30"
		mustCreateUserViaService(t, account, name, password)

		accessToken, cookieHeader := loginAndGetAuth(t, h, ip, account, name, password)

		authHeaders := func(token, cookies string) []ut.Header {
			return []ut.Header{
				{Key: "X-Forwarded-For", Value: ip},
				{Key: "Authorization", Value: token},
				{Key: "Cookie", Value: cookies},
			}
		}

		t.Run("jwt.ValidateMW拦截: Authorization为空返回401", func(t *testing.T) {
			rr := perform(h, http.MethodGet, "/api/v1/user/info", "", ut.Header{Key: "X-Forwarded-For", Value: ip})
			assert.DeepEqual(t, http.StatusUnauthorized, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.Unauthorized.Code()), resp.Code)
		})

		t.Run("jwt.ValidateMW拦截: token存在性校验失败返回401", func(t *testing.T) {
			deleteAccessTokenExistKey(t, accessToken)
			rr := perform(h, http.MethodGet, "/api/v1/user/info", "", authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusUnauthorized, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.Unauthorized.Code()), resp.Code)
		})

		t.Run("正常: 获取用户信息返回200", func(t *testing.T) {
			accessToken, cookieHeader = loginAndGetAuth(t, h, ip, account, name, password)
			rr := perform(h, http.MethodGet, "/api/v1/user/info", "", authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)
			data, ok := resp.Data.(map[string]any)
			assert.True(t, ok)
			assert.DeepEqual(t, account, data["account"])
			assert.DeepEqual(t, name, data["name"])
		})

		t.Run("UpdateInfo handler拦截: 参数校验失败返回400", func(t *testing.T) {
			accessToken, cookieHeader = loginAndGetAuth(t, h, ip, account, name, password)
			rr := perform(h, http.MethodPost, "/api/v1/user/update_info", `{"name":"a"}`, authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusBadRequest, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.ParamError.Code()), resp.Code)
		})

		t.Run("正常: UpdateInfo成功后再次/info能读到新name", func(t *testing.T) {
			accessToken, cookieHeader = loginAndGetAuth(t, h, ip, account, name, password)
			newName := "name_new30"
			rr := perform(h, http.MethodPost, "/api/v1/user/update_info", `{"name":"`+newName+`"}`, authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			rr2 := perform(h, http.MethodGet, "/api/v1/user/info", "", authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusOK, rr2.Code)
			resp2 := decodeCommonResp(t, rr2.Body.Bytes())
			assert.True(t, resp2.Success)
			data2, ok := resp2.Data.(map[string]any)
			assert.True(t, ok)
			assert.DeepEqual(t, newName, data2["name"])
		})

		t.Run("UpdatePassword handler拦截: 旧密码错误返回业务错误", func(t *testing.T) {
			accessToken, cookieHeader = loginAndGetAuth(t, h, ip, account, name, password)
			rr := perform(h, http.MethodPost, "/api/v1/user/update_password",
				`{"old_password":"badpassword","new_password":"password31"}`,
				authHeaders(accessToken, cookieHeader)...,
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.False(t, resp.Success)
			assert.DeepEqual(t, int(errs.PasswordIncorrect.Code()), resp.Code)
		})

		t.Run("串联: 改密成功后同一session访问/info被CredentialCheck拦截，重新登录后恢复", func(t *testing.T) {
			accessToken, cookieHeader = loginAndGetAuth(t, h, ip, account, name, password)

			rr := perform(h, http.MethodPost, "/api/v1/user/update_password",
				`{"old_password":"`+password+`","new_password":"password31"}`,
				authHeaders(accessToken, cookieHeader)...,
			)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			rr2 := perform(h, http.MethodGet, "/api/v1/user/info", "", authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusForbidden, rr2.Code)
			resp2 := decodeCommonResp(t, rr2.Body.Bytes())
			assert.False(t, resp2.Success)
			assert.DeepEqual(t, int(errs.SessionExpired.Code()), resp2.Code)

			accessToken2, cookieHeader2 := loginAndGetAuth(t, h, ip, account, name, "password31")
			rr3 := perform(h, http.MethodGet, "/api/v1/user/info", "", authHeaders(accessToken2, cookieHeader2)...)
			assert.DeepEqual(t, http.StatusOK, rr3.Code)
			resp3 := decodeCommonResp(t, rr3.Body.Bytes())
			assert.True(t, resp3.Success)
		})

		t.Run("Logout正常: 登出成功并清理refresh_token cookie", func(t *testing.T) {
			logoutAccount := "account_logout30"
			logoutName := "name_logout30"
			logoutPassword := "password_logout30"
			mustCreateUserViaService(t, logoutAccount, logoutName, logoutPassword)
			accessToken, cookieHeader = loginAndGetAuth(t, h, ip, logoutAccount, logoutName, logoutPassword)
			rr := perform(h, http.MethodPost, "/api/v1/user/logout", `{}`, authHeaders(accessToken, cookieHeader)...)
			assert.DeepEqual(t, http.StatusOK, rr.Code)
			resp := decodeCommonResp(t, rr.Body.Bytes())
			assert.True(t, resp.Success)

			cookies := cookiesFromRecorder(t, rr)
			_, ok := cookies["refresh_token"]
			assert.True(t, ok)
			assert.True(t, cookies["refresh_token"] == "")
		})
	})
}
