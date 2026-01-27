package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Settings struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"info" validate:"oneof=trace,debug,info,warn,error,fatal"`
}

func determineLogLevel(level string) slog.Level {
	switch l := strings.ToUpper(level); l {
	case "TRACE":
		return slog.Level(-8)
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "FATAL":
		return slog.Level(12)
	default:
		return slog.LevelInfo
	}
}

func NewLogger() *slog.Logger {
	var settings Settings
	if err := env.Parse(&settings); err != nil {
		panic(err)
	}

	logLevel := determineLogLevel(settings.LogLevel)

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
}
