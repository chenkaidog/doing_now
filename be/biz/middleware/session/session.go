package session

import (
	"context"
	"doing_now/be/biz/config"
	"doing_now/be/biz/db/redis"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/sessions"
	"github.com/rbcervilla/redisstore/v9"
)

func New() app.HandlerFunc {
	conf := config.GetSessionConf()

	store := NewRedisStore(conf.StorePrefix)
	store.Options(sessions.Options{
		Path:     defaultString(conf.Path, "/"),
		Domain:   conf.Domain,
		MaxAge:   defaultInt(conf.MaxAge, 7*24*3600),
		Secure:   conf.Secure,
		HttpOnly: conf.HTTPOnly,
		SameSite: parseSameSite(conf.SameSite),
	})

	return sessions.New(defaultString(conf.Name, "auth_session_id"), store)
}

func Remove(c *app.RequestContext) error {
	conf := config.GetSessionConf()

	sess := sessions.Default(c)
	sess.Options(sessions.Options{
		Path:     defaultString(conf.Path, "/"),
		Domain:   conf.Domain,
		MaxAge:   -1,
		Secure:   conf.Secure,
		HttpOnly: conf.HTTPOnly,
		SameSite: parseSameSite(conf.SameSite),
	})
	return sess.Save()
}

type RedisStore struct {
	*redisstore.RedisStore
}

func (r *RedisStore) Options(opts sessions.Options) {
	r.RedisStore.Options(*opts.ToGorillaOptions())
}

func NewRedisStore(prefix string) *RedisStore {
	redisStore, err := redisstore.NewRedisStore(context.Background(), redis.GetRedisClient())
	if err != nil {
		panic(err)
	}
	if prefix == "" {
		prefix = "auth_session:"
	}
	redisStore.KeyPrefix(prefix)
	return &RedisStore{
		RedisStore: redisStore,
	}
}

func defaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func defaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func parseSameSite(v string) http.SameSite {
	switch v {
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	case "Strict":
		fallthrough
	default:
		return http.SameSiteStrictMode
	}
}
