-- name: CreateInvoice :one
INSERT INTO ar.ar_invoice (
    organization_id, invoice_no, customer_id, invoice_date, due_date,
    currency_code, total_amount, outstanding_amount, sales_order_ref, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, invoice_date, status, row_version;

-- name: GetInvoice :one
SELECT * FROM ar.ar_invoice
WHERE id = $1 AND invoice_date = $2 AND deleted_at IS NULL;

-- name: GetInvoiceByID :one
SELECT * FROM ar.ar_invoice
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListInvoices :many
SELECT id, invoice_no, customer_id, invoice_date, due_date, status,
       currency_code, total_amount, outstanding_amount, row_version
FROM ar.ar_invoice
WHERE deleted_at IS NULL
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
ORDER BY invoice_date DESC, invoice_no
LIMIT $1 OFFSET $2;

-- name: CountInvoices :one
SELECT COUNT(*)
FROM ar.ar_invoice
WHERE deleted_at IS NULL
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text);

-- name: ListInvoicesByStatus :many
SELECT id, invoice_no, customer_id, invoice_date, due_date, status,
       currency_code, total_amount, outstanding_amount, row_version
FROM ar.ar_invoice
WHERE organization_id = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY invoice_date DESC, invoice_no
LIMIT $3 OFFSET $4;

-- name: CreateInvoiceLine :exec
INSERT INTO ar.ar_invoice_line (
    organization_id, ar_invoice_id, invoice_date, line_no, account_id,
    item_id, quantity, unit_price, amount, description, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: ListInvoiceLines :many
SELECT id, line_no, account_id, item_id, quantity, unit_price, amount, description
FROM ar.ar_invoice_line
WHERE ar_invoice_id = $1 AND invoice_date = $2 AND deleted_at IS NULL
ORDER BY line_no;

-- name: InvoiceLineTotal :one
SELECT COALESCE(SUM(amount), 0)::numeric(20,4) AS total_amount
FROM ar.ar_invoice_line
WHERE ar_invoice_id = $1 AND invoice_date = $2 AND deleted_at IS NULL;

-- name: MarkInvoicePaid :execrows
UPDATE ar.ar_invoice
SET status = 'Paid', outstanding_amount = 0, updated_by = $4
WHERE id = $1 AND invoice_date = $2 AND status = 'Issued' AND row_version = $3 AND deleted_at IS NULL;

-- name: DeleteDraftInvoice :execrows
UPDATE ar.ar_invoice
SET deleted_at = now(), updated_by = $4
WHERE id = $1 AND invoice_date = $2 AND status = 'Draft' AND row_version = $3 AND deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO ar.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM ar.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE ar.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO ar.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertMasterdataCache :exec
INSERT INTO ar.masterdata_cache (organization_id, party_id, code, name, party_type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (organization_id, party_id) DO UPDATE
SET code = EXCLUDED.code, name = EXCLUDED.name, party_type = EXCLUDED.party_type, updated_at = now();
