// ============================================================
// definitions/declarations/erp_staging_sources.js
// Source declarations for ERP Staging & CDC tables
// ============================================================

const stagingTables = [
  "stg_gl_journal_cdc",
  "stg_sales_order_cdc",
  "stg_inventory_snapshot",
  "stg_organizations_cdc",
  "stg_exchange_rates_cdc",
  "stg_chart_of_accounts_cdc",
  "stg_items_cdc",
  "stg_holding_coa_mapping",
  "stg_fixed_assets_cdc",
  "stg_budgets_cdc",
  "stg_projects_cdc",
  "stg_crm_opportunities_cdc"
];

stagingTables.forEach((tableName) => {
  declare({
    database: dataform.projectConfig.defaultDatabase,
    schema: "erp_staging",
    name: tableName,
    description: `ERP Staging CDC table: ${tableName}`
  });
});
