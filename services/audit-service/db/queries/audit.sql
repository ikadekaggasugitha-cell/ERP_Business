-- name: GetAuditLog :one
SELECT * FROM audit.audit_log
WHERE id = $1 AND occurred_at = $2;

-- name: GetAuditLogByID :one
SELECT * FROM audit.audit_log
WHERE id = $1
ORDER BY occurred_at DESC
LIMIT 1;

-- Recent audit entries visible to the caller's scope (RLS-filtered), paginated.
-- name: ListAuditLogs :many
SELECT id, occurred_at, actor_id, action, entity_type, entity_id, correlation_id
FROM audit.audit_log
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;

-- Filtered list of audit entries with optional entity_type / action / time-range / entity_id / correlation_id predicates.
-- name: ListAuditLogsFiltered :many
SELECT id, occurred_at, organization_id, actor_id, action, entity_type, entity_id, correlation_id
FROM audit.audit_log
WHERE (sqlc.arg(entity_type)::text = '' OR entity_type = sqlc.arg(entity_type)::text)
  AND (sqlc.arg(action)::text = '' OR action = sqlc.arg(action)::text)
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR occurred_at >= sqlc.narg(from_ts)::timestamptz)
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR occurred_at <= sqlc.narg(to_ts)::timestamptz)
  AND (sqlc.narg(entity_id)::uuid IS NULL OR entity_id = sqlc.narg(entity_id)::uuid)
  AND (sqlc.narg(correlation_id)::uuid IS NULL OR correlation_id = sqlc.narg(correlation_id)::uuid)
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;

-- Count matching audit entries inside the caller's RLS scope for pagination metadata.
-- name: CountAuditLogsFiltered :one
SELECT COUNT(*)
FROM audit.audit_log
WHERE (sqlc.arg(entity_type)::text = '' OR entity_type = sqlc.arg(entity_type)::text)
  AND (sqlc.arg(action)::text = '' OR action = sqlc.arg(action)::text)
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR occurred_at >= sqlc.narg(from_ts)::timestamptz)
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR occurred_at <= sqlc.narg(to_ts)::timestamptz)
  AND (sqlc.narg(entity_id)::uuid IS NULL OR entity_id = sqlc.narg(entity_id)::uuid)
  AND (sqlc.narg(correlation_id)::uuid IS NULL OR correlation_id = sqlc.narg(correlation_id)::uuid);

-- name: ListAuditLogsByEntity :many
SELECT id, occurred_at, actor_id, action, entity_type, entity_id, correlation_id
FROM audit.audit_log
WHERE organization_id = $1 AND entity_type = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- Append path of the consumer: persists one consumed domain event into the
-- append-only audit trail. RETURNING the composite PK (id, occurred_at).
-- name: RecordAuditLog :one
INSERT INTO audit.audit_log (
    occurred_at, organization_id, actor_id, action, entity_type,
    entity_id, before_value, after_value, reason, correlation_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, occurred_at;

-- Consumer idempotency ledger. Returns rows affected (0 = already processed);
-- ON CONFLICT DO NOTHING makes re-delivery a no-op.
-- name: MarkEventProcessed :execrows
INSERT INTO audit.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;
