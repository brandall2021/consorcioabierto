// Entrypoint del worker (outbox, PDFs, envíos). Comparte los paquetes internal
// con apps/api ([ADR-0001]).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/brandall2021/consorcioabierto/internal/config"
	"github.com/brandall2021/consorcioabierto/internal/database"
	"github.com/brandall2021/consorcioabierto/internal/logger"
)

func main() {
	log := logger.New(os.Getenv("LOG_FORMAT"))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("base de datos", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Fase posterior: consumo del outbox (documentos, notificaciones, envíos).
	log.Info("worker iniciado", "env", cfg.Env)
	<-ctx.Done()
	log.Info("worker detenido")
}
