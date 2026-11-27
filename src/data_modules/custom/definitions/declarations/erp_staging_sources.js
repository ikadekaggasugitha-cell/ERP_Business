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
  "stg_crm_opportunities_cdc",
  "stg_ap_vendor_invoice_cdc",
  "stg_ar_customer_invoice_cdc",
  "stg_tax_invoice_cdc",
  "stg_treasury_payment_cdc",
  "stg_procurement_po_cdc",
  "stg_manufacturing_wo_cdc",
  "stg_payroll_run_cdc"
];

stagingTables.forEach((tableName) => {
  declare({
    database: dataform.projectConfig.defaultDatabase,
    schema: "erp_staging",
    name: tableName,
    description: `ERP Staging CDC table: ${tableName}`
  });
});
