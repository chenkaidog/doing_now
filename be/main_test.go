package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	be "doing_now/be"
	"doing_now/be/biz/config"
	redisdb "doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/domain"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	usersvc "doing_now/be/biz/service/user"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/common/ut"
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
  access_token_secret: "test-secret"
  refresh_token_secret: "test-secret"
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

func TestRegister_ParamError(t *testing.T) {
	h := newTestServer(t)

	w := perform(h, http.MethodPost, "/api/v1/user/register", "{")
	resp := w.Result()
	assert.DeepEqual(t, http.StatusBadRequest, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.False(t, r.Success)
	assert.DeepEqual(t, int(errs.ParamError.Code()), r.Code)
}

func TestRegister_AccountTooLong(t *testing.T) {
	h := newTestServer(t)

	longAcc := strings.Repeat("a", 65)
	body := `{"account":"` + longAcc + `","name":"name","password":"pwd"}`
	w := perform(h, http.MethodPost, "/api/v1/user/register", body)
	resp := w.Result()
	assert.DeepEqual(t, http.StatusBadRequest, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.False(t, r.Success)
	assert.DeepEqual(t, int(errs.ParamError.Code()), r.Code)
}

func TestRegister_SuccessAndBizError(t *testing.T) {
	h := newTestServer(t)

	patchCtor := mockey.Mock(usersvc.NewDefault).Return(&usersvc.Service{}).Build()
	defer patchCtor.UnPatch()

	patchReg := mockey.Mock((*usersvc.Service).Register).
		Return(&domain.User{UserID: "u1"}, nil).
		Build()
	defer patchReg.UnPatch()

	body := `{"account":"acc","name":"name","password":"pwd"}`
	w := perform(h, http.MethodPost, "/api/v1/user/register", body)
	resp := w.Result()
	assert.DeepEqual(t, http.StatusOK, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.True(t, r.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r.Code)

	dataBytes, err := json.Marshal(r.Data)
	assert.Nil(t, err)
	var reg dto.RegisterResp
	err = json.Unmarshal(dataBytes, &reg)
	assert.Nil(t, err)
	assert.DeepEqual(t, "u1", reg.UserID)

	patchReg.UnPatch()
	patchReg = mockey.Mock((*usersvc.Service).Register).
		Return(nil, errs.UserNameDuplicatedErr).
		Build()
	defer patchReg.UnPatch()

	w2 := perform(h, http.MethodPost, "/api/v1/user/register", body)
	resp2 := w2.Result()
	assert.DeepEqual(t, http.StatusOK, resp2.StatusCode())

	r2 := decodeCommonResp(t, resp2.Body())
	assert.False(t, r2.Success)
	assert.DeepEqual(t, int(errs.UserNameDuplicatedErr.Code()), r2.Code)
}

func TestLogin_ParamError(t *testing.T) {
	h := newTestServer(t)

	w := perform(h, http.MethodPost, "/api/v1/user/login", "{")
	resp := w.Result()
	assert.DeepEqual(t, http.StatusBadRequest, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.False(t, r.Success)
	assert.DeepEqual(t, int(errs.ParamError.Code()), r.Code)
}

func TestLogin_BizError(t *testing.T) {
	h := newTestServer(t)

	patchCtor := mockey.Mock(usersvc.NewDefault).Return(&usersvc.Service{}).Build()
	defer patchCtor.UnPatch()

	patchLogin := mockey.Mock((*usersvc.Service).Login).
		Return((*domain.User)(nil), errs.UserNotExist).
		Build()
	defer patchLogin.UnPatch()

	body := `{"account":"acc","password":"pwd"}`
	w := perform(h, http.MethodPost, "/api/v1/user/login", body)
	resp := w.Result()
	assert.DeepEqual(t, http.StatusOK, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.False(t, r.Success)
	assert.DeepEqual(t, int(errs.UserNotExist.Code()), r.Code)
}

func TestGetUserInfo_Unauthorized(t *testing.T) {
	h := newTestServer(t)

	w := perform(h, http.MethodGet, "/api/v1/user/info", "")
	resp := w.Result()
	assert.DeepEqual(t, http.StatusUnauthorized, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.False(t, r.Success)
	assert.DeepEqual(t, int(errs.Unauthorized.Code()), r.Code)
}

func TestLoginGetUserInfoAndLogout_SuccessFlow(t *testing.T) {
	h := newTestServer(t)

	patchCtor := mockey.Mock(usersvc.NewDefault).Return(&usersvc.Service{}).Build()
	defer patchCtor.UnPatch()

	u := &domain.User{
		UserID:  "u1",
		Account: "acc",
		Name:    "name",
	}

	patchLogin := mockey.Mock((*usersvc.Service).Login).
		Return(u, nil).
		Build()
	defer patchLogin.UnPatch()

	patchGetByID := mockey.Mock((*usersvc.Service).GetByUserID).
		Return(u, nil).
		Build()
	defer patchGetByID.UnPatch()

	loginBody := `{"account":"acc","password":"pwd"}`
	w := perform(h, http.MethodPost, "/api/v1/user/login", loginBody)
	resp := w.Result()
	assert.DeepEqual(t, http.StatusOK, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.True(t, r.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r.Code)

	setCookie := string(resp.Header.Peek("Set-Cookie"))
	if setCookie == "" {
		t.Fatalf("no set-cookie header")
	}

	dataBytes, err := json.Marshal(r.Data)
	assert.Nil(t, err)
	var loginResp dto.LoginResp
	err = json.Unmarshal(dataBytes, &loginResp)
	assert.Nil(t, err)
	if loginResp.AccessToken == "" {
		t.Fatalf("empty access token, resp=%s", string(resp.Body()))
	}

	w2 := perform(h, http.MethodGet, "/api/v1/user/info", "",
		ut.Header{Key: "Authorization", Value: loginResp.AccessToken},
		ut.Header{Key: "Cookie", Value: setCookie},
	)
	resp2 := w2.Result()
	assert.DeepEqual(t, http.StatusOK, resp2.StatusCode())

	r2 := decodeCommonResp(t, resp2.Body())
	assert.True(t, r2.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r2.Code)

	dataBytes2, err := json.Marshal(r2.Data)
	assert.Nil(t, err)
	var info dto.GetUserInfoResp
	err = json.Unmarshal(dataBytes2, &info)
	assert.Nil(t, err)
	assert.DeepEqual(t, u.UserID, info.UserID)
	assert.DeepEqual(t, u.Account, info.Account)
	assert.DeepEqual(t, u.Name, info.Name)

	w3 := perform(h, http.MethodPost, "/api/v1/user/logout", "{}",
		ut.Header{Key: "Authorization", Value: loginResp.AccessToken},
		ut.Header{Key: "Cookie", Value: setCookie},
	)
	resp3 := w3.Result()
	assert.DeepEqual(t, http.StatusOK, resp3.StatusCode())

	r3 := decodeCommonResp(t, resp3.Body())
	assert.True(t, r3.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r3.Code)
}

func TestGetUserInfo_BizError(t *testing.T) {
	h := newTestServer(t)

	patchCtor := mockey.Mock(usersvc.NewDefault).Return(&usersvc.Service{}).Build()
	defer patchCtor.UnPatch()

	u := &domain.User{
		UserID:  "u1",
		Account: "acc",
		Name:    "name",
	}

	patchLogin := mockey.Mock((*usersvc.Service).Login).
		Return(u, nil).
		Build()
	defer patchLogin.UnPatch()

	// first login to get valid token and cookie
	loginBody := `{"account":"acc","password":"pwd"}`
	w := perform(h, http.MethodPost, "/api/v1/user/login", loginBody)
	resp := w.Result()
	assert.DeepEqual(t, http.StatusOK, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.True(t, r.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r.Code)

	setCookie := string(resp.Header.Peek("Set-Cookie"))
	if setCookie == "" {
		t.Fatalf("no set-cookie header")
	}

	dataBytes, err := json.Marshal(r.Data)
	assert.Nil(t, err)
	var loginResp dto.LoginResp
	err = json.Unmarshal(dataBytes, &loginResp)
	assert.Nil(t, err)
	if loginResp.AccessToken == "" {
		t.Fatalf("empty access token, resp=%s", string(resp.Body()))
	}

	patchGetByID := mockey.Mock((*usersvc.Service).GetByUserID).
		Return((*domain.User)(nil), errs.UserNotExist).
		Build()
	defer patchGetByID.UnPatch()

	w2 := perform(h, http.MethodGet, "/api/v1/user/info", "",
		ut.Header{Key: "Authorization", Value: loginResp.AccessToken},
		ut.Header{Key: "Cookie", Value: setCookie},
	)
	resp2 := w2.Result()
	assert.DeepEqual(t, http.StatusOK, resp2.StatusCode())

	r2 := decodeCommonResp(t, resp2.Body())
	assert.False(t, r2.Success)
	assert.DeepEqual(t, int(errs.UserNotExist.Code()), r2.Code)
}

func TestRefreshToken_SuccessFlow(t *testing.T) {
	h := newTestServer(t)

	patchCtor := mockey.Mock(usersvc.NewDefault).Return(&usersvc.Service{}).Build()
	defer patchCtor.UnPatch()

	u := &domain.User{
		UserID:  "u1",
		Account: "acc",
		Name:    "name",
	}

	patchLogin := mockey.Mock((*usersvc.Service).Login).
		Return(u, nil).
		Build()
	defer patchLogin.UnPatch()

	patchGetByID := mockey.Mock((*usersvc.Service).GetByUserID).
		Return(u, nil).
		Build()
	defer patchGetByID.UnPatch()

	loginBody := `{"account":"acc","password":"pwd"}`
	w := perform(h, http.MethodPost, "/api/v1/user/login", loginBody)
	resp := w.Result()
	assert.DeepEqual(t, http.StatusOK, resp.StatusCode())

	r := decodeCommonResp(t, resp.Body())
	assert.True(t, r.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r.Code)

	setCookie := string(resp.Header.Peek("Set-Cookie"))
	if setCookie == "" {
		t.Fatalf("no set-cookie header")
	}

	dataBytes, err := json.Marshal(r.Data)
	assert.Nil(t, err)
	var loginResp dto.LoginResp
	err = json.Unmarshal(dataBytes, &loginResp)
	assert.Nil(t, err)

	w2 := perform(h, http.MethodPost, "/api/v1/user/refresh_token", "{}",
		ut.Header{Key: "Cookie", Value: setCookie},
	)
	resp2 := w2.Result()
	assert.DeepEqual(t, http.StatusOK, resp2.StatusCode())

	r2 := decodeCommonResp(t, resp2.Body())
	assert.True(t, r2.Success)
	assert.DeepEqual(t, int(errs.Success.Code()), r2.Code)

	dataBytes2, err := json.Marshal(r2.Data)
	assert.Nil(t, err)
	var refreshResp dto.RefreshTokenResp
	err = json.Unmarshal(dataBytes2, &refreshResp)
	assert.Nil(t, err)
	if refreshResp.AccessToken == "" {
		t.Fatalf("empty access token, resp=%s", string(resp2.Body()))
	}
	if refreshResp.RefreshToken == "" {
		t.Fatalf("empty refresh token, resp=%s", string(resp2.Body()))
	}

	w3 := perform(h, http.MethodGet, "/api/v1/user/info", "",
		ut.Header{Key: "Authorization", Value: refreshResp.AccessToken},
		ut.Header{Key: "Cookie", Value: setCookie},
	)
	resp3 := w3.Result()
	assert.DeepEqual(t, http.StatusOK, resp3.StatusCode())
}
