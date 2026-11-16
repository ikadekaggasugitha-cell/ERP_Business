-- name: CreateCostCenter :one
INSERT INTO cc.cost_center (organization_id, code, name, description, parent_id, owner_ref, status, effective_from, effective_to, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, row_version;

-- name: GetCostCenter :one
SELECT * FROM cc.cost_center WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCostCenters :many
SELECT * FROM cc.cost_center
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountCostCenters :one
SELECT COUNT(*) FROM cc.cost_center
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: UpdateCostCenter :one
UPDATE cc.cost_center
SET name = $3, description = $4, parent_id = $5, owner_ref = $6, status = $7,
    effective_from = $8, effective_to = $9, updated_by = $10
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteCostCenter :execrows
UPDATE cc.cost_center SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateProfitCenter :one
INSERT INTO pc.profit_center (organization_id, code, name, parent_id, status, effective_from, effective_to, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, row_version;

-- name: GetProfitCenter :one
SELECT * FROM pc.profit_center WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProfitCenters :many
SELECT * FROM pc.profit_center
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountProfitCenters :one
SELECT COUNT(*) FROM pc.profit_center
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: UpdateProfitCenter :one
UPDATE pc.profit_center
SET name = $3, parent_id = $4, status = $5, effective_from = $6, effective_to = $7, updated_by = $8
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteProfitCenter :execrows
UPDATE pc.profit_center SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateAllocationRule :one
INSERT INTO cc.cost_allocation_rule (organization_id, source_cost_center_id, name, allocation_basis, targets, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: GetAllocationRule :one
SELECT * FROM cc.cost_allocation_rule WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAllocationRules :many
SELECT * FROM cc.cost_allocation_rule
WHERE deleted_at IS NULL
ORDER BY created_at
LIMIT $1 OFFSET $2;

-- name: CountAllocationRules :one
SELECT COUNT(*) FROM cc.cost_allocation_rule WHERE deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO cc.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM cc.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE cc.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

