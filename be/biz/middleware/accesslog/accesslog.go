package accesslog

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/logger/accesslog"
)

func New() app.HandlerFunc {
	return accesslog.New(
		accesslog.WithAccessLogFunc(hlog.CtxInfof),
		accesslog.WithFormat("${status} ${latency} ${method} ${path} ${queryParams}"),
	)
}
