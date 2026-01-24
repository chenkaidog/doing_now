package logger

import (
	"context"
	"doing_now/be/biz/util/trace_info"
	"fmt"
	"io"
	"path"
	"runtime"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/sirupsen/logrus"
)

const depth = 4

type logrusLogger struct {
	*logrus.Logger
}

func NewLogrusLogger() hlog.FullLogger {
	logger := &logrusLogger{
		Logger: logrus.New(),
	}

	logger.SetFormatter(
		&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		},
	)

	logger.AddHook(new(TraceHook))
	logger.AddHook(new(ConsoleHook))

	return logger
}

func (logger *logrusLogger) entryWithLoc() *logrus.Entry {
	_, file, line, ok := runtime.Caller(depth)
	if ok {
		return logger.Logger.WithFields(
			logrus.Fields{
				"location": fmt.Sprintf("%s:%d", path.Base(file), line),
			})
	}

	return logger.Logger.WithFields(logrus.Fields{})
}

// CtxDebugf implements hlog.FullLogger.
func (logger *logrusLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Debugf(format, v...)
}

// CtxErrorf implements hlog.FullLogger.
func (logger *logrusLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Errorf(format, v...)
}

// CtxFatalf implements hlog.FullLogger.
func (logger *logrusLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Fatalf(format, v...)
}

// CtxInfof implements hlog.FullLogger.
func (logger *logrusLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Infof(format, v...)
}

// CtxNoticef implements hlog.FullLogger.
func (logger *logrusLogger) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Infof(format, v...)
}

// CtxTracef implements hlog.FullLogger.
func (logger *logrusLogger) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Tracef(format, v...)
}

// CtxWarnf implements hlog.FullLogger.
func (logger *logrusLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	logger.entryWithLoc().WithContext(ctx).Warnf(format, v...)
}

// Debug implements hlog.FullLogger.
func (logger *logrusLogger) Debug(v ...interface{}) {
	logger.entryWithLoc().Debug(v...)
}

// Debugf implements hlog.FullLogger.
func (logger *logrusLogger) Debugf(format string, v ...interface{}) {
	logger.entryWithLoc().Debugf(format, v...)
}

// Error implements hlog.FullLogger.
func (logger *logrusLogger) Error(v ...interface{}) {
	logger.entryWithLoc().Error(v...)
}

// Errorf implements hlog.FullLogger.
func (logger *logrusLogger) Errorf(format string, v ...interface{}) {
	logger.entryWithLoc().Errorf(format, v...)
}

// Fatal implements hlog.FullLogger.
func (logger *logrusLogger) Fatal(v ...interface{}) {
	logger.entryWithLoc().Fatal(v...)
}

// Fatalf implements hlog.FullLogger.
func (logger *logrusLogger) Fatalf(format string, v ...interface{}) {
	logger.entryWithLoc().Fatalf(format, v...)
}

// Info implements hlog.FullLogger.
func (logger *logrusLogger) Info(v ...interface{}) {
	logger.entryWithLoc().Info(v...)
}

// Infof implements hlog.FullLogger.
func (logger *logrusLogger) Infof(format string, v ...interface{}) {
	logger.entryWithLoc().Infof(format, v...)
}

// Notice implements hlog.FullLogger.
func (logger *logrusLogger) Notice(v ...interface{}) {
	logger.entryWithLoc().Info(v...)
}

// Noticef implements hlog.FullLogger.
func (logger *logrusLogger) Noticef(format string, v ...interface{}) {
	logger.entryWithLoc().Infof(format, v...)
}

// Trace implements hlog.FullLogger.
func (logger *logrusLogger) Trace(v ...interface{}) {
	logger.entryWithLoc().Trace(v...)
}

// Tracef implements hlog.FullLogger.
func (logger *logrusLogger) Tracef(format string, v ...interface{}) {
	logger.entryWithLoc().Tracef(format, v...)
}

// Warn implements hlog.FullLogger.
func (logger *logrusLogger) Warn(v ...interface{}) {
	logger.entryWithLoc().Warn(v...)
}

// Warnf implements hlog.FullLogger.
func (logger *logrusLogger) Warnf(format string, v ...interface{}) {
	logger.entryWithLoc().Warnf(format, v...)
}

func (logger *logrusLogger) SetLevel(level hlog.Level) {
	switch level {
	case hlog.LevelTrace:
		logger.Logger.SetLevel(logrus.TraceLevel)
	case hlog.LevelDebug:
		logger.Logger.SetLevel(logrus.DebugLevel)
	case hlog.LevelInfo, hlog.LevelNotice:
		logger.Logger.SetLevel(logrus.InfoLevel)
	case hlog.LevelWarn:
		logger.Logger.SetLevel(logrus.WarnLevel)
	case hlog.LevelError:
		logger.Logger.SetLevel(logrus.ErrorLevel)
	case hlog.LevelFatal:
		logger.Logger.SetLevel(logrus.FatalLevel)
	}
}

// SetOutput implements hlog.FullLogger.
func (logger *logrusLogger) SetOutput(output io.Writer) {
	logger.Logger.SetOutput(output)
}

type TraceHook struct{}

// Fire implements logrus.Hook.
func (*TraceHook) Fire(entry *logrus.Entry) error {
	if entry.Context != nil {
		entry.Data["log_id"] = trace_info.GetLogId(entry.Context)
	}

	return nil
}

// Levels implements logrus.Hook.
func (*TraceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

type ConsoleHook struct{}

func (hook *ConsoleHook) Fire(entry *logrus.Entry) error {
	if entry != nil {
		logId, _ := entry.Data["log_id"].(string)
		location, _ := entry.Data["location"].(string)
		fmt.Printf("[%s] %s %s %s %s\n",
			entry.Level,
			entry.Time.Format(time.RFC3339),
			logId,
			location,
			entry.Message,
		)
	}
	return nil
}

func (hook *ConsoleHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
