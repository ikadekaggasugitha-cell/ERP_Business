-- name: CreateInvoice :one
INSERT INTO ap.ap_invoice (
    organization_id, invoice_no, vendor_id, invoice_date, due_date,
    currency_code, total_amount, outstanding_amount, goods_receipt_ref, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, invoice_date, status, row_version;

-- name: GetInvoice :one
SELECT * FROM ap.ap_invoice
WHERE id = $1 AND invoice_date = $2 AND deleted_at IS NULL;

-- name: GetInvoiceByID :one
SELECT * FROM ap.ap_invoice
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetInvoiceByRef :one
SELECT * FROM ap.ap_invoice
WHERE organization_id = $1 AND deleted_at IS NULL
  AND (id::text = sqlc.arg(invoice_ref)::text OR invoice_no = sqlc.arg(invoice_ref)::text)
ORDER BY invoice_date DESC
LIMIT 1;

-- name: ListInvoices :many
SELECT id, invoice_no, vendor_id, invoice_date, due_date, status,
       currency_code, total_amount, outstanding_amount, row_version
FROM ap.ap_invoice
WHERE deleted_at IS NULL
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
ORDER BY invoice_date DESC, invoice_no
LIMIT $1 OFFSET $2;

-- name: CountInvoices :one
SELECT COUNT(*)
FROM ap.ap_invoice
WHERE deleted_at IS NULL
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text);

-- name: ListInvoicesByStatus :many
SELECT id, invoice_no, vendor_id, invoice_date, due_date, status,
       currency_code, total_amount, outstanding_amount, row_version
FROM ap.ap_invoice
WHERE organization_id = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY invoice_date DESC, invoice_no
LIMIT $3 OFFSET $4;

-- name: CreateInvoiceLine :exec
INSERT INTO ap.ap_invoice_line (
    organization_id, ap_invoice_id, invoice_date, line_no, account_id,
    item_id, quantity, unit_price, amount, description, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: ListInvoiceLines :many
SELECT id, line_no, account_id, item_id, quantity, unit_price, amount, description
FROM ap.ap_invoice_line
WHERE ap_invoice_id = $1 AND invoice_date = $2 AND deleted_at IS NULL
ORDER BY line_no;

-- name: DeleteInvoiceLines :exec
DELETE FROM ap.ap_invoice_line
WHERE ap_invoice_id = $1 AND invoice_date = $2;

-- name: InvoiceLineTotal :one
SELECT COALESCE(SUM(amount), 0)::numeric(20,4) AS total_amount
FROM ap.ap_invoice_line
WHERE ap_invoice_id = $1 AND invoice_date = $2 AND deleted_at IS NULL;

-- name: InvoiceLineCount :one
SELECT COUNT(*)
FROM ap.ap_invoice_line
WHERE ap_invoice_id = $1 AND invoice_date = $2 AND deleted_at IS NULL;

-- name: UpdateDraftInvoice :one
UPDATE ap.ap_invoice
SET vendor_id = $4,
    due_date = $5,
    currency_code = $6,
    total_amount = $7,
    outstanding_amount = $7,
    goods_receipt_ref = $8,
    updated_by = $9
WHERE id = $1 AND invoice_date = $2 AND status = 'Draft' AND row_version = $3 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteDraftInvoice :execrows
UPDATE ap.ap_invoice
SET deleted_at = now(), updated_by = $4
WHERE id = $1 AND invoice_date = $2 AND status = 'Draft' AND row_version = $3 AND deleted_at IS NULL;

-- name: RecordInvoice :execrows
UPDATE ap.ap_invoice
SET status = 'Recorded', updated_by = $4
WHERE id = $1 AND invoice_date = $2 AND status = 'Draft' AND row_version = $3 AND deleted_at IS NULL;

-- name: CreatePaymentAllocation :exec
INSERT INTO ap.ap_payment_allocation (
    organization_id, ap_invoice_id, invoice_date, payment_id, allocated_amount, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (ap_invoice_id, payment_id) DO NOTHING;

-- name: ApplyPaymentToInvoice :execrows
UPDATE ap.ap_invoice
SET outstanding_amount = GREATEST(outstanding_amount - sqlc.arg(paid_amount)::numeric(20,4), 0),
    status = CASE
        WHEN outstanding_amount - sqlc.arg(paid_amount)::numeric(20,4) <= 0 THEN 'Paid'
        ELSE 'PartiallyPaid'
    END,
    updated_by = $3
WHERE id = $1 AND invoice_date = $2 AND status IN ('Recorded', 'PartiallyPaid') AND deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO ap.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM ap.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE ap.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO ap.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertMasterdataCache :exec
INSERT INTO ap.masterdata_cache (organization_id, party_id, code, name, party_type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (organization_id, party_id) DO UPDATE
SET code = EXCLUDED.code, name = EXCLUDED.name, party_type = EXCLUDED.party_type, updated_at = now();
