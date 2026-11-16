-- sqlc queries for payroll-service (runs + encrypted results + statutory BPJS/PPh21 + outbox).

-- name: CreatePayrollRun :one
INSERT INTO pay.payroll_run (
    organization_id, run_no, period_month, period_year, correlation_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, organization_id, run_no, period_month, period_year, status, gl_journal_ref, correlation_id, row_version;

-- name: GetPayrollRun :one
SELECT id, organization_id, run_no, period_month, period_year, status, gl_journal_ref, correlation_id, row_version
FROM pay.payroll_run
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPayrollRuns :many
SELECT id, organization_id, run_no, period_month, period_year, status, gl_journal_ref, row_version
FROM pay.payroll_run
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY period_year DESC, period_month DESC, run_no DESC
LIMIT $1 OFFSET $2;

-- name: CountPayrollRuns :one
SELECT count(*)
FROM pay.payroll_run
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: DeletePayrollResultsByRun :exec
DELETE FROM pay.payroll_result
WHERE payroll_run_id = $1;

-- name: CreatePayrollResult :one
INSERT INTO pay.payroll_result (
    organization_id, payroll_run_id, employee_id,
    gross_amount, deduction_amount, tax_amount, net_amount,
    encrypted_gross, encrypted_net, encrypted_tax, pii_key_version, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING id, row_version;

-- name: ListPayrollResultsByRun :many
SELECT id, organization_id, payroll_run_id, employee_id,
       gross_amount, deduction_amount, tax_amount, net_amount,
       encrypted_gross, encrypted_net, encrypted_tax, pii_key_version, row_version
FROM pay.payroll_result
WHERE payroll_run_id = $1 AND deleted_at IS NULL
ORDER BY created_at;

-- name: CreatePayrollResultLine :exec
INSERT INTO pay.payroll_result_line (
    organization_id, payroll_result_id, component_code, amount, created_by
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListPayrollResultLinesByRun :many
SELECT l.id, l.payroll_result_id, l.component_code, l.amount, l.row_version
FROM pay.payroll_result_line l
JOIN pay.payroll_result r ON r.id = l.payroll_result_id
WHERE r.payroll_run_id = $1 AND l.deleted_at IS NULL AND r.deleted_at IS NULL
ORDER BY l.payroll_result_id, l.component_code;

-- name: CreateStatutoryContribution :exec
INSERT INTO pay.statutory_contribution (
    organization_id, payroll_result_id, contribution_type, employee_amount, employer_amount, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: ListStatutoryContributionsByRun :many
SELECT s.id, s.payroll_result_id, s.contribution_type, s.employee_amount, s.employer_amount, s.row_version
FROM pay.statutory_contribution s
JOIN pay.payroll_result r ON r.id = s.payroll_result_id
WHERE r.payroll_run_id = $1 AND s.deleted_at IS NULL AND r.deleted_at IS NULL
ORDER BY s.payroll_result_id, s.contribution_type;

-- name: CalculatePayrollRun :one
UPDATE pay.payroll_run
SET status = 'Calculated', updated_by = $3
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: ApprovePayrollRun :one
UPDATE pay.payroll_run
SET status = 'Approved', updated_by = $3
WHERE id = $1 AND status = 'Calculated' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: PostPayrollRun :one
UPDATE pay.payroll_run
SET status = 'Posted', gl_journal_ref = $4, updated_by = $3
WHERE id = $1 AND status IN ('Calculated', 'Approved') AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: SoftDeleteDraftPayrollRun :execrows
UPDATE pay.payroll_run
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL;

-- name: InsertOutbox :exec
INSERT INTO pay.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM pay.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE pay.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
