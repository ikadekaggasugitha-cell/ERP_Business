-- sqlc queries for sales-service (SO + lines + deliveries + outbox + consumer dedup + cache).

-- name: CreateSalesOrder :one
INSERT INTO so.sales_order (
    organization_id, so_no, customer_id, order_date, currency_code,
    total_amount, correlation_id, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, status, row_version;

-- name: CreateSalesOrderLine :exec
INSERT INTO so.so_line (
    organization_id, sales_order_id, line_no, item_id, order_qty, unit_price, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetSalesOrder :one
SELECT id, organization_id, so_no, customer_id, order_date, currency_code, total_amount, status, correlation_id, row_version
FROM so.sales_order
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListSalesOrderLines :many
SELECT id, organization_id, sales_order_id, line_no, item_id, order_qty, unit_price, delivered_qty, row_version
FROM so.so_line
WHERE sales_order_id = $1 AND deleted_at IS NULL
ORDER BY line_no;

-- name: ListSalesOrders :many
SELECT id, organization_id, so_no, customer_id, order_date, currency_code, total_amount, status, row_version
FROM so.sales_order
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY order_date DESC, so_no DESC
LIMIT $1 OFFSET $2;

-- name: CountSalesOrders :one
SELECT count(*)
FROM so.sales_order
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: UpdateDraftSalesOrder :one
UPDATE so.sales_order
SET customer_id = $3,
    order_date = $4,
    currency_code = $5,
    total_amount = $6,
    updated_by = $7
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteSalesOrderLines :exec
DELETE FROM so.so_line
WHERE sales_order_id = $1;

-- name: SoftDeleteDraftSalesOrder :execrows
UPDATE so.sales_order
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL;

-- name: ConfirmSalesOrder :one
UPDATE so.sales_order
SET status = 'Confirmed', updated_by = $3
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: CancelSalesOrder :execrows
UPDATE so.sales_order
SET status = 'Cancelled', updated_by = $2
WHERE id = $1 AND status <> 'Cancelled' AND deleted_at IS NULL;

-- name: CreateDelivery :one
INSERT INTO so.delivery (
    organization_id, delivery_no, sales_order_id, warehouse_id, delivery_date, status, correlation_id, created_by
) VALUES ($1, $2, $3, $4, $5, 'Delivered', $6, $7)
RETURNING id, delivery_no, status, row_version;

-- name: CreateDeliveryLine :exec
INSERT INTO so.delivery_line (
    organization_id, delivery_id, so_line_id, line_no, item_id, delivered_qty, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: MarkSOLinesDelivered :exec
UPDATE so.so_line
SET delivered_qty = order_qty, updated_by = $2
WHERE sales_order_id = $1 AND deleted_at IS NULL;

-- name: MarkSalesOrderDelivered :one
UPDATE so.sales_order
SET status = 'Delivered', updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Confirmed' AND deleted_at IS NULL
RETURNING row_version;

-- name: GetDelivery :one
SELECT id, organization_id, delivery_no, sales_order_id, warehouse_id, delivery_date, status, correlation_id, row_version
FROM so.delivery
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListDeliveryLines :many
SELECT id, organization_id, delivery_id, so_line_id, line_no, item_id, delivered_qty, row_version
FROM so.delivery_line
WHERE delivery_id = $1 AND deleted_at IS NULL
ORDER BY line_no;

-- name: ListDeliveries :many
SELECT id, organization_id, delivery_no, sales_order_id, warehouse_id, delivery_date, status, row_version
FROM so.delivery
WHERE deleted_at IS NULL
ORDER BY delivery_date DESC, delivery_no DESC
LIMIT $1 OFFSET $2;

-- name: CountDeliveries :one
SELECT count(*)
FROM so.delivery
WHERE deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO so.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM so.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE so.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO so.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertMasterdataCache :exec
INSERT INTO so.masterdata_cache (organization_id, party_id, code, name, party_type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (organization_id, party_id) DO UPDATE
SET code = EXCLUDED.code, name = EXCLUDED.name, party_type = EXCLUDED.party_type, updated_at = now();
