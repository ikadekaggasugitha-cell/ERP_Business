-- name: CreateBudget :one
INSERT INTO bud.budget (
    organization_id, code, name, financial_organization_ref, fiscal_year_ref, version, status, is_active, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, status, row_version;

-- name: GetBudget :one
SELECT * FROM bud.budget WHERE id = $1 AND deleted_at IS NULL;

-- name: GetActiveApprovedBudget :one
SELECT * FROM bud.budget
WHERE organization_id = $1 AND status = 'Approved' AND is_active = TRUE AND deleted_at IS NULL
ORDER BY version DESC, created_at DESC
LIMIT 1;

-- name: ListBudgets :many
SELECT * FROM bud.budget
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY code, version DESC
LIMIT $1 OFFSET $2;

-- name: CountBudgets :one
SELECT COUNT(*) FROM bud.budget
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: TransitionBudget :one
UPDATE bud.budget
SET status = $3, is_active = $4, updated_by = $5
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteDraftBudget :execrows
UPDATE bud.budget SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL;

-- name: CreateBudgetLine :one
INSERT INTO bud.budget_line (
    organization_id, budget_id, account_ref, cost_center_ref, project_ref, period_no, budget_amount, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, row_version;

-- name: ListBudgetLines :many
SELECT * FROM bud.budget_line
WHERE budget_id = $1 AND deleted_at IS NULL
ORDER BY period_no, account_ref;

-- name: SumBudgetLines :one
SELECT COALESCE(SUM(budget_amount), 0)::numeric(20,4) AS total_budget
FROM bud.budget_line
WHERE budget_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('account_ref')::uuid IS NULL OR account_ref = sqlc.narg('account_ref')::uuid)
  AND (sqlc.narg('period_no')::int IS NULL OR period_no = sqlc.narg('period_no')::int);

-- name: CreateBudgetPolicy :one
INSERT INTO bud.budget_control_policy (organization_id, account_ref, transaction_type, control_type, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, row_version;

-- name: ListBudgetPolicies :many
SELECT * FROM bud.budget_control_policy WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountBudgetPolicies :one
SELECT COUNT(*) FROM bud.budget_control_policy WHERE deleted_at IS NULL;

-- name: ResolveBudgetControlType :one
SELECT control_type FROM bud.budget_control_policy
WHERE organization_id = $1 AND deleted_at IS NULL
  AND (account_ref IS NULL OR account_ref = sqlc.narg('account_ref')::uuid)
  AND (transaction_type IS NULL OR transaction_type = sqlc.narg('transaction_type')::text)
ORDER BY (account_ref IS NOT NULL) DESC, (transaction_type IS NOT NULL) DESC, created_at DESC
LIMIT 1;

-- name: CreateBudgetCommitment :one
INSERT INTO bud.budget_commitment (
    organization_id, budget_id, source_document_ref, committed_amount, status, created_by
) VALUES ($1, $2, $3, $4, 'Open', $5)
RETURNING id, status, row_version;

-- name: ListBudgetCommitments :many
SELECT * FROM bud.budget_commitment
WHERE deleted_at IS NULL
  AND (sqlc.narg('budget_id')::uuid IS NULL OR budget_id = sqlc.narg('budget_id')::uuid)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountBudgetCommitments :one
SELECT COUNT(*) FROM bud.budget_commitment
WHERE deleted_at IS NULL
  AND (sqlc.narg('budget_id')::uuid IS NULL OR budget_id = sqlc.narg('budget_id')::uuid);

-- name: SumActiveCommitments :one
SELECT COALESCE(SUM(committed_amount), 0)::numeric(20,4) AS total_committed
FROM bud.budget_commitment
WHERE budget_id = $1 AND status IN ('Open', 'Consumed') AND deleted_at IS NULL;

-- name: TransitionBudgetCommitment :one
UPDATE bud.budget_commitment
SET status = $3, updated_by = $4
WHERE id = $1 AND row_version = $2 AND status = 'Open' AND deleted_at IS NULL
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO bud.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM bud.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE bud.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO bud.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;
