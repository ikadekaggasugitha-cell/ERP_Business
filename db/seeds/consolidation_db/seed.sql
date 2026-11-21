-- Seed: consolidation_db (reference). Idempotent. Run: psql -d consolidation_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO cons.consolidation_group (id, organization_id, code, name, parent_group_id, presentation_currency_code, created_by) VALUES
  ('c0050000-0000-4000-8000-000000000001', :'org_id', 'GRP-HOLDING', 'Holding Consolidated Group (Reference)', NULL, 'IDR', :'actor')
ON CONFLICT (organization_id, code) DO NOTHING;

INSERT INTO cons.elimination_rule (id, organization_id, consolidation_group_id, rule_type, config, is_active, created_by)
SELECT v.id, :'org_id'::uuid, g.id, v.rule_type, v.config, true, :'actor'::uuid
FROM (VALUES
  ('c0051000-0000-4000-8000-000000000001'::uuid, 'ICReceivablePayable', '{"description":"Eliminate intercompany AR/AP balances","matchOn":"counterparty"}'::jsonb),
  ('c0051000-0000-4000-8000-000000000002'::uuid, 'ICSalesPurchase', '{"description":"Eliminate intercompany revenue and COGS","matchOn":"counterparty"}'::jsonb),
  ('c0051000-0000-4000-8000-000000000003'::uuid, 'UnrealizedProfit', '{"description":"Eliminate unrealized intercompany inventory profit"}'::jsonb),
  ('c0051000-0000-4000-8000-000000000004'::uuid, 'InvestmentEquity', '{"description":"Eliminate parent investment against subsidiary equity"}'::jsonb)
) AS v(id, rule_type, config)
JOIN cons.consolidation_group g
  ON g.organization_id = :'org_id'::uuid AND g.code = 'GRP-HOLDING'
ON CONFLICT (id) DO NOTHING;
