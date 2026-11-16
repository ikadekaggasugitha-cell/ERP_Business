-- sqlc queries for hr-service (encrypted PII columns + blind index + HR master + outbox).

-- name: CreateEmployee :one
INSERT INTO hr.employee (
    organization_id, employee_no, full_name, national_id, date_of_birth,
    email, hire_date, user_ref_id,
    encrypted_full_name, encrypted_national_id, encrypted_dob, encrypted_email, national_id_hash,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING id, organization_id, employee_no, full_name, national_id, date_of_birth, email,
          encrypted_full_name, encrypted_national_id, encrypted_dob, encrypted_email, national_id_hash,
          hire_date, status, user_ref_id, row_version;

-- name: GetEmployee :one
SELECT id, organization_id, employee_no, full_name, national_id, date_of_birth, email,
       encrypted_full_name, encrypted_national_id, encrypted_dob, encrypted_email, national_id_hash,
       hire_date, status, user_ref_id, row_version
FROM hr.employee
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetEmployeeByNationalIDHash :one
SELECT id, organization_id, employee_no, full_name, national_id, date_of_birth, email,
       encrypted_full_name, encrypted_national_id, encrypted_dob, encrypted_email, national_id_hash,
       hire_date, status, user_ref_id, row_version
FROM hr.employee
WHERE organization_id = $1 AND national_id_hash = $2 AND deleted_at IS NULL;

-- name: ListEmployees :many
SELECT id, organization_id, employee_no, full_name, encrypted_full_name, encrypted_national_id,
       encrypted_dob, encrypted_email, hire_date, status, user_ref_id, row_version
FROM hr.employee
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY employee_no
LIMIT $1 OFFSET $2;

-- name: CountEmployees :one
SELECT count(*)
FROM hr.employee
WHERE deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: UpdateEmployee :one
UPDATE hr.employee
SET full_name = $3,
    national_id = $4,
    email = $5,
    encrypted_full_name = $6,
    encrypted_national_id = $7,
    encrypted_dob = $8,
    encrypted_email = $9,
    national_id_hash = $10,
    user_ref_id = $11,
    updated_by = $12
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: TerminateEmployee :one
UPDATE hr.employee
SET status = 'Terminated', updated_by = $3
WHERE id = $1 AND status = 'Active' AND row_version = $2 AND deleted_at IS NULL
RETURNING row_version;

-- name: SoftDeleteEmployee :execrows
UPDATE hr.employee
SET deleted_at = now(), updated_by = $3
WHERE id = $1 AND row_version = $2 AND deleted_at IS NULL;

-- name: CreatePosition :one
INSERT INTO hr.position (
    organization_id, code, title, organization_ref_id, parent_position_id, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, organization_id, code, title, organization_ref_id, parent_position_id, row_version;

-- name: ListPositions :many
SELECT id, organization_id, code, title, organization_ref_id, parent_position_id, row_version
FROM hr.position
WHERE deleted_at IS NULL
ORDER BY code
LIMIT $1 OFFSET $2;

-- name: CountPositions :one
SELECT count(*)
FROM hr.position
WHERE deleted_at IS NULL;

-- name: CreateEmployment :one
INSERT INTO hr.employment (
    organization_id, employee_id, position_id, employment_type,
    effective_from, effective_to, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, organization_id, employee_id, position_id, employment_type, effective_from, effective_to, row_version;

-- name: ListEmploymentsByEmployee :many
SELECT id, organization_id, employee_id, position_id, employment_type, effective_from, effective_to, row_version
FROM hr.employment
WHERE employee_id = $1 AND deleted_at IS NULL
ORDER BY effective_from DESC;

-- name: CreateCompensation :one
INSERT INTO hr.employee_compensation (
    organization_id, employee_id, component_code, amount, effective_from, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, organization_id, employee_id, component_code, amount, effective_from, row_version;

-- name: ListCompensationsByEmployee :many
SELECT id, organization_id, employee_id, component_code, amount, effective_from, row_version
FROM hr.employee_compensation
WHERE employee_id = $1 AND deleted_at IS NULL
ORDER BY effective_from DESC, component_code;

-- name: InsertOutbox :exec
INSERT INTO hr.outbox (aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DrainOutbox :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, organization_id, correlation_id, created_at
FROM hr.outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE hr.outbox SET published_at = now()
WHERE id = $1 AND created_at = $2;
