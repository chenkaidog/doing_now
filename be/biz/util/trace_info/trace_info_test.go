package trace_info

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceInfo(t *testing.T) {
	ctx := context.Background()
	logId := "123456zbcd"
	ctx = WithLogId(ctx, logId)

	assert.Equal(t, logId, GetLogId(ctx))
}
