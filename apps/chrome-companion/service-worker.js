// ERP Enterprise Companion — Background Service Worker (Manifest V3)
// Adheres strictly to ephemeral lifecycle: state in chrome.storage.session/local, alarms for scheduling
// Ref: ADR-012 (Enterprise Chrome Extension Companion)

const DEFAULT_API_URL = 'http://localhost:8000/api/v1';
const DEFAULT_ORG_ID = '11111111-1111-1111-1111-111111111111';

const DEFAULT_DEV_APPROVALS = [
  {
    id: 'PO-2026-004',
    type: 'Purchase Order',
    vendor: 'PT Steel Pipe Industry of Indonesia',
    amount: 'Rp 450.000.000',
    requestedBy: 'Budi Santoso (Procurement Lead)',
  },
  {
    id: 'JV-2026-012',
    type: 'Journal Entry',
    vendor: 'Revaluasi Selisih Kurs Valas (USD)',
    amount: 'USD 25,000.00 (Rp 406.250.000)',
    requestedBy: 'Siti Rahma (GL Accountant)',
  },
  {
    id: 'PAY-2026-003',
    type: 'Payment Voucher',
    vendor: 'Pelunasan Sewa Gedung Kantor Cabang',
    amount: 'Rp 85.000.000',
    requestedBy: 'Ahmad Fauzi (Treasury)',
  },
];

async function getExtensionConfig() {
  const localData =
    (chrome.storage?.local?.get &&
      (await chrome.storage.local.get(['token', 'apiUrl', 'orgId', 'userId']))) ||
    {};
  let sessionToken = '';
  if (chrome.storage?.session?.get) {
    const sessionData = await chrome.storage.session.get(['token']);
    sessionToken = sessionData?.token || '';
  }

  const token = String(sessionToken || localData?.token || '').trim();
  const apiUrl =
    String(localData?.apiUrl || DEFAULT_API_URL)
      .trim()
      .replace(/\/+$/, '') || DEFAULT_API_URL;
  const orgId = String(localData?.orgId || DEFAULT_ORG_ID).trim() || DEFAULT_ORG_ID;
  const userId = String(localData?.userId || '').trim();

  return { token, apiUrl, orgId, userId };
}

async function saveStoredToken(token) {
  const cleanToken = String(token || '').trim();
  if (!cleanToken) return;
  if (chrome.storage?.session?.set) {
    await chrome.storage.session.set({ token: cleanToken });
    if (chrome.storage?.local?.remove) {
      await chrome.storage.local.remove('token');
    }
  } else if (chrome.storage?.local?.set) {
    await chrome.storage.local.set({ token: cleanToken });
  }
}

async function clearStoredToken() {
  if (chrome.storage?.session?.remove) {
    await chrome.storage.session.remove('token');
  }
  if (chrome.storage?.local?.remove) {
    await chrome.storage.local.remove('token');
  }
}

function buildGatewayHeaders({
  token,
  orgId,
  userId,
  contentType = false,
  idempotencyKey = '',
  ifMatch = undefined,
}) {
  const headers = {
    'X-Correlation-Id': crypto.randomUUID(),
    'X-Organization-Id': orgId || DEFAULT_ORG_ID,
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (userId) {
    headers['X-User-Id'] = userId;
  }
  if (contentType) {
    headers['Content-Type'] = 'application/json';
  }
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey;
  }
  if (ifMatch !== undefined && ifMatch !== null && String(ifMatch).trim() !== '') {
    headers['If-Match'] = String(ifMatch).trim();
  }
  return headers;
}

async function parseGatewayError(response, fallbackMsg) {
  try {
    const errJson = await response.json();
    const msg = errJson?.error?.message || errJson?.message;
    if (msg) return `${msg} (HTTP ${response.status})`;
  } catch {
    // Ignore non-JSON error bodies
  }
  return fallbackMsg || `Gateway mengembalikan HTTP ${response.status}`;
}

function buildHealthUrl(apiUrl) {
  try {
    const parsed = new URL(apiUrl || DEFAULT_API_URL);
    const basePath = parsed.pathname
      .replace(/\/api\/v\d+\/?$/i, '')
      .replace(/\/v\d+\/?$/i, '')
      .replace(/\/+$/, '');
    return `${parsed.origin}${basePath}/healthz`;
  } catch {
    return 'http://localhost:8000/healthz';
  }
}

function getExtensionVersion() {
  try {
    return chrome.runtime?.getManifest?.()?.version || '1.0.0';
  } catch {
    return '1.0.0';
  }
}

// 1. Enable Side Panel opening on action icon click
if (typeof chrome !== 'undefined' && chrome.sidePanel) {
  chrome.sidePanel
    .setPanelBehavior({ openPanelOnActionClick: true })
    .catch((error) => console.error('[ERP Companion] Failed to set panel behavior:', error));
}

// 2. Setup periodic background alarm on install/update
if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.onInstalled) {
  chrome.runtime.onInstalled.addListener(async (details) => {
    console.log('[ERP Companion] Lifecycle event:', details.reason);

    await chrome.storage.session.set({
      lastSyncTime: Date.now(),
      pendingCount: DEFAULT_DEV_APPROVALS.length,
      pendingDocs: DEFAULT_DEV_APPROVALS,
      approvalMode: 'dev-offline',
      lastSyncError: null,
    });

    // Schedule alarm every 5 minutes (replaces setInterval per MV3 lifecycle rules)
    await chrome.alarms.create('syncApprovalsAlarm', {
      periodInMinutes: 5,
    });

    await syncPendingApprovals();
  });
}

// 3. Setup browser startup listener for badge recovery
if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.onStartup) {
  chrome.runtime.onStartup.addListener(async () => {
    console.log('[ERP Companion] Browser startup: restoring badge and state');
    const { pendingCount = 0 } = await chrome.storage.session.get('pendingCount');
    await updateBadgeCount(pendingCount);
    await syncPendingApprovals();
  });
}

// 4. Listen to periodic alarms
if (typeof chrome !== 'undefined' && chrome.alarms && chrome.alarms.onAlarm) {
  chrome.alarms.onAlarm.addListener(async (alarm) => {
    if (alarm.name === 'syncApprovalsAlarm') {
      await syncPendingApprovals();
    }
  });
}

// 5. Update Extension Badge helper
async function updateBadgeCount(count) {
  if (typeof chrome === 'undefined' || !chrome.action) return;
  if (count > 0) {
    await chrome.action.setBadgeText({ text: String(count) });
    await chrome.action.setBadgeBackgroundColor({ color: '#2563eb' });
  } else {
    await chrome.action.setBadgeText({ text: '' });
  }
}

// 6. Synchronize pending approvals from ERP API Gateway (Kong)
async function syncPendingApprovals() {
  try {
    const { token, apiUrl, orgId, userId } = await getExtensionConfig();
    const sessionState = (await chrome.storage.session.get(['pendingDocs', 'approvalMode'])) || {};

    if (token) {
      const headers = buildGatewayHeaders({ token, orgId, userId });
      const response = await fetch(`${apiUrl}/workflow-tasks?status=PENDING`, {
        headers,
      });

      if (!response.ok) {
        const errMsg = await parseGatewayError(response);
        const existingLiveDocs =
          sessionState.approvalMode === 'live' && Array.isArray(sessionState.pendingDocs)
            ? sessionState.pendingDocs
            : [];
        await chrome.storage.session.set({
          pendingDocs: existingLiveDocs,
          pendingCount: existingLiveDocs.length,
          lastSyncTime: Date.now(),
          approvalMode: 'live-error',
          lastSyncError: errMsg,
        });
        await updateBadgeCount(existingLiveDocs.length);
        return {
          docs: existingLiveDocs,
          count: existingLiveDocs.length,
          mode: 'live-error',
          error: errMsg,
        };
      }

      const json = await response.json();
      let docs = [];
      if (Array.isArray(json)) {
        docs = json;
      } else if (Array.isArray(json?.data)) {
        docs = json.data;
      }

      const count = docs.length;
      await chrome.storage.session.set({
        pendingDocs: docs,
        pendingCount: count,
        lastSyncTime: Date.now(),
        approvalMode: 'live',
        lastSyncError: null,
      });
      await updateBadgeCount(count);
      return { docs, count, mode: 'live', error: null };
    }

    // Explicit dev offline placeholders when unauthenticated
    const docs =
      sessionState.approvalMode === 'dev-offline' && Array.isArray(sessionState.pendingDocs)
        ? sessionState.pendingDocs
        : DEFAULT_DEV_APPROVALS;
    const count = docs.length;

    await chrome.storage.session.set({
      pendingDocs: docs,
      pendingCount: count,
      lastSyncTime: Date.now(),
      approvalMode: 'dev-offline',
      lastSyncError: null,
    });

    await updateBadgeCount(count);
    return { docs, count, mode: 'dev-offline', error: null };
  } catch (err) {
    console.error('[ERP Companion] Failed to sync pending approvals:', err);
    const { token } = await getExtensionConfig();
    if (token) {
      await chrome.storage.session.set({
        pendingDocs: [],
        pendingCount: 0,
        lastSyncTime: Date.now(),
        approvalMode: 'live-error',
        lastSyncError: err.message,
      });
      await updateBadgeCount(0);
      return { docs: [], count: 0, mode: 'live-error', error: err.message };
    }

    const { pendingDocs = DEFAULT_DEV_APPROVALS } = await chrome.storage.session.get('pendingDocs');
    await updateBadgeCount(pendingDocs.length);
    return { docs: pendingDocs, count: pendingDocs.length, mode: 'dev-offline', error: null };
  }
}

// Helper to submit approval or rejection decision with audit metadata & optimistic lock If-Match
async function submitWorkflowDecision(documentId, decision, notes, rowVersion) {
  const { token, apiUrl, orgId, userId } = await getExtensionConfig();
  const sessionState = (await chrome.storage.session.get('pendingDocs')) || {};
  const pendingDocs = Array.isArray(sessionState.pendingDocs)
    ? sessionState.pendingDocs
    : DEFAULT_DEV_APPROVALS;
  const matchedDoc = pendingDocs.find((doc) => String(doc.id) === String(documentId));
  const resolvedRowVersion =
    rowVersion !== undefined && rowVersion !== null && String(rowVersion).trim() !== ''
      ? rowVersion
      : matchedDoc?.rowVersion;

  if (token) {
    const headers = buildGatewayHeaders({
      token,
      orgId,
      userId,
      contentType: true,
      idempotencyKey: `${decision}-${documentId}-${Date.now()}`,
      ifMatch: resolvedRowVersion,
    });

    const action = decision.toLowerCase() === 'reject' ? 'reject' : 'approve';
    const response = await fetch(`${apiUrl}/workflow-tasks/${encodeURIComponent(documentId)}/${action}`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        notes,
        origin: 'chrome-extension',
        extension_version: getExtensionVersion(),
      }),
    });

    if (!response.ok) {
      throw new Error(await parseGatewayError(response));
    }
  }

  const nextDocs = pendingDocs.filter((doc) => String(doc.id) !== String(documentId));
  const nextCount = nextDocs.length;

  await chrome.storage.session.set({
    pendingDocs: nextDocs,
    pendingCount: nextCount,
  });
  await updateBadgeCount(nextCount);

  return { nextDocs, nextCount };
}

// Fetch live e-Faktur tax invoices from tax-service via Kong API Gateway
async function getTaxInvoices(params = {}) {
  const { token, apiUrl, orgId, userId } = await getExtensionConfig();
  if (!token) {
    return {
      success: true,
      authenticated: false,
      offline: true,
      invoices: [],
      message: 'Belum login / token Bearer belum dikonfigurasi.',
    };
  }

  try {
    const qs = new URLSearchParams();
    if (params?.efakturStatus) {
      qs.set('efakturStatus', String(params.efakturStatus).trim());
    }
    const querySuffix = qs.toString() ? `?${qs.toString()}` : '';
    const response = await fetch(`${apiUrl}/tax-invoices${querySuffix}`, {
      headers: buildGatewayHeaders({ token, orgId, userId }),
    });

    if (!response.ok) {
      return {
        success: false,
        authenticated: true,
        offline: false,
        statusCode: response.status,
        invoices: [],
        error: await parseGatewayError(response),
      };
    }

    const json = await response.json();
    const invoices = Array.isArray(json)
      ? json
      : Array.isArray(json?.data)
        ? json.data
        : [];

    return {
      success: true,
      authenticated: true,
      offline: false,
      invoices,
    };
  } catch (err) {
    return {
      success: false,
      authenticated: true,
      offline: true,
      invoices: [],
      error: err.message,
    };
  }
}

// Report e-Faktur NTTE / NSFP confirmation to tax-service POST /tax-invoices/{id}/report
async function reportTaxInvoice(input = {}) {
  const targetId = String(input.id || input.invoiceId || '').trim();
  if (!targetId) {
    return { success: false, error: 'ID faktur pajak wajib dipilih.' };
  }

  const { token, apiUrl, orgId, userId } = await getExtensionConfig();
  if (!token) {
    return {
      success: false,
      authenticated: false,
      error: 'Token Bearer diperlukan untuk melaporkan e-Faktur ke API Gateway.',
    };
  }

  const rowVersion = input.rowVersion;
  if (rowVersion === undefined || rowVersion === null || String(rowVersion).trim() === '') {
    return {
      success: false,
      authenticated: true,
      error: 'rowVersion (If-Match) faktur pajak wajib tersedia untuk pelaporan.',
    };
  }

  // Strict JSON DTO for services/tax-service reportReq (DisallowUnknownFields)
  const payload = {};
  if (input.efakturNo !== undefined && input.efakturNo !== null && String(input.efakturNo).trim() !== '') {
    payload.efakturNo = String(input.efakturNo).trim();
  }
  if (input.ntte !== undefined && input.ntte !== null && String(input.ntte).trim() !== '') {
    payload.ntte = String(input.ntte).trim();
  }

  try {
    const headers = buildGatewayHeaders({
      token,
      orgId,
      userId,
      contentType: true,
      idempotencyKey: `tax-report-${targetId}-${Date.now()}`,
      ifMatch: rowVersion,
    });

    const response = await fetch(`${apiUrl}/tax-invoices/${encodeURIComponent(targetId)}/report`, {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      return {
        success: false,
        authenticated: true,
        statusCode: response.status,
        error: await parseGatewayError(response),
      };
    }

    const invoice = await response.json();
    return {
      success: true,
      authenticated: true,
      invoice,
      message: `Faktur ${invoice?.sourceDocumentRef || targetId} berhasil dilaporkan.`,
    };
  } catch (err) {
    return {
      success: false,
      authenticated: true,
      offline: true,
      error: err.message,
    };
  }
}

// Probe Kong /api-gateway /healthz and report honest auth/token status
async function checkGatewayStatus() {
  const { token, apiUrl, orgId } = await getExtensionConfig();
  const healthUrl = buildHealthUrl(apiUrl);
  const controller = typeof AbortController !== 'undefined' ? new AbortController() : null;
  const timer = controller ? setTimeout(() => controller.abort(), 3000) : null;

  let gatewayReachable = false;
  let gatewayStatusCode = null;
  let gatewayService = null;
  let gatewayError = null;

  try {
    const response = await fetch(healthUrl, {
      method: 'GET',
      ...(controller ? { signal: controller.signal } : {}),
    });
    gatewayStatusCode = response.status;
    if (response.ok) {
      gatewayReachable = true;
      try {
        const json = await response.json();
        gatewayService = json?.service || null;
      } catch {
        // Empty 200 body still counts as reachable
      }
    } else {
      gatewayError = `HTTP ${response.status}`;
    }
  } catch (err) {
    gatewayError = err?.name === 'AbortError' ? 'Timeout (/healthz)' : err?.message || 'Tidak terjangkau';
  } finally {
    if (timer) clearTimeout(timer);
  }

  return {
    success: true,
    gatewayReachable,
    gatewayStatusCode,
    gatewayService,
    gatewayError,
    healthUrl,
    apiUrl,
    orgId,
    hasToken: Boolean(token),
    authMode: token ? 'LIVE_BEARER' : 'DEV_OFFLINE',
  };
}

async function handleMessage(message, sender = {}) {
  const msgType = message?.action || message?.type;

  if (msgType === 'OPEN_SIDE_PANEL') {
    let windowId = sender?.tab?.windowId;
    if (!windowId && chrome.windows?.getLastFocused) {
      const focusedWindow = await chrome.windows.getLastFocused();
      windowId = focusedWindow?.id;
    }
    if (windowId && chrome.sidePanel?.open) {
      await chrome.sidePanel.open({ windowId });
    }
    return { success: true };
  }

  if (msgType === 'GET_PENDING_APPROVALS') {
    const { token } = await getExtensionConfig();
    const {
      pendingDocs = DEFAULT_DEV_APPROVALS,
      pendingCount = pendingDocs.length,
      lastSyncTime,
      approvalMode = token ? 'live' : 'dev-offline',
      lastSyncError = null,
    } = await chrome.storage.session.get([
      'pendingDocs',
      'pendingCount',
      'lastSyncTime',
      'approvalMode',
      'lastSyncError',
    ]);
    return {
      success: true,
      docs: pendingDocs,
      count: pendingCount ?? pendingDocs.length,
      lastSyncTime,
      approvalMode,
      lastSyncError,
      hasToken: Boolean(token),
    };
  }

  if (msgType === 'SYNC_NOW') {
    const { docs, count, mode, error } = await syncPendingApprovals();
    const { lastSyncTime } = await chrome.storage.session.get('lastSyncTime');
    return {
      success: mode !== 'live-error',
      docs,
      count,
      mode,
      error,
      lastSyncTime,
    };
  }

  if (msgType === 'APPROVE_DOCUMENT') {
    const { documentId, notes = '', rowVersion } = message;
    const { nextDocs, nextCount } = await submitWorkflowDecision(documentId, 'APPROVE', notes, rowVersion);
    return {
      success: true,
      docs: nextDocs,
      count: nextCount,
      message: `Dokumen ${documentId} berhasil disetujui.`,
    };
  }

  if (msgType === 'REJECT_DOCUMENT') {
    const { documentId, notes = '', rowVersion } = message;
    const { nextDocs, nextCount } = await submitWorkflowDecision(documentId, 'REJECT', notes, rowVersion);
    return {
      success: true,
      docs: nextDocs,
      count: nextCount,
      message: `Dokumen ${documentId} ditolak.`,
    };
  }

  if (msgType === 'GET_TAX_INVOICES') {
    return getTaxInvoices(message.params || message);
  }

  if (msgType === 'REPORT_TAX_INVOICE') {
    return reportTaxInvoice(message.payload || message);
  }

  if (msgType === 'HARDWARE_BARCODE_SCANNED') {
    const cleanBarcode = String(message?.barcode || '').trim();
    if (!cleanBarcode) {
      return { success: false, error: 'Barcode kosong.' };
    }
    const scanRecord = {
      barcode: cleanBarcode,
      source: String(message?.source || 'UNKNOWN'),
      timestamp: Number(message?.timestamp) || Date.now(),
    };
    if (chrome.storage?.session?.set) {
      await chrome.storage.session.set({ lastBarcodeScan: scanRecord });
    }
    if (chrome.storage?.local?.set) {
      await chrome.storage.local.set({ lastBarcodeScan: scanRecord });
    }
    return { success: true, scan: scanRecord };
  }

  if (msgType === 'GET_LAST_BARCODE_SCAN') {
    let scan = null;
    if (chrome.storage?.session?.get) {
      const sess = await chrome.storage.session.get('lastBarcodeScan');
      if (sess?.lastBarcodeScan) scan = sess.lastBarcodeScan;
    }
    if (!scan && chrome.storage?.local?.get) {
      const local = await chrome.storage.local.get('lastBarcodeScan');
      if (local?.lastBarcodeScan) scan = local.lastBarcodeScan;
    }
    return { success: true, scan };
  }

  if (msgType === 'CHECK_GATEWAY_STATUS' || msgType === 'GET_CONNECTION_STATUS') {
    return checkGatewayStatus();
  }

  if (msgType === 'BANK_STATEMENT_EXTRACTED') {
    const { transactions = [], bankName = 'Unknown Bank', accountNumber = '' } = message;
    const extractionRecord = {
      transactions,
      bankName,
      accountNumber,
      extractedAt: Date.now(),
    };

    await chrome.storage.local.set({ lastBankExtraction: extractionRecord });

    // Forward to Kong / treasury-cashbank-service when live token is present
    const { token, apiUrl, orgId, userId } = await getExtensionConfig();

    let syncedToErp = false;
    if (token) {
      const headers = buildGatewayHeaders({
        token,
        orgId,
        userId,
        contentType: true,
        idempotencyKey: `bank-extract-${bankName}-${Date.now()}`,
      });

      const resp = await fetch(`${apiUrl}/bank-statement-lines`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          lines: transactions,
          bankName,
          accountNumber,
          origin: 'chrome-extension',
          extension_version: getExtensionVersion(),
        }),
      });
      syncedToErp = resp.ok;
    }

    return {
      success: true,
      count: transactions.length,
      syncedToErp,
      extraction: extractionRecord,
    };
  }

  if (msgType === 'GET_LAST_BANK_EXTRACTION') {
    const { lastBankExtraction = null } = await chrome.storage.local.get('lastBankExtraction');
    return { success: true, extraction: lastBankExtraction };
  }

  if (msgType === 'SAVE_SETTINGS') {
    const {
      apiUrl = DEFAULT_API_URL,
      orgId = DEFAULT_ORG_ID,
      token,
      clearToken = false,
    } = message.payload || {};
    const cleanApiUrl =
      String(apiUrl || DEFAULT_API_URL)
        .trim()
        .replace(/\/+$/, '') || DEFAULT_API_URL;
    const cleanOrgId = String(orgId || DEFAULT_ORG_ID).trim() || DEFAULT_ORG_ID;

    await chrome.storage.local.set({ apiUrl: cleanApiUrl, orgId: cleanOrgId });

    if (clearToken) {
      await clearStoredToken();
    } else if (typeof token === 'string' && token.trim() !== '') {
      await saveStoredToken(token.trim());
    }

    const { token: activeToken } = await getExtensionConfig();
    return {
      success: true,
      apiUrl: cleanApiUrl,
      orgId: cleanOrgId,
      hasToken: Boolean(activeToken),
    };
  }

  if (msgType === 'CLEAR_TOKEN') {
    await clearStoredToken();
    return { success: true, hasToken: false };
  }

  if (msgType === 'GET_SETTINGS') {
    const { token, apiUrl, orgId } = await getExtensionConfig();
    return { success: true, apiUrl, orgId, hasToken: Boolean(token) };
  }

  return { success: false, error: 'Unknown action' };
}

// 7. Message Passing Handler (async/await IIFE with return true)
if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.onMessage) {
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    (async () => {
      try {
        const result = await handleMessage(message, sender);
        sendResponse(result);
      } catch (error) {
        console.error('[ERP Companion] Service worker error:', error);
        sendResponse({ success: false, error: error.message });
      }
    })();

    return true; // Keep message channel open for async response
  });
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    DEFAULT_API_URL,
    DEFAULT_ORG_ID,
    DEFAULT_DEV_APPROVALS,
    buildHealthUrl,
    buildGatewayHeaders,
    getExtensionConfig,
    saveStoredToken,
    clearStoredToken,
    checkGatewayStatus,
    syncPendingApprovals,
    submitWorkflowDecision,
    getTaxInvoices,
    reportTaxInvoice,
    handleMessage,
  };
}
