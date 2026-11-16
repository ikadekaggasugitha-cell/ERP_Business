#!/bin/sh
# One-shot: apply least-privilege grants (db/ops/apply_grants.sql) per database,
# after migrations have created the tables.
set -e
PGHOST="${PGHOST:-db}"
PGUSER="${POSTGRES_USER:-postgres}"
export PGPASSWORD="${POSTGRES_PASSWORD:-postgres}"

ALL_DBS="gl_db ap_db ar_db coa_db tax_db treasury_db user_db security_db masterdata_db org_db workflow_db procurement_db inventory_db manufacturing_db sales_db hr_db payroll_db reporting_db audit_db currency_db asset_db budget_db consolidation_db crm_db dimension_db financialclose_db fixedasset_db localization_db maintenance_db project_db"

for d in $ALL_DBS; do
  psql -h "$PGHOST" -U "$PGUSER" -v ON_ERROR_STOP=1 -d "$d" -f /ops/apply_grants.sql
  echo "grants: applied on $d"
done
echo "grants: done (30 DBs)"
