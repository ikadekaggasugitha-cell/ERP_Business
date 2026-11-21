-- name: CreateMaintenanceSchedule :one
INSERT INTO mnt.maintenance_schedule (
    organization_id, asset_ref, schedule_type, interval_value, interval_unit,
    next_due_date, is_active, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, row_version;

-- name: GetMaintenanceSchedule :one
SELECT * FROM mnt.maintenance_schedule WHERE id = $1 AND deleted_at IS NULL;

-- name: ListMaintenanceSchedules :many
SELECT * FROM mnt.maintenance_schedule
WHERE deleted_at IS NULL
ORDER BY next_due_date ASC NULLS LAST, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountMaintenanceSchedules :one
SELECT count(*) FROM mnt.maintenance_schedule WHERE deleted_at IS NULL;

-- name: UpdateMaintenanceSchedule :one
UPDATE mnt.maintenance_schedule
SET schedule_type = $2, interval_value = $3, interval_unit = $4, next_due_date = $5, is_active = $6, updated_by = $7
WHERE id = $1 AND row_version = $8 AND deleted_at IS NULL
RETURNING row_version;

-- name: SoftDeleteMaintenanceSchedule :execrows
UPDATE mnt.maintenance_schedule
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND deleted_at IS NULL;

-- name: CreateWorkOrder :one
INSERT INTO mnt.work_order (
    organization_id, wo_no, asset_ref, maintenance_type, status,
    assignee_ref, vendor_ref, scheduled_date, created_by
) VALUES ($1, $2, $3, $4, 'Open', $5, $6, $7, $8)
RETURNING id, row_version;

-- name: GetWorkOrder :one
SELECT * FROM mnt.work_order WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkOrders :many
SELECT * FROM mnt.work_order
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkOrders :one
SELECT count(*) FROM mnt.work_order
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: AssignWorkOrder :one
UPDATE mnt.work_order
SET status = 'Assigned', assignee_ref = $2, vendor_ref = $3, updated_by = $4
WHERE id = $1 AND row_version = $5 AND status = 'Open' AND deleted_at IS NULL
RETURNING row_version;

-- name: StartWorkOrder :one
UPDATE mnt.work_order
SET status = 'InProgress', downtime_start = COALESCE(downtime_start, now()), updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Assigned' AND deleted_at IS NULL
RETURNING row_version;

-- name: CompleteWorkOrder :one
UPDATE mnt.work_order
SET status = 'Completed', downtime_end = COALESCE(downtime_end, now()), updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'InProgress' AND deleted_at IS NULL
RETURNING row_version;

-- name: CloseWorkOrder :one
UPDATE mnt.work_order
SET status = 'Closed', updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Completed' AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateWorkOrderPart :one
INSERT INTO mnt.work_order_part (
    organization_id, work_order_id, item_ref, quantity, unit_cost, created_by
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: ListWorkOrderParts :many
SELECT * FROM mnt.work_order_part
WHERE work_order_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: CreateWorkOrderChecklist :one
INSERT INTO mnt.work_order_checklist (
    organization_id, work_order_id, step_no, description, is_done, created_by
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: ListWorkOrderChecklist :many
SELECT * FROM mnt.work_order_checklist
WHERE work_order_id = $1 AND deleted_at IS NULL
ORDER BY step_no ASC;

-- name: UpdateWorkOrderChecklist :one
UPDATE mnt.work_order_checklist
SET description = $2, is_done = $3, updated_by = $4
WHERE id = $1 AND row_version = $5 AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateMaintenanceCost :one
INSERT INTO mnt.maintenance_cost (
    organization_id, work_order_id, cost_type, amount, created_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, row_version;

-- name: ListMaintenanceCosts :many
SELECT * FROM mnt.maintenance_cost
WHERE work_order_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: InsertOutbox :exec
INSERT INTO mnt.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ClaimUnpublishedOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM mnt.outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE mnt.outbox SET published_at = now() WHERE id = $1 AND created_at = $2;
