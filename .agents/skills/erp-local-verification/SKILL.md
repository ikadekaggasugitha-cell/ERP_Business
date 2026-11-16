---
name: erp-local-verification
description: >-
  Run local verification gates across the erp.business workspace (Go microservices,
  sqlc diff, local Dataform compile, React 19 SPA build/lint/tests, Chrome extension
  tests, and Docker Compose E2E smoke tests). Use when verifying changes across
  any ERP phase or diagnosing local build/disk issues.
---

# ERP Local Verification Runbook

## 1. Go Microservices (`services/`)
- On hosts with tight disk space (e.g., macOS `/System/Volumes/Data` >= 95%), run `go clean -cache -testcache` first and pass `-p 2` to prevent parallel linker `no space left on device` errors.
- Validate all 22 Go modules:
  ```bash
  for dir in services/*; do
    [ -f "$dir/go.mod" ] || continue
    go -C "$dir" vet ./...
    [ -f "$dir/sqlc.yaml" ] && sqlc diff -f "$dir/sqlc.yaml"
    go -C "$dir" test -p 2 -count=1 ./...
  done
  ```
- Re-run changed services with `-race`: `go -C services/<service> test -p 2 -count=1 -race ./...`

## 2. Dataform (`src/data_modules/custom/`) — Local Compile Only
- Never execute `dataform run` against BigQuery from verification runs.
  ```bash
  ./src/data_modules/custom/node_modules/.bin/dataform compile ./src/data_modules/custom
  ./src/data_modules/custom/node_modules/.bin/dataform compile ./src/data_modules/custom --json
  ```
- Confirm exit code `0` and `graphErrors.compilationErrors` is empty (`32` datasets, `4` assertions, `31` declarations).

## 3. Web SPA (`apps/web-spa/`) & Chrome Companion (`apps/chrome-companion/`)
- Web SPA:
  ```bash
  cd apps/web-spa
  ./node_modules/.bin/tsc --noEmit
  ./node_modules/.bin/eslint .
  ./node_modules/.bin/vite build
  node --experimental-strip-types --test test/*.test.js
  ```
- Chrome Companion:
  ```bash
  node --test apps/chrome-companion/test/companion.test.js
  ```

## 4. Docker Compose & Smoke Suite (`deploy/smoke-test.sh`)
- Validate compose & script syntax statically first (`docker compose config -q`, `bash -n deploy/smoke-test.sh`).
- Ensure Docker Desktop VM has sufficient host disk space, then bring up services and run the 88-check E2E suite:
  ```bash
  docker compose up -d --build
  bash deploy/smoke-test.sh
  ```

