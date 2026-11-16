-- name: CreateJournalEntry :one
INSERT INTO gl.journal_entry (
    organization_id, journal_no, financial_organization_id, ledger_id, fiscal_period_id,
    posting_date, journal_source, source_module, source_document_ref, currency_code,
    exchange_rate, reversed_entry_id, correlation_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING id, posting_date, status, row_version;

-- name: GetJournalEntry :one
SELECT * FROM gl.journal_entry
WHERE id = $1 AND posting_date = $2 AND deleted_at IS NULL;

-- name: GetJournalEntryByID :one
SELECT * FROM gl.journal_entry
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListJournalEntries :many
SELECT id, journal_no, financial_organization_id, ledger_id, fiscal_period_id,
       posting_date, journal_source, currency_code, status, source_document_ref,
       reversed_entry_id, row_version, posted_at, created_at
FROM gl.journal_entry
WHERE deleted_at IS NULL
ORDER BY posting_date DESC, created_at DESC, journal_no
LIMIT $1 OFFSET $2;

-- name: CountJournalEntries :one
SELECT count(*)::bigint AS total
FROM gl.journal_entry
WHERE deleted_at IS NULL;

-- name: ListJournalEntriesByPeriod :many
SELECT id, journal_no, financial_organization_id, ledger_id, fiscal_period_id,
       posting_date, journal_source, currency_code, status, source_document_ref,
       reversed_entry_id, row_version, posted_at, created_at
FROM gl.journal_entry
WHERE organization_id = $1 AND fiscal_period_id = $2 AND deleted_at IS NULL
ORDER BY posting_date, journal_no
LIMIT $3 OFFSET $4;

-- name: CreateJournalLine :exec
INSERT INTO gl.journal_line (
    organization_id, journal_entry_id, posting_date, line_no, account_id,
    debit_amount, credit_amount, functional_amount, description, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: ListJournalLines :many
SELECT id, line_no, account_id, debit_amount, credit_amount, functional_amount, description
FROM gl.journal_line
WHERE journal_entry_id = $1 AND posting_date = $2 AND deleted_at IS NULL
ORDER BY line_no;

-- name: UpdateJournalEntryDraft :execrows
UPDATE gl.journal_entry
SET currency_code = $3,
    exchange_rate = $4,
    source_document_ref = $5,
    updated_by = $6
WHERE id = $1 AND posting_date = $2 AND status = 'Draft' AND deleted_at IS NULL AND row_version = $7;

-- name: DeleteJournalLines :exec
DELETE FROM gl.journal_line
WHERE journal_entry_id = $1 AND posting_date = $2;

-- name: SoftDeleteJournalEntry :execrows
UPDATE gl.journal_entry
SET deleted_at = now(), updated_by = $4
WHERE id = $1 AND posting_date = $2 AND status = 'Draft' AND deleted_at IS NULL AND row_version = $3;

-- name: JournalEntryTotals :one
SELECT COALESCE(SUM(debit_amount), 0)::numeric(20,4)  AS total_debit,
       COALESCE(SUM(credit_amount), 0)::numeric(20,4) AS total_credit,
       count(*)::bigint AS line_count
FROM gl.journal_line
WHERE journal_entry_id = $1 AND posting_date = $2 AND deleted_at IS NULL;

-- name: PostJournalEntry :execrows
UPDATE gl.journal_entry
SET status = 'Posted', posted_at = now(), updated_by = $4
WHERE id = $1 AND posting_date = $2 AND status = 'Draft' AND deleted_at IS NULL AND row_version = $3;

-- name: MarkJournalReversed :execrows
UPDATE gl.journal_entry
SET status = 'Reversed', updated_by = $4
WHERE id = $1 AND posting_date = $2 AND status = 'Posted' AND deleted_at IS NULL AND row_version = $3;

-- name: UpsertAccountBalance :exec
INSERT INTO gl.account_balance (
    organization_id, account_id, fiscal_period_id, ledger_id,
    opening_balance, period_debit, period_credit, closing_balance, created_by
) VALUES (
    $1, $2, $3, $4, 0, $5, $6, ($5::numeric(20,4) - $6::numeric(20,4)), $7
)
ON CONFLICT (organization_id, ledger_id, account_id, fiscal_period_id)
DO UPDATE SET
    period_debit    = gl.account_balance.period_debit + EXCLUDED.period_debit,
    period_credit   = gl.account_balance.period_credit + EXCLUDED.period_credit,
    closing_balance = gl.account_balance.opening_balance + (gl.account_balance.period_debit + EXCLUDED.period_debit) - (gl.account_balance.period_credit + EXCLUDED.period_credit),
    updated_by      = EXCLUDED.created_by;

-- name: GetAccountBalance :one
SELECT closing_balance
FROM gl.account_balance
WHERE organization_id = $1 AND ledger_id = $2 AND account_id = $3 AND fiscal_period_id = $4 AND deleted_at IS NULL;

-- name: ListAccountBalances :many
SELECT id, organization_id, ledger_id, account_id, fiscal_period_id,
       opening_balance, period_debit, period_credit, closing_balance, row_version, updated_at
FROM gl.account_balance
WHERE deleted_at IS NULL
  AND (sqlc.arg(ledger_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ledger_id = sqlc.arg(ledger_id))
  AND (sqlc.arg(fiscal_period_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR fiscal_period_id = sqlc.arg(fiscal_period_id))
ORDER BY account_id
LIMIT $1 OFFSET $2;

-- name: CountAccountBalances :one
SELECT count(*)::bigint AS total
FROM gl.account_balance
WHERE deleted_at IS NULL
  AND (sqlc.arg(ledger_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ledger_id = sqlc.arg(ledger_id))
  AND (sqlc.arg(fiscal_period_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR fiscal_period_id = sqlc.arg(fiscal_period_id));

-- name: InsertOutbox :exec
INSERT INTO gl.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM gl.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE gl.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO gl.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;
