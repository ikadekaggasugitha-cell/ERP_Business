-- name: CreateAppUser :one
-- No password column — authentication is delegated to the IdP (SAD BAB 7.3).
-- username/email uniqueness is enforced by the DB (uq_app_user_username/email).
INSERT INTO usr.app_user (
    organization_id, username, email, full_name, default_organization_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, status, row_version;

-- name: GetAppUser :one
SELECT id, organization_id, username, email, full_name, status, default_organization_id,
       created_at, created_by, updated_at, updated_by, deleted_at, row_version
FROM usr.app_user
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAppUsers :many
SELECT id, organization_id, username, email, full_name, status, default_organization_id, row_version
FROM usr.app_user
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY username
LIMIT $1 OFFSET $2;

-- name: CountAppUsers :one
SELECT count(*)::bigint
FROM usr.app_user
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text);

-- name: UpdateAppUser :one
UPDATE usr.app_user
SET email = $3,
    full_name = $4,
    default_organization_id = $5,
    updated_by = $6
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING status, row_version;

-- name: ChangeUserStatus :one
UPDATE usr.app_user
SET status = $3, updated_by = $4
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid
RETURNING status, row_version;

-- name: DeleteAppUser :execrows
UPDATE usr.app_user
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
  AND organization_id <> '00000000-0000-0000-0000-000000000000'::uuid;

-- name: CreateUserOrgAssignment :one
-- Scope-based access grant (INT-012). role_id/organization_ref_id are logical
-- refs to security_db/org_db (no physical FK).
INSERT INTO usr.user_org_assignment (
    organization_id, user_id, organization_ref_id, role_id, created_by
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (user_id, organization_ref_id, role_id) DO UPDATE
SET deleted_at = NULL, updated_by = EXCLUDED.created_by
RETURNING id, row_version;

-- name: ListUserOrgAssignments :many
SELECT id, user_id, organization_ref_id, role_id, row_version
FROM usr.user_org_assignment
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at;

-- name: RemoveUserOrgAssignment :execrows
UPDATE usr.user_org_assignment
SET deleted_at = now(), updated_by = $4
WHERE user_id = $1 AND role_id = $2 AND organization_ref_id = $3 AND deleted_at IS NULL;

-- name: InsertOutbox :exec
-- Written inside the domain transaction (SDD-022): no dual-write.
INSERT INTO usr.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
-- Relay poll — uses the partial index idx_outbox_unpublished with SKIP LOCKED.
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM usr.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE usr.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
