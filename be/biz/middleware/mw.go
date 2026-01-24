package middleware

import (
	"doing_now/be/biz/middleware/accesslog"
	"doing_now/be/biz/middleware/cors"
	"doing_now/be/biz/middleware/ratelimit"
	"doing_now/be/biz/middleware/recovery"
	"doing_now/be/biz/middleware/session"
	"doing_now/be/biz/middleware/trace"

	"github.com/cloudwego/hertz/pkg/app"
)

func Suite() []app.HandlerFunc {
	return []app.HandlerFunc{
		recovery.New(),  // panic handler
		trace.New(),     // 链路ID
		accesslog.New(), // 接口日志
		cors.New(),      // 跨域请求
		session.New(),   // 会话
		ratelimit.New(), // 限流
	}
}
