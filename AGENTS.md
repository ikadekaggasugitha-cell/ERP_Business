# ERP Business Workspace Guardrails

## 1. Frontend & Documentation Standard (CONS-013)
- The ratified Backoffice SPA stack is **React 19 + Vite + TanStack (Router, Query v5, Virtual) + Zustand + Tailwind CSS v4**.
- Never introduce or reintroduce Nuxt 3 / Vue references into application code or `docs/`, `PROGRESS.md`, `CHANGELOG.md`, or `BLUEPRINT.md`.
- Preserve fixed-point decimal string arithmetic (`apps/web-spa/src/lib/decimal.ts`) for monetary amounts, exchange rates, and quantities. Never compute ledger totals with IEEE-754 floats.

## 2. Cortex Dataform (`src/data_modules/custom/`)
- Keep `@dataform/cli` and `@dataform/core` inside local `package.json` (`dependencies` / `devDependencies`), not globally installed.
- **Do NOT set `dataformCoreVersion` in `workflow_settings.yaml`**: `@dataform/cli` uses `createIsolatedEnvironment` when `dataformCoreVersion` is present and rejects local `node_modules/`. Omitting `dataformCoreVersion` triggers `useProjectEnvironment(projectDir)` for offline compilation.
- Keep `.sqlx` as the primary format for models and assertions; use `.js` (`declare()` / `publish()`) strictly for genuine dynamic loops or complex conditional logic.
- Always verify locally with `./node_modules/.bin/dataform compile`. Never run `dataform run` or deploy against live BigQuery without explicit user confirmation.

## 3. Go Microservices & Dev API Gateway (`services/`)
- **Transactional Outbox (`DrainAndPublish`)**: Outbox partition tables do not enforce RLS; always open transactions with `r.db.Pool.Begin(ctx)` (never `WithOrgScope(nil)`), return early when `len(rows) == 0`, and `break` + commit already-published rows on partial batch errors instead of returning early and rolling back the batch.
- **`currency-service` System Scope**: For non-empty `orgScope`, include system organization UUID `00000000-0000-0000-0000-000000000000` in RLS scope (`ScopeWithSystemOrg`) so global ISO-4217 currencies and rate types (`SPOT`, `CLOSING`, `AVERAGE`) remain visible on both gateway (`:8000`) and direct (`:8100`) requests while keeping empty scopes failing closed.
- **API Gateway Auth, RBAC, & SSE**:
  - Grant both dot-separated (`gl.*`, `reporting.read`) and colon-separated (`ap:read`, `currency:read`, etc.) permissions in `rolePermissions` to match downstream `RequirePermission` checks.
  - Keep `services/api-gateway/internal/sse/hub.go` `DomainTopics` synchronized with `services/audit-service/internal/consumer/consumer.go`.
  - Browser SSE clients must connect to `/api/v1/stream/events` using a short-lived `ticket` from `POST /api/v1/auth/stream-ticket` or the `HttpOnly` session cookie—never place long-lived JWTs in URL query parameters.

