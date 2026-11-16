-- sqlc queries for manufacturing-service (BOM + BOM lines + Work Orders + Outbox).

-- name: CreateBillOfMaterial :one
INSERT INTO mfg.bill_of_material (
    organization_id, code, product_item_id, version, created_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, code, product_item_id, version, is_active, row_version;

-- name: CreateBomLine :exec
INSERT INTO mfg.bom_line (
    organization_id, bom_id, line_no, component_item_id, quantity, uom, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetBillOfMaterial :one
SELECT id, organization_id, code, product_item_id, version, is_active, row_version
FROM mfg.bill_of_material
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListBomLines :many
SELECT id, organization_id, bom_id, line_no, component_item_id, quantity, uom, row_version
FROM mfg.bom_line
WHERE bom_id = $1 AND deleted_at IS NULL
ORDER BY line_no;

-- name: ListBillsOfMaterial :many
SELECT id, organization_id, code, product_item_id, version, is_active, row_version
FROM mfg.bill_of_material
WHERE deleted_at IS NULL
ORDER BY code, version DESC
LIMIT $1 OFFSET $2;

-- name: CountBillsOfMaterial :one
SELECT count(*)
FROM mfg.bill_of_material
WHERE deleted_at IS NULL;

-- name: CreateWorkOrder :one
INSERT INTO mfg.work_order (
    organization_id, wo_no, product_item_id, bom_id, planned_qty,
    sales_order_ref, correlation_id, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, status, row_version;

-- name: GetWorkOrder :one
SELECT id, organization_id, wo_no, product_item_id, bom_id, planned_qty, produced_qty, status, sales_order_ref, correlation_id, row_version
FROM mfg.work_order
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkOrders :many
SELECT id, organization_id, wo_no, product_item_id, bom_id, planned_qty, produced_qty, status, sales_order_ref, row_version
FROM mfg.work_order
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY created_at DESC, wo_no DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkOrders :one
SELECT count(*)
FROM mfg.work_order
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: UpdatePlannedWorkOrder :one
UPDATE mfg.work_order
SET product_item_id = $3,
    bom_id = $4,
    planned_qty = $5,
    sales_order_ref = $6,
    updated_by = $7
WHERE id = $1 AND row_version = $2 AND status = 'Planned' AND deleted_at IS NULL
RETURNING row_version;

-- name: SoftDeletePlannedWorkOrder :execrows
UPDATE mfg.work_order
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Planned' AND deleted_at IS NULL;

-- name: ReleaseWorkOrder :one
UPDATE mfg.work_order
SET status = 'Released', updated_by = $3
WHERE id = $1 AND status = 'Planned' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: CompleteWorkOrder :one
UPDATE mfg.work_order
SET status = 'Completed',
    produced_qty = $3,
    updated_by = $4
WHERE id = $1 AND status IN ('Released', 'InProgress') AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO mfg.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM mfg.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE mfg.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
