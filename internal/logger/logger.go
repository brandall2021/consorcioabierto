// Package logger crea el logger estructurado de la aplicación (slog).
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New devuelve un *slog.Logger con salida JSON (por defecto) o texto plano.
// El nivel sale de LOG_LEVEL (debug|info|warn|error; defecto info).
// Los logs no incluyen secretos, tokens ni PII ([ADR-0009] §9).
func New(format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level(os.Getenv("LOG_LEVEL"))}
	if format == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func level(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
