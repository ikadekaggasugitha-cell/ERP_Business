// ERP Enterprise Companion — Side Panel Logic (Manifest V3)
// Complies with MV3 strict CSP: no inline scripts, no inline onclick handlers.

const INVOICE_CATALOG = {
  'INV-2026-0089': {
    invoiceNo: 'INV-2026-0089',
    customerNPWP: '01.345.678.9-012.000',
    customerName: 'PT Maju Bersama Nusantara',
    dppAmount: '150000000',
    ppnAmount: '16500000',
    taxInvoiceDate: '2026-11-11',
  },
  'INV-2026-0090': {
    invoiceNo: 'INV-2026-0090',
    customerNPWP: '02.876.543.2-054.000',
    customerName: 'CV Mandiri Jaya Abadi',
    dppAmount: '38636364',
    ppnAmount: '3863636',
    taxInvoiceDate: '2026-11-11',
  },
};

let pendingDocs = [];
let activeRejectId = null;

document.addEventListener('DOMContentLoaded', async () => {
  initTabs();
  initListeners();
  initSandboxBridge();
  await Promise.all([
    loadPendingApprovals(),
    loadLastBankExtraction(),
    loadSettings(),
  ]);
});

function escapeHtml(str) {
  return String(str ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
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
      showToast('Gagal sinkronisasi data.', 'error');
    }
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

  // Inject tax data to active DJP tab
  document.getElementById('injectTaxBtn')?.addEventListener('click', async () => {
    const invoiceNo = document.getElementById('invoiceSelect')?.value || 'INV-2026-0089';
    const payload = INVOICE_CATALOG[invoiceNo] || INVOICE_CATALOG['INV-2026-0089'];

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
        showToast(`Data faktur ${invoiceNo} (${response.filled?.length || 0} kolom) berhasil diinjeksi!`, 'success');
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

  // Save Kong Gateway & Org Scope settings
  document.getElementById('saveSettingsBtn')?.addEventListener('click', async () => {
    const apiUrl = document.getElementById('apiUrlInput')?.value.trim() || 'http://localhost:8000/api/v1';
    const orgId = document.getElementById('orgIdInput')?.value.trim() || '11111111-1111-1111-1111-111111111111';

    const resp = await chrome.runtime.sendMessage({
      action: 'SAVE_SETTINGS',
      payload: { apiUrl, orgId },
    });

    if (resp?.success) {
      showToast('Konfigurasi Gateway & Scope RLS disimpan.', 'success');
    } else {
      showToast('Gagal menyimpan konfigurasi.', 'error');
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

async function loadSettings() {
  const resp = await chrome.runtime.sendMessage({ action: 'GET_SETTINGS' });
  if (resp?.success) {
    const apiInput = document.getElementById('apiUrlInput');
    const orgInput = document.getElementById('orgIdInput');
    if (apiInput) apiInput.value = resp.apiUrl || 'http://localhost:8000/api/v1';
    if (orgInput) orgInput.value = resp.orgId || '11111111-1111-1111-1111-111111111111';
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
