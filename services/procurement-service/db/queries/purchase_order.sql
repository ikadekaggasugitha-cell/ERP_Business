-- sqlc queries for procurement-service (PO + PO lines + PR + GR + outbox + processed_event + masterdata_cache).

-- name: CreatePurchaseRequisition :one
INSERT INTO pr.purchase_requisition (
    organization_id, pr_no, requester_id, status, workflow_ref, needed_date, created_by
) VALUES ($1, $2, $3, 'Draft', $4, $5, $6)
RETURNING id, status, row_version;

-- name: CreatePurchaseRequisitionLine :exec
INSERT INTO pr.pr_line (
    organization_id, purchase_requisition_id, line_no, item_id, request_qty, needed_date, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPurchaseRequisition :one
SELECT id, organization_id, pr_no, requester_id, status, workflow_ref, needed_date, created_by, updated_by, row_version
FROM pr.purchase_requisition
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPurchaseRequisitionLines :many
SELECT id, organization_id, purchase_requisition_id, line_no, item_id, request_qty, needed_date, row_version
FROM pr.pr_line
WHERE purchase_requisition_id = $1 AND deleted_at IS NULL
ORDER BY line_no;

-- name: ListPurchaseRequisitions :many
SELECT id, organization_id, pr_no, requester_id, status, workflow_ref, needed_date, row_version
FROM pr.purchase_requisition
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY created_at DESC, pr_no DESC
LIMIT $1 OFFSET $2;

-- name: CountPurchaseRequisitions :one
SELECT count(*)
FROM pr.purchase_requisition
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: TransitionPurchaseRequisition :one
UPDATE pr.purchase_requisition
SET status = $3, updated_by = $4
WHERE id = $1 AND row_version = $2 AND status = $5 AND deleted_at IS NULL
RETURNING status, row_version;

-- name: CreatePurchaseOrder :one
INSERT INTO po.purchase_order (
    organization_id, po_no, vendor_id, purchase_requisition_id,
    order_date, currency_code, total_amount, status, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'Draft', $8)
RETURNING id, status, row_version;

-- name: CreatePurchaseOrderLine :exec
INSERT INTO po.po_line (
    organization_id, purchase_order_id, line_no, item_id, order_qty, unit_price, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPurchaseOrder :one
SELECT id, organization_id, po_no, vendor_id, purchase_requisition_id, order_date, currency_code, total_amount, status, created_by, updated_by, row_version
FROM po.purchase_order
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPurchaseOrderLines :many
SELECT id, organization_id, purchase_order_id, line_no, item_id, order_qty, received_qty, unit_price, row_version
FROM po.po_line
WHERE purchase_order_id = $1 AND deleted_at IS NULL
ORDER BY line_no;

-- name: ListPurchaseOrders :many
SELECT id, organization_id, po_no, vendor_id, purchase_requisition_id, order_date, currency_code, total_amount, status, row_version
FROM po.purchase_order
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY order_date DESC, po_no DESC
LIMIT $1 OFFSET $2;

-- name: CountPurchaseOrders :one
SELECT count(*)
FROM po.purchase_order
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: UpdateDraftPurchaseOrder :one
UPDATE po.purchase_order
SET vendor_id = $3,
    purchase_requisition_id = $4,
    order_date = $5,
    currency_code = $6,
    total_amount = $7,
    updated_by = $8
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: DeletePurchaseOrderLines :exec
DELETE FROM po.po_line
WHERE purchase_order_id = $1;

-- name: SoftDeleteDraftPurchaseOrder :execrows
UPDATE po.purchase_order
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL;

-- name: ApprovePurchaseOrder :one
UPDATE po.purchase_order
SET status = 'Approved', updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: ReceivePurchaseOrder :one
UPDATE po.purchase_order
SET status = 'Received', updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Approved' AND deleted_at IS NULL
RETURNING row_version;

-- name: MarkPOLinesReceived :exec
UPDATE po.po_line
SET received_qty = order_qty, updated_by = $2
WHERE purchase_order_id = $1 AND deleted_at IS NULL;

-- name: CreateGoodsReceipt :one
INSERT INTO gr.goods_receipt (
    organization_id, gr_no, purchase_order_id, warehouse_id,
    receipt_date, status, correlation_id, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, gr_no, status, row_version;

-- name: CreateGoodsReceiptLine :exec
INSERT INTO gr.gr_line (
    organization_id, goods_receipt_id, po_line_id, line_no, received_qty, created_by
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetGoodsReceipt :one
SELECT id, organization_id, gr_no, purchase_order_id, warehouse_id, receipt_date, status, correlation_id, row_version
FROM gr.goods_receipt
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListGoodsReceiptLines :many
SELECT id, organization_id, goods_receipt_id, po_line_id, line_no, received_qty, row_version
FROM gr.gr_line
WHERE goods_receipt_id = $1 AND deleted_at IS NULL
ORDER BY line_no;

-- name: ListGoodsReceipts :many
SELECT id, organization_id, gr_no, purchase_order_id, warehouse_id, receipt_date, status, correlation_id, row_version
FROM gr.goods_receipt
WHERE deleted_at IS NULL
ORDER BY receipt_date DESC, gr_no DESC
LIMIT $1 OFFSET $2;

-- name: CountGoodsReceipts :one
SELECT count(*)
FROM gr.goods_receipt
WHERE deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO pr.outbox (
    aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM pr.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE pr.outbox
SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO pr.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertMasterdataCache :exec
INSERT INTO pr.masterdata_cache (organization_id, party_id, code, name, party_type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (organization_id, party_id) DO UPDATE
SET code = EXCLUDED.code, name = EXCLUDED.name, party_type = EXCLUDED.party_type, updated_at = now();
