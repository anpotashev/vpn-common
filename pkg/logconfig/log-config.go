package logconfig

import (
	"log/slog"
	"os"
)

const TraceLogLevel = slog.Level(-8)

func ConfigLogs(logLevel string) {
	logHandler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: getLogLevel(logLevel),
		})
	slog.SetDefault(slog.New(logHandler))
}

func getLogLevel(level string) slog.Leveler {
	switch level {
	case "trace":
		return TraceLogLevel
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		panic("Invalid log level")
	}
}
