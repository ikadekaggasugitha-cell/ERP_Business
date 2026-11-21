-- Encrypt remaining payroll PII amounts (deductions, result lines, statutory contributions)
-- and add local employee status cache populated from hr.employee.updated.

ALTER TABLE pay.payroll_result
    ADD COLUMN IF NOT EXISTS encrypted_deduction text;

ALTER TABLE pay.payroll_result_line
    ADD COLUMN IF NOT EXISTS encrypted_amount text;

ALTER TABLE pay.statutory_contribution
    ADD COLUMN IF NOT EXISTS encrypted_employee_amount text,
    ADD COLUMN IF NOT EXISTS encrypted_employer_amount text;

CREATE TABLE IF NOT EXISTS pay.employee_cache (
    organization_id uuid        NOT NULL,
    employee_id     uuid        NOT NULL,
    employee_no     varchar(50) NOT NULL DEFAULT '',
    status          varchar(20) NOT NULL DEFAULT 'Active',
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, employee_id)
);

ALTER TABLE pay.employee_cache ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS scope_isolation ON pay.employee_cache;
CREATE POLICY scope_isolation ON pay.employee_cache
    USING (organization_id = ANY (public.current_org_scope()))
    WITH CHECK (organization_id = ANY (public.current_org_scope()));
