# ConsorcioAbierto — Makefile

.DEFAULT_GOAL := help

## ── Entorno ────────────────────────────────────────────────
up:            ## Levanta el stack local (Compose: postgres, minio, mailpit)
	docker compose -f deploy/compose.yaml up -d

down:          ## Detiene el stack local
	docker compose -f deploy/compose.yaml down

logs:          ## Logs del stack local
	docker compose -f deploy/compose.yaml logs -f

## ── Backend ────────────────────────────────────────────────
gen:           ## Regenera sqlc y el cliente OpenAPI
	sqlc generate
	cd apps/web && npm run generate:client

migrate-up:    ## Aplica migraciones (goose)
	go run ./apps/api migrate up

migrate-down:  ## Revierte una migración
	go run ./apps/api migrate down 1

dev-key:       ## Genera clave privada JWT de desarrollo (PKCS8 PEM)
	mkdir -p deploy/keys
	openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out deploy/keys/jwt_private_dev.pem 2>/dev/null
	openssl pkcs8 -topk8 -nocrypt -in deploy/keys/jwt_private_dev.pem -out deploy/keys/jwt_private_dev.pem.tmp 2>/dev/null
	mv deploy/keys/jwt_private_dev.pem.tmp deploy/keys/jwt_private_dev.pem
	@echo "Clave generada en deploy/keys/jwt_private_dev.pem"

api:           ## Ejecuta el API en modo desarrollo
	go run ./apps/api

worker:        ## Ejecuta el worker (outbox, pdf, envíos)
	go run ./apps/worker

test-back:     ## Tests backend con -race
	go test -race ./...

lint:          ## Lint backend y web
	golangci-lint run ./...
	cd apps/web && npm run lint

build:         ## Compila backend y web
	go build ./...
	cd apps/web && npm run build

## ── Frontend ───────────────────────────────────────────────
dev-web:       ## Vite dev server
	cd apps/web && npm run dev

test-web:      ## Vitest + Testing Library
	cd apps/web && npm run test

e2e:           ## Playwright
	cd apps/web && npm run test:e2e

## ── Contrato ───────────────────────────────────────────────
check-openapi: ## Valida el OpenAPI generando el cliente (falla si el contrato o el cliente quedan desactualizados)
	cd apps/web && npm run generate:client
	git diff --exit-code -- api/openapi.yaml apps/web/src/api/generated.d.ts

## ── Help ───────────────────────────────────────────────────
help:          ## Muestra esta ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
