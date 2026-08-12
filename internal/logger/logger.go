// Package logger crea el logger estructurado de la aplicación (slog).
package logger

import (
	"log/slog"
	"os"
)

// New devuelve un *slog.Logger con salida JSON (por defecto) o texto plano.
// Los logs no incluyen secretos, tokens ni PII ([ADR] §9).
func New(format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if format == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
