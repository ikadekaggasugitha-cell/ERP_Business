-- name: CreateConsolidationGroup :one
INSERT INTO cons.consolidation_group (organization_id, code, name, parent_group_id, presentation_currency_code, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: GetConsolidationGroup :one
SELECT * FROM cons.consolidation_group WHERE id = $1 AND deleted_at IS NULL;

-- name: ListConsolidationGroups :many
SELECT * FROM cons.consolidation_group WHERE deleted_at IS NULL ORDER BY code LIMIT $1 OFFSET $2;

-- name: CountConsolidationGroups :one
SELECT COUNT(*) FROM cons.consolidation_group WHERE deleted_at IS NULL;

-- name: UpdateConsolidationGroup :one
UPDATE cons.consolidation_group
SET name = $3, parent_group_id = $4, presentation_currency_code = $5, updated_by = $6
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteConsolidationGroup :execrows
UPDATE cons.consolidation_group SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateConsolidationMember :one
INSERT INTO cons.consolidation_member (
    organization_id, consolidation_group_id, company_ref, ownership_percentage, consolidation_method, created_by
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: ListConsolidationMembers :many
SELECT * FROM cons.consolidation_member
WHERE consolidation_group_id = $1 AND deleted_at IS NULL
ORDER BY created_at;

-- name: DeleteConsolidationMember :execrows
UPDATE cons.consolidation_member SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateEliminationRule :one
INSERT INTO cons.elimination_rule (organization_id, consolidation_group_id, rule_type, config, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: ListEliminationRules :many
SELECT * FROM cons.elimination_rule
WHERE deleted_at IS NULL
  AND (sqlc.narg('consolidation_group_id')::uuid IS NULL OR consolidation_group_id = sqlc.narg('consolidation_group_id')::uuid)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountEliminationRules :one
SELECT COUNT(*) FROM cons.elimination_rule
WHERE deleted_at IS NULL
  AND (sqlc.narg('consolidation_group_id')::uuid IS NULL OR consolidation_group_id = sqlc.narg('consolidation_group_id')::uuid);

-- name: CreateConsolidationRun :one
INSERT INTO cons.consolidation_run (organization_id, consolidation_group_id, fiscal_period_ref, status, version, created_by)
VALUES ($1, $2, $3, 'Draft', $4, $5)
RETURNING id, status, row_version;

-- name: GetConsolidationRun :one
SELECT * FROM cons.consolidation_run WHERE id = $1 AND deleted_at IS NULL;

-- name: ListConsolidationRuns :many
SELECT * FROM cons.consolidation_run
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountConsolidationRuns :one
SELECT COUNT(*) FROM cons.consolidation_run WHERE deleted_at IS NULL;

-- name: TransitionConsolidationRun :one
UPDATE cons.consolidation_run
SET status = $3, updated_by = $4
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateEliminationEntry :one
INSERT INTO cons.elimination_entry (
    organization_id, consolidation_run_id, elimination_rule_id, account_ref, debit_amount, credit_amount, description, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, row_version;

-- name: ListEliminationEntriesByRun :many
SELECT * FROM cons.elimination_entry WHERE consolidation_run_id = $1 AND deleted_at IS NULL ORDER BY created_at;

-- name: CreateConsolidationAdjustment :one
INSERT INTO cons.consolidation_adjustment (
    organization_id, consolidation_run_id, account_ref, debit_amount, credit_amount, adjustment_type, notes, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, row_version;

-- name: ListConsolidationAdjustmentsByRun :many
SELECT * FROM cons.consolidation_adjustment WHERE consolidation_run_id = $1 AND deleted_at IS NULL ORDER BY created_at;

-- name: InsertOutbox :exec
INSERT INTO cons.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM cons.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE cons.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
