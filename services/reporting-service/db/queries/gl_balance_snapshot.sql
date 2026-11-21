-- name: UpsertGLBalanceSnapshot :one
INSERT INTO rpt.gl_balance_snapshot (
    organization_id, fiscal_period_id, account_code, closing_balance
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (organization_id, fiscal_period_id, account_code)
DO UPDATE SET closing_balance = EXCLUDED.closing_balance, refreshed_at = now()
RETURNING id, refreshed_at;

-- name: AccumulateGLBalanceSnapshot :one
INSERT INTO rpt.gl_balance_snapshot (
    organization_id, fiscal_period_id, account_code, closing_balance
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (organization_id, fiscal_period_id, account_code)
DO UPDATE SET closing_balance = rpt.gl_balance_snapshot.closing_balance + EXCLUDED.closing_balance, refreshed_at = now()
RETURNING id, refreshed_at;

-- name: GetGLBalanceSnapshot :one
SELECT * FROM rpt.gl_balance_snapshot
WHERE id = $1;

-- name: ListGLBalanceSnapshotsByPeriod :many
SELECT id, fiscal_period_id, account_code, closing_balance, refreshed_at
FROM rpt.gl_balance_snapshot
WHERE organization_id = $1 AND fiscal_period_id = $2
ORDER BY account_code
LIMIT $3 OFFSET $4;

-- name: ListGLBalanceSnapshots :many
SELECT id, fiscal_period_id, account_code, closing_balance, refreshed_at
FROM rpt.gl_balance_snapshot
ORDER BY refreshed_at DESC, account_code
LIMIT $1 OFFSET $2;

-- name: CountGLBalanceSnapshots :one
SELECT count(*)::bigint AS total
FROM rpt.gl_balance_snapshot;

-- name: UpsertSalesSummary :one
INSERT INTO rpt.sales_summary (
    organization_id, period_year, period_month, customer_id, total_sales_amount, order_count
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (organization_id, period_year, period_month, customer_id)
DO UPDATE SET
    total_sales_amount = rpt.sales_summary.total_sales_amount + EXCLUDED.total_sales_amount,
    order_count        = rpt.sales_summary.order_count + EXCLUDED.order_count,
    refreshed_at       = now()
RETURNING id, refreshed_at;

-- name: ListSalesSummaries :many
SELECT id, organization_id, period_year, period_month, customer_id, total_sales_amount, order_count, refreshed_at
FROM rpt.sales_summary
ORDER BY period_year DESC, period_month DESC, refreshed_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSalesSummaries :one
SELECT count(*)::bigint AS total
FROM rpt.sales_summary;

-- name: UpsertInventoryPosition :one
INSERT INTO rpt.inventory_position (
    organization_id, warehouse_code, item_code, on_hand_qty, available_qty
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (organization_id, warehouse_code, item_code)
DO UPDATE SET
    on_hand_qty   = rpt.inventory_position.on_hand_qty + EXCLUDED.on_hand_qty,
    available_qty = rpt.inventory_position.available_qty + EXCLUDED.available_qty,
    refreshed_at  = now()
RETURNING id, refreshed_at;

-- name: ListInventoryPositions :many
SELECT id, organization_id, warehouse_code, item_code, on_hand_qty, available_qty, refreshed_at
FROM rpt.inventory_position
ORDER BY warehouse_code, item_code
LIMIT $1 OFFSET $2;

-- name: CountInventoryPositions :one
SELECT count(*)::bigint AS total
FROM rpt.inventory_position;

-- name: MarkEventProcessed :execrows
INSERT INTO rpt.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertCOACache :exec
INSERT INTO rpt.coa_cache (
    organization_id, account_id, account_code, account_name, account_type
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (organization_id, account_id)
DO UPDATE SET
    account_code = EXCLUDED.account_code,
    account_name = EXCLUDED.account_name,
    account_type = EXCLUDED.account_type,
    refreshed_at = now();

-- name: GetCOACacheCode :one
SELECT account_code
FROM rpt.coa_cache
WHERE organization_id = $1 AND account_id = $2;
