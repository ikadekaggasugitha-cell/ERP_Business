# Go Microservices — HTTP & Kafka Handler Conventions

Standard Operating Procedure (SOP) for authoring and maintaining HTTP handlers (`internal/httpapi/`) and Kafka consumers (`internal/consumer/`) across all 30 ERP Go microservices (`services/`).

---

## 1. File Organization (`internal/httpapi/`)

- Keep every file in `internal/httpapi/` **under 800 lines** (target ~250–500 lines per file).
- `handlers.go` owns only:
  - The `Store` interface (for unit-test decoupling).
  - The `Server` / `Handler` struct and `NewServer` / `NewRouter` constructor.
  - Route registration (`Register(mux *http.ServeMux)`).
  - Tiny shared helper functions if not placed in `dto.go`.
- Split domain endpoints into focused per-resource files (e.g., `employees.go`, `leave_attendance.go`, `organizations.go`, `attributes_units.go`, `dto.go`).
- Never return raw `sqlc` structs containing `pgtype.Numeric`, `pgtype.Date`, or `pgtype.Timestamptz` directly to clients — map to explicit `camelCase` DTO structs with fixed-point decimal strings (`platform.FormatNumeric`) and ISO-8601 dates (`platform.FormatDate`). Initialize list slices with `make([]DTO, 0, len(rows))` so empty lists serialize as `[]`, never `null`.

---

## 2. Shared Middleware & Trust Boundary (`services/shared/httpx`)

Every microservice wraps its `http.ServeMux` with `metrics.Wrap(httpx.Middleware(mux))`:

1. **Request Timeout & Panic Recovery**: `httpx.Middleware` enforces a bounded request context timeout (30s default via `ServeHTTP` read/write timeouts) and recovers panics into a masked `500 INTERNAL_ERROR` JSON envelope with `correlationId`.
2. **Distributed Tracing (`W3C traceparent`)**: `httpx.Middleware` parses/propagates W3C `traceparent`/`tracestate` headers via `services/shared/telemetry` and logs `traceId` alongside `X-Correlation-Id`.
3. **Gateway HMAC Trust Boundary**: When `GATEWAY_HMAC_SECRET` is configured, `api-gateway` signs injected identity headers (`X-User-Id`, `X-Organization-Id`, `X-Org-Scope`, `X-Roles`, `X-Permissions`) via `httpx.SignGatewayRequest`, and `httpx.Middleware` verifies `X-Gateway-Signature` before trusting caller headers.

---

## 3. Auth, Multi-Tenancy (RLS) & RBAC

Every non-health endpoint MUST enforce scope and permission checks at the top of the handler:

- **Tenant Scope**:
  - Use `scope, err := httpx.RequireScope(r)` for read paths (fail-closed `401` if absent).
  - Use `orgID, scope, err := httpx.RequireTenantOrg(r)` on tenant writes to block `platform.SystemOrgID` (`00000000-0000-0000-0000-000000000000`) mutations with `403 FORBIDDEN_SCOPE`.
  - Use `platform.WithOrgReadScope` / `platform.ScopeWithSystemOrg` when reading seeded system-reference data (e.g., currencies, tax codes, org types, languages, categories), and `platform.WithTenantWriteScope` on writes.
- **Permission Check**:
  - Call `httpx.RequirePermission(r, "<module>:<action>")` (e.g., `hr:read`, `hr:write`, `ap:approve`).
  - Enforce `REQUIRE_RBAC=true` in staging/production so missing gateway role headers fail closed with `403`.
- **3-Tier Admin Hierarchy**:
  - Gate privileged cross-tenant or tier-restricted actions (`HOLDING_ADMIN` tier 1, `SUBSIDIARY_ADMIN` tier 2, `BRANCH_ADMIN` tier 3) with `httpx.RequireMaxAdminTier(r, maxTier)` and `httpx.CanDelegateTier(callerTier, targetTier)`.
- **Separation of Duties (SoD)**:
  - Approval/posting endpoints must verify `actorID != record.CreatedBy` (when `CreatedBy != uuid.Nil`).

---

## 4. Request Parsing, Idempotency & Optimistic Locking

- **Path UUIDs**: Use `httpx.ParsePathUUID(r, "id")` (or `httpx.ParseUUIDParam(field, raw)`) — returns structured `400 VALIDATION_ERROR`.
- **JSON Decoding**:
  - Use `httpx.DecodeJSON(w, r, &req)` (1 MiB `MaxBytesReader`, `DisallowUnknownFields`, rejects trailing tokens).
  - Use `httpx.DecodeOptionalJSON(w, r, &req)` for action endpoints where body is optional.
- **Mutating `POST` Idempotency**:
  - Always guard `POST` create and state-transition endpoints with `httpx.BeginIdempotent(w, r, s.idem, scope, &req)` + `defer release()` and finalize with `httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, status, version, resp)`.
  - `IdemStore` persists to PostgreSQL `public.idempotency_key` (pruned hourly by `db/ops/prune_idempotency_keys.sql`) and scopes keys by `(scope, method, path, Idempotency-Key)`.
- **Optimistic Locking (`ETag` / `If-Match`)**:
  - Return `ETag` on `GET /{resource}/{id}` and mutations via `httpx.WriteIDResponse(w, status, rowVersion, dto)`.
  - Parse required `If-Match` on `PUT`, `DELETE`, and state-transition `POST` endpoints via `httpx.IfMatch(r)` (`428 PRECONDITION_REQUIRED` if missing, `409 VERSION_CONFLICT` on stale version).
- **Fixed-Point Monetary Arithmetic**:
  - Never use `float64` for money, rates, or quantities. Parse and compute with `*big.Rat` via `services/shared/platform` decimal helpers.

---

## 5. CSV Exports & Formula-Injection Protection

- Never define local `sanitizeCSVCell` helpers in service packages.
- Always use **`httpx.SanitizeCSVCell(v)`** (or **`httpx.WriteCSV(w, filename, headers, rows)`**) from `services/shared/httpx/helpers.go`, which neutralizes leading `=`, `+`, `-`, `@`, `\t`, `\r` formula characters while preserving valid negative decimal strings (e.g., `-500.0000`).

---

## 6. Kafka Consumers & Transactional Outbox

- **Outbox Relay (`DrainAndPublish`)**:
  - Open transactions with `r.db.Pool.Begin(ctx)` (never `WithOrgScope` — outbox partitions do not enforce RLS).
  - Return early when `len(rows) == 0`.
  - On partial publish error, `break` and commit already-published rows rather than rolling back the batch.
- **Consumers**:
  - Record processed `event_id` in `<schema>.processed_event` inside the same DB transaction as the state mutation.
  - Return `eventbus.ErrPoison{Err: ...}` for deterministic malformed/invalid payloads so `eventbus.Consumer` routes them to `<topic>.dlq` without blocking partition progress.
  - Use `services/shared/cmd/dlq-replay` to inspect or redrive dead-lettered messages.

---

## 7. Verification Gate

Before merging any handler change, run:

```bash
cd services/<service-name>
go vet ./...
go test -race -count=1 ./...
```
