-- name: CreateLanguage :one
INSERT INTO loc.language (organization_id, code, name, is_active, is_fallback, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: GetLanguage :one
SELECT * FROM loc.language WHERE id = $1 AND deleted_at IS NULL;

-- name: GetLanguageByCode :one
SELECT * FROM loc.language WHERE code = $1 AND deleted_at IS NULL
ORDER BY (organization_id = '00000000-0000-0000-0000-000000000000'::uuid) ASC LIMIT 1;

-- name: GetFallbackLanguage :one
SELECT * FROM loc.language WHERE is_fallback = TRUE AND is_active = TRUE AND deleted_at IS NULL
ORDER BY (organization_id = '00000000-0000-0000-0000-000000000000'::uuid) ASC LIMIT 1;

-- name: ListLanguages :many
SELECT * FROM loc.language WHERE deleted_at IS NULL ORDER BY code LIMIT $1 OFFSET $2;

-- name: CountLanguages :one
SELECT COUNT(*) FROM loc.language WHERE deleted_at IS NULL;

-- name: UpdateLanguage :one
UPDATE loc.language SET name = $3, is_active = $4, is_fallback = $5, updated_by = $6
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: ClearFallbackExcept :exec
UPDATE loc.language SET is_fallback = FALSE
WHERE is_fallback = TRUE AND deleted_at IS NULL AND id <> $1 AND organization_id = $2;

-- name: CreateTranslation :one
INSERT INTO loc.translation (organization_id, language_code, resource_key, resource_value, module, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: GetTranslation :one
SELECT * FROM loc.translation WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTranslations :many
SELECT * FROM loc.translation
WHERE deleted_at IS NULL
  AND (sqlc.narg('language_code')::text IS NULL OR language_code = sqlc.narg('language_code')::text)
  AND (sqlc.narg('module')::text IS NULL OR module = sqlc.narg('module')::text)
ORDER BY language_code, resource_key
LIMIT $1 OFFSET $2;

-- name: CountTranslations :one
SELECT COUNT(*) FROM loc.translation
WHERE deleted_at IS NULL
  AND (sqlc.narg('language_code')::text IS NULL OR language_code = sqlc.narg('language_code')::text)
  AND (sqlc.narg('module')::text IS NULL OR module = sqlc.narg('module')::text);

-- name: ListTranslationsByLanguage :many
SELECT DISTINCT ON (resource_key) resource_key, resource_value FROM loc.translation
WHERE language_code = $1 AND deleted_at IS NULL
ORDER BY resource_key, (organization_id = '00000000-0000-0000-0000-000000000000'::uuid) ASC;

-- name: UpdateTranslation :one
UPDATE loc.translation SET resource_value = $3, module = $4, updated_by = $5
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: DeleteTranslation :execrows
UPDATE loc.translation SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreateLocaleFormat :one
INSERT INTO loc.locale_format (organization_id, locale_code, date_format, number_format, currency_format, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, row_version;

-- name: GetLocaleFormat :one
SELECT * FROM loc.locale_format WHERE id = $1 AND deleted_at IS NULL;

-- name: ListLocaleFormats :many
SELECT * FROM loc.locale_format WHERE deleted_at IS NULL ORDER BY locale_code LIMIT $1 OFFSET $2;

-- name: CountLocaleFormats :one
SELECT COUNT(*) FROM loc.locale_format WHERE deleted_at IS NULL;

-- name: UpdateLocaleFormat :one
UPDATE loc.locale_format SET date_format = $3, number_format = $4, currency_format = $5, updated_by = $6
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: InsertOutbox :exec
INSERT INTO loc.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM loc.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE loc.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
