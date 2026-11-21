DROP TABLE IF EXISTS pay.employee_cache;

ALTER TABLE pay.statutory_contribution
    DROP COLUMN IF EXISTS encrypted_employee_amount,
    DROP COLUMN IF EXISTS encrypted_employer_amount;

ALTER TABLE pay.payroll_result_line
    DROP COLUMN IF EXISTS encrypted_amount;

ALTER TABLE pay.payroll_result
    DROP COLUMN IF EXISTS encrypted_deduction;
