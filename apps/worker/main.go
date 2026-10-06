// Entrypoint del worker (outbox, PDFs, envíos). Comparte los paquetes internal
// con apps/api ([ADR-0001]).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/brandall2021/consorcioabierto/internal/config"
	"github.com/brandall2021/consorcioabierto/internal/database"
	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/logger"
	"github.com/brandall2021/consorcioabierto/internal/outbox"
)

func main() {
	log := logger.New(os.Getenv("LOG_FORMAT"))
	slog.SetDefault(log)

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

	mailDriver := outbox.MailDriver(&outbox.MockDriver{Log: log})
	if cfg.MailDriver == "mailpit" {
		mailDriver = &outbox.MailpitDriver{BaseURL: "http://localhost:8025"}
	}

	w := &outbox.Worker{Log: log, Pool: pool, Queries: db.New(pool), Mail: mailDriver, PDFGen: &outbox.SimplePDFGenerator{}}
	go w.Run(ctx)
	log.Info("worker iniciado", "env", cfg.Env)
	<-ctx.Done()
	log.Info("worker detenido")
}
