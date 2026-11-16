#!/bin/sh
# One-shot: create all ERP slice & extended databases, cluster roles (db/ops/roles.sql),
# and a LOGIN role the services use (member of erp_app_rw => non-owner => RLS applies).
set -e
PGHOST="${PGHOST:-db}"
PGUSER="${POSTGRES_USER:-postgres}"
export PGPASSWORD="${POSTGRES_PASSWORD:-postgres}"
APP_USER="${ERP_APP_USER:-erp_app}"
APP_PASSWORD="${ERP_APP_PASSWORD:-app}"
PSQL="psql -h $PGHOST -U $PGUSER -v ON_ERROR_STOP=1"

ALL_DBS="gl_db ap_db ar_db coa_db tax_db treasury_db user_db security_db masterdata_db org_db workflow_db procurement_db inventory_db manufacturing_db sales_db hr_db payroll_db reporting_db audit_db currency_db asset_db budget_db consolidation_db crm_db dimension_db financialclose_db fixedasset_db localization_db maintenance_db project_db"

echo "bootstrap: waiting for postgres at $PGHOST..."
until pg_isready -h "$PGHOST" -U "$PGUSER" >/dev/null 2>&1; do sleep 1; done

for d in $ALL_DBS; do
  if ! $PSQL -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$d'" | grep -q 1; then
    $PSQL -d postgres -c "CREATE DATABASE $d"
    echo "bootstrap: created $d"
  fi
done

$PSQL -d postgres -f /ops/roles.sql
$PSQL -d postgres -c "
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$APP_USER') THEN
    EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L IN ROLE erp_app_rw', '$APP_USER', '$APP_PASSWORD');
  END IF;
END \$\$;
"
echo "bootstrap: done (30 DBs + roles + $APP_USER login)"
