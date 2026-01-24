package logger

import (
	"context"
	"io"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func Init() {
	hlog.SetLogger(&hertzLogger{
		loggerInf: NewLogrusLogger(),
	})
	hlog.SetOutput(newOutput())
	hlog.SetLevel(newLevel())
}

type hertzLogger struct {
	loggerInf hlog.FullLogger
}

// SetLevel implements hlog.FullLogger.
func (h *hertzLogger) SetLevel(level hlog.Level) {
	h.loggerInf.SetLevel(level)
}

// SetOutput implements hlog.FullLogger.
func (h *hertzLogger) SetOutput(output io.Writer) {
	h.loggerInf.SetOutput(output)
}

// CtxDebugf implements hlog.FullLogger.
func (h *hertzLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxDebugf(ctx, format, v...)
}

// CtxErrorf implements hlog.FullLogger.
func (h *hertzLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxErrorf(ctx, format, v...)
}

// CtxFatalf implements hlog.FullLogger.
func (h *hertzLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxFatalf(ctx, format, v...)
}

// CtxInfof implements hlog.FullLogger.
func (h *hertzLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxInfof(ctx, format, v...)
}

// CtxNoticef implements hlog.FullLogger.
func (h *hertzLogger) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxInfof(ctx, format, v...)
}

// CtxTracef implements hlog.FullLogger.
func (h *hertzLogger) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxTracef(ctx, format, v...)
}

// CtxWarnf implements hlog.FullLogger.
func (h *hertzLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	h.loggerInf.CtxWarnf(ctx, format, v...)
}

// Debug implements hlog.FullLogger.
func (h *hertzLogger) Debug(v ...interface{}) {
	h.loggerInf.Debug(v...)
}

// Debugf implements hlog.FullLogger.
func (h *hertzLogger) Debugf(format string, v ...interface{}) {
	h.loggerInf.Debugf(format, v...)
}

// Error implements hlog.FullLogger.
func (h *hertzLogger) Error(v ...interface{}) {
	h.loggerInf.Error(v...)
}

// Errorf implements hlog.FullLogger.
func (h *hertzLogger) Errorf(format string, v ...interface{}) {
	h.loggerInf.Errorf(format, v...)
}

// Fatal implements hlog.FullLogger.
func (h *hertzLogger) Fatal(v ...interface{}) {
	h.loggerInf.Fatal(v...)
}

// Fatalf implements hlog.FullLogger.
func (h *hertzLogger) Fatalf(format string, v ...interface{}) {
	h.loggerInf.Fatalf(format, v...)
}

// Info implements hlog.FullLogger.
func (h *hertzLogger) Info(v ...interface{}) {
	h.loggerInf.Info(v...)
}

// Infof implements hlog.FullLogger.
func (h *hertzLogger) Infof(format string, v ...interface{}) {
	h.loggerInf.Infof(format, v...)
}

// Notice implements hlog.FullLogger.
func (h *hertzLogger) Notice(v ...interface{}) {
	h.loggerInf.Info(v...)
}

// Noticef implements hlog.FullLogger.
func (h *hertzLogger) Noticef(format string, v ...interface{}) {
	h.loggerInf.Infof(format, v...)
}

// Trace implements hlog.FullLogger.
func (h *hertzLogger) Trace(v ...interface{}) {
	h.loggerInf.Trace(v...)
}

// Tracef implements hlog.FullLogger.
func (h *hertzLogger) Tracef(format string, v ...interface{}) {
	h.loggerInf.Tracef(format, v...)
}

// Warn implements hlog.FullLogger.
func (h *hertzLogger) Warn(v ...interface{}) {
	h.loggerInf.Warn(v...)
}

// Warnf implements hlog.FullLogger.
func (h *hertzLogger) Warnf(format string, v ...interface{}) {
	h.loggerInf.Warnf(format, v...)
}
