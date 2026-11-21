-- Seed: budget_db (reference). Idempotent. Run: psql -d budget_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO bud.budget_control_policy (id, organization_id, account_ref, transaction_type, control_type, created_by)
SELECT
  CASE WHEN :'org_id'::uuid = '00000000-0000-0000-0000-000000000000'::uuid THEN v.sys_id ELSE md5(:'org_id' || ':bud:policy:' || v.tx_type)::uuid END,
  :'org_id'::uuid,
  NULL,
  v.tx_type,
  v.ctrl_type,
  :'actor'::uuid
FROM (VALUES
  ('b0000000-0000-4000-8000-000000000001'::uuid, 'PurchaseOrder', 'HardStop'),
  ('b0000000-0000-4000-8000-000000000002'::uuid, 'APInvoice',     'ApprovalOverride'),
  ('b0000000-0000-4000-8000-000000000003'::uuid, 'JournalEntry',  'SoftWarning')
) AS v(sys_id, tx_type, ctrl_type)
WHERE NOT EXISTS (
  SELECT 1 FROM bud.budget_control_policy p
  WHERE p.organization_id = :'org_id'::uuid
    AND p.account_ref IS NULL
    AND p.transaction_type = v.tx_type
)
ON CONFLICT (id) DO NOTHING;
