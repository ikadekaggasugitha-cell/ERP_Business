# Seed Data Framework

Idempotent **reference / master data** seeds (SAD BAB 6.10 "Data Migration"). Safe to run
repeatedly — every statement uses `ON CONFLICT (<natural key>) DO NOTHING`, so re-running never
duplicates or overwrites.

## What is seeded

| DB               | Data |
| ---------------- | ---- |
| `currency_db`    | ISO-4217 currencies (IDR, USD, EUR, …) + exchange rate types (Spot/Closing/Average/Historical/Budget) |
| `localization_db`| Languages (en fallback, id) + locale formats (id-ID, en-US) |
| `org_db`         | Organization types (Holding→Project Site) |
| `security_db`    | Permissions (`<module>.<action>` for 14 modules) + system roles (SYSTEM_ADMIN, FINANCE_MANAGER, ACCOUNTANT, AUDITOR, VIEWER) + role→permission grants |
| `tax_db`         | Indonesian tax codes (PPN 11%, PPh 21/22/23/4(2), …) |
| `coa_db`         | Fiscal calendar template `STD-MONTHLY` + FY2026 + 12 monthly periods |
| `fixedasset_db`  | Asset categories (building, vehicle, machine, office equipment, computer, land) with default depreciation method / useful life |
| `budget_db`      | Default budget control policies (`HardStop` PO, `ApprovalOverride` APInvoice, `SoftWarning` JournalEntry) |
| `financialclose_db` | Standard closing checklist templates (bank reconciliation → financial statement preparation, 10 tasks) |
| `consolidation_db` | Reference holding consolidation group (`GRP-HOLDING`) + 4 standard elimination rules |
| `project_db`     | Reference CapEx project template (`TMPL-CAPEX`) + 3 standard WBS phases |
| `asset_db`       | Reference operational asset templates (`AST-TMPL-HVAC`, `AST-TMPL-GEN`) |
| `maintenance_db` | Standard preventive maintenance schedules (30-day HVAC, 6-month Generator) |
| `crm_db`         | Customer segments (SMB, Enterprise, Key Account, Government, Retail) with segment criteria |

Transactional/demo data is **not** seeded (chosen scope: standard reference only).

## Org scoping convention

Reference data is seeded under a **system organization** (`00000000-0000-0000-0000-000000000000`)
with actor `00000000-0000-0000-0000-000000000001`. Because RLS filters by `app.org_scope`, the
application should **include the system org id in every session scope** so shared reference data
is visible, e.g. `SET LOCAL app.org_scope = '{<system>,<tenant-path...>}'`.

Alternatively, seed **per tenant** by overriding the variables:
```bash
psql -d currency_db -v org_id=<tenant-uuid> -v actor=<user-uuid> -f db/seeds/currency_db/seed.sql
```

## Run order (after migrations + grants)

```bash
for db in currency_db localization_db org_db security_db tax_db coa_db fixedasset_db budget_db financialclose_db consolidation_db project_db asset_db maintenance_db crm_db; do
  psql -U postgres -d "$db" -f "db/seeds/$db/seed.sql"
done
```

Run as the owner/`erp_migrator` (RLS bypassed for owner). Idempotency verified on PostgreSQL 16:
running twice yields identical row counts and zero FK/enum violations.
