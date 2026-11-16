// ============================================================
// definitions/declarations/sap_ddic_sources.js
// Source declarations for replicated SAP DDIC tables
// ============================================================

const sapTables = [
  "bkpf", "bseg", "acdoca", "mara", "makt", "marc", "mard", "mbew",
  "matdoc", "vbak", "vbap", "ekko", "ekpo", "t001", "t009b", "tcurr",
  "tcurx", "tvko", "ska1", "skat", "dd02l", "dd03l", "dd08l"
];

sapTables.forEach((tableName) => {
  declare({
    database: dataform.projectConfig.defaultDatabase,
    schema: "sap_ddic_replica",
    name: tableName,
    description: `Raw replicated SAP DDIC table: ${tableName.toUpperCase()}`
  });
});
