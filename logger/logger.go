package logger

import (
	"log/slog"
	"os"
)

// Init sets up the default slog logger with JSON output to stdout.
func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
