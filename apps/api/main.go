package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/cobranzas"
	"github.com/brandall2021/consorcioabierto/internal/config"
	"github.com/brandall2021/consorcioabierto/internal/database"
	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/documentos"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/brandall2021/consorcioabierto/internal/logger"
	"github.com/brandall2021/consorcioabierto/internal/observability"
	"github.com/brandall2021/consorcioabierto/internal/outbox"
	"github.com/brandall2021/consorcioabierto/internal/server"
)

func main() {
	log := logger.New(os.Getenv("LOG_FORMAT"))
	slog.SetDefault(log)

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(log, os.Args[2:])
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.InitTracing("consorcioabierto-api", cfg.Env, cfg.OTELExporter, cfg.OTELExporterEndpoint)
	if err != nil {
		log.Error("tracing", "error", err)
		os.Exit(1)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("base de datos", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	observability.RegisterDBPool(ctx, pool)
	observability.RegisterOutbox(ctx, pool)

	identityManager := identity.NewAuthManager(cfg, nil, pool)
	if keyPem := []byte(cfg.JWTPrivateKey); len(keyPem) > 0 {
		privateKey, err := identity.ParseRSAPrivateKeyFromPEM(keyPem)
		if err != nil {
			log.Error("clave privada JWT inválida", "error", err)
			os.Exit(1)
		}
		identityManager = identity.NewAuthManager(cfg, privateKey, pool)
	}

	docsEnv, err := documentos.DocsEnvFromConfig(
		cfg.StorageDriver, cfg.ScanDriver,
		cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region, cfg.S3Bucket,
		cfg.S3UseSSL, cfg.S3SignedTTL, cfg.MaxUploadBytes,
	)
	if err != nil {
		log.Error("config documentos", "error", err)
		os.Exit(1)
	}

	workerEnabled := os.Getenv("WORKER_ENABLED")
	if workerEnabled == "" {
		workerEnabled = "true"
	}
	if workerEnabled != "false" {
		mailDriver := outbox.MailDriver(&outbox.MockDriver{Log: log})
		if cfg.MailDriver == "mailpit" {
			mailDriver = &outbox.MailpitDriver{BaseURL: "http://localhost:8025"}
		}
		w := &outbox.Worker{
			Log:     log,
			Pool:    pool,
			Queries: db.New(pool),
			Mail:    mailDriver,
			PDFGen:  &outbox.SimplePDFGenerator{},
		}
		go w.Run(ctx)
	}

	r := server.New(log, cfg.Env, identityManager, audit.New(pool), docsEnv, cobranzas.NewPSP(cfg.PSPDriver, cfg.BaseURL), cfg.MercadoPagoWebhookSecret)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("api iniciado", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "error", err)
	}
	log.Info("api detenido")
}

// runMigrate ejecuta `go run ./apps/api migrate up|down [n]` con goose.
func runMigrate(log *slog.Logger, args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}
	if len(args) < 1 {
		log.Error("uso: go run ./apps/api migrate [up|down <n>]")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dbURL := cfg.DatabaseURLAdmin
	dir := "db/migrations"
	switch args[0] {
	case "up":
		err = database.Up(ctx, dbURL, dir)
	case "down":
		n := 1
		if len(args) > 1 {
			n, err = strconv.Atoi(args[1])
			if err != nil {
				log.Error("paso inválido", "error", err)
				os.Exit(1)
			}
		}
		err = database.Down(ctx, dbURL, dir, n)
	default:
		log.Error("comando desconocido", "cmd", args[0])
		os.Exit(1)
	}
	if err != nil {
		log.Error("migración", "error", err)
		os.Exit(1)
	}
	log.Info("migración ok")
}
