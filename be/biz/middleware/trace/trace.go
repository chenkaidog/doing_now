package trace

import (
	"context"
	"doing_now/be/biz/util/id_gen"
	"doing_now/be/biz/util/trace_info"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	headerKeyLogId = "X-Log-ID"
)

func New() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		logID := c.Request.Header.Get(headerKeyLogId)
		if logID == "" {
			logID = id_gen.NewID()
		}
		ctx = trace_info.WithLogId(ctx, logID)
		c.Next(ctx)
		c.Header(headerKeyLogId, logID)
	}
}
