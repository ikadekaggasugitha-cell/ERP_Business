# currency-service — Currency Management & Real-Time Triangulation Engine (SVC-010-CUR)

Backend service for `currency_db`, built from the standard platform blueprint.
Stack: **Go 1.23+ pgx v5 + sqlc + golang-migrate + transactional outbox** (ADR-001, ADR-008, ADR-014).

```
services/currency-service/
├── cmd/server/main.go                  # entrypoint: pool + /healthz + /readyz + /metrics + relay
├── db/
│   ├── schema/schema.sql               # sqlc catalog (plain DDL; source of truth = db/migrations/currency_db)
│   └── queries/currency.sql            # SQL with -- name: annotations (sqlc input)
├── internal/
│   ├── currencydb/                     # sqlc-GENERATED — do not edit
│   ├── repository/
│   │   ├── currency_repository.go      # RLS WithOrgScope + Outbox + CRUD
│   │   ├── triangulation.go            # Direct/Inverse/Triangulated USD Anchor rational math
│   │   └── triangulation_test.go       # High-precision rational conversion tests
│   └── httpapi/
│       ├── handlers.go                 # REST endpoints (Currencies, Rate Types, Rates, Convert)
│       └── handlers_test.go            # Comprehensive unit tests (200, 201, 400, 404, 409, 422)
├── sqlc.yaml   Makefile   go.mod   go.sum
```

## Features & Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/currencies` | List active/inactive currencies with pagination (`?isActive=true&page=1&size=25`) |
| `POST` | `/currencies` | Create ISO 4217 currency (`code`, `name`, `precision`, `numericCode`) |
| `GET` | `/currencies/{id}` | Get currency by UUID |
| `PUT` | `/currencies/{id}` | Update currency (`If-Match` optimistic locking) |
| `DELETE` | `/currencies/{id}` | Soft-delete currency (`If-Match` optimistic locking) |
| `GET` | `/exchange-rate-types` | List exchange rate types (`SPOT`, `CLOSING`, `AVERAGE`, `HISTORICAL`, `BUDGET`) |
| `POST` | `/exchange-rate-types` | Create custom exchange rate type |
| `GET` | `/exchange-rates` | List exchange rates (`?from=USD&to=IDR&type=SPOT&onDate=2026-11-13`) |
| `POST` | `/exchange-rates` | Create dated exchange rate with date validation and rate > 0 |
| `PUT` | `/exchange-rates/{id}` | Update exchange rate (`If-Match` optimistic locking) |
| `DELETE` | `/exchange-rates/{id}` | Soft-delete exchange rate (`If-Match` optimistic locking) |
| `POST` | `/exchange-rates/convert`| High-precision financial conversion with automated USD anchor triangulation |

## Triangulation Engine (ADR-014)

Converts amounts between any currency pair via rational arithmetic (`math/big` `Rat`):
1. **Identity**: `from == to` -> rate = 1.00000000
2. **Direct**: Active rate found for `from -> to`
3. **Inverse**: Active rate found for `to -> from` -> rate = `1 / inverse_rate`
4. **Triangulation**: Via anchor currency (default `USD`): `rate = (from -> USD) * (USD -> to)`

## Migration & Pipeline

```bash
make sqlc           # Regenerate type-safe queries
make sqlc-verify    # Fail if generated code is stale (CI gate)
make migrate-up     # Apply database migrations
make seed           # Load reference ISO currencies & rate types
make test           # Run unit & race tests
```

