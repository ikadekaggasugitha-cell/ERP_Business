// ============================================================
// content/bank-extractor.js
// Content script — injected on corporate internet banking portals
// Treasury Bank Statement Extractor for cash/bank reconciliation
// Ref: ADR-012 (Chrome Extension Companion)
// ============================================================

(function () {
  'use strict';

  const ERP_COMPANION_ID = 'erp-companion-bank-extractor';

  // Prevent double-injection
  if (typeof document !== 'undefined' && document.getElementById && document.getElementById(ERP_COMPANION_ID)) return;

  // --- Bank Detection ---
  /**
   * Detect which bank portal we're on based on hostname.
   * Returns a normalized bank name or null if unknown.
   */
  function detectBank() {
    const host = location.hostname.toLowerCase();
    if (host.includes('klikbca'))       return 'BCA';
    if (host.includes('bni'))           return 'BNI';
    if (host.includes('bankmandiri'))   return 'Bank Mandiri';
    return 'Unknown Bank';
  }

  // --- Table Detection & Extraction ---

  /**
   * Column header pattern matchers for Indonesian banking portals.
   */
  const COLUMN_PATTERNS = {
    date:        /tanggal|date|tgl|posting|value\s*date/i,
    description: /keterangan|description|desc|uraian|narasi|remark|berita/i,
    debit:       /debit|debet|keluar|withdrawal|mutasi\s*db|pengeluaran/i,
    credit:      /kredit|credit|masuk|deposit|mutasi\s*cr|penerimaan/i,
    balance:     /saldo|balance|sisa/i,
  };

  /**
   * Detect the column mapping from a header row.
   * @param {NodeList|Array} headerCells - TH or TD elements from the header row
   * @returns {Object|null} Map of semantic names to column indices
   */
  function detectColumns(headerCells) {
    const headers = Array.from(headerCells).map((cell) =>
      cell.textContent.trim().toLowerCase()
    );

    const map = {};
    let matchCount = 0;

    headers.forEach((text, idx) => {
      for (const [key, pattern] of Object.entries(COLUMN_PATTERNS)) {
        if (pattern.test(text) && !(key in map)) {
          map[key] = idx;
          matchCount++;
        }
      }
    });

    // Require at least 2 matching columns to be considered a transaction table
    return matchCount >= 2 ? map : null;
  }

  /**
   * Parse an Indonesian currency amount string to a number.
   * Handles: 1.234.567,89 / 1,234,567.89 / 1234,56 / 1234.56 / Rp prefixes
   */
  function parseAmount(raw) {
    if (!raw || String(raw).trim() === '' || String(raw).trim() === '-') return 0;

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
        cleaned = cleaned.replace(/\./g, '').replace(',', '.');
      } else {
        cleaned = cleaned.replace(/,/g, '');
      }
    } else if (hasComma) {
      const commaCount = (cleaned.match(/,/g) || []).length;
      if (commaCount === 1) {
        const parts = cleaned.split(',');
        if (parts[1].length === 3 && parts[0].length <= 3) {
          cleaned = cleaned.replace(/,/g, '');
        } else {
          cleaned = parts[0] + '.' + parts[1];
        }
      } else {
        cleaned = cleaned.replace(/,/g, '');
      }
    } else if (hasDot) {
      const dotCount = (cleaned.match(/\./g) || []).length;
      if (dotCount === 1) {
        const parts = cleaned.split('.');
        if (parts[1].length === 3 && parts[0].length <= 3) {
          cleaned = cleaned.replace(/\./g, '');
        }
      } else {
        cleaned = cleaned.replace(/\./g, '');
      }
    }

    const value = parseFloat(cleaned);
    if (isNaN(value)) return 0;
    return isNegative ? -value : value;
  }

  /**
   * Normalize date string to YYYY-MM-DD.
   */
  function normalizeDate(raw) {
    if (!raw) return '';

    const ddmmyyyy = raw.match(/(\d{1,2})[/\-.](\d{1,2})[/\-.](\d{4})/);
    if (ddmmyyyy) {
      const [, d, m, y] = ddmmyyyy;
      return `${y}-${m.padStart(2, '0')}-${d.padStart(2, '0')}`;
    }

    const months = {
      jan: '01', feb: '02', mar: '03', apr: '04', mei: '05', may: '05',
      jun: '06', jul: '07', agu: '08', aug: '08', sep: '09', okt: '10',
      oct: '10', nov: '11', des: '12', dec: '12',
    };
    const dmy = raw.match(/(\d{1,2})\s+([a-zA-Z]{3})\s+(\d{4})/);
    if (dmy) {
      const [, d, mon, y] = dmy;
      const mm = months[mon.toLowerCase().substring(0, 3)] || '01';
      return `${y}-${mm}-${d.padStart(2, '0')}`;
    }

    return raw;
  }

  /**
   * Scan the page for transaction tables and extract rows.
   * @returns {Array<Object>} Extracted transactions
   */
  function extractTransactions() {
    const transactions = [];
    const tables = document.querySelectorAll('table');

    for (const table of tables) {
      const rows = table.querySelectorAll('tr');
      if (rows.length < 2) continue;

      // Try first row as header
      const firstRowCells = rows[0].querySelectorAll('th, td');
      const columnMap = detectColumns(firstRowCells);
      if (!columnMap) continue;

      for (let i = 1; i < rows.length; i++) {
        const cells = rows[i].querySelectorAll('td');
        if (cells.length < 3) continue;

        const getText = (key) =>
          key in columnMap && columnMap[key] < cells.length
            ? cells[columnMap[key]].textContent.trim()
            : '';

        const description = getText('description');
        if (!description) continue;

        transactions.push({
          date: normalizeDate(getText('date')),
          description: description,
          debit: parseAmount(getText('debit')),
          credit: parseAmount(getText('credit')),
          balance: parseAmount(getText('balance')),
        });
      }
    }

    return transactions;
  }

  /**
   * Attempt to detect account number from page text.
   * Looks for common patterns in Indonesian banking portals.
   */
  function detectAccountNumber() {
    const bodyText = document.body.innerText || '';

    // Pattern: "No. Rekening: 1234567890" or "Account: 1234567890"
    const patterns = [
      /(?:no\.?\s*rekening|nomor\s*rekening|account\s*(?:no|number)?)\s*[:\-]?\s*(\d[\d\s\-.]{6,})/i,
      /(?:rek|acct)\s*[:\-]?\s*(\d[\d\s\-.]{6,})/i,
    ];

    for (const pattern of patterns) {
      const match = bodyText.match(pattern);
      if (match) {
        return match[1].replace(/[\s\-.]/g, '');
      }
    }

    return '';
  }

  // --- Floating Action Button ---

  function createFAB() {
    const fab = document.createElement('div');
    fab.id = ERP_COMPANION_ID;
    fab.style.cssText = `
      position: fixed; bottom: 20px; right: 20px; z-index: 99999;
      background: linear-gradient(135deg, #059669, #10b981);
      color: #fff; border-radius: 12px;
      padding: 10px 16px; font-family: system-ui, sans-serif;
      font-size: 13px; box-shadow: 0 4px 12px rgba(0,0,0,0.25);
      cursor: pointer; display: flex; align-items: center; gap: 8px;
      transition: transform 0.2s, box-shadow 0.2s;
      user-select: none;
    `;

    const bankName = detectBank();

    const iconSpan = document.createElement('span');
    iconSpan.style.fontSize = '16px';
    iconSpan.textContent = '🏦';

    const titleSpan = document.createElement('span');
    titleSpan.textContent = 'ERP Treasury';

    const bankSpan = document.createElement('span');
    bankSpan.style.opacity = '0.75';
    bankSpan.style.fontSize = '11px';
    bankSpan.textContent = `| ${bankName}`;

    fab.replaceChildren(iconSpan, titleSpan, bankSpan);

    // Hover effects
    fab.addEventListener('mouseenter', () => {
      fab.style.transform = 'scale(1.05)';
      fab.style.boxShadow = '0 6px 20px rgba(5,150,105,0.4)';
    });
    fab.addEventListener('mouseleave', () => {
      fab.style.transform = 'scale(1)';
      fab.style.boxShadow = '0 4px 12px rgba(0,0,0,0.25)';
    });

    // Click handler: extract + send
    fab.addEventListener('click', handleExtraction);

    document.body.appendChild(fab);
    return fab;
  }

  /**
   * Extraction flow triggered by FAB click.
   */
  async function handleExtraction() {
    const fab = document.getElementById(ERP_COMPANION_ID);
    const originalChildren = Array.from(fab.childNodes).map((node) => node.cloneNode(true));

    // Visual feedback: extracting
    const busyIcon = document.createElement('span');
    busyIcon.style.fontSize = '16px';
    busyIcon.textContent = '⏳';
    const busyText = document.createElement('span');
    busyText.textContent = 'Mengekstrak...';
    fab.replaceChildren(busyIcon, busyText);
    fab.style.pointerEvents = 'none';

    try {
      const transactions = extractTransactions();
      const bankName = detectBank();
      const accountNumber = detectAccountNumber();

      if (transactions.length === 0) {
        showNotification('⚠️ Tidak ditemukan tabel transaksi pada halaman ini.', 'warning');
        return;
      }

      // Send extracted data to service worker
      const response = await chrome.runtime.sendMessage({
        action: 'BANK_STATEMENT_EXTRACTED',
        transactions,
        bankName,
        accountNumber,
      });

      if (response?.success) {
        showNotification(
          `✅ ${transactions.length} transaksi ${bankName} berhasil diekstrak.`,
          'success'
        );
      } else {
        showNotification('❌ Gagal menyimpan data ekstraksi.', 'error');
      }

      // Also request side panel to open
      chrome.runtime.sendMessage({ action: 'OPEN_SIDE_PANEL' }).catch(() => {});
    } catch (err) {
      console.error('[ERP Companion] Bank extraction error:', err);
      showNotification('❌ Terjadi kesalahan saat mengekstrak data.', 'error');
    } finally {
      // Restore FAB
      fab.replaceChildren(...originalChildren);
      fab.style.pointerEvents = '';
    }
  }

  // --- Notification Toast ---

  function showNotification(message, type = 'info') {
    const toast = document.createElement('div');
    const colors = {
      success: { bg: '#ecfdf5', border: '#059669', text: '#065f46' },
      warning: { bg: '#fffbeb', border: '#d97706', text: '#92400e' },
      error:   { bg: '#fef2f2', border: '#dc2626', text: '#991b1b' },
      info:    { bg: '#eff6ff', border: '#2563eb', text: '#1e40af' },
    };
    const c = colors[type] || colors.info;

    toast.style.cssText = `
      position: fixed; top: 20px; right: 20px; z-index: 999999;
      background: ${c.bg}; color: ${c.text};
      border: 1px solid ${c.border}; border-left: 4px solid ${c.border};
      border-radius: 8px; padding: 12px 16px;
      font-family: system-ui, sans-serif; font-size: 13px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.15);
      max-width: 350px; animation: erpFadeIn 0.3s ease;
    `;
    toast.textContent = message;

    // Inject animation keyframes if not already present
    if (!document.getElementById('erp-toast-style')) {
      const style = document.createElement('style');
      style.id = 'erp-toast-style';
      style.textContent = `
        @keyframes erpFadeIn { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }
        @keyframes erpFadeOut { from { opacity: 1; } to { opacity: 0; transform: translateY(-10px); } }
      `;
      document.head.appendChild(style);
    }

    document.body.appendChild(toast);

    // Auto-dismiss after 4 seconds
    setTimeout(() => {
      toast.style.animation = 'erpFadeOut 0.3s ease forwards';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
  }

  // --- Message listener from service worker / side panel ---
  if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.onMessage) {
    chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
      const msgType = msg?.action || msg?.type;

      if (msgType === 'EXTRACT_BANK_STATEMENT') {
        const transactions = extractTransactions();
        const bankName = detectBank();
        const accountNumber = detectAccountNumber();
        sendResponse({
          success: true,
          transactions,
          bankName,
          accountNumber,
          count: transactions.length,
        });
        return false;
      }

      if (msgType === 'PING_EXTRACTOR') {
        sendResponse({ active: true, bank: detectBank(), url: location.href });
        return false;
      }

      return false;
    });
  }

  // --- Init ---
  if (typeof document !== 'undefined') {
    createFAB();
  }
  if (typeof console !== 'undefined' && console.log && typeof location !== 'undefined') {
    console.log(
      `[ERP Companion] Bank Statement Extractor loaded — ${detectBank()} (${location.hostname})`
    );
  }

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
      parseAmount,
      normalizeDate,
    };
  }
})();
