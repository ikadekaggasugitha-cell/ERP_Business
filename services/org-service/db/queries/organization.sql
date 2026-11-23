-- name: CreateOrganizationType :one
INSERT INTO org.organization_type (organization_id, code, name, level_hint, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: ListOrganizationTypes :many
SELECT id, organization_id, code, name, level_hint, is_active, row_version
FROM org.organization_type
WHERE deleted_at IS NULL
ORDER BY level_hint NULLS LAST, code
LIMIT $1 OFFSET $2;

-- name: CountOrganizationTypes :one
SELECT count(*)::bigint
FROM org.organization_type
WHERE deleted_at IS NULL;

-- name: GetOrganizationTypeByID :one
SELECT id, organization_id, code, name, level_hint, is_active, row_version
FROM org.organization_type
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpsertOrganizationAttribute :exec
INSERT INTO org.organization_attribute (
    organization_id, organization_ref_id, attr_key, attr_value, created_by
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (organization_ref_id, attr_key) DO UPDATE
SET attr_value = EXCLUDED.attr_value,
    updated_by = EXCLUDED.created_by,
    deleted_at = NULL;

-- name: ListOrganizationAttributes :many
SELECT attr_key, attr_value
FROM org.organization_attribute
WHERE organization_ref_id = $1 AND deleted_at IS NULL
ORDER BY attr_key;

-- name: CreateOrganization :one
INSERT INTO org.organization (
    organization_id, code, name, organization_type_id, parent_id, path,
    effective_from, effective_to, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id, status, row_version;

-- name: InsertSelfClosure :exec
INSERT INTO org.organization_closure (organization_id, ancestor_id, descendant_id, depth, created_by)
VALUES ($1, $2, $2, 0, $3)
ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;

-- name: InsertAncestorClosures :exec
INSERT INTO org.organization_closure (organization_id, ancestor_id, descendant_id, depth, created_by)
SELECT $1, c.ancestor_id, $2, c.depth + 1, $3
FROM org.organization_closure c
WHERE c.descendant_id = $4
ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;

-- name: GetOrganization :one
SELECT id, organization_id, code, name, organization_type_id, parent_id, path,
       status, effective_from, effective_to,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM org.organization
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListOrganizations :many
SELECT id, organization_id, code, name, organization_type_id, parent_id, path,
       status, effective_from, effective_to, row_version
FROM org.organization
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(parent_id)::uuid IS NULL OR parent_id = sqlc.narg(parent_id)::uuid)
ORDER BY path, code
LIMIT $1 OFFSET $2;

-- name: CountOrganizations :one
SELECT count(*)::bigint
FROM org.organization
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(parent_id)::uuid IS NULL OR parent_id = sqlc.narg(parent_id)::uuid);

-- name: ListOrganizationDescendants :many
SELECT o.id, o.code, o.name, o.organization_type_id, o.parent_id, o.path,
       o.status, o.effective_from, o.effective_to, o.row_version, c.depth
FROM org.organization_closure c
JOIN org.organization o ON o.id = c.descendant_id
WHERE c.ancestor_id = $1 AND o.deleted_at IS NULL
ORDER BY c.depth, o.path, o.code;

-- name: CountChildOrganizations :one
SELECT count(*)::bigint
FROM org.organization
WHERE parent_id = $1 AND deleted_at IS NULL;

-- name: UpdateOrganization :one
UPDATE org.organization
SET name = $3,
    organization_type_id = $4,
    effective_from = $5,
    effective_to = $6,
    updated_by = $7
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING status, row_version;

-- name: ActivateOrganization :one
UPDATE org.organization
SET status = 'Active', updated_by = $3
WHERE id = $1 AND status = 'Draft' AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: DeleteOrganization :execrows
UPDATE org.organization
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: InsertOutbox :exec
INSERT INTO org.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM org.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE org.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
