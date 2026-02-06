package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type LogConfig struct {
	Level string `env:"LEVEL" envDefault:"DEBUG"`
}

type loggerCtx struct{}

var loggerKey loggerCtx

func Setup(cfg LogConfig) *slog.Logger {
	level := slog.LevelDebug
	switch strings.ToUpper(cfg.Level) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	return logger
}

func Logger() *slog.Logger {
	return slog.Default()
}

func WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerKey, Logger())
}

func FromContext(ctx context.Context, attrs ...slog.Attr) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		logger = slog.Default()
	}
	if len(attrs) > 0 {
		logger = logger.With(any(attrs).([]any)...)
	}
	return logger
}
