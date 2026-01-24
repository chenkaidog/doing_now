package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "deploy.yml")
	if err := os.WriteFile(p, []byte(`mysql:
  db_name: ""
  ip: "127.0.0.1"
  port: 3306
  username: ""
  password: ""

redis:
  ip: "127.0.0.1"
  port: 6379
  password: ""
  db: 0

jwt:
  access_expiration: 10
  refresh_expiration: 20
  access_token_secret: "test_secret"
  refresh_token_secret: "test_secret"
  issuer: "test_issuer"

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
`), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	Init(p)
	if got := GetJWTConfig().Issuer; got != "test_issuer" {
		t.Fatalf("issuer mismatch: got=%q", got)
	}
}
