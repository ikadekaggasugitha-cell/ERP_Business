-- name: CreateAsset :one
INSERT INTO ast.asset (
    organization_id, asset_code, name, category, specification,
    acquisition_date, organization_ref, fixed_asset_ref, barcode, status, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING id, row_version;

-- name: GetAsset :one
SELECT * FROM ast.asset WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAssets :many
SELECT * FROM ast.asset
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY asset_code ASC
LIMIT $1 OFFSET $2;

-- name: CountAssets :one
SELECT count(*) FROM ast.asset
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'));

-- name: UpdateAsset :one
UPDATE ast.asset
SET name = $2, category = $3, specification = $4, barcode = $5, status = $6, updated_by = $7
WHERE id = $1 AND row_version = $8 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: UpdateAssetOrganizationRef :exec
UPDATE ast.asset
SET organization_ref = $2, updated_by = $3
WHERE id = $1 AND deleted_at IS NULL AND status <> 'Disposed'
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: SoftDeleteAsset :execrows
UPDATE ast.asset
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateLocationHistory :one
INSERT INTO ast.asset_location_history (
    organization_id, asset_id, organization_ref, warehouse_ref, location_note, effective_from, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: ListLocationHistory :many
SELECT * FROM ast.asset_location_history
WHERE asset_id = $1 AND deleted_at IS NULL
ORDER BY effective_from DESC, created_at DESC;

-- name: CreateAssetMovement :one
INSERT INTO ast.asset_movement (
    organization_id, asset_id, from_organization_ref, to_organization_ref, status, created_by
) VALUES ($1, $2, $3, $4, 'Draft', $5)
RETURNING id, row_version;

-- name: GetAssetMovement :one
SELECT * FROM ast.asset_movement WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAssetMovements :many
SELECT * FROM ast.asset_movement
WHERE deleted_at IS NULL
  AND (sqlc.narg('asset_id')::uuid IS NULL OR asset_id = sqlc.narg('asset_id'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAssetMovements :one
SELECT count(*) FROM ast.asset_movement
WHERE deleted_at IS NULL
  AND (sqlc.narg('asset_id')::uuid IS NULL OR asset_id = sqlc.narg('asset_id'));

-- name: ApproveAssetMovement :one
UPDATE ast.asset_movement
SET status = 'Approved', approved_by = $2, updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Draft' AND deleted_at IS NULL
RETURNING row_version;

-- name: CompleteAssetMovement :one
UPDATE ast.asset_movement
SET status = 'Completed', moved_at = now(), updated_by = $2
WHERE id = $1 AND row_version = $3 AND status = 'Approved' AND deleted_at IS NULL
RETURNING row_version;

-- name: CreateAssetVerification :one
INSERT INTO ast.asset_verification (
    organization_id, asset_id, verification_date, system_status, physical_status, discrepancy_note, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, row_version;

-- name: ListAssetVerifications :many
SELECT * FROM ast.asset_verification
WHERE deleted_at IS NULL
  AND (sqlc.narg('asset_id')::uuid IS NULL OR asset_id = sqlc.narg('asset_id'))
ORDER BY verification_date DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAssetVerifications :one
SELECT count(*) FROM ast.asset_verification
WHERE deleted_at IS NULL
  AND (sqlc.narg('asset_id')::uuid IS NULL OR asset_id = sqlc.narg('asset_id'));

-- name: InsertOutbox :exec
INSERT INTO ast.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ClaimUnpublishedOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM ast.outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE ast.outbox SET published_at = now() WHERE id = $1 AND created_at = $2;
