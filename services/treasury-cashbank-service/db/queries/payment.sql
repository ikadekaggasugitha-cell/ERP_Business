-- name: CreateBankAccount :one
INSERT INTO trs.bank_account (
    organization_id, code, bank_name, account_no, currency_code, gl_account_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, row_version;

-- name: GetBankAccount :one
SELECT id, organization_id, code, bank_name, account_no, currency_code, gl_account_id,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM trs.bank_account
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListBankAccounts :many
SELECT id, organization_id, code, bank_name, account_no, currency_code, gl_account_id,
       created_at, updated_at, row_version
FROM trs.bank_account
WHERE deleted_at IS NULL
  AND (sqlc.narg(currency_code)::text IS NULL OR currency_code = sqlc.narg(currency_code)::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountBankAccounts :one
SELECT count(*)::bigint
FROM trs.bank_account
WHERE deleted_at IS NULL
  AND (sqlc.narg(currency_code)::text IS NULL OR currency_code = sqlc.narg(currency_code)::text);

-- Idempotent per-org system clearing account for consumer-scheduled payments
-- (INT-002/003). Upsert-returning-id so the FK on trs.payment resolves.
-- name: EnsureSystemBankAccount :one
INSERT INTO trs.bank_account (organization_id, code, bank_name, account_no, currency_code, gl_account_id, created_by)
VALUES ($1, 'SYS-BANK', 'System Clearing', '00000000', 'IDR', $2, $3)
ON CONFLICT (organization_id, code) DO UPDATE SET code = EXCLUDED.code
RETURNING id;

-- payment_no uniqueness is DB-enforced by uq_payment_no (no registry trigger).
-- status defaults to 'Draft'.
-- name: CreatePayment :one
INSERT INTO trs.payment (
    organization_id, payment_no, payment_type, bank_account_id, party_id,
    amount, payment_date, external_ref, correlation_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, status, row_version;

-- name: GetPayment :one
SELECT id, organization_id, payment_no, payment_type, bank_account_id, party_id,
       amount, payment_date, status, external_ref, correlation_id,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM trs.payment
WHERE id = $1 AND deleted_at IS NULL;

-- Scope-listed (RLS filters to the caller's org scope), paginated.
-- name: ListPayments :many
SELECT id, organization_id, payment_no, payment_type, bank_account_id, party_id, amount,
       payment_date, status, external_ref, row_version
FROM trs.payment
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(bank_account_id)::uuid IS NULL OR bank_account_id = sqlc.narg(bank_account_id)::uuid)
ORDER BY payment_date DESC, payment_no
LIMIT $1 OFFSET $2;

-- name: CountPayments :one
SELECT count(*)::bigint
FROM trs.payment
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(bank_account_id)::uuid IS NULL OR bank_account_id = sqlc.narg(bank_account_id)::uuid);

-- name: UpdateDraftPayment :one
UPDATE trs.payment
SET payment_type    = $3,
    bank_account_id = $4,
    party_id        = $5,
    amount          = $6,
    payment_date    = $7,
    external_ref    = $8,
    updated_by      = $9
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteDraftPayment :execrows
UPDATE trs.payment
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL;

-- name: CreatePaymentAllocation :exec
INSERT INTO trs.payment_allocation (
    organization_id, payment_id, target_document_ref, allocated_amount, created_by
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListPaymentAllocations :many
SELECT id, payment_id, target_document_ref, allocated_amount, allocated_at, row_version
FROM trs.payment_allocation
WHERE payment_id = $1 AND deleted_at IS NULL
ORDER BY created_at;

-- Optimistic-locked Draft->Settled transition.
-- name: SettlePayment :one
UPDATE trs.payment
SET status = 'Settled', updated_by = $3
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- Optimistic-locked Settled->Reconciled transition.
-- name: ReconcilePayment :one
UPDATE trs.payment
SET status = 'Reconciled', updated_by = $3
WHERE id = $1 AND status = 'Settled' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateBankStatementLine :one
INSERT INTO trs.bank_statement_line (
    organization_id, bank_account_id, statement_date, description, amount, external_ref, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, is_reconciled, row_version;

-- name: ListBankStatementLines :many
SELECT id, organization_id, bank_account_id, statement_date, description, amount,
       external_ref, matched_payment_id, is_reconciled, row_version
FROM trs.bank_statement_line
WHERE deleted_at IS NULL
  AND (sqlc.narg(bank_account_id)::uuid IS NULL OR bank_account_id = sqlc.narg(bank_account_id)::uuid)
  AND (sqlc.narg(is_reconciled)::boolean IS NULL OR is_reconciled = sqlc.narg(is_reconciled)::boolean)
ORDER BY statement_date DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountBankStatementLines :one
SELECT count(*)::bigint
FROM trs.bank_statement_line
WHERE deleted_at IS NULL
  AND (sqlc.narg(bank_account_id)::uuid IS NULL OR bank_account_id = sqlc.narg(bank_account_id)::uuid)
  AND (sqlc.narg(is_reconciled)::boolean IS NULL OR is_reconciled = sqlc.narg(is_reconciled)::boolean);

-- name: ListUnreconciledStatementLines :many
SELECT id, organization_id, bank_account_id, statement_date, description, amount,
       external_ref, row_version
FROM trs.bank_statement_line
WHERE bank_account_id = $1 AND is_reconciled = false AND deleted_at IS NULL
ORDER BY statement_date, created_at
FOR UPDATE;

-- name: ListSettledPaymentsForBank :many
SELECT id, organization_id, payment_no, payment_type, bank_account_id,
       amount, payment_date, external_ref, correlation_id, row_version
FROM trs.payment
WHERE bank_account_id = $1 AND status = 'Settled' AND deleted_at IS NULL
ORDER BY payment_date, created_at
FOR UPDATE;

-- name: MarkStatementLineReconciled :execrows
UPDATE trs.bank_statement_line
SET matched_payment_id = $3, is_reconciled = true, updated_by = $4
WHERE id = $1 AND row_version = $2 AND is_reconciled = false AND deleted_at IS NULL;

-- Written inside the domain transaction (SDD-022): no dual-write.
-- name: InsertOutbox :exec
INSERT INTO trs.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- Relay poll — uses the partial index idx_outbox_unpublished with SKIP LOCKED.
-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM trs.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE trs.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- Consumer idempotency ledger (SDD-030). Returns rows affected (0 = already
-- processed); ON CONFLICT DO NOTHING makes at-least-once re-delivery a no-op.
-- name: MarkEventProcessed :execrows
INSERT INTO trs.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;
