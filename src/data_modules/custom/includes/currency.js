// ============================================================
// includes/currency.js
// Cortex V7 Data Modeling Standard: SAP Currency Decimal Shift (TCURX)
// Ref: data_modeling_standards, ADR-014 (Tri-Currency Engine)
// ============================================================

/**
 * Builds a CTE expression from replicated SAP TCURX table to compute
 * the scaling factor per currency key (SAP stores 0-decimal currencies
 * such as IDR/JPY shifted by 2 decimals unless adjusted by TCURX).
 *
 * @param {string} tcurxRef - Dataform resolved table reference for TCURX
 * @returns {string} SQL query for currency_decimal CTE
 */
function currencyDecimalShift(tcurxRef) {
  return `
    SELECT
      TRIM(currency_key) AS currkey,
      CAST(decimal_places AS INT64) AS currdec,
      CAST(POWER(10, 2 - COALESCE(CAST(decimal_places AS INT64), 2)) AS NUMERIC) AS decimal_shift_multiplier
    FROM ${tcurxRef}
  `;
}

/**
 * Applies currency decimal shift multiplier to an SAP raw monetary amount.
 * Falls back to multiplier 1 when currency key is not listed in TCURX (default 2 decimals).
 *
 * @param {string} amountExpr - SQL column expression (e.g., "l.WRBTR")
 * @param {string} cteAlias   - Joined alias of currency_decimal CTE
 * @returns {string} Shifted NUMERIC SQL expression
 */
function amountWithDecimalShift(amountExpr, cteAlias) {
  return `(CAST(${amountExpr} AS NUMERIC) * COALESCE(${cteAlias}.decimal_shift_multiplier, NUMERIC '1'))`;
}

module.exports = {
  currencyDecimalShift,
  amountWithDecimalShift,
};

