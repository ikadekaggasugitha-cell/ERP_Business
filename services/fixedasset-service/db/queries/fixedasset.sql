-- name: CreateAssetCategory :one
INSERT INTO fa.asset_category (
    organization_id, code, name, default_depreciation_method, default_useful_life_months, is_active, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: GetAssetCategory :one
SELECT * FROM fa.asset_category WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAssetCategories :many
SELECT * FROM fa.asset_category WHERE deleted_at IS NULL ORDER BY code LIMIT $1 OFFSET $2;

-- name: CountAssetCategories :one
SELECT COUNT(*) FROM fa.asset_category WHERE deleted_at IS NULL;

-- name: UpdateAssetCategory :one
UPDATE fa.asset_category
SET name = $3, default_depreciation_method = $4, default_useful_life_months = $5, is_active = $6, updated_by = $7
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteAssetCategory :execrows
UPDATE fa.asset_category SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateFixedAsset :one
INSERT INTO fa.fixed_asset (
    organization_id, asset_no, name, asset_category_id, financial_organization_ref,
    acquisition_date, acquisition_cost, salvage_value, useful_life_months, depreciation_method,
    depreciation_start_date, asset_account_id, accum_depreciation_account_id,
    depreciation_expense_account_id, cost_center_ref, net_book_value, parent_asset_id, status, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
RETURNING id, status, row_version;

-- name: GetFixedAsset :one
SELECT * FROM fa.fixed_asset WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFixedAssets :many
SELECT * FROM fa.fixed_asset
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY asset_no
LIMIT $1 OFFSET $2;

-- name: CountFixedAssets :one
SELECT COUNT(*) FROM fa.fixed_asset
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: UpdateFixedAssetDraft :one
UPDATE fa.fixed_asset
SET name = $3, salvage_value = $4, useful_life_months = $5, depreciation_method = $6, cost_center_ref = $7, updated_by = $8
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: TransitionFixedAsset :one
UPDATE fa.fixed_asset
SET status = $3, net_book_value = $4, updated_by = $5
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteFixedAssetDraft :execrows
UPDATE fa.fixed_asset SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND status = 'Draft' AND deleted_at IS NULL;

-- name: CreateDepreciationEntry :one
INSERT INTO fa.depreciation_entry (
    organization_id, fixed_asset_id, fiscal_period_ref, depreciation_amount,
    accumulated_depreciation, net_book_value, is_posted, gl_journal_ref, run_date, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, row_version;

-- name: ListDepreciationEntriesByAsset :many
SELECT * FROM fa.depreciation_entry
WHERE fixed_asset_id = $1 AND deleted_at IS NULL
ORDER BY run_date DESC, created_at DESC;

-- name: CreateAssetTransaction :one
INSERT INTO fa.asset_transaction (
    organization_id, fixed_asset_id, transaction_type, transaction_date, amount,
    gain_loss_amount, from_financial_organization_ref, to_financial_organization_ref,
    gl_journal_ref, notes, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, row_version;

-- name: ListAssetTransactionsByAsset :many
SELECT * FROM fa.asset_transaction
WHERE fixed_asset_id = $1 AND deleted_at IS NULL
ORDER BY transaction_date DESC, created_at DESC;

-- name: InsertOutbox :exec
INSERT INTO fa.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM fa.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE fa.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;

-- name: MarkEventProcessed :execrows
INSERT INTO fa.processed_event (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;
