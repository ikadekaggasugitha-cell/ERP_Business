-- name: CreateLead :one
INSERT INTO crm.lead (
    organization_id, code, name, source, status, estimated_value, customer_ref, created_by
) VALUES ($1, $2, $3, $4, 'New', $5, $6, $7)
RETURNING id, row_version;

-- name: GetLead :one
SELECT * FROM crm.lead WHERE id = $1 AND deleted_at IS NULL;

-- name: ListLeads :many
SELECT * FROM crm.lead
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountLeads :one
SELECT count(*) FROM crm.lead
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: UpdateLead :one
UPDATE crm.lead
SET name = $2, source = $3, estimated_value = $4, customer_ref = $5, updated_by = $6
WHERE id = $1 AND row_version = $7 AND status IN ('New', 'Qualified') AND deleted_at IS NULL
RETURNING row_version;

-- name: TransitionLeadStatus :one
UPDATE crm.lead
SET status = $2, customer_ref = COALESCE($3, customer_ref), updated_by = $4
WHERE id = $1 AND row_version = $5 AND status = $6 AND deleted_at IS NULL
RETURNING row_version;

-- name: LoseLead :one
UPDATE crm.lead
SET status = 'Lost', updated_by = $2
WHERE id = $1 AND row_version = $3 AND status IN ('New', 'Qualified') AND deleted_at IS NULL
RETURNING row_version;

-- name: SoftDeleteLead :execrows
UPDATE crm.lead
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND deleted_at IS NULL;

-- name: CreateOpportunity :one
INSERT INTO crm.opportunity (
    organization_id, code, lead_id, customer_ref, stage,
    probability_pct, estimated_value, expected_close_date, status, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'Open', $9)
RETURNING id, row_version;

-- name: GetOpportunity :one
SELECT * FROM crm.opportunity WHERE id = $1 AND deleted_at IS NULL;

-- name: ListOpportunities :many
SELECT * FROM crm.opportunity
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountOpportunities :one
SELECT count(*) FROM crm.opportunity
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: UpdateOpportunity :one
UPDATE crm.opportunity
SET stage = $2, probability_pct = $3, estimated_value = $4, expected_close_date = $5, customer_ref = $6, updated_by = $7
WHERE id = $1 AND row_version = $8 AND status = 'Open' AND deleted_at IS NULL
RETURNING row_version;

-- name: CloseOpportunity :one
UPDATE crm.opportunity
SET status = $2, updated_by = $3
WHERE id = $1 AND row_version = $4 AND status = 'Open' AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateCustomerSegment :one
INSERT INTO crm.customer_segment (
    organization_id, code, name, criteria, created_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, row_version;

-- name: GetCustomerSegment :one
SELECT * FROM crm.customer_segment WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCustomerSegments :many
SELECT * FROM crm.customer_segment
WHERE deleted_at IS NULL
ORDER BY code ASC
LIMIT $1 OFFSET $2;

-- name: CountCustomerSegments :one
SELECT count(*) FROM crm.customer_segment WHERE deleted_at IS NULL;

-- name: UpdateCustomerSegment :one
UPDATE crm.customer_segment
SET name = $2, criteria = $3, updated_by = $4
WHERE id = $1 AND row_version = $5 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: SoftDeleteCustomerSegment :execrows
UPDATE crm.customer_segment
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateInteraction :one
INSERT INTO crm.interaction (
    organization_id, customer_ref, lead_id, interaction_type, occurred_at, notes, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: ListInteractions :many
SELECT * FROM crm.interaction
WHERE deleted_at IS NULL
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;

-- name: CountInteractions :one
SELECT count(*) FROM crm.interaction WHERE deleted_at IS NULL;

-- name: CreateSupportCase :one
INSERT INTO crm.support_case (
    organization_id, case_no, customer_ref, subject, status, created_by
) VALUES ($1, $2, $3, $4, 'Open', $5)
RETURNING id, row_version;

-- name: GetSupportCase :one
SELECT * FROM crm.support_case WHERE id = $1 AND deleted_at IS NULL;

-- name: ListSupportCases :many
SELECT * FROM crm.support_case
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY opened_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSupportCases :one
SELECT count(*) FROM crm.support_case
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: EscalateSupportCase :one
UPDATE crm.support_case
SET status = 'Escalated', updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Open' AND deleted_at IS NULL
RETURNING row_version;

-- name: ResolveSupportCase :one
UPDATE crm.support_case
SET status = 'Resolved', resolution = $2, resolved_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $4 AND status IN ('Open', 'Escalated') AND deleted_at IS NULL
RETURNING row_version;

-- name: CloseSupportCase :one
UPDATE crm.support_case
SET status = 'Closed', updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Resolved' AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateActivityTask :one
INSERT INTO crm.activity_task (
    organization_id, assignee_ref, related_type, related_ref, due_date, status, created_by
) VALUES ($1, $2, $3, $4, $5, 'Pending', $6)
RETURNING id, row_version;

-- name: ListActivityTasks :many
SELECT * FROM crm.activity_task
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY due_date ASC NULLS LAST, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountActivityTasks :one
SELECT count(*) FROM crm.activity_task
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: CompleteActivityTask :one
UPDATE crm.activity_task
SET status = 'Done', updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Pending' AND deleted_at IS NULL
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO crm.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ClaimUnpublishedOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM crm.outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE crm.outbox SET published_at = now() WHERE id = $1 AND created_at = $2;
