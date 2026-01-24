package logger

import (
	"doing_now/be/biz/config"
	"io"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newOutput() io.Writer {
	conf := config.GetLoggerConf()
	dir := conf.Dir
	if dir == "" {
		dir = "./log"
	}
	filename := conf.FileName
	if filename == "" {
		filename = "hertz.log"
	}

	maxSize := conf.MaxSize
	if maxSize == 0 {
		maxSize = 512
	}

	maxBackups := conf.MaxBackups
	if maxBackups == 0 {
		maxBackups = 10
	}

	maxAge := conf.MaxAge
	if maxAge == 0 {
		maxAge = 14
	}

	return io.MultiWriter(
		&lumberjack.Logger{
			Filename:   filepath.Join(dir, filename),
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
			LocalTime:  true,
			Compress:   false,
		},
	)
}

func newLevel() hlog.Level {
	conf := config.GetLoggerConf()
	switch conf.Level {
	case "trace":
		return hlog.LevelTrace
	case "debug":
		return hlog.LevelDebug
	case "info":
		return hlog.LevelInfo
	case "notice":
		return hlog.LevelNotice
	case "warn":
		return hlog.LevelWarn
	case "error":
		return hlog.LevelError
	case "fatal":
		return hlog.LevelFatal
	}

	return hlog.LevelTrace
}
