// ============================================================
// includes/sap_helpers.js
// Cortex V7 Data Modeling Standard: SAP Dummy Dates & Audit Watermarks
// Ref: data_modeling_standards, ADR-013 (SAP DDIC Coexistence)
// ============================================================

/**
 * Safely parses SAP DATS fields (YYYYMMDD or DATE) while converting SAP dummy
 * date '00000000' or empty strings into SQL NULL (prevents BigQuery cast errors).
 *
 * @param {string} dateExpr - Column or SQL expression representing an SAP date
 * @returns {string} BigQuery DATE expression
 */
function safeSapDate(dateExpr) {
  return `COALESCE(
    SAFE_CAST(${dateExpr} AS DATE),
    SAFE.PARSE_DATE('%Y%m%d', NULLIF(NULLIF(TRIM(CAST(${dateExpr} AS STRING)), ''), '00000000'))
  )`;
}

/**
 * Generates standardized audit column expression `source_last_updated_at`
 * taking the GREATEST timestamp across joined source recordstamps.
 *
 * @param {string[]} timestampExprs - Array of recordstamp SQL expressions
 * @returns {string} SQL expression resolving to TIMESTAMP
 */
function latestRecordstamp(timestampExprs) {
  const safeTerms = timestampExprs.map(
    (expr) => `IFNULL(CAST(${expr} AS TIMESTAMP), TIMESTAMP('1900-01-01 00:00:00+00'))`
  );
  if (safeTerms.length === 1) {
    return safeTerms[0];
  }
  return `GREATEST(\n    ${safeTerms.join(',\n    ')}\n  )`;
}

module.exports = {
  safeSapDate,
  latestRecordstamp,
};

