#!/bin/sh
# One-shot: apply idempotent reference seeds (db/seeds/<db>/seed.sql) across all seeded DBs.
set -e
PGHOST="${PGHOST:-db}"
PGUSER="${POSTGRES_USER:-postgres}"
export PGPASSWORD="${POSTGRES_PASSWORD:-postgres}"

for db in currency_db localization_db org_db security_db tax_db coa_db fixedasset_db budget_db financialclose_db consolidation_db project_db asset_db maintenance_db crm_db; do
  if [ -f "/seeds/$db/seed.sql" ]; then
    psql -h "$PGHOST" -U "$PGUSER" -v ON_ERROR_STOP=1 -d "$db" -f "/seeds/$db/seed.sql"
    echo "seeds: loaded $db"
  fi
done
echo "seeds: done"

