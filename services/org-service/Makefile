# org-service — dev & migration pipeline (mirrors ap-service; swap DB + migrations).
# Requires: go, sqlc, golang-migrate (`migrate`). Docker equivalents in comments.
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/org_db?sslmode=disable
APP_URL      ?= postgres://erp_app_rw@localhost:5432/org_db?sslmode=disable
MIGRATIONS   := ../../db/migrations/org_db

.PHONY: sqlc sqlc-verify migrate-up migrate-down migrate-version grants seed tidy build test run

sqlc:            ## regenerate type-safe queries from SQL
	sqlc generate      # or: docker run --rm -v "$$PWD":/src -w /src sqlc/sqlc generate

sqlc-verify:     ## fail if generated code is stale (CI gate)
	sqlc diff          # or: docker run --rm -v "$$PWD":/src -w /src sqlc/sqlc diff

migrate-up:      ## apply all migrations (expand phase)
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down:    ## roll back one migration
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

migrate-version: ## show current schema version
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" version

grants:          ## apply least-privilege grants (run after migrate-up)
	psql "$(DATABASE_URL)" -f ../../db/ops/apply_grants.sql

seed:            ## load reference data (idempotent; run after migrate-up as owner)
	psql "$(DATABASE_URL)" -f ../../db/seeds/org_db/seed.sql

tidy:            ## resolve module deps
	go mod tidy

build:           ## compile
	go build ./...

test:            ## unit tests
	go test ./...

run:             ## run the service (connects as erp_app_rw so RLS applies)
	DATABASE_URL="$(APP_URL)" go run ./cmd/server
