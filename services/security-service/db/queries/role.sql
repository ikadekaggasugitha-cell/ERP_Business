-- name: CreateRole :one
INSERT INTO sec.role (
    organization_id, code, name, is_system, created_by
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, row_version;

-- name: GetRole :one
SELECT id, organization_id, code, name, is_system,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM sec.role
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListRoles :many
SELECT id, organization_id, code, name, is_system, row_version
FROM sec.role
WHERE deleted_at IS NULL
  AND (sqlc.narg(is_system)::boolean IS NULL OR is_system = sqlc.narg(is_system)::boolean)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountRoles :one
SELECT count(*)::bigint
FROM sec.role
WHERE deleted_at IS NULL
  AND (sqlc.narg(is_system)::boolean IS NULL OR is_system = sqlc.narg(is_system)::boolean);

-- name: UpdateRole :one
UPDATE sec.role
SET name = $3, is_system = $4, updated_by = $5
WHERE id = $1 AND row_version = $2 AND is_system = false AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING row_version;

-- name: DeleteRole :execrows
UPDATE sec.role
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND is_system = false AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreatePermission :one
INSERT INTO sec.permission (
    organization_id, code, module, action, created_by
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, row_version;

-- name: ListPermissions :many
SELECT id, organization_id, code, module, action, row_version
FROM sec.permission
WHERE deleted_at IS NULL
  AND (sqlc.narg(module)::text IS NULL OR module = sqlc.narg(module)::text)
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountPermissions :one
SELECT count(*)::bigint
FROM sec.permission
WHERE deleted_at IS NULL
  AND (sqlc.narg(module)::text IS NULL OR module = sqlc.narg(module)::text);

-- name: CreateRolePermission :execrows
INSERT INTO sec.role_permission (
    organization_id, role_id, permission_id, created_by
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: ListRolePermissions :many
SELECT p.id, p.code, p.module, p.action
FROM sec.role_permission rp
JOIN sec.permission p ON p.id = rp.permission_id
WHERE rp.role_id = $1 AND p.deleted_at IS NULL
ORDER BY p.code;

-- name: DeleteRolePermission :execrows
DELETE FROM sec.role_permission
WHERE role_id = $1 AND permission_id = $2
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: InsertOutbox :exec
INSERT INTO sec.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM sec.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE sec.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
