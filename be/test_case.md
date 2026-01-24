# HTTP 测试用例：登录相关（curl）

服务默认监听：`http://127.0.0.1:8000`（见 [main.go](file:///e:/Users/go/src/doing_now/be/main.go)）。

## 通用约定

- 统一响应结构：`{ "success": bool, "code": int, "message": string, "data": any }`（见 [common.go](file:///e:/Users/go/src/doing_now/be/biz/model/dto/common.go) 与 [resp.go](file:///e:/Users/go/src/doing_now/be/biz/util/resp/resp.go)）。
- 常见错误码：
  - 参数错误：`code=10002`，通常 **HTTP 400**
  - 未授权：`code=10003`，在 JWT 中间件场景通常 **HTTP 401**
  - 账号不存在或密码错误：`code=20001`，通常 **HTTP 200**
- 重要鉴权说明：
  - 受保护接口（如 `/api/v1/user/info`、`/api/v1/user/logout`）不仅需要 `Authorization`，还依赖登录时服务端下发的 **session cookie**（默认名 `auth_session_id`）。
  - `Authorization` 头要求直接放 **token 字符串本体**，不支持 `Bearer <token>` 前缀（见 [jwt.go](file:///e:/Users/go/src/doing_now/be/biz/middleware/jwt/jwt.go) 的 `exactJWT` 实现）。

## 预置数据

登录测试需要一个已存在用户。可先调用注册接口创建账号（如已有账号可跳过）。

```bash
curl -sS -X POST "http://127.0.0.1:8000/api/v1/user/register" \
  -H "Content-Type: application/json" \
  -d '{"account":"tc_login_user","name":"TC User","password":"tc_login_pass"}'
```

## 测试用 cookie 文件

建议用 `-c`/`-b` 保存并复用 cookie（同时包含 session 与 refresh_token）。

```bash
COOKIE_JAR=./cookiejar.txt
```

---

## TC-L-001 登录成功（返回 access_token + 下发 cookie）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -c "$COOKIE_JAR" \
  -d '{"account":"tc_login_user","password":"tc_login_pass"}'
```

预期：
- HTTP 200
- Body：`success=true`、`code=0`，且 `data.access_token`、`data.expires_at` 存在
- Headers：包含 `Set-Cookie`，至少应看到 `auth_session_id` 与 `refresh_token`

---

## TC-L-002 缺少 password（参数校验失败）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"tc_login_user"}'
```

预期：
- HTTP 400
- Body：`success=false`、`code=10002`、`message="param error"`

---

## TC-L-003 password 超长（>128）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"tc_login_user","password":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
```

预期：
- HTTP 400
- Body：`success=false`、`code=10002`

---

## TC-L-004 account 超长（>64）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","password":"tc_login_pass"}'
```

预期：
- HTTP 400
- Body：`success=false`、`code=10002`

---

## TC-L-005 非法 JSON（解析失败）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"tc_login_user","password":'
```

预期：
- HTTP 400
- Body：`success=false`、`code=10002`

---

## TC-L-006 密码错误

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"tc_login_user","password":"wrong_pass"}'
```

预期：
- HTTP 200
- Body：`success=false`、`code=20001`、`message="user not exist or password incorrect"`

---

## TC-L-007 账号不存在

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"tc_login_user_not_exist","password":"any"}'
```

预期：
- HTTP 200
- Body：`success=false`、`code=20001`、`message="user not exist or password incorrect"`

---

## TC-L-008 account 为空字符串（不会触发 required 校验）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"","password":"tc_login_pass"}'
```

预期（当前实现）：
- HTTP 200
- Body：`success=false`、`code=20001`

---

## TC-A-001 不带 Authorization 访问受保护接口（未授权）

```bash
curl -sS -i -X GET "http://127.0.0.1:8000/api/v1/user/info"
```

预期：
- HTTP 401
- Body：`success=false`、`code=10003`、`message="user unauthorized"`

---

## TC-A-002 使用 Bearer 前缀（不被支持，未授权）

先完成 TC-L-001，从响应中拷贝 `data.access_token` 作为 `<ACCESS_TOKEN>`。

```bash
curl -sS -i -X GET "http://127.0.0.1:8000/api/v1/user/info" \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

预期：
- HTTP 401
- Body：`success=false`、`code=10003`

---

## TC-A-003 只带 Authorization 不带 cookie（session 校验失败，未授权）

```bash
curl -sS -i -X GET "http://127.0.0.1:8000/api/v1/user/info" \
  -H "Authorization: <ACCESS_TOKEN>"
```

预期：
- HTTP 401
- Body：`success=false`、`code=10003`

---

## TC-A-004 Authorization + cookie 正常访问 /info

```bash
curl -sS -i -X GET "http://127.0.0.1:8000/api/v1/user/info" \
  -b "$COOKIE_JAR" \
  -H "Authorization: <ACCESS_TOKEN>"
```

预期：
- HTTP 200
- Body：`success=true`、`code=0`，且 `data.user_id`、`data.account`、`data.name` 存在

---

## TC-R-001 刷新 token（需要 session cookie + refresh_token cookie）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/refresh_token" \
  -H "Content-Type: application/json" \
  -b "$COOKIE_JAR" \
  -c "$COOKIE_JAR" \
  -d '{}'
```

预期：
- HTTP 200
- Body：`success=true`、`code=0`，且 `data.access_token`、`data.refresh_token` 存在
- Headers：`Set-Cookie` 会更新 `refresh_token`

---

## TC-R-002 刷新 token 不带 cookie（未授权）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/refresh_token" \
  -H "Content-Type: application/json" \
  -d '{}'
```

预期（当前实现）：
- HTTP 200
- Body：`success=false`、`code=10003`

---

## TC-O-001 登出成功（需要 Authorization + cookie）

```bash
curl -sS -i -X POST "http://127.0.0.1:8000/api/v1/user/logout" \
  -H "Content-Type: application/json" \
  -b "$COOKIE_JAR" \
  -c "$COOKIE_JAR" \
  -H "Authorization: <ACCESS_TOKEN>" \
  -d '{}'
```

预期：
- HTTP 200
- Body：`success=true`、`code=0`
- Headers：`Set-Cookie` 应清理 `refresh_token`，session cookie 也会被删除/失效

---

## TC-O-002 登出后再次访问 /info（未授权）

```bash
curl -sS -i -X GET "http://127.0.0.1:8000/api/v1/user/info" \
  -b "$COOKIE_JAR" \
  -H "Authorization: <ACCESS_TOKEN>"
```

预期：
- HTTP 401
- Body：`success=false`、`code=10003`

