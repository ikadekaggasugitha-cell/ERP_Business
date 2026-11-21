-- name: CreateProject :one
INSERT INTO prj.project (
    organization_id, code, name, project_type, organization_ref,
    currency_code, start_date, end_date, status, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 'Draft', $9
) RETURNING id, row_version;

-- name: GetProject :one
SELECT * FROM prj.project WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjects :many
SELECT * FROM prj.project
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY code ASC
LIMIT $1 OFFSET $2;

-- name: CountProjects :one
SELECT count(*) FROM prj.project
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: UpdateProject :one
UPDATE prj.project
SET name = $2, project_type = $3, start_date = $4, end_date = $5, updated_by = $6
WHERE id = $1 AND row_version = $7 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: TransitionProjectStatus :one
UPDATE prj.project
SET status = $2, updated_by = $3
WHERE id = $1 AND row_version = $4 AND status = $5 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: SoftDeleteProject :execrows
UPDATE prj.project
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Draft' AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: ShiftProjectEndDate :exec
UPDATE prj.project
SET end_date = (end_date + ($2 * interval '1 day'))::date, updated_by = $3
WHERE id = $1 AND end_date IS NOT NULL AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateWBSElement :one
INSERT INTO prj.wbs_element (
    organization_id, project_id, parent_id, code, name, level_no, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: GetWBSElement :one
SELECT * FROM prj.wbs_element WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWBSElementsByProject :many
SELECT * FROM prj.wbs_element
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY level_no ASC, code ASC;

-- name: UpdateWBSElement :one
UPDATE prj.wbs_element
SET parent_id = $2, name = $3, level_no = $4, updated_by = $5
WHERE id = $1 AND row_version = $6 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: CountWBSChildren :one
SELECT count(*) FROM prj.wbs_element
WHERE parent_id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteWBSElement :execrows
UPDATE prj.wbs_element
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateProjectBudget :one
INSERT INTO prj.project_budget (
    organization_id, project_id, wbs_element_id, version, budget_amount, status, is_baseline, created_by
) VALUES ($1, $2, $3, $4, $5, 'Draft', $6, $7)
RETURNING id, row_version;

-- name: GetProjectBudget :one
SELECT * FROM prj.project_budget WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjectBudgets :many
SELECT * FROM prj.project_budget
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY version DESC, created_at DESC;

-- name: ApproveProjectBudget :one
UPDATE prj.project_budget
SET status = 'Approved', updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateProjectCost :one
INSERT INTO prj.project_cost (
    organization_id, project_id, wbs_element_id, cost_type, amount, source_document_ref, posting_date, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, row_version;

-- name: ListProjectCosts :many
SELECT * FROM prj.project_cost
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY posting_date DESC, created_at DESC;

-- name: UpsertProjectProgress :one
INSERT INTO prj.project_progress (
    organization_id, wbs_element_id, period_date, physical_progress_pct, created_by
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (wbs_element_id, period_date) DO UPDATE
SET physical_progress_pct = EXCLUDED.physical_progress_pct,
    recorded_at = now(),
    updated_by = EXCLUDED.created_by
RETURNING id, row_version;

-- name: ListProjectProgress :many
SELECT * FROM prj.project_progress
WHERE wbs_element_id = $1 AND deleted_at IS NULL
ORDER BY period_date DESC;

-- name: CreateProjectBilling :one
INSERT INTO prj.project_billing (
    organization_id, project_id, billing_type, amount, ar_invoice_ref, status, created_by
) VALUES ($1, $2, $3, $4, $5, 'Draft', $6)
RETURNING id, row_version;

-- name: GetProjectBilling :one
SELECT * FROM prj.project_billing WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjectBillings :many
SELECT * FROM prj.project_billing
WHERE deleted_at IS NULL
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountProjectBillings :one
SELECT count(*) FROM prj.project_billing
WHERE deleted_at IS NULL
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id'));

-- name: InvoiceProjectBilling :one
UPDATE prj.project_billing
SET status = 'Invoiced', ar_invoice_ref = $2, updated_by = $3
WHERE id = $1 AND row_version = $4 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateChangeOrder :one
INSERT INTO prj.change_order (
    organization_id, project_id, description, budget_impact_amount, schedule_impact_days, status, created_by
) VALUES ($1, $2, $3, $4, $5, 'Draft', $6)
RETURNING id, row_version;

-- name: GetChangeOrder :one
SELECT * FROM prj.change_order WHERE id = $1 AND deleted_at IS NULL;

-- name: ListChangeOrders :many
SELECT * FROM prj.change_order
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: DecideChangeOrder :one
UPDATE prj.change_order
SET status = $2, updated_by = $3
WHERE id = $1 AND row_version = $4 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO prj.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ClaimUnpublishedOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM prj.outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE prj.outbox SET published_at = now() WHERE id = $1 AND created_at = $2;
