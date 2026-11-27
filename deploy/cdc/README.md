# Debezium CDC & BigQuery Sink (`deploy/cdc/`)

Templates for streaming PostgreSQL logical replication (`pgoutput`, `wal_level=logical`) from the ERP PostgreSQL databases into Kafka and sinking into BigQuery `erp_staging.stg_*_cdc` tables consumed by Cortex Dataform (`src/data_modules/custom/`).

## Files

- [`debezium-postgres-connectors.json`](./debezium-postgres-connectors.json) — Debezium PostgreSQL source connector definitions (`pgoutput`, `decimal.handling.mode: "string"` for fixed-point monetary precision, optional `EventRouter` SMT for transactional outbox).
- [`bigquery-sink-erp-staging.json`](./bigquery-sink-erp-staging.json) — Kafka Connect BigQuery Sink mapping CDC topics 1:1 to all 12 `erp_staging` declarations in `src/data_modules/custom/definitions/declarations/erp_staging_sources.js`.

## Topic → Dataform Staging Mapping

| Kafka CDC Topic | BigQuery `erp_staging` Table | Consuming Dataform Model |
|---|---|---|
| `erp.cdc.gl.gl.journal_line` | `stg_gl_journal_cdc` | `ann_gl_document` → `fact_gl_transaction` |
| `erp.cdc.sales.so.sales_order_line` | `stg_sales_order_cdc` | `ann_sales_order` → `fact_sales_order` |
| `erp.cdc.inventory.inv.stock_balance` | `stg_inventory_snapshot` | `fact_inventory_snapshot` |
| `erp.cdc.org.org.organization` | `stg_organizations_cdc` | `dim_organization`, `dim_organization_closure` |
| `erp.cdc.currency.cur.exchange_rate` | `stg_exchange_rates_cdc` | `dim_currency` |
| `erp.cdc.coa.fin.account` | `stg_chart_of_accounts_cdc` | `dim_chart_of_accounts` |
| `erp.cdc.masterdata.md.item` | `stg_items_cdc` | `ann_material` |
| `erp.cdc.coa.fin.holding_coa_mapping` | `stg_holding_coa_mapping` | `dim_chart_of_accounts` |
| `erp.cdc.fixedasset.fa.depreciation_schedule` | `stg_fixed_assets_cdc` | `fact_fixed_asset_depreciation` |
| `erp.cdc.budget.bud.budget_line` | `stg_budgets_cdc` | `fact_budget_vs_actual` |
| `erp.cdc.project.prj.project` | `stg_projects_cdc` | `fact_project_profitability` |
| `erp.cdc.crm.crm.opportunity` | `stg_crm_opportunities_cdc` | `fact_crm_pipeline` |

## Registering Connectors (Kafka Connect REST API)

```bash
# Register each PostgreSQL Debezium source connector
jq -c '.connectors[]' deploy/cdc/debezium-postgres-connectors.json | while read -r c; do
  name=$(printf '%s' "$c" | jq -r '.name')
  curl -fsS -X PUT "http://kafka-connect:8083/connectors/${name}/config" \
    -H "Content-Type: application/json" \
    -d "$(printf '%s' "$c" | jq -c '.config')"
done

# Register BigQuery sink
curl -fsS -X PUT "http://kafka-connect:8083/connectors/erp-bq-sink-erp-staging/config" \
  -H "Content-Type: application/json" \
  -d "$(jq -c '.config' deploy/cdc/bigquery-sink-erp-staging.json)"
```

## Offline Verification

```bash
python3 -m json.tool deploy/cdc/debezium-postgres-connectors.json >/dev/null
python3 -m json.tool deploy/cdc/bigquery-sink-erp-staging.json >/dev/null
(cd src/data_modules/custom && ./node_modules/.bin/dataform compile)
```
