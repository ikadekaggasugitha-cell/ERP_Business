-- name: CreateParty :one
INSERT INTO md.party (
    organization_id, code, name, party_type, tax_id, is_shared, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, row_version;

-- name: GetParty :one
SELECT id, organization_id, code, name, party_type, tax_id, is_shared,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM md.party
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListParties :many
SELECT id, organization_id, code, name, party_type, tax_id, is_shared, row_version
FROM md.party
WHERE deleted_at IS NULL
  AND (sqlc.narg(party_type)::text IS NULL OR party_type = sqlc.narg(party_type)::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountParties :one
SELECT count(*)::bigint
FROM md.party
WHERE deleted_at IS NULL
  AND (sqlc.narg(party_type)::text IS NULL OR party_type = sqlc.narg(party_type)::text);

-- name: UpdateParty :one
UPDATE md.party
SET name = $3, party_type = $4, tax_id = $5, is_shared = $6, updated_by = $7
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteParty :execrows
UPDATE md.party
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreatePartyAddress :one
INSERT INTO md.party_address (
    organization_id, party_id, address_type, line1, line2, city,
    state_province, postal_code, country_code, is_primary, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id, row_version;

-- name: ListPartyAddresses :many
SELECT id, party_id, address_type, line1, line2, city, state_province, postal_code, country_code, is_primary, row_version
FROM md.party_address
WHERE party_id = $1 AND deleted_at IS NULL
ORDER BY is_primary DESC, created_at;

-- name: CreateItem :one
INSERT INTO md.item (
    organization_id, code, name, item_type, base_uom, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, row_version;

-- name: GetItem :one
SELECT id, organization_id, code, name, item_type, base_uom,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM md.item
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListItems :many
SELECT id, organization_id, code, name, item_type, base_uom, row_version
FROM md.item
WHERE deleted_at IS NULL
  AND (sqlc.narg(item_type)::text IS NULL OR item_type = sqlc.narg(item_type)::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountItems :one
SELECT count(*)::bigint
FROM md.item
WHERE deleted_at IS NULL
  AND (sqlc.narg(item_type)::text IS NULL OR item_type = sqlc.narg(item_type)::text);

-- name: UpdateItem :one
UPDATE md.item
SET name = $3, item_type = $4, base_uom = $5, updated_by = $6
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteItem :execrows
UPDATE md.item
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateItemUOM :exec
INSERT INTO md.item_uom (
    organization_id, item_id, uom_code, conversion_to_base, is_base, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: ListItemUOMs :many
SELECT id, item_id, uom_code, conversion_to_base, is_base, row_version
FROM md.item_uom
WHERE item_id = $1 AND deleted_at IS NULL
ORDER BY is_base DESC, uom_code;

-- name: InsertOutbox :exec
INSERT INTO md.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM md.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE md.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
