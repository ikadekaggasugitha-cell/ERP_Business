-- name: CreateTaxCode :one
INSERT INTO tax.tax_code (
    organization_id, code, rate, tax_type, gl_account_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, row_version;

-- name: GetTaxCode :one
SELECT * FROM tax.tax_code
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTaxCodes :many
SELECT id, code, rate, tax_type, gl_account_id, row_version
FROM tax.tax_code
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountTaxCodes :one
SELECT COUNT(*)
FROM tax.tax_code
WHERE deleted_at IS NULL;

-- name: CreateTaxTransaction :one
INSERT INTO tax.tax_transaction (
    organization_id, source_document_ref, tax_code_id, base_amount, tax_amount,
    efaktur_no, efaktur_status, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, row_version;

-- name: GetTaxTransaction :one
SELECT * FROM tax.tax_transaction
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTaxTransactionsByCode :many
SELECT id, source_document_ref, tax_code_id, base_amount, tax_amount,
       efaktur_no, efaktur_status, row_version
FROM tax.tax_transaction
WHERE organization_id = $1 AND tax_code_id = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListTaxTransactions :many
SELECT id, source_document_ref, tax_code_id, base_amount, tax_amount,
       efaktur_no, efaktur_status, row_version
FROM tax.tax_transaction
WHERE deleted_at IS NULL
  AND (sqlc.arg(efaktur_status)::text = '' OR COALESCE(efaktur_status, '') = sqlc.arg(efaktur_status)::text)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTaxTransactions :one
SELECT COUNT(*)
FROM tax.tax_transaction
WHERE deleted_at IS NULL
  AND (sqlc.arg(efaktur_status)::text = '' OR COALESCE(efaktur_status, '') = sqlc.arg(efaktur_status)::text);

-- name: ReportTaxTransaction :one
UPDATE tax.tax_transaction
SET efaktur_no = COALESCE(NULLIF(sqlc.arg(efaktur_no)::text, ''), efaktur_no),
    efaktur_status = 'Reported',
    updated_by = $3
WHERE id = $1
  AND row_version = $2
  AND deleted_at IS NULL
  AND COALESCE(efaktur_status, 'Draft') <> 'Reported'
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO tax.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM tax.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE tax.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO tax.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: EnsureSystemTaxCode :one
INSERT INTO tax.tax_code (organization_id, code, rate, tax_type, created_by)
VALUES ($1, 'SYS-VAT', 11.0, 'VAT', $2)
ON CONFLICT (organization_id, code) DO UPDATE SET code = EXCLUDED.code
RETURNING id, code, rate;

-- name: GetTaxCodeByCode :one
SELECT id, code, rate, tax_type
FROM tax.tax_code
WHERE organization_id IN ($1, '00000000-0000-0000-0000-000000000000'::uuid)
  AND code = $2
  AND deleted_at IS NULL
ORDER BY (organization_id = $1) DESC
LIMIT 1;

-- name: GetDefaultVATTaxCode :one
SELECT id, code, rate, tax_type
FROM tax.tax_code
WHERE organization_id IN ($1, '00000000-0000-0000-0000-000000000000'::uuid)
  AND tax_type = 'VAT'
  AND deleted_at IS NULL
ORDER BY (organization_id = $1) DESC, created_at ASC
LIMIT 1;
