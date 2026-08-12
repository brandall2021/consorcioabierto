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

	"github.com/brandall2021/consorcioabierto/internal/config"
	"github.com/brandall2021/consorcioabierto/internal/database"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/brandall2021/consorcioabierto/internal/logger"
	"github.com/brandall2021/consorcioabierto/internal/server"
)

func main() {
	log := logger.New(os.Getenv("LOG_FORMAT"))

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

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("base de datos", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	identityManager := identity.NewAuthManager(cfg, nil, pool)
	if keyPem := []byte(cfg.JWTPrivateKey); len(keyPem) > 0 {
		privateKey, err := identity.ParseRSAPrivateKeyFromPEM(keyPem)
		if err != nil {
			log.Error("clave privada JWT inválida", "error", err)
			os.Exit(1)
		}
		identityManager = identity.NewAuthManager(cfg, privateKey, pool)
	}

	r := server.New(log, cfg.Env, identityManager)
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
