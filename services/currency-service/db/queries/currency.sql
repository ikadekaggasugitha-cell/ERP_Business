-- name: CreateCurrency :one
INSERT INTO cur.currency (organization_id, code, numeric_code, name, precision, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: GetCurrency :one
SELECT * FROM cur.currency
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCurrencyByCode :one
SELECT * FROM cur.currency
WHERE code = $1 AND deleted_at IS NULL
ORDER BY (organization_id = '00000000-0000-0000-0000-000000000000'::uuid) ASC
LIMIT 1;

-- name: ListCurrencies :many
SELECT id, organization_id, code, numeric_code, name, precision, is_active, row_version
FROM cur.currency
WHERE deleted_at IS NULL
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active')::boolean)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountCurrencies :one
SELECT COUNT(*)
FROM cur.currency
WHERE deleted_at IS NULL
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active')::boolean);

-- name: UpdateCurrency :one
UPDATE cur.currency
SET name = $3, numeric_code = $4, precision = $5, is_active = $6, updated_by = $7
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: DeleteCurrency :execrows
UPDATE cur.currency
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateExchangeRateType :one
INSERT INTO cur.exchange_rate_type (organization_id, code, name, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, row_version;

-- name: GetExchangeRateType :one
SELECT * FROM cur.exchange_rate_type
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetExchangeRateTypeByCode :one
SELECT * FROM cur.exchange_rate_type
WHERE code = $1 AND deleted_at IS NULL
ORDER BY (organization_id = '00000000-0000-0000-0000-000000000000'::uuid) ASC
LIMIT 1;

-- name: ListExchangeRateTypes :many
SELECT id, organization_id, code, name, row_version
FROM cur.exchange_rate_type
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountExchangeRateTypes :one
SELECT COUNT(*)
FROM cur.exchange_rate_type
WHERE deleted_at IS NULL;

-- name: CreateExchangeRate :one
INSERT INTO cur.exchange_rate (
    organization_id, from_currency, to_currency, rate_type_id, rate, valid_from, valid_to, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, row_version;

-- name: GetExchangeRate :one
SELECT er.id, er.organization_id, er.from_currency, er.to_currency, er.rate_type_id, er.rate, er.valid_from, er.valid_to, er.row_version, rt.code as rate_type_code, rt.name as rate_type_name
FROM cur.exchange_rate er
JOIN cur.exchange_rate_type rt ON rt.id = er.rate_type_id
WHERE er.id = $1 AND er.deleted_at IS NULL;

-- name: ListExchangeRates :many
SELECT er.id, er.organization_id, er.from_currency, er.to_currency, er.rate_type_id, er.rate, er.valid_from, er.valid_to, er.row_version, rt.code as rate_type_code, rt.name as rate_type_name
FROM cur.exchange_rate er
JOIN cur.exchange_rate_type rt ON rt.id = er.rate_type_id
WHERE er.deleted_at IS NULL
  AND (sqlc.arg(from_currency)::text = '' OR er.from_currency = sqlc.arg(from_currency)::text)
  AND (sqlc.arg(to_currency)::text = '' OR er.to_currency = sqlc.arg(to_currency)::text)
  AND (sqlc.arg(rate_type_id)::text = '' OR er.rate_type_id::text = sqlc.arg(rate_type_id)::text OR UPPER(rt.code) = UPPER(sqlc.arg(rate_type_id)::text))
  AND (sqlc.narg('on_date')::date IS NULL OR (er.valid_from <= sqlc.narg('on_date')::date AND (er.valid_to IS NULL OR er.valid_to >= sqlc.narg('on_date')::date)))
ORDER BY er.valid_from DESC, er.from_currency, er.to_currency
LIMIT $1 OFFSET $2;

-- name: CountExchangeRates :one
SELECT COUNT(*)
FROM cur.exchange_rate er
JOIN cur.exchange_rate_type rt ON rt.id = er.rate_type_id
WHERE er.deleted_at IS NULL
  AND (sqlc.arg(from_currency)::text = '' OR er.from_currency = sqlc.arg(from_currency)::text)
  AND (sqlc.arg(to_currency)::text = '' OR er.to_currency = sqlc.arg(to_currency)::text)
  AND (sqlc.arg(rate_type_id)::text = '' OR er.rate_type_id::text = sqlc.arg(rate_type_id)::text OR UPPER(rt.code) = UPPER(sqlc.arg(rate_type_id)::text))
  AND (sqlc.narg('on_date')::date IS NULL OR (er.valid_from <= sqlc.narg('on_date')::date AND (er.valid_to IS NULL OR er.valid_to >= sqlc.narg('on_date')::date)));

-- name: GetLatestRate :one
SELECT er.id, er.organization_id, er.from_currency, er.to_currency, er.rate_type_id, er.rate, er.valid_from, er.valid_to, er.row_version
FROM cur.exchange_rate er
WHERE er.deleted_at IS NULL
  AND er.from_currency = $1
  AND er.to_currency = $2
  AND er.rate_type_id = $3
  AND er.valid_from <= $4
  AND (er.valid_to IS NULL OR er.valid_to >= $4)
ORDER BY er.valid_from DESC, (er.organization_id = '00000000-0000-0000-0000-000000000000'::uuid) ASC
LIMIT 1;

-- name: UpdateExchangeRate :one
UPDATE cur.exchange_rate
SET rate = $3, valid_to = $4, updated_by = $5
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: DeleteExchangeRate :execrows
UPDATE cur.exchange_rate
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: InsertOutbox :exec
INSERT INTO cur.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM cur.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE cur.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

