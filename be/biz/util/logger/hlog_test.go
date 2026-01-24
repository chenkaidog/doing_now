package logger

import (
	"context"
	"doing_now/be/biz/util/random"
	"doing_now/be/biz/util/trace_info"
	"testing"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func TestHlog(t *testing.T) {
	Init()

	ctx := trace_info.WithLogId(context.Background(), random.RandStr(32))

	hlog.CtxInfof(ctx, "test info data: %d, %s", 123, "ttt")
	hlog.CtxErrorf(ctx, "test error data: %d, %s", 123, "ttt")

	hlog.Infof("test info data: %d, %s", 123, "ttt")
	hlog.Errorf("test error data: %d, %s", 123, "ttt")
}
