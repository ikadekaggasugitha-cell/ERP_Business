// ERP Enterprise Companion — Side Panel Logic (Manifest V3)
// Complies with MV3 strict CSP: no inline scripts, no inline onclick handlers.

let pendingDocs = [];
let activeRejectId = null;
let taxInvoices = [];

if (typeof document !== 'undefined') {
  document.addEventListener('DOMContentLoaded', async () => {
    initTabs();
    initListeners();
    initSandboxBridge();
    await Promise.all([
      loadPendingApprovals(),
      loadTaxInvoices(),
      loadLastBankExtraction(),
      loadSettings(),
      loadConnectionStatus(),
    ]);
  });
}

function escapeHtml(str) {
  return String(str ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function mapTaxInvoiceToAutofillPayload(inv = {}) {
  return {
    invoiceNo: inv.sourceDocumentRef || inv.invoiceNo || inv.efakturNo || inv.id || '',
    customerNPWP: inv.customerNPWP || inv.npwp || '',
    customerName: inv.customerName || inv.counterpartyName || '',
    customerAddress: inv.customerAddress || '',
    dppAmount: inv.baseAmount ?? inv.dppAmount ?? '',
    ppnAmount: inv.taxAmount ?? inv.ppnAmount ?? '',
    taxInvoiceDate: inv.taxInvoiceDate || (typeof inv.createdAt === 'string' ? inv.createdAt.slice(0, 10) : ''),
  };
}

function formatTaxInvoiceOptionLabel(inv = {}) {
  const ref = inv.sourceDocumentRef || inv.efakturNo || inv.id || 'Faktur';
  const dpp = inv.baseAmount ?? inv.dppAmount ?? '0';
  const ppn = inv.taxAmount ?? inv.ppnAmount ?? '0';
  const status = inv.efakturStatus || 'DRAFT';
  return `${ref} — DPP ${dpp} / PPN ${ppn} (${status})`;
}

function initTabs() {
  const tabBtns = document.querySelectorAll('.tab-btn');
  tabBtns.forEach((btn) => {
    btn.addEventListener('click', () => {
      tabBtns.forEach((b) => b.classList.remove('active'));
      document.querySelectorAll('.tab-pane').forEach((p) => p.classList.remove('active'));

      btn.classList.add('active');
      const targetId = `tab-${btn.dataset.tab}`;
      const targetPane = document.getElementById(targetId);
      if (targetPane) targetPane.classList.add('active');
    });
  });
}

function initListeners() {
  // Sync button
  document.getElementById('refreshBtn')?.addEventListener('click', async () => {
    showToast('Menyinkronkan data dengan Kong Gateway...', 'info');
    const resp = await chrome.runtime.sendMessage({ action: 'SYNC_NOW' });
    if (resp?.success) {
      pendingDocs = resp.docs || [];
      renderPendingApprovals();
      updateLastSyncLabel(resp.lastSyncTime);
      showToast('Data berhasil diperbarui.', 'success');
    } else {
      showToast(resp?.error || 'Gagal sinkronisasi data.', 'error');
    }
    await Promise.all([loadTaxInvoices(), loadConnectionStatus()]);
  });

  // Event delegation for approval cards (MV3 CSP-compliant, zero inline onclick)
  document.getElementById('approvalsList')?.addEventListener('click', async (event) => {
    const button = event.target.closest('button[data-action]');
    if (!button) return;

    const { action, docId } = button.dataset;
    if (!docId) return;

    if (action === 'approve') {
      await handleApprove(docId);
    } else if (action === 'open-reject') {
      activeRejectId = activeRejectId === docId ? null : docId;
      renderPendingApprovals();
    } else if (action === 'cancel-reject') {
      activeRejectId = null;
      renderPendingApprovals();
    } else if (action === 'confirm-reject') {
      const reasonInput = document.getElementById(`reject-reason-${docId}`);
      const reason = reasonInput?.value.trim() || 'Data dokumen tidak lengkap / revisi';
      await handleReject(docId, reason);
    }
  });

  // Reload live tax invoices from tax-service
  document.getElementById('reloadTaxInvoicesBtn')?.addEventListener('click', async () => {
    await loadTaxInvoices();
  });

  // Sync e-Faktur number input when selected invoice changes
  document.getElementById('invoiceSelect')?.addEventListener('change', () => {
    syncSelectedTaxInvoiceFields();
  });

  // Inject live tax invoice data to active DJP tab
  document.getElementById('injectTaxBtn')?.addEventListener('click', async () => {
    const selected = getSelectedTaxInvoice();
    if (!selected) {
      showToast('Pilih faktur pajak live dari tax-service terlebih dahulu.', 'error');
      return;
    }

    const payload = mapTaxInvoiceToAutofillPayload(selected);
    const labelRef = payload.invoiceNo || selected.id;

    try {
      const [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
      if (!activeTab?.id) {
        showToast('Tidak ada tab aktif.', 'error');
        return;
      }

      const response = await chrome.tabs.sendMessage(activeTab.id, {
        action: 'INJECT_TAX_INVOICE',
        payload,
      });

      if (response?.success) {
        showToast(`Data faktur ${labelRef} (${response.filled?.length || 0} kolom) berhasil diinjeksi!`, 'success');
      } else {
        showToast('Kolom formulir e-Faktur tidak ditemukan pada halaman ini.', 'error');
      }
    } catch (err) {
      console.warn('[ERP Companion] Tax content script unreachable:', err);
      showToast('Buka halaman resmi DJP Online (pajak.go.id) untuk menginjeksi faktur.', 'error');
    }
  });

  // Scan form fields on active DJP tab
  document.getElementById('scanFieldsBtn')?.addEventListener('click', async () => {
    try {
      const [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
      if (!activeTab?.id) {
        showToast('Tidak ada tab aktif.', 'error');
        return;
      }

      const response = await chrome.tabs.sendMessage(activeTab.id, { action: 'SCAN_FIELDS' });
      const detected = response?.detected || [];
      if (detected.length > 0) {
        showToast(`Terdeteksi ${detected.length} kolom: ${detected.join(', ')}`, 'success');
      } else {
        showToast('Tidak ada kolom e-Faktur yang terdeteksi di tab ini.', 'info');
      }
    } catch {
      showToast('Buka halaman DJP Online (pajak.go.id) untuk memindai formulir.', 'error');
    }
  });

  // Extract NSFP & NTTE confirmation from active DJP Online tab
  document.getElementById('extractConfirmationBtn')?.addEventListener('click', async () => {
    try {
      const [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
      if (!activeTab?.id) {
        showToast('Tidak ada tab aktif.', 'error');
        return;
      }

      const response = await chrome.tabs.sendMessage(activeTab.id, {
        action: 'EXTRACT_TAX_CONFIRMATION',
      });
      const confirmation = response?.confirmation;
      if (confirmation?.nsfp || confirmation?.ntte) {
        const efakturInput = document.getElementById('efakturNoInput');
        const ntteInput = document.getElementById('ntteInput');
        if (efakturInput && confirmation.nsfp) efakturInput.value = confirmation.nsfp;
        if (ntteInput && confirmation.ntte) ntteInput.value = confirmation.ntte;
        showToast('Konfirmasi NSFP/NTTE berhasil diambil dari halaman DJP.', 'success');
      } else {
        showToast('NSFP atau NTTE tidak ditemukan pada halaman tab aktif.', 'info');
      }
    } catch {
      showToast('Buka halaman konfirmasi DJP Online (pajak.go.id) untuk mengambil NSFP/NTTE.', 'error');
    }
  });

  // Report e-Faktur confirmation back to tax-service (REPORT_TAX_INVOICE)
  document.getElementById('reportTaxBtn')?.addEventListener('click', async () => {
    const selected = getSelectedTaxInvoice();
    if (!selected) {
      showToast('Pilih faktur pajak yang akan dilaporkan terlebih dahulu.', 'error');
      return;
    }

    const efakturNo = document.getElementById('efakturNoInput')?.value.trim() || selected.efakturNo || '';
    const ntte = document.getElementById('ntteInput')?.value.trim() || '';

    const resp = await chrome.runtime.sendMessage({
      action: 'REPORT_TAX_INVOICE',
      payload: {
        id: selected.id,
        rowVersion: selected.rowVersion,
        efakturNo,
        ntte,
      },
    });

    if (resp?.success) {
      showToast(resp.message || 'Faktur pajak berhasil dilaporkan ke ERP.', 'success');
      const ntteInput = document.getElementById('ntteInput');
      if (ntteInput) ntteInput.value = '';
      await loadTaxInvoices();
    } else {
      showToast(resp?.error || 'Gagal melaporkan faktur pajak ke tax-service.', 'error');
    }
  });

  // Extract bank statement from active banking portal tab
  document.getElementById('extractBankBtn')?.addEventListener('click', async () => {
    try {
      const [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
      if (!activeTab?.id) {
        showToast('Tidak ada tab aktif.', 'error');
        return;
      }

      const extracted = await chrome.tabs.sendMessage(activeTab.id, { action: 'EXTRACT_BANK_STATEMENT' });
      if (!extracted?.success) {
        showToast('Gagal membaca tabel mutasi bank pada halaman ini.', 'error');
        return;
      }

      if ((extracted.transactions || []).length === 0) {
        showToast('Tabel transaksi tidak ditemukan di halaman bank ini.', 'info');
        return;
      }

      const saved = await chrome.runtime.sendMessage({
        action: 'BANK_STATEMENT_EXTRACTED',
        transactions: extracted.transactions,
        bankName: extracted.bankName,
        accountNumber: extracted.accountNumber,
      });

      if (saved?.success && saved.extraction) {
        renderBankSummary(saved.extraction);
        showToast(`${saved.count} mutasi ${extracted.bankName} siap direkonsiliasi.`, 'success');
      }
    } catch {
      showToast('Buka portal internet banking (BCA/BNI/Mandiri) atau gunakan Sandbox HTML parser.', 'error');
    }
  });

  // Parse pasted HTML table using isolated MV3 sandbox iframe
  document.getElementById('parseSandboxBtn')?.addEventListener('click', () => {
    const html = document.getElementById('rawBankHtmlInput')?.value || '';
    const sandboxFrame = document.getElementById('parserSandbox');
    if (!html.trim()) {
      showToast('Tempel HTML tabel mutasi bank terlebih dahulu.', 'info');
      return;
    }
    if (!sandboxFrame?.contentWindow) {
      showToast('Sandbox parser belum siap.', 'error');
      return;
    }

    sandboxFrame.contentWindow.postMessage(
      { type: 'PARSE_BANK_STATEMENT', payload: { html } },
      '*'
    );
  });

  // Save Kong Gateway, Org Scope, Manual Bearer Token, & OIDC config
  document.getElementById('saveSettingsBtn')?.addEventListener('click', async () => {
    const apiUrl = document.getElementById('apiUrlInput')?.value.trim() || 'http://localhost:8000/api/v1';
    const orgId = document.getElementById('orgIdInput')?.value.trim() || '11111111-1111-1111-1111-111111111111';
    const tokenInput = document.getElementById('bearerTokenInput');
    const token = tokenInput?.value.trim() || '';
    const oidcAuthUrl = document.getElementById('oidcAuthUrlInput')?.value.trim() || '';
    const oidcTokenUrl = document.getElementById('oidcTokenUrlInput')?.value.trim() || '';
    const oidcClientId = document.getElementById('oidcClientIdInput')?.value.trim() || '';

    const resp = await chrome.runtime.sendMessage({
      action: 'SAVE_SETTINGS',
      payload: {
        apiUrl,
        orgId,
        ...(token ? { token } : {}),
        oidcAuthUrl,
        oidcTokenUrl,
        oidcClientId,
      },
    });

    if (resp?.success) {
      if (tokenInput) tokenInput.value = '';
      updateTokenHint(Boolean(resp.hasToken));
      showToast('Konfigurasi Gateway & Token disimpan.', 'success');
      await Promise.all([loadConnectionStatus(), loadTaxInvoices(), loadPendingApprovals()]);
    } else {
      showToast(resp?.error || 'Gagal menyimpan konfigurasi.', 'error');
    }
  });

  // Clear stored Bearer token
  document.getElementById('clearTokenBtn')?.addEventListener('click', async () => {
    const resp = await chrome.runtime.sendMessage({ action: 'CLEAR_TOKEN' });
    const tokenInput = document.getElementById('bearerTokenInput');
    if (tokenInput) tokenInput.value = '';

    if (resp?.success) {
      updateTokenHint(false);
      showToast('Token sesi berhasil dihapus.', 'info');
      await Promise.all([loadConnectionStatus(), loadTaxInvoices(), loadPendingApprovals()]);
    } else {
      showToast('Gagal menghapus token sesi.', 'error');
    }
  });

  // Launch OIDC PKCE login via chrome.identity.launchWebAuthFlow
  document.getElementById('oidcLoginBtn')?.addEventListener('click', async () => {
    const authUrl = document.getElementById('oidcAuthUrlInput')?.value.trim() || '';
    const tokenUrl = document.getElementById('oidcTokenUrlInput')?.value.trim() || '';
    const clientId = document.getElementById('oidcClientIdInput')?.value.trim() || '';

    const resp = await chrome.runtime.sendMessage({
      action: 'START_OIDC_PKCE',
      payload: { authUrl, tokenUrl, clientId },
    });

    if (resp?.success) {
      updateTokenHint(true);
      showToast(resp.message || 'Login SSO OIDC PKCE berhasil.', 'success');
      await Promise.all([loadConnectionStatus(), loadTaxInvoices(), loadPendingApprovals()]);
    } else {
      showToast(resp?.error || 'Login SSO OIDC PKCE gagal.', 'error');
    }
  });
}

function initSandboxBridge() {
  const sandboxFrame = document.getElementById('parserSandbox');
  window.addEventListener('message', async (event) => {
    if (!sandboxFrame || event.source !== sandboxFrame.contentWindow) return;

    const { type, success, transactions = [], error } = event.data || {};
    if (type !== 'BANK_STATEMENT_PARSED') return;

    if (!success) {
      showToast(`Sandbox parser error: ${error}`, 'error');
      return;
    }

    if (transactions.length === 0) {
      showToast('Tidak ada baris transaksi terdeteksi dari HTML.', 'info');
      return;
    }

    const saved = await chrome.runtime.sendMessage({
      action: 'BANK_STATEMENT_EXTRACTED',
      transactions,
      bankName: 'Sandbox HTML Import',
      accountNumber: '',
    });

    if (saved?.success && saved.extraction) {
      renderBankSummary(saved.extraction);
      showToast(`${transactions.length} transaksi diparsing aman di Sandbox.`, 'success');
    }
  });
}

function getSelectedTaxInvoice() {
  const select = document.getElementById('invoiceSelect');
  const selectedId = select?.value || '';
  if (!selectedId) return null;
  return taxInvoices.find((inv) => String(inv.id) === String(selectedId)) || null;
}

function syncSelectedTaxInvoiceFields() {
  const selected = getSelectedTaxInvoice();
  const efakturInput = document.getElementById('efakturNoInput');
  if (efakturInput) {
    efakturInput.value = selected?.efakturNo || '';
  }
}

function setTaxInvoiceBanner(message, variant = '') {
  const banner = document.getElementById('taxInvoiceState');
  if (!banner) return;
  banner.textContent = message;
  banner.className = `tax-state-banner${variant ? ` state-${variant}` : ''}`;
}

async function loadTaxInvoices() {
  const select = document.getElementById('invoiceSelect');
  const injectBtn = document.getElementById('injectTaxBtn');
  const reportBtn = document.getElementById('reportTaxBtn');

  setTaxInvoiceBanner('Memuat faktur pajak live dari tax-service...');

  let resp;
  try {
    resp = await chrome.runtime.sendMessage({ action: 'GET_TAX_INVOICES' });
  } catch (err) {
    resp = { success: false, authenticated: true, error: err?.message || 'Service worker tidak merespons.' };
  }

  taxInvoices = [];

  if (!resp || resp.authenticated === false) {
    setTaxInvoiceBanner(
      resp?.message || 'Belum login. Simpan Bearer Token atau login OIDC PKCE di tab Sesi & Status untuk memuat faktur pajak.',
      'warning'
    );
    if (select) {
      select.innerHTML = '<option value="">— Login diperlukan untuk memuat faktur —</option>';
      select.disabled = true;
    }
    if (injectBtn) injectBtn.disabled = true;
    if (reportBtn) reportBtn.disabled = true;
    syncSelectedTaxInvoiceFields();
    return;
  }

  if (!resp.success) {
    setTaxInvoiceBanner(
      `Gagal memuat faktur pajak: ${resp.error || 'Kesalahan gateway.'}`,
      'error'
    );
    if (select) {
      select.innerHTML = '<option value="">— Gagal memuat data tax-service —</option>';
      select.disabled = true;
    }
    if (injectBtn) injectBtn.disabled = true;
    if (reportBtn) reportBtn.disabled = true;
    syncSelectedTaxInvoiceFields();
    return;
  }

  taxInvoices = Array.isArray(resp.invoices) ? resp.invoices : [];
  if (taxInvoices.length === 0) {
    setTaxInvoiceBanner('Tidak ada faktur pajak ditemukan pada organisasi ini.', 'warning');
    if (select) {
      select.innerHTML = '<option value="">— Tidak ada faktur pajak —</option>';
      select.disabled = true;
    }
    if (injectBtn) injectBtn.disabled = true;
    if (reportBtn) reportBtn.disabled = true;
    syncSelectedTaxInvoiceFields();
    return;
  }

  setTaxInvoiceBanner(`${taxInvoices.length} faktur pajak live dimuat dari tax-service.`, 'success');
  if (select) {
    select.innerHTML = taxInvoices
      .map(
        (inv) =>
          `<option value="${escapeHtml(inv.id)}">${escapeHtml(formatTaxInvoiceOptionLabel(inv))}</option>`
      )
      .join('');
    select.disabled = false;
  }
  if (injectBtn) injectBtn.disabled = false;
  if (reportBtn) reportBtn.disabled = false;
  syncSelectedTaxInvoiceFields();
}

async function loadPendingApprovals() {
  const response = await chrome.runtime.sendMessage({ action: 'GET_PENDING_APPROVALS' });
  if (response?.success) {
    pendingDocs = response.docs || [];
    updateLastSyncLabel(response.lastSyncTime);
  }
  renderPendingApprovals();
}

function renderPendingApprovals() {
  const container = document.getElementById('approvalsList');
  const badge = document.getElementById('approvalBadge');
  if (!container) return;

  if (badge) badge.textContent = String(pendingDocs.length);

  if (pendingDocs.length === 0) {
    container.innerHTML = `
      <div class="card card-empty">
        <p>Tidak ada dokumen yang menunggu persetujuan Anda saat ini.</p>
      </div>
    `;
    return;
  }

  container.innerHTML = pendingDocs
    .map((doc) => {
      const id = escapeHtml(doc.id);
      const isRejecting = activeRejectId === doc.id;
      return `
        <div class="card" id="card-${id}">
          <div class="card-header">
            <span class="card-id">${id}</span>
            <span class="card-type">${escapeHtml(doc.type)}</span>
          </div>
          <div class="card-desc">${escapeHtml(doc.vendor)} · Diajukan oleh: ${escapeHtml(doc.requestedBy)}</div>
          <div class="card-amount">${escapeHtml(doc.amount)}</div>
          <div class="card-actions">
            <button class="btn-approve" type="button" data-action="approve" data-doc-id="${id}">
              Setujui Dokumen
            </button>
            <button class="btn-reject" type="button" data-action="open-reject" data-doc-id="${id}">
              Tolak
            </button>
          </div>
          ${
            isRejecting
              ? `
            <div class="reject-box">
              <input id="reject-reason-${id}" type="text" class="text-input" placeholder="Alasan penolakan (wajib untuk audit)..." value="Data dokumen tidak lengkap / revisi">
              <div class="reject-actions">
                <button class="btn-reject" type="button" data-action="cancel-reject" data-doc-id="${id}">Batal</button>
                <button class="btn-reject-confirm" type="button" data-action="confirm-reject" data-doc-id="${id}">Konfirmasi Tolak</button>
              </div>
            </div>
          `
              : ''
          }
        </div>
      `;
    })
    .join('');
}

async function handleApprove(docId) {
  const card = document.getElementById(`card-${docId}`);
  if (card) {
    card.style.opacity = '0.5';
    card.style.pointerEvents = 'none';
  }

  try {
    const response = await chrome.runtime.sendMessage({
      action: 'APPROVE_DOCUMENT',
      documentId: docId,
      notes: 'Disetujui via ERP Chrome Companion Side Panel',
    });

    if (response?.success) {
      pendingDocs = response.docs || pendingDocs.filter((d) => d.id !== docId);
      renderPendingApprovals();
      showToast(`Dokumen ${docId} berhasil disetujui (Selective EOS).`, 'success');
    } else {
      throw new Error(response?.error || 'Approval gagal');
    }
  } catch (err) {
    showToast(`Gagal menyetujui ${docId}: ${err.message}`, 'error');
    if (card) {
      card.style.opacity = '1';
      card.style.pointerEvents = 'auto';
    }
  }
}

async function handleReject(docId, reason) {
  try {
    const response = await chrome.runtime.sendMessage({
      action: 'REJECT_DOCUMENT',
      documentId: docId,
      notes: reason,
    });

    if (response?.success) {
      activeRejectId = null;
      pendingDocs = response.docs || pendingDocs.filter((d) => d.id !== docId);
      renderPendingApprovals();
      showToast(`Dokumen ${docId} ditolak.`, 'info');
    } else {
      throw new Error(response?.error || 'Penolakan gagal');
    }
  } catch (err) {
    showToast(`Gagal menolak ${docId}: ${err.message}`, 'error');
  }
}

async function loadLastBankExtraction() {
  const resp = await chrome.runtime.sendMessage({ action: 'GET_LAST_BANK_EXTRACTION' });
  if (resp?.success && resp.extraction) {
    renderBankSummary(resp.extraction);
  }
}

function renderBankSummary(extraction) {
  const box = document.getElementById('bankSummaryBox');
  const title = document.getElementById('bankSummaryTitle');
  const count = document.getElementById('bankSummaryCount');
  const meta = document.getElementById('bankSummaryMeta');
  const list = document.getElementById('bankPreviewList');
  if (!box || !list) return;

  const txns = extraction.transactions || [];
  box.hidden = false;
  if (title) title.textContent = extraction.bankName || 'Mutasi Bank';
  if (count) count.textContent = `${txns.length} baris`;
  if (meta) {
    const acct = extraction.accountNumber ? `Rek: ${extraction.accountNumber} · ` : '';
    const time = extraction.extractedAt ? new Date(extraction.extractedAt).toLocaleTimeString('id-ID') : '-';
    meta.textContent = `${acct}Diekstrak pukul ${time}`;
  }

  const previewItems = txns.slice(0, 5);
  list.innerHTML = previewItems
    .map((tx) => {
      const isCredit = Number(tx.credit) > 0;
      const amt = isCredit ? Number(tx.credit) : Number(tx.debit);
      const formatted = new Intl.NumberFormat('id-ID').format(amt || 0);
      return `
        <li class="bank-preview-item">
          <span class="bank-preview-desc">${escapeHtml(tx.date || '-')} · ${escapeHtml(tx.description)}</span>
          <span class="bank-preview-amt ${isCredit ? 'credit' : 'debit'}">${isCredit ? '+' : '-'}${escapeHtml(formatted)}</span>
        </li>
      `;
    })
    .join('');
}

function updateTokenHint(hasToken) {
  const hint = document.getElementById('tokenStatusHint');
  if (!hint) return;
  hint.textContent = hasToken
    ? 'Token Bearer aktif tersimpan di chrome.storage.session.'
    : 'Belum ada token aktif (mode dev-offline untuk antrean, tax-service nonaktif).';
}

async function loadSettings() {
  const resp = await chrome.runtime.sendMessage({ action: 'GET_SETTINGS' });
  if (resp?.success) {
    const apiInput = document.getElementById('apiUrlInput');
    const orgInput = document.getElementById('orgIdInput');
    const oidcAuthInput = document.getElementById('oidcAuthUrlInput');
    const oidcTokenInput = document.getElementById('oidcTokenUrlInput');
    const oidcClientInput = document.getElementById('oidcClientIdInput');
    if (apiInput) apiInput.value = resp.apiUrl || 'http://localhost:8000/api/v1';
    if (orgInput) orgInput.value = resp.orgId || '11111111-1111-1111-1111-111111111111';
    if (oidcAuthInput) oidcAuthInput.value = resp.oidcAuthUrl || '';
    if (oidcTokenInput) oidcTokenInput.value = resp.oidcTokenUrl || '';
    if (oidcClientInput) oidcClientInput.value = resp.oidcClientId || '';
    updateTokenHint(Boolean(resp.hasToken));
  }
}

async function loadConnectionStatus() {
  const gwLabel = document.getElementById('gatewayStatusLabel');
  const authLabel = document.getElementById('authSessionStatusLabel');
  const resp = await chrome.runtime.sendMessage({ action: 'CHECK_GATEWAY_STATUS' });
  if (!resp?.success) return;

  if (gwLabel) {
    if (resp.gatewayReachable) {
      gwLabel.textContent = `Terhubung (${resp.gatewayService || 'Gateway'})`;
      gwLabel.className = 'status-value green';
    } else {
      gwLabel.textContent = `Tidak Terjangkau (${resp.gatewayError || 'Offline'})`;
      gwLabel.className = 'status-value red';
    }
  }

  if (authLabel) {
    if (resp.hasToken) {
      authLabel.textContent = 'Aktif (Bearer / OIDC)';
      authLabel.className = 'status-value green';
    } else {
      authLabel.textContent = 'Belum Login (Dev Offline)';
      authLabel.className = 'status-value amber';
    }
  }
}

function updateLastSyncLabel(timestamp) {
  const label = document.getElementById('lastSyncLabel');
  if (label) {
    const d = timestamp ? new Date(timestamp) : new Date();
    label.textContent = `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`;
  }
}

function showToast(message, type = 'info') {
  const container = document.getElementById('toastContainer');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = `toast ${type === 'success' ? 'success' : type === 'error' ? 'error' : ''}`;
  toast.textContent = message;

  container.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    mapTaxInvoiceToAutofillPayload,
    formatTaxInvoiceOptionLabel,
  };
}
