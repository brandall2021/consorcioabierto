// Package database gestiona el pool pgx y las migraciones goose.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para database/sql (goose)
)

// Connect crea un pool pgx y verifica conectividad.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parsear DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("crear pool pgx: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping a la base: %w", err)
	}
	return pool, nil
}

// Up aplica todas las migraciones pendientes en dir.
func Up(ctx context.Context, url, dir string) error {
	return withProvider(ctx, url, dir, func(p *goose.Provider) error {
		_, err := p.Up(ctx)
		return err
	})
}

// Down revierte `steps` migraciones.
func Down(ctx context.Context, url, dir string, steps int) error {
	return withProvider(ctx, url, dir, func(p *goose.Provider) error {
		v, err := p.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		to := v - int64(steps)
		if to < 0 {
			to = 0
		}
		if _, err := p.DownTo(ctx, to); err != nil {
			return err
		}
		return nil
	})
}

func withProvider(ctx context.Context, url, dir string, fn func(*goose.Provider) error) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("abrir conexión de migración: %w", err)
	}
	defer func() { _ = db.Close() }()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping de migración: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(dir))
	if err != nil {
		return fmt.Errorf("crear proveedor goose: %w", err)
	}
	return fn(provider)
}
