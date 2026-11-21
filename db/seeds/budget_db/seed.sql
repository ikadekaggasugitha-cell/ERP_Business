-- Seed: budget_db (reference). Idempotent. Run: psql -d budget_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO bud.budget_control_policy (id, organization_id, account_ref, transaction_type, control_type, created_by) VALUES
  ('b0000000-0000-4000-8000-000000000001', :'org_id', NULL, 'PurchaseOrder', 'HardStop', :'actor'),
  ('b0000000-0000-4000-8000-000000000002', :'org_id', NULL, 'APInvoice', 'ApprovalOverride', :'actor'),
  ('b0000000-0000-4000-8000-000000000003', :'org_id', NULL, 'JournalEntry', 'SoftWarning', :'actor')
ON CONFLICT (id) DO NOTHING;
