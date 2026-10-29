// Package logger configures structured JSON logging (slog).
package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
	slog.SetDefault(l)
	return l
}
