-- name: CreateChartOfAccounts :one
INSERT INTO coa.chart_of_accounts (organization_id, code, name, structure, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, row_version;

-- name: GetChartOfAccounts :one
SELECT * FROM coa.chart_of_accounts
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListChartsOfAccounts :many
SELECT id, organization_id, code, name, structure, row_version
FROM coa.chart_of_accounts
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountChartsOfAccounts :one
SELECT COUNT(*)
FROM coa.chart_of_accounts
WHERE deleted_at IS NULL;

-- name: UpdateChartOfAccounts :one
UPDATE coa.chart_of_accounts
SET name = $3, structure = $4, updated_by = $5
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: DeleteChartOfAccounts :execrows
UPDATE coa.chart_of_accounts
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateAccount :one
INSERT INTO coa.account (
    organization_id, chart_of_accounts_id, code, name, account_type,
    account_class, is_control, is_monetary, allow_posting, parent_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id, row_version;

-- name: GetAccount :one
SELECT * FROM coa.account
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAccounts :many
SELECT id, chart_of_accounts_id, code, name, account_type, account_class,
       is_control, is_monetary, allow_posting, parent_id, row_version
FROM coa.account
WHERE deleted_at IS NULL
  AND (sqlc.arg(account_type)::text = '' OR account_type = sqlc.arg(account_type)::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountAccounts :one
SELECT COUNT(*)
FROM coa.account
WHERE deleted_at IS NULL
  AND (sqlc.arg(account_type)::text = '' OR account_type = sqlc.arg(account_type)::text);

-- name: ListAccountsByType :many
SELECT id, chart_of_accounts_id, code, name, account_type, account_class,
       is_control, is_monetary, allow_posting, parent_id, row_version
FROM coa.account
WHERE organization_id = $1 AND account_type = $2 AND deleted_at IS NULL
ORDER BY code
LIMIT $3 OFFSET $4;

-- name: UpdateAccount :one
UPDATE coa.account
SET name = $3,
    account_type = $4,
    account_class = $5,
    is_control = $6,
    is_monetary = $7,
    allow_posting = $8,
    parent_id = $9,
    updated_by = $10
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: DeleteAccount :execrows
UPDATE coa.account
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CountChildAccounts :one
SELECT COUNT(*)
FROM coa.account
WHERE parent_id = $1 AND deleted_at IS NULL;

-- name: CreatePostingRule :one
INSERT INTO coa.posting_rule (
    organization_id, chart_of_accounts_id, source_module, event_key,
    debit_account_rule, credit_account_rule, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, row_version;

-- name: ListPostingRulesBySource :many
SELECT id, chart_of_accounts_id, source_module, event_key,
       debit_account_rule, credit_account_rule, row_version
FROM coa.posting_rule
WHERE organization_id = $1 AND source_module = $2 AND deleted_at IS NULL
ORDER BY event_key
LIMIT $3 OFFSET $4;

-- name: CreateFinancialOrganization :one
INSERT INTO fin.financial_organization (
    organization_id, code, name, organization_ref_id, functional_currency_code,
    chart_of_accounts_id, fiscal_calendar_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, row_version;

-- name: GetFinancialOrganization :one
SELECT * FROM fin.financial_organization
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFinancialOrganizations :many
SELECT id, code, name, organization_ref_id, functional_currency_code,
       chart_of_accounts_id, fiscal_calendar_id, row_version
FROM fin.financial_organization
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountFinancialOrganizations :one
SELECT COUNT(*)
FROM fin.financial_organization
WHERE deleted_at IS NULL;

-- name: CreateLedger :one
INSERT INTO fin.ledger (
    organization_id, financial_organization_id, code, ledger_type, currency_code, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, row_version;

-- name: ListLedgers :many
SELECT id, financial_organization_id, code, ledger_type, currency_code, row_version
FROM fin.ledger
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountLedgers :one
SELECT COUNT(*)
FROM fin.ledger
WHERE deleted_at IS NULL;

-- name: CreateFiscalCalendar :one
INSERT INTO fin.fiscal_calendar (organization_id, code, name, period_type, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, row_version;

-- name: ListFiscalCalendars :many
SELECT id, code, name, period_type, row_version
FROM fin.fiscal_calendar
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountFiscalCalendars :one
SELECT COUNT(*)
FROM fin.fiscal_calendar
WHERE deleted_at IS NULL;

-- name: CreateFiscalYear :one
INSERT INTO fin.fiscal_year (organization_id, fiscal_calendar_id, year_no, name, start_date, end_date, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, 'Open', $7)
RETURNING id, status, row_version;

-- name: ListFiscalYears :many
SELECT id, fiscal_calendar_id, year_no, name, start_date, end_date, status, row_version
FROM fin.fiscal_year
WHERE deleted_at IS NULL
ORDER BY year_no DESC
LIMIT $1 OFFSET $2;

-- name: CountFiscalYears :one
SELECT COUNT(*)
FROM fin.fiscal_year
WHERE deleted_at IS NULL;

-- name: CreateFiscalPeriod :one
INSERT INTO fin.fiscal_period (organization_id, fiscal_year_id, period_no, start_date, end_date, status, created_by)
VALUES ($1, $2, $3, $4, $5, 'Open', $6)
RETURNING id, status, row_version;

-- name: GetFiscalPeriod :one
SELECT * FROM fin.fiscal_period
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFiscalPeriods :many
SELECT id, fiscal_year_id, period_no, start_date, end_date, status, row_version
FROM fin.fiscal_period
WHERE deleted_at IS NULL
  AND (sqlc.arg(fiscal_year_id)::text = '' OR fiscal_year_id::text = sqlc.arg(fiscal_year_id)::text)
ORDER BY start_date DESC, period_no
LIMIT $1 OFFSET $2;

-- name: CountFiscalPeriods :one
SELECT COUNT(*)
FROM fin.fiscal_period
WHERE deleted_at IS NULL
  AND (sqlc.arg(fiscal_year_id)::text = '' OR fiscal_year_id::text = sqlc.arg(fiscal_year_id)::text);

-- name: UpdateFiscalPeriodStatus :one
UPDATE fin.fiscal_period
SET status = sqlc.arg(new_status)::varchar(20), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = sqlc.arg(expected_status)::varchar(20) AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: SyncFiscalPeriodStatus :execrows
UPDATE fin.fiscal_period
SET status = sqlc.arg(new_status)::varchar(20), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: MarkEventProcessed :execrows
INSERT INTO coa.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: InsertOutbox :exec
INSERT INTO coa.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM coa.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE coa.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
