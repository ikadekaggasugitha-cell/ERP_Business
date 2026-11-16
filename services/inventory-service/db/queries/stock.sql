-- sqlc queries for inventory-service (warehouses, stock_balance, stock_movement, stock_reservation, outbox, cache).

-- name: CreateWarehouse :one
INSERT INTO inv.warehouse (organization_id, code, name, organization_ref_id, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, code, name, organization_ref_id, row_version;

-- name: GetWarehouse :one
SELECT id, organization_id, code, name, organization_ref_id, row_version
FROM inv.warehouse
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWarehouses :many
SELECT id, organization_id, code, name, organization_ref_id, row_version
FROM inv.warehouse
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountWarehouses :one
SELECT count(*)
FROM inv.warehouse
WHERE deleted_at IS NULL;

-- name: GetDefaultWarehouse :one
SELECT id, organization_id, code, name, organization_ref_id, row_version
FROM inv.warehouse
WHERE deleted_at IS NULL
ORDER BY created_at
LIMIT 1;

-- name: LockStockBalanceForItemWarehouse :one
SELECT id, organization_id, warehouse_id, item_id, on_hand_qty, reserved_qty, available_qty, avg_cost, row_version
FROM inv.stock_balance
WHERE organization_id = $1 AND warehouse_id = $2 AND item_id = $3 AND deleted_at IS NULL
FOR UPDATE;

-- name: LockBestStockBalanceForItem :one
SELECT id, organization_id, warehouse_id, item_id, on_hand_qty, reserved_qty, available_qty, avg_cost, row_version
FROM inv.stock_balance
WHERE organization_id = $1 AND item_id = $2 AND deleted_at IS NULL
ORDER BY (on_hand_qty - reserved_qty) DESC
LIMIT 1
FOR UPDATE;

-- name: UpsertReceiptBalance :one
INSERT INTO inv.stock_balance (
    organization_id, warehouse_id, item_id, on_hand_qty, reserved_qty, avg_cost, created_by
) VALUES ($1, $2, $3, $4, 0, $5, $6)
ON CONFLICT (organization_id, warehouse_id, item_id) DO UPDATE
SET on_hand_qty = inv.stock_balance.on_hand_qty + EXCLUDED.on_hand_qty,
    avg_cost = CASE
        WHEN EXCLUDED.avg_cost > 0 THEN EXCLUDED.avg_cost
        ELSE inv.stock_balance.avg_cost
    END,
    updated_by = $6
RETURNING id, warehouse_id, item_id, on_hand_qty, reserved_qty, available_qty, avg_cost, row_version;

-- name: ReserveStockBalance :one
UPDATE inv.stock_balance
SET reserved_qty = reserved_qty + $2,
    updated_by = $3
WHERE id = $1
  AND (on_hand_qty - reserved_qty) >= $2
  AND deleted_at IS NULL
RETURNING row_version;

-- name: ReleaseStockBalance :one
UPDATE inv.stock_balance
SET reserved_qty = GREATEST(reserved_qty - $2, 0),
    updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING row_version;

-- name: ConsumeReservedStockBalance :one
UPDATE inv.stock_balance
SET on_hand_qty = on_hand_qty - $2,
    reserved_qty = GREATEST(reserved_qty - $2, 0),
    updated_by = $3
WHERE id = $1 AND on_hand_qty >= $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeductAvailableStockBalance :one
UPDATE inv.stock_balance
SET on_hand_qty = on_hand_qty - $2,
    updated_by = $3
WHERE id = $1
  AND (on_hand_qty - reserved_qty) >= $2
  AND deleted_at IS NULL
RETURNING row_version;

-- name: ListStockBalances :many
SELECT id, organization_id, warehouse_id, item_id, on_hand_qty, reserved_qty, available_qty, avg_cost, row_version
FROM inv.stock_balance
WHERE deleted_at IS NULL
  AND (sqlc.narg(warehouse_id)::uuid IS NULL OR warehouse_id = sqlc.narg(warehouse_id))
  AND (sqlc.narg(item_id)::uuid IS NULL OR item_id = sqlc.narg(item_id))
ORDER BY warehouse_id, item_id
LIMIT $1 OFFSET $2;

-- name: CountStockBalances :one
SELECT count(*)
FROM inv.stock_balance
WHERE deleted_at IS NULL
  AND (sqlc.narg(warehouse_id)::uuid IS NULL OR warehouse_id = sqlc.narg(warehouse_id))
  AND (sqlc.narg(item_id)::uuid IS NULL OR item_id = sqlc.narg(item_id));

-- name: CreateStockMovement :one
INSERT INTO inv.stock_movement (
    organization_id, movement_date, warehouse_id, item_id, movement_type,
    quantity, source_document_ref, correlation_id, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, movement_date, row_version;

-- name: GetStockMovement :one
SELECT id, organization_id, movement_date, warehouse_id, item_id, movement_type, quantity, source_document_ref, correlation_id, row_version
FROM inv.stock_movement
WHERE id = $1 AND movement_date = $2 AND deleted_at IS NULL;

-- name: ListStockMovements :many
SELECT id, organization_id, movement_date, warehouse_id, item_id, movement_type, quantity, source_document_ref, correlation_id, row_version
FROM inv.stock_movement
WHERE deleted_at IS NULL
  AND (sqlc.narg(warehouse_id)::uuid IS NULL OR warehouse_id = sqlc.narg(warehouse_id))
  AND (sqlc.narg(item_id)::uuid IS NULL OR item_id = sqlc.narg(item_id))
ORDER BY movement_date DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountStockMovements :one
SELECT count(*)
FROM inv.stock_movement
WHERE deleted_at IS NULL
  AND (sqlc.narg(warehouse_id)::uuid IS NULL OR warehouse_id = sqlc.narg(warehouse_id))
  AND (sqlc.narg(item_id)::uuid IS NULL OR item_id = sqlc.narg(item_id));

-- name: CreateReservation :one
INSERT INTO inv.stock_reservation (
    organization_id, item_id, warehouse_id, reserved_qty, sales_order_ref, created_by
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, status, row_version;

-- name: ReleaseReservation :one
UPDATE inv.stock_reservation
SET status = 'Released', updated_by = $3
WHERE id = $1 AND status = 'Reserved' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: IssueReservation :one
UPDATE inv.stock_reservation
SET status = 'Consumed', updated_by = $3
WHERE id = $1 AND status = 'Reserved' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: GetReservation :one
SELECT id, organization_id, item_id, warehouse_id, reserved_qty, sales_order_ref, status, row_version
FROM inv.stock_reservation
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListReservations :many
SELECT id, organization_id, item_id, warehouse_id, reserved_qty, sales_order_ref, status, row_version
FROM inv.stock_reservation
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountReservations :one
SELECT count(*)
FROM inv.stock_reservation
WHERE deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO inv.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM inv.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE inv.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO inv.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertMasterdataCache :exec
INSERT INTO inv.masterdata_cache (organization_id, party_id, code, name, party_type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (organization_id, party_id) DO UPDATE
SET code = EXCLUDED.code, name = EXCLUDED.name, party_type = EXCLUDED.party_type, updated_at = now();
