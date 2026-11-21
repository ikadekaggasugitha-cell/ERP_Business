-- name: CreateWorkflowDefinition :one
INSERT INTO wf.workflow_definition (organization_id, code, document_type, version, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: CreateWorkflowStep :exec
INSERT INTO wf.workflow_step (
    organization_id, definition_id, step_order, approver_rule, escalation_rule, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: ListWorkflowDefinitions :many
SELECT id, organization_id, code, document_type, version, is_active, row_version
FROM wf.workflow_definition
WHERE deleted_at IS NULL
  AND (sqlc.narg(document_type)::text IS NULL OR document_type = sqlc.narg(document_type)::text)
ORDER BY code, version DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkflowDefinitions :one
SELECT count(*)::bigint
FROM wf.workflow_definition
WHERE deleted_at IS NULL
  AND (sqlc.narg(document_type)::text IS NULL OR document_type = sqlc.narg(document_type)::text);

-- name: CreateWorkflowInstance :one
INSERT INTO wf.workflow_instance (
    organization_id, definition_id, document_ref, current_step, created_by
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, status, row_version;

-- name: GetWorkflowInstance :one
SELECT id, organization_id, definition_id, document_ref, status, current_step,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM wf.workflow_instance
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkflowInstances :many
SELECT id, organization_id, definition_id, document_ref, status, current_step, row_version, created_at
FROM wf.workflow_instance
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkflowInstances :one
SELECT count(*)::bigint
FROM wf.workflow_instance
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text);

-- name: CreateWorkflowTask :one
INSERT INTO wf.workflow_task (
    organization_id, instance_id, assignee_id, created_by
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, row_version;

-- name: GetWorkflowTask :one
SELECT id, organization_id, instance_id, assignee_id, decision, decided_at,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM wf.workflow_task
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkflowTasks :many
SELECT t.id, t.organization_id, t.instance_id, t.assignee_id, t.decision, t.decided_at,
       t.created_by, t.row_version, i.document_ref, i.status AS instance_status
FROM wf.workflow_task t
JOIN wf.workflow_instance i ON i.id = t.instance_id
WHERE t.deleted_at IS NULL AND i.deleted_at IS NULL
  AND (sqlc.narg(instance_id)::uuid IS NULL OR t.instance_id = sqlc.narg(instance_id)::uuid)
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR t.assignee_id = sqlc.narg(assignee_id)::uuid)
  AND (sqlc.narg(pending_only)::boolean IS NULL OR (sqlc.narg(pending_only)::boolean = true AND t.decision IS NULL) OR (sqlc.narg(pending_only)::boolean = false AND t.decision IS NOT NULL))
ORDER BY t.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkflowTasks :one
SELECT count(*)::bigint
FROM wf.workflow_task t
JOIN wf.workflow_instance i ON i.id = t.instance_id
WHERE t.deleted_at IS NULL AND i.deleted_at IS NULL
  AND (sqlc.narg(instance_id)::uuid IS NULL OR t.instance_id = sqlc.narg(instance_id)::uuid)
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR t.assignee_id = sqlc.narg(assignee_id)::uuid)
  AND (sqlc.narg(pending_only)::boolean IS NULL OR (sqlc.narg(pending_only)::boolean = true AND t.decision IS NULL) OR (sqlc.narg(pending_only)::boolean = false AND t.decision IS NOT NULL));

-- name: DecideWorkflowTask :one
UPDATE wf.workflow_task
SET decision = $3,
    decided_at = now(),
    updated_by = $4
WHERE id = $1 AND row_version = $2 AND decision IS NULL AND deleted_at IS NULL
RETURNING row_version;

-- name: CountPendingTasksByInstance :one
SELECT count(*)::bigint
FROM wf.workflow_task
WHERE instance_id = $1 AND decision IS NULL AND deleted_at IS NULL;

-- name: DecideWorkflowInstance :one
UPDATE wf.workflow_instance
SET status = $3, updated_by = $4
WHERE id = $1 AND status = 'Pending' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO wf.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM wf.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE wf.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
