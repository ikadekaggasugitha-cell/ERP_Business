#!/usr/bin/env bash
# Logical backup of every ERP service database (custom format) + global roles.
# Env: PGHOST/PGPORT/PGUSER/PGPASSWORD (libpq), BACKUP_DIR, DBS (space-separated).
# Usage: BACKUP_DIR=/var/backups/erp ./logical_backup.sh
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/erp}"
DBS="${DBS:-org_db user_db security_db workflow_db masterdata_db audit_db coa_db currency_db \
gl_db ap_db ar_db tax_db treasury_db procurement_db inventory_db sales_db manufacturing_db \
hr_db payroll_db reporting_db fixedasset_db dimension_db budget_db financialclose_db \
consolidation_db project_db asset_db maintenance_db crm_db localization_db}"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$BACKUP_DIR/$TS"
mkdir -p "$OUT"

echo "[$(date -u +%FT%TZ)] backup -> $OUT"
pg_dumpall --globals-only > "$OUT/globals.sql"          # roles/grants (see db/ops/roles.sql)
for db in $DBS; do
  echo "  dumping $db"
  # -Fc custom format: parallel-restorable, compressed; --no-owner for portability
  pg_dump -Fc --no-owner -d "$db" -f "$OUT/$db.dump"
done
sha256sum "$OUT"/*.dump > "$OUT/SHA256SUMS" 2>/dev/null || shasum -a 256 "$OUT"/*.dump > "$OUT/SHA256SUMS"
echo "[$(date -u +%FT%TZ)] done. Ship $OUT to durable/offsite storage (S3, etc.)."
echo "$TS"
