package cors

import (
	"doing_now/be/biz/config"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/cors"
)

func New() app.HandlerFunc {
	corsConf := config.GetCORSConf()

	cfg := cors.Config{
		AllowMethods:     defaultIfEmpty(corsConf.AllowMethods, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}),
		AllowHeaders:     defaultIfEmpty(corsConf.AllowHeaders, []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"}),
		AllowCredentials: corsConf.AllowCredentials,
		MaxAge:           time.Duration(corsConf.MaxAge) * time.Second,
	}

	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 12 * time.Hour
	}

	if len(corsConf.AllowOrigins) == 0 {
		cfg.AllowOriginFunc = func(origin string) bool {
			// 未来改成动态配置
			return true
		}
	} else if contains(corsConf.AllowOrigins, "*") {
		if corsConf.AllowCredentials {
			cfg.AllowOriginFunc = func(origin string) bool {
				return true
			}
		} else {
			cfg.AllowAllOrigins = true
		}
	} else {
		cfg.AllowOrigins = corsConf.AllowOrigins
	}

	return cors.New(cfg)
}

func defaultIfEmpty(v, def []string) []string {
	if len(v) == 0 {
		return def
	}
	return v
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
