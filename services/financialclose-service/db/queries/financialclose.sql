-- name: CreateChecklistTemplate :one
INSERT INTO fcl.closing_checklist_template (organization_id, code, name, financial_organization_ref, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, row_version;

-- name: CreateTaskTemplate :one
INSERT INTO fcl.closing_task_template (organization_id, template_id, task_order, task_name, is_mandatory, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: ListChecklistTemplates :many
SELECT * FROM fcl.closing_checklist_template WHERE deleted_at IS NULL ORDER BY code LIMIT $1 OFFSET $2;

-- name: CountChecklistTemplates :one
SELECT COUNT(*) FROM fcl.closing_checklist_template WHERE deleted_at IS NULL;

-- name: ListTaskTemplatesByTemplate :many
SELECT * FROM fcl.closing_task_template WHERE template_id = $1 AND deleted_at IS NULL ORDER BY task_order;

-- name: CreateClosingRun :one
INSERT INTO fcl.closing_run (organization_id, financial_organization_ref, fiscal_period_ref, run_type, status, created_by)
VALUES ($1, $2, $3, $4, 'Open', $5)
RETURNING id, status, row_version;

-- name: GetClosingRun :one
SELECT * FROM fcl.closing_run WHERE id = $1 AND deleted_at IS NULL;

-- name: ListClosingRuns :many
SELECT * FROM fcl.closing_run
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountClosingRuns :one
SELECT COUNT(*) FROM fcl.closing_run
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: CreateClosingTask :one
INSERT INTO fcl.closing_task (organization_id, closing_run_id, task_name, is_mandatory, assignee_ref, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, status, row_version;

-- name: ListClosingTasksByRun :many
SELECT * FROM fcl.closing_task WHERE closing_run_id = $1 AND deleted_at IS NULL ORDER BY created_at;

-- name: CountIncompleteMandatoryTasks :one
SELECT COUNT(*) FROM fcl.closing_task
WHERE closing_run_id = $1 AND is_mandatory = TRUE AND status <> 'Completed' AND deleted_at IS NULL;

-- name: CompleteClosingTask :one
UPDATE fcl.closing_task
SET status = 'Completed', completed_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status <> 'Completed' AND deleted_at IS NULL
RETURNING row_version;

-- name: TransitionClosingRun :one
UPDATE fcl.closing_run
SET status = $3, closed_at = CASE WHEN $3 IN ('Closed','Locked') THEN COALESCE(closed_at, now()) ELSE closed_at END,
    closed_by = CASE WHEN $3 IN ('Closed','Locked') THEN COALESCE(closed_by, $4) ELSE closed_by END,
    updated_by = $4
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: UpsertPeriodStatus :one
INSERT INTO fcl.period_status (organization_id, financial_organization_ref, fiscal_period_ref, module, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (organization_id, financial_organization_ref, fiscal_period_ref, module) DO UPDATE
SET status = EXCLUDED.status, updated_by = EXCLUDED.created_by, deleted_at = NULL
RETURNING id, status, row_version;

-- name: ListPeriodStatuses :many
SELECT * FROM fcl.period_status WHERE deleted_at IS NULL ORDER BY module LIMIT $1 OFFSET $2;

-- name: CountPeriodStatuses :one
SELECT COUNT(*) FROM fcl.period_status WHERE deleted_at IS NULL;

-- name: CreatePeriodReopening :one
INSERT INTO fcl.period_reopening (organization_id, financial_organization_ref, fiscal_period_ref, requested_by, reason, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: GetPeriodReopening :one
SELECT * FROM fcl.period_reopening WHERE id = $1 AND deleted_at IS NULL;

-- name: ApprovePeriodReopening :one
UPDATE fcl.period_reopening
SET approved_by = $3, reopened_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND approved_by IS NULL AND deleted_at IS NULL
RETURNING row_version;

-- name: ListPeriodReopenings :many
SELECT * FROM fcl.period_reopening WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountPeriodReopenings :one
SELECT COUNT(*) FROM fcl.period_reopening WHERE deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO fcl.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM fcl.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE fcl.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
