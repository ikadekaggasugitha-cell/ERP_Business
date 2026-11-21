#!/usr/bin/env bash
# Automated monthly partition maintenance across all 30 ERP PostgreSQL databases.
# Runs public.ensure_future_partitions(parent, MONTHS_AHEAD) for every partitioned
# table discovered in pg_partitioned_table, and optionally detaches partitions
# older than DETACH_KEEP_MONTHS when explicitly configured.
set -euo pipefail

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${POSTGRES_USER:-postgres}"
export PGPASSWORD="${POSTGRES_PASSWORD:-postgres}"
MONTHS_AHEAD="${MONTHS_AHEAD:-6}"
DETACH_KEEP_MONTHS="${DETACH_KEEP_MONTHS:-}"
DRY_RUN="${DRY_RUN:-false}"

ALL_DBS=(
  gl_db ap_db ar_db coa_db tax_db treasury_db user_db security_db
  masterdata_db org_db workflow_db procurement_db inventory_db
  manufacturing_db sales_db hr_db payroll_db reporting_db audit_db
  currency_db asset_db budget_db consolidation_db crm_db dimension_db
  financialclose_db fixedasset_db localization_db maintenance_db project_db
)

if ! [[ "$MONTHS_AHEAD" =~ ^[0-9]+$ ]] || [ "$MONTHS_AHEAD" -lt 1 ]; then
  echo "ERROR: MONTHS_AHEAD must be a positive integer (got '$MONTHS_AHEAD')" >&2
  exit 1
fi

if [ -n "$DETACH_KEEP_MONTHS" ] && { ! [[ "$DETACH_KEEP_MONTHS" =~ ^[0-9]+$ ]] || [ "$DETACH_KEEP_MONTHS" -lt 1 ]; }; then
  echo "ERROR: DETACH_KEEP_MONTHS must be a positive integer when set (got '$DETACH_KEEP_MONTHS')" >&2
  exit 1
fi

maintained_dbs=0
maintained_tables=0

for db in "${ALL_DBS[@]}"; do
  exists=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db'" 2>/dev/null || true)
  if [ "$exists" != "1" ]; then
    echo "skip: database '$db' does not exist"
    continue
  fi

  has_fn=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$db" -tAc "SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='public' AND p.proname='ensure_future_partitions' LIMIT 1" || true)
  if [ "$has_fn" != "1" ]; then
    echo "skip: $db (no public.ensure_future_partitions function)"
    continue
  fi

  # Order child partitioned tables (e.g. gl.journal_line) BEFORE referenced parent tables (e.g. gl.journal_entry)
  tables=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$db" -tAc "
    SELECT format('%I.%I', n.nspname, c.relname)
    FROM pg_partitioned_table pt
    JOIN pg_class c ON c.oid = pt.partrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
    ORDER BY EXISTS (
      SELECT 1 FROM pg_constraint fk
      WHERE fk.contype = 'f' AND fk.confrelid = c.oid AND fk.conrelid <> c.oid
    ) ASC, 1
  ")

  if [ -z "$tables" ]; then
    continue
  fi

  maintained_dbs=$((maintained_dbs + 1))
  while IFS= read -r tbl; do
    [ -z "$tbl" ] && continue
    maintained_tables=$((maintained_tables + 1))
    if [ "$DRY_RUN" = "true" ]; then
      echo "[dry-run] $db: SELECT public.ensure_future_partitions('$tbl'::regclass, $MONTHS_AHEAD);"
      if [ -n "$DETACH_KEEP_MONTHS" ]; then
        echo "[dry-run] $db: SELECT public.detach_old_partitions('$tbl'::regclass, $DETACH_KEEP_MONTHS);"
      fi
      continue
    fi
    psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$db" -v ON_ERROR_STOP=1 -c "
      SELECT public.ensure_future_partitions('$tbl'::regclass, $MONTHS_AHEAD);
      DO \$\$
      DECLARE p text;
      BEGIN
        FOR p IN SELECT format('%I.%I', n.nspname, c.relname)
                 FROM pg_inherits i
                 JOIN pg_class c ON c.oid = i.inhrelid
                 JOIN pg_namespace n ON n.oid = c.relnamespace
                 WHERE i.inhparent = '$tbl'::regclass LOOP
          IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'erp_app_rw') THEN
            EXECUTE format('REVOKE ALL ON %s FROM PUBLIC, erp_app_rw, erp_app_ro', p);
          ELSE
            EXECUTE format('REVOKE ALL ON %s FROM PUBLIC', p);
          END IF;
        END LOOP;
      END \$\$;
    " >/dev/null
    echo "maintained: $db.$tbl (+${MONTHS_AHEAD}m)"
    if [ -n "$DETACH_KEEP_MONTHS" ]; then
      psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$db" -v ON_ERROR_STOP=1 -c "
        DO \$\$
        DECLARE cutoff date := (date_trunc('month', CURRENT_DATE) - ($DETACH_KEEP_MONTHS || ' months')::interval)::date;
        BEGIN
          IF '$tbl' = 'ap.ap_invoice' THEN
            CREATE TABLE IF NOT EXISTS ap.ap_payment_allocation_archive (LIKE ap.ap_payment_allocation);
            WITH moved AS (DELETE FROM ap.ap_payment_allocation WHERE invoice_date < cutoff RETURNING *)
            INSERT INTO ap.ap_payment_allocation_archive SELECT * FROM moved;
            CREATE TABLE IF NOT EXISTS ap.ap_invoice_line_archive (LIKE ap.ap_invoice_line);
            WITH moved AS (DELETE FROM ap.ap_invoice_line WHERE invoice_date < cutoff RETURNING *)
            INSERT INTO ap.ap_invoice_line_archive SELECT * FROM moved;
          ELSIF '$tbl' = 'ar.ar_invoice' THEN
            CREATE TABLE IF NOT EXISTS ar.ar_receipt_allocation_archive (LIKE ar.ar_receipt_allocation);
            WITH moved AS (DELETE FROM ar.ar_receipt_allocation WHERE invoice_date < cutoff RETURNING *)
            INSERT INTO ar.ar_receipt_allocation_archive SELECT * FROM moved;
            CREATE TABLE IF NOT EXISTS ar.ar_invoice_line_archive (LIKE ar.ar_invoice_line);
            WITH moved AS (DELETE FROM ar.ar_invoice_line WHERE invoice_date < cutoff RETURNING *)
            INSERT INTO ar.ar_invoice_line_archive SELECT * FROM moved;
          ELSIF '$tbl' = 'gl.journal_entry' THEN
            ALTER TABLE gl.journal_line DROP CONSTRAINT IF EXISTS fk_journal_line_entry;
            PERFORM public.detach_old_partitions('gl.journal_entry'::regclass, $DETACH_KEEP_MONTHS);
            ALTER TABLE gl.journal_line
              ADD CONSTRAINT fk_journal_line_entry FOREIGN KEY (journal_entry_id, posting_date)
              REFERENCES gl.journal_entry (id, posting_date) ON DELETE CASCADE;
            RETURN;
          END IF;
          PERFORM public.detach_old_partitions('$tbl'::regclass, $DETACH_KEEP_MONTHS);
        END \$\$;
      " >/dev/null
      echo "detached:   $db.$tbl (keep=${DETACH_KEEP_MONTHS}m)"
    fi
  done <<< "$tables"
done

echo "partition-maintenance: done (${maintained_tables} partitioned tables across ${maintained_dbs} databases)"
