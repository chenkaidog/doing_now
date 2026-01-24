package trace_info

import (
	"context"
)

type logIdKey struct{}

func WithLogId(ctx context.Context, logId string) context.Context {
	return context.WithValue(ctx, logIdKey{}, logId)
}

func GetLogId(ctx context.Context) string {
	logId, ok := ctx.Value(logIdKey{}).(string)
	if ok {
		return logId
	}
	return ""
}
