// ============================================================
// content/tax-helper.js
// Content script — injected on https://*.pajak.go.id/*
// Provides e-Faktur form auto-fill assistance & field scanning
// Ref: ADR-012 (Enterprise Chrome Extension Companion)
// ============================================================

(function () {
  'use strict';

  const ERP_COMPANION_ID = 'erp-companion-tax-helper';

  // Prevent double-injection
  if (typeof document !== 'undefined' && document.getElementById && document.getElementById(ERP_COMPANION_ID)) return;

  // --- Floating Companion Badge ---
  function createBadge() {
    if (typeof document === 'undefined' || !document.createElement) return;
    const badge = document.createElement('div');
    badge.id = ERP_COMPANION_ID;
    badge.style.cssText = `
      position: fixed; bottom: 20px; right: 20px; z-index: 99999;
      background: #1e40af; color: #fff; border-radius: 12px;
      padding: 10px 16px; font-family: system-ui, sans-serif;
      font-size: 13px; box-shadow: 0 4px 12px rgba(0,0,0,0.25);
      cursor: pointer; display: flex; align-items: center; gap: 8px;
      transition: transform 0.2s; user-select: none;
    `;

    const titleSpan = document.createElement('span');
    titleSpan.textContent = '🏢 ERP Companion';

    const subtitleSpan = document.createElement('span');
    subtitleSpan.style.opacity = '0.75';
    subtitleSpan.textContent = '| e-Faktur Helper';

    badge.append(titleSpan, subtitleSpan);
    badge.addEventListener('mouseenter', () => {
      badge.style.transform = 'scale(1.05)';
    });
    badge.addEventListener('mouseleave', () => {
      badge.style.transform = 'scale(1)';
    });
    badge.addEventListener('click', openHelperPanel);
    document.body.appendChild(badge);
  }

  function openHelperPanel() {
    if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.sendMessage) {
      chrome.runtime.sendMessage({ action: 'OPEN_SIDE_PANEL' }).catch((err) => {
        console.warn('[ERP Companion] Could not open side panel:', err);
      });
    }
  }

  function showNotification(message, type = 'info') {
    if (typeof document === 'undefined' || !document.createElement) return;
    const colors = {
      success: { bg: '#ecfdf5', border: '#059669', text: '#065f46' },
      warning: { bg: '#fffbeb', border: '#d97706', text: '#92400e' },
      error: { bg: '#fef2f2', border: '#dc2626', text: '#991b1b' },
      info: { bg: '#eff6ff', border: '#2563eb', text: '#1e40af' },
    };
    const c = colors[type] || colors.info;

    const toast = document.createElement('div');
    toast.style.cssText = `
      position: fixed; top: 20px; right: 20px; z-index: 999999;
      background: ${c.bg}; color: ${c.text};
      border: 1px solid ${c.border}; border-left: 4px solid ${c.border};
      border-radius: 8px; padding: 12px 16px;
      font-family: system-ui, sans-serif; font-size: 13px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.15);
      max-width: 360px; transition: opacity 0.25s ease;
    `;
    toast.textContent = message;
    document.body.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      setTimeout(() => toast.remove(), 250);
    }, 3500);
  }

  // --- Field Detection & Mapping ---
  function detectEfakturFields() {
    if (typeof document === 'undefined' || !document.querySelectorAll) return {};
    const fields = {};
    const inputs = document.querySelectorAll('input, select, textarea');

    inputs.forEach((el) => {
      if (el.type === 'hidden' || el.disabled) return;
      const hint = `${el.name || ''} ${el.id || ''} ${el.placeholder || ''} ${el.getAttribute('aria-label') || ''}`.toLowerCase();

      if (!fields.npwp && /npwp|tax.?id/.test(hint)) fields.npwp = el;
      if (!fields.nama && /(nama|name|lawan.?transaksi|customer)/.test(hint)) fields.nama = el;
      if (!fields.alamat && /alamat|address/.test(hint)) fields.alamat = el;
      if (!fields.dpp && /(dpp|dasar.?pengenaan)/.test(hint)) fields.dpp = el;
      if (!fields.ppn && /(ppn|vat|tax.?amount)/.test(hint)) fields.ppn = el;
      if (!fields.tanggal && /(tanggal|tgl|date)/.test(hint)) fields.tanggal = el;
      if (!fields.referensi && /(referensi|invoice|nomor.?faktur|no.?dok)/.test(hint)) fields.referensi = el;
    });

    return fields;
  }

  function normalizePayload(payload = {}) {
    return {
      npwp: payload.customerNPWP ?? payload.npwp ?? '',
      nama: payload.customerName ?? payload.nama ?? '',
      alamat: payload.customerAddress ?? payload.alamat ?? '',
      dpp: payload.dppAmount ?? payload.dpp ?? '',
      ppn: payload.ppnAmount ?? payload.ppn ?? '',
      tanggal: payload.taxInvoiceDate ?? payload.tanggal ?? '',
      referensi: payload.invoiceNo ?? payload.referensi ?? '',
    };
  }

  /**
   * Sets value on React / CoreTax controlled HTML input elements
   * using native prototype property descriptor to trigger state changes.
   */
  function setNativeValue(element, value) {
    if (!element) return;
    const stringValue = String(value);

    const prototype = Object.getPrototypeOf(element);
    const prototypeValueSetter = Object.getOwnPropertyDescriptor(prototype, 'value')?.set;
    const ownValueSetter = Object.getOwnPropertyDescriptor(element, 'value')?.set;

    if (prototypeValueSetter && ownValueSetter !== prototypeValueSetter) {
      prototypeValueSetter.call(element, stringValue);
    } else if (ownValueSetter) {
      ownValueSetter.call(element, stringValue);
    } else {
      element.value = stringValue;
    }

    element.dispatchEvent(new Event('input', { bubbles: true }));
    element.dispatchEvent(new Event('change', { bubbles: true }));
  }

  // Batch DOM mutations inside requestAnimationFrame per MV3 performance best practices
  async function autoFillFromERP(rawPayload) {
    const fields = detectEfakturFields();
    const normalized = normalizePayload(rawPayload);
    const entries = Object.entries(normalized).filter(([, value]) => value !== '');
    const filled = [];
    const notFound = [];

    await new Promise((resolve) => {
      const schedule = typeof requestAnimationFrame !== 'undefined' ? requestAnimationFrame : (fn) => setTimeout(fn, 0);
      schedule(() => {
        for (const [key, value] of entries) {
          const target = fields[key];
          if (target) {
            setNativeValue(target, value);
            filled.push(key);
          } else {
            notFound.push(key);
          }
        }
        resolve();
      });
    });

    if (filled.length > 0) {
      showNotification(
        `✅ Auto-fill e-Faktur selesai (${filled.length} kolom diisi${normalized.referensi ? `: ${normalized.referensi}` : ''}).`,
        'success'
      );
    } else {
      showNotification(
        '⚠️ Formulir e-Faktur tidak terdeteksi di halaman ini. Pastikan halaman pembuatan faktur terbuka.',
        'warning'
      );
    }

    return { filled, notFound };
  }

  /**
   * Reverse synchronization: extract generated NTTE, NSFP, or QR code from DJP confirmation page
   */
  function extractTaxConfirmation() {
    if (typeof document === 'undefined') return { ntte: null, nsfp: null, qrCodeUrl: null, timestamp: Date.now() };

    const bodyText = document.body ? document.body.innerText || '' : '';
    const ntteMatch = bodyText.match(/NTTE[:\s]+([A-Z0-9]{16,32})/i);
    const nsfpMatch = bodyText.match(/(?:NSFP|Nomor Seri)[:\s]+(\d{3}[.-]?\d{3}[.-]?\d{8})/i);
    const qrImg = document.querySelector ? document.querySelector('img[src*="qr"], img[alt*="qr"], img[src*="barcode"]') : null;

    return {
      ntte: ntteMatch ? ntteMatch[1] : null,
      nsfp: nsfpMatch ? nsfpMatch[1].replace(/[.-]/g, '') : null,
      qrCodeUrl: qrImg ? qrImg.src : null,
      timestamp: Date.now(),
    };
  }

  // --- Message listener from Side Panel / Service Worker ---
  if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.onMessage) {
    chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
      const msgType = msg?.action || msg?.type;

      if (msgType === 'INJECT_TAX_INVOICE' || msgType === 'AUTOFILL_EFAKTUR') {
        (async () => {
          try {
            const result = await autoFillFromERP(msg.payload);
            sendResponse({
              success: result.filled.length > 0,
              status: result.filled.length > 0 ? 'ok' : 'no_fields_matched',
              filled: result.filled,
              notFound: result.notFound,
            });
          } catch (err) {
            sendResponse({ success: false, error: err.message });
          }
        })();
        return true;
      }

      if (msgType === 'SCAN_FIELDS') {
        const fields = detectEfakturFields();
        sendResponse({
          success: true,
          status: 'ok',
          detected: Object.keys(fields),
        });
        return false;
      }

      if (msgType === 'EXTRACT_TAX_CONFIRMATION') {
        const confirmation = extractTaxConfirmation();
        sendResponse({
          success: true,
          status: 'ok',
          confirmation,
        });
        return false;
      }

      return false;
    });
  }

  // --- Init ---
  if (typeof document !== 'undefined') {
    createBadge();
  }
  if (typeof console !== 'undefined' && console.log) {
    console.log('[ERP Companion] DJP e-Faktur Tax Helper content script loaded.');
  }

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
      detectEfakturFields,
      normalizePayload,
      setNativeValue,
      extractTaxConfirmation,
    };
  }
})();
