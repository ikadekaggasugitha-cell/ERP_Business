// ============================================================
// sandbox/sandbox.js
// Sandboxed execution context for parsing third-party HTML/documents
// Runs inside an <iframe> with strict CSP — no access to Chrome APIs
// Communication via window.postMessage only
// Ref: ADR-012 (Chrome Extension Companion)
// ============================================================

(function () {
  'use strict';

  /**
   * Parse bank statement HTML table rows into structured JSON.
   * Handles common internet banking table formats.
   *
   * @param {string} html - Raw HTML string containing transaction tables
   * @returns {Array<Object>} Parsed transaction objects
   */
  function parseBankStatementHTML(html) {
    if (typeof DOMParser === 'undefined') return [];
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');
    const transactions = [];

    // Find all table rows across any table on the page
    const tables = doc.querySelectorAll('table');

    for (const table of tables) {
      const rows = table.querySelectorAll('tr');
      if (rows.length < 2) continue; // Skip tables without data rows

      // Detect header row to map column indices
      const headerRow = rows[0];
      const headers = Array.from(headerRow.querySelectorAll('th, td')).map(
        (cell) => cell.textContent.trim().toLowerCase()
      );

      const columnMap = detectColumnMapping(headers);
      if (!columnMap) continue; // Skip tables that don't look like transaction data

      // Parse data rows
      for (let i = 1; i < rows.length; i++) {
        const cells = rows[i].querySelectorAll('td');
        if (cells.length < 3) continue; // Skip rows with too few columns

        const transaction = extractTransaction(cells, columnMap);
        if (transaction) {
          transactions.push(transaction);
        }
      }
    }

    return transactions;
  }

  /**
   * Detect column mapping from header text.
   * Returns an object mapping semantic names to column indices.
   */
  function detectColumnMapping(headers) {
    const map = { date: -1, description: -1, debit: -1, credit: -1, balance: -1 };

    const patterns = {
      date: /tanggal|date|tgl|posting/i,
      description: /keterangan|description|desc|uraian|narasi|remark/i,
      debit: /debit|debet|keluar|withdrawal|mutasi\s*db/i,
      credit: /kredit|credit|masuk|deposit|mutasi\s*cr/i,
      balance: /saldo|balance|sisa/i,
    };

    headers.forEach((header, idx) => {
      for (const [key, regex] of Object.entries(patterns)) {
        if (regex.test(header) && map[key] === -1) {
          map[key] = idx;
        }
      }
    });

    // At minimum we need both date and description to consider it valid
    if (map.date === -1 || map.description === -1) return null;

    return map;
  }

  /**
   * Extract a transaction object from table cells using the column map.
   */
  function extractTransaction(cells, columnMap) {
    const getText = (idx) =>
      idx >= 0 && idx < cells.length ? cells[idx].textContent.trim() : '';

    const rawDate = getText(columnMap.date);
    const description = getText(columnMap.description);

    if (!description) return null;

    return {
      date: normalizeDate(rawDate),
      description: description,
      debit: parseAmount(getText(columnMap.debit)),
      credit: parseAmount(getText(columnMap.credit)),
      balance: parseAmount(getText(columnMap.balance)),
    };
  }

  /**
   * Normalize date string to ISO-like format.
   * Handles DD/MM/YYYY, DD-MM-YYYY, and similar Indonesian banking formats.
   */
  function normalizeDate(raw) {
    if (!raw) return '';

    // DD/MM/YYYY or DD-MM-YYYY
    const ddmmyyyy = raw.match(/(\d{1,2})[/\-.](\d{1,2})[/\-.](\d{4})/);
    if (ddmmyyyy) {
      const [, day, month, year] = ddmmyyyy;
      return `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`;
    }

    // DD MMM YYYY (e.g., 15 Nov 2026)
    const months = {
      jan: '01', feb: '02', mar: '03', apr: '04', mei: '05', may: '05',
      jun: '06', jul: '07', agu: '08', aug: '08', sep: '09', okt: '10',
      oct: '10', nov: '11', des: '12', dec: '12',
    };
    const dmy = raw.match(/(\d{1,2})\s+([a-zA-Z]{3})\s+(\d{4})/);
    if (dmy) {
      const [, day, mon, year] = dmy;
      const mm = months[mon.toLowerCase().substring(0, 3)] || '01';
      return `${year}-${mm}-${day.padStart(2, '0')}`;
    }

    return raw; // Return as-is if no pattern matches
  }

  /**
   * Parse currency amount string to numeric value without precision loss or 100x distortion.
   * Accurately parses both Indonesian format (1.234.567,89 and 1234,56)
   * and standard international format (1,234.56 and 1234.56).
   */
  function parseAmount(raw) {
    if (!raw || raw === '-' || raw === '') return 0;

    let cleaned = String(raw).replace(/[Rp\s]/gi, '').trim();
    if (!cleaned || cleaned === '-') return 0;

    const isNegative = cleaned.startsWith('-') || /^\(.*\)$/.test(cleaned);
    cleaned = cleaned.replace(/[()\-]/g, '').trim();

    const hasComma = cleaned.includes(',');
    const hasDot = cleaned.includes('.');

    if (hasComma && hasDot) {
      const lastComma = cleaned.lastIndexOf(',');
      const lastDot = cleaned.lastIndexOf('.');
      if (lastComma > lastDot) {
        // Indonesian format: 1.234.567,89 -> remove dots, replace comma with dot
        cleaned = cleaned.replace(/\./g, '').replace(',', '.');
      } else {
        // Standard format: 1,234,567.89 -> remove commas
        cleaned = cleaned.replace(/,/g, '');
      }
    } else if (hasComma) {
      // Only comma present: e.g. "1234,56" or "1,234,567"
      const commaCount = (cleaned.match(/,/g) || []).length;
      if (commaCount === 1) {
        const parts = cleaned.split(',');
        if (parts[1].length === 3 && parts[0].length <= 3) {
          cleaned = cleaned.replace(/,/g, '');
        } else {
          // e.g. "1234,56" -> "1234.56"
          cleaned = parts[0] + '.' + parts[1];
        }
      } else {
        cleaned = cleaned.replace(/,/g, '');
      }
    } else if (hasDot) {
      // Only dot present: e.g. "1234.56" or "1.234.567"
      const dotCount = (cleaned.match(/\./g) || []).length;
      if (dotCount === 1) {
        const parts = cleaned.split('.');
        if (parts[1].length === 3 && parts[0].length <= 3) {
          cleaned = cleaned.replace(/\./g, '');
        }
        // else standard decimal like "1234.56", leave as-is
      } else {
        cleaned = cleaned.replace(/\./g, '');
      }
    }

    const value = parseFloat(cleaned);
    if (isNaN(value)) return 0;
    return isNegative ? -value : value;
  }

  // --- Message listener: receive data from parent (side panel) ---
  if (typeof window !== 'undefined' && window.addEventListener) {
    window.addEventListener('message', (event) => {
      if (window.parent !== window && event.source !== window.parent) return;

      const targetOrigin = event.origin && event.origin !== 'null' ? event.origin : '*';
      const { type, payload } = event.data || {};

      if (type === 'PARSE_BANK_STATEMENT') {
        try {
          const transactions = parseBankStatementHTML(payload?.html || '');
          event.source.postMessage(
            {
              type: 'BANK_STATEMENT_PARSED',
              success: true,
              transactions: transactions,
              count: transactions.length,
            },
            targetOrigin
          );
        } catch (err) {
          event.source.postMessage(
            {
              type: 'BANK_STATEMENT_PARSED',
              success: false,
              error: err.message,
              transactions: [],
              count: 0,
            },
            targetOrigin
          );
        }
        return;
      }

      if (type === 'PARSE_GENERIC_TABLE') {
        try {
          const transactions = parseBankStatementHTML(payload?.html || '');
          event.source.postMessage(
            {
              type: 'GENERIC_TABLE_PARSED',
              success: true,
              data: transactions,
              count: transactions.length,
            },
            targetOrigin
          );
        } catch (err) {
          event.source.postMessage(
            {
              type: 'GENERIC_TABLE_PARSED',
              success: false,
              error: err.message,
              data: [],
              count: 0,
            },
            targetOrigin
          );
        }
        return;
      }

      // Ping/healthcheck
      if (type === 'PING') {
        event.source.postMessage({ type: 'PONG', timestamp: Date.now() }, targetOrigin);
      }
    });
  }

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
      parseBankStatementHTML,
      detectColumnMapping,
      normalizeDate,
      parseAmount,
    };
  }

  if (typeof console !== 'undefined' && console.log) {
    console.log('[ERP Companion] Sandbox loaded — isolated parsing context ready.');
  }
})();
