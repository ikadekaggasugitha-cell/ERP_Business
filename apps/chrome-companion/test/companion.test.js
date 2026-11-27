const { test, describe } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');

const sandbox = require('../sandbox/sandbox.js');
const bankExtractor = require('../content/bank-extractor.js');
const taxHelper = require('../content/tax-helper.js');
const sidepanel = require('../sidepanel/sidepanel.js');
const serviceWorker = require('../service-worker.js');

describe('Bank Parser Amount Normalization & 100x Bug Fix', () => {
  test('sandbox.parseAmount: correctly parses 1234,56 without 100x overstatement', () => {
    const result = sandbox.parseAmount('1234,56');
    assert.strictEqual(result, 1234.56, '1234,56 must equal 1234.56, not 123456');
  });

  test('bankExtractor.parseAmount: correctly parses 1234,56 without 100x overstatement', () => {
    const result = bankExtractor.parseAmount('1234,56');
    assert.strictEqual(result, 1234.56, '1234,56 must equal 1234.56, not 123456');
  });

  test('sandbox.parseAmount: correctly parses Indonesian format with dots as thousands and comma as decimal', () => {
    assert.strictEqual(sandbox.parseAmount('1.234.567,89'), 1234567.89);
    assert.strictEqual(sandbox.parseAmount('Rp 1.000.000,00'), 1000000.00);
    assert.strictEqual(sandbox.parseAmount('500,00'), 500.00);
    assert.strictEqual(sandbox.parseAmount('12,5'), 12.5);
  });

  test('sandbox.parseAmount: correctly parses standard international format', () => {
    assert.strictEqual(sandbox.parseAmount('1,234.56'), 1234.56);
    assert.strictEqual(sandbox.parseAmount('1234.56'), 1234.56);
    assert.strictEqual(sandbox.parseAmount('1000000'), 1000000);
  });

  test('sandbox.parseAmount: handles edge cases like dashes, negatives, empty strings', () => {
    assert.strictEqual(sandbox.parseAmount(''), 0);
    assert.strictEqual(sandbox.parseAmount('-'), 0);
    assert.strictEqual(sandbox.parseAmount(null), 0);
    assert.strictEqual(sandbox.parseAmount(undefined), 0);
    assert.strictEqual(sandbox.parseAmount('-1234,56'), -1234.56);
  });
});

describe('Bank Statement Date & Column Normalization', () => {
  test('sandbox.normalizeDate: normalizes DD/MM/YYYY and DD-MM-YYYY to YYYY-MM-DD', () => {
    assert.strictEqual(sandbox.normalizeDate('15/11/2026'), '2026-11-15');
    assert.strictEqual(sandbox.normalizeDate('01-08-2026'), '2026-08-01');
  });

  test('sandbox.normalizeDate: normalizes DD MMM YYYY in Indonesian', () => {
    assert.strictEqual(sandbox.normalizeDate('15 Nov 2026'), '2026-11-15');
    assert.strictEqual(sandbox.normalizeDate('20 Mei 2026'), '2026-05-20');
  });

  test('sandbox.detectColumnMapping: requires BOTH date and description (line 79 fix)', () => {
    // Only description, missing date
    const onlyDesc = sandbox.detectColumnMapping(['no', 'keterangan', 'debet']);
    assert.strictEqual(onlyDesc, null, 'Must return null if date column is missing');

    // Only date, missing description
    const onlyDate = sandbox.detectColumnMapping(['tanggal', 'nominal', 'saldo']);
    assert.strictEqual(onlyDate, null, 'Must return null if description column is missing');

    // Both present
    const valid = sandbox.detectColumnMapping(['tanggal', 'keterangan', 'debet', 'kredit', 'saldo']);
    assert.notStrictEqual(valid, null);
    assert.strictEqual(valid.date, 0);
    assert.strictEqual(valid.description, 1);
  });
});

describe('Manifest V3 Compliance & Permissions', () => {
  const manifestPath = path.resolve(__dirname, '../manifest.json');
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));

  test('manifest specifies manifest_version 3', () => {
    assert.strictEqual(manifest.manifest_version, 3);
  });

  test('manifest specifies all_frames: true for all content scripts', () => {
    assert.ok(Array.isArray(manifest.content_scripts), 'content_scripts must be an array');
    assert.ok(manifest.content_scripts.length >= 2, 'Must have at least 2 content scripts');
    for (const cs of manifest.content_scripts) {
      assert.strictEqual(
        cs.all_frames,
        true,
        `Content script for ${cs.matches} must have all_frames: true for banking/tax iframes`
      );
    }
  });
});

describe('UUID & Gateway Parity', () => {
  const expectedOrgId = '11111111-1111-1111-1111-111111111111';

  test('service-worker.js uses valid UUID for DEFAULT_ORG_ID', () => {
    const swContent = fs.readFileSync(path.resolve(__dirname, '../service-worker.js'), 'utf8');
    assert.ok(
      swContent.includes(`const DEFAULT_ORG_ID = '${expectedOrgId}';`),
      `DEFAULT_ORG_ID must be ${expectedOrgId}`
    );
  });

  test('no occurrences of deprecated org-001 exist in extension code', () => {
    const files = [
      '../manifest.json',
      '../service-worker.js',
      '../sidepanel/sidepanel.js',
      '../sidepanel/sidepanel.html',
      '../content/tax-helper.js',
      '../content/bank-extractor.js',
      '../sandbox/sandbox.js',
    ];

    for (const f of files) {
      const content = fs.readFileSync(path.resolve(__dirname, f), 'utf8');
      assert.ok(!content.includes('org-001'), `File ${f} must not contain deprecated org-001`);
    }
  });

  test('service-worker.js routes approvals to /workflow-tasks and bank extraction to /bank-statement-lines', () => {
    const swContent = fs.readFileSync(path.resolve(__dirname, '../service-worker.js'), 'utf8');
    assert.ok(
      swContent.includes('/workflow-tasks?status=PENDING'),
      'Must target /workflow-tasks?status=PENDING'
    );
    assert.ok(
      swContent.includes('/workflow-tasks/${encodeURIComponent(documentId)}/${action}'),
      'Must target /workflow-tasks/{id}/approve and /workflow-tasks/{id}/reject'
    );
    assert.ok(
      swContent.includes('/bank-statement-lines'),
      'Must target /bank-statement-lines'
    );
  });
});

describe('Tax Helper & Native Form Setter', () => {
  test('normalizePayload maps ERP tax invoice fields accurately', () => {
    const payload = {
      customerNPWP: '01.234.567.8-901.000',
      customerName: 'PT Sukses Makmur',
      customerAddress: 'Jl. Sudirman No. 1, Jakarta',
      dppAmount: '10000000',
      ppnAmount: '1100000',
      taxInvoiceDate: '2026-11-13',
      invoiceNo: 'INV-2026-001',
    };

    const normalized = taxHelper.normalizePayload(payload);
    assert.strictEqual(normalized.npwp, '01.234.567.8-901.000');
    assert.strictEqual(normalized.nama, 'PT Sukses Makmur');
    assert.strictEqual(normalized.dpp, '10000000');
    assert.strictEqual(normalized.ppn, '1100000');
    assert.strictEqual(normalized.referensi, 'INV-2026-001');
  });

  test('extractTaxConfirmation returns structure with timestamp', () => {
    const conf = taxHelper.extractTaxConfirmation();
    assert.ok('timestamp' in conf);
    assert.ok('ntte' in conf);
    assert.ok('nsfp' in conf);
  });
});

function createMockChrome(initialLocal = {}, initialSession = {}, identityOverride = {}) {
  const localStore = { ...initialLocal };
  const sessionStore = { ...initialSession };

  const makeArea = (store) => ({
    async get(keys) {
      if (Array.isArray(keys)) {
        const out = {};
        for (const k of keys) out[k] = store[k];
        return out;
      }
      if (typeof keys === 'string') {
        return { [keys]: store[keys] };
      }
      return { ...store };
    },
    async set(obj) {
      Object.assign(store, obj);
    },
    async remove(key) {
      delete store[key];
    },
  });

  return {
    localStore,
    sessionStore,
    chrome: {
      storage: {
        local: makeArea(localStore),
        session: makeArea(sessionStore),
      },
      runtime: {
        getManifest: () => ({ version: '1.0.0' }),
      },
      identity: {
        getRedirectURL: (suffix = '') => `https://ext-id.chromiumapp.org/${suffix}`,
        ...identityOverride,
      },
    },
  };
}

describe('Live Tax Invoice Wiring & OIDC PKCE Controls', () => {
  test('sidepanel.js and sidepanel.html contain zero hardcoded INVOICE_CATALOG fixtures', () => {
    const js = fs.readFileSync(path.resolve(__dirname, '../sidepanel/sidepanel.js'), 'utf8');
    const html = fs.readFileSync(path.resolve(__dirname, '../sidepanel/sidepanel.html'), 'utf8');

    assert.ok(!js.includes('INVOICE_CATALOG'), 'sidepanel.js must not define hardcoded INVOICE_CATALOG');
    assert.ok(!js.includes('INV-2026-0089'), 'sidepanel.js must not hardcode INV-2026-0089');
    assert.ok(!html.includes('INV-2026-0089'), 'sidepanel.html must not hardcode INV-2026-0089');
    assert.ok(!html.includes('INV-2026-0090'), 'sidepanel.html must not hardcode INV-2026-0090');

    assert.ok(js.includes('GET_TAX_INVOICES'), 'sidepanel.js must fetch live invoices via GET_TAX_INVOICES');
    assert.ok(js.includes('REPORT_TAX_INVOICE'), 'sidepanel.js must report invoices via REPORT_TAX_INVOICE');
    assert.ok(html.includes('id="taxInvoiceState"'), 'sidepanel.html must expose explicit tax invoice state banner');
    assert.ok(html.includes('id="bearerTokenInput"'), 'sidepanel.html must expose manual Bearer token input');
    assert.ok(html.includes('id="clearTokenBtn"'), 'sidepanel.html must expose clear token button');
    assert.ok(html.includes('id="oidcLoginBtn"'), 'sidepanel.html must expose OIDC PKCE login button');
  });

  test('sidepanel maps tax-service DTO cleanly to autofill payload and option label', () => {
    const dto = {
      id: 'tax-inv-1',
      sourceDocumentRef: 'AR-2026-0101',
      baseAmount: '150000000.00',
      taxAmount: '16500000.00',
      efakturNo: '010.000-26.00000001',
      efakturStatus: 'DRAFT',
      rowVersion: 2,
    };

    const payload = sidepanel.mapTaxInvoiceToAutofillPayload(dto);
    assert.strictEqual(payload.invoiceNo, 'AR-2026-0101');
    assert.strictEqual(payload.dppAmount, '150000000.00');
    assert.strictEqual(payload.ppnAmount, '16500000.00');

    const label = sidepanel.formatTaxInvoiceOptionLabel(dto);
    assert.ok(label.includes('AR-2026-0101'));
    assert.ok(label.includes('150000000.00'));
    assert.ok(label.includes('DRAFT'));
  });

  test('service-worker getTaxInvoices returns explicit unauthenticated/live/error states without fake fallback', async () => {
    const prevChrome = global.chrome;
    const prevFetch = global.fetch;

    try {
      // 1. Unauthenticated
      const unauthEnv = createMockChrome({ apiUrl: 'https://api.erp.test/api/v1' }, {});
      global.chrome = unauthEnv.chrome;
      const unauthResp = await serviceWorker.getTaxInvoices();
      assert.strictEqual(unauthResp.authenticated, false);
      assert.deepStrictEqual(unauthResp.invoices, []);

      // 2. Live fetch
      const authEnv = createMockChrome(
        { apiUrl: 'https://api.erp.test/api/v1' },
        { token: 'live-jwt-token' }
      );
      global.chrome = authEnv.chrome;
      let capturedUrl = '';
      let capturedAuth = '';
      global.fetch = async (url, init) => {
        capturedUrl = url;
        capturedAuth = init.headers.Authorization;
        return {
          ok: true,
          status: 200,
          async json() {
            return {
              data: [
                {
                  id: 'ti-1',
                  sourceDocumentRef: 'AR-2026-0200',
                  baseAmount: '10000000.00',
                  taxAmount: '1100000.00',
                  efakturStatus: 'DRAFT',
                  rowVersion: 1,
                },
              ],
            };
          },
        };
      };

      const liveResp = await serviceWorker.getTaxInvoices();
      assert.strictEqual(capturedUrl, 'https://api.erp.test/api/v1/tax-invoices');
      assert.strictEqual(capturedAuth, 'Bearer live-jwt-token');
      assert.strictEqual(liveResp.success, true);
      assert.strictEqual(liveResp.authenticated, true);
      assert.strictEqual(liveResp.invoices.length, 1);
      assert.strictEqual(liveResp.invoices[0].id, 'ti-1');

      // 3. Upstream error returns explicit error and empty list
      global.fetch = async () => ({
        ok: false,
        status: 502,
        async json() {
          return { error: { message: 'tax-service unavailable' } };
        },
      });
      const errResp = await serviceWorker.getTaxInvoices();
      assert.strictEqual(errResp.success, false);
      assert.deepStrictEqual(errResp.invoices, []);
      assert.ok(errResp.error.includes('tax-service unavailable'));
    } finally {
      global.chrome = prevChrome;
      global.fetch = prevFetch;
    }
  });

  test('service-worker reportTaxInvoice sends If-Match, Idempotency-Key, and strict DTO', async () => {
    const prevChrome = global.chrome;
    const prevFetch = global.fetch;

    try {
      const authEnv = createMockChrome(
        { apiUrl: 'https://api.erp.test/api/v1' },
        { token: 'live-jwt-token' }
      );
      global.chrome = authEnv.chrome;

      let reqUrl = '';
      let reqInit = null;
      global.fetch = async (url, init) => {
        reqUrl = url;
        reqInit = init;
        return {
          ok: true,
          status: 200,
          async json() {
            return {
              id: 'ti-1',
              sourceDocumentRef: 'AR-2026-0200',
              efakturStatus: 'REPORTED',
              rowVersion: 2,
            };
          },
        };
      };

      const resp = await serviceWorker.handleMessage({
        action: 'REPORT_TAX_INVOICE',
        payload: {
          id: 'ti-1',
          rowVersion: 1,
          efakturNo: '010.000-26.00000001',
          ntte: 'NTTE1234567890ABCDEF',
        },
      });

      assert.strictEqual(resp.success, true);
      assert.strictEqual(reqUrl, 'https://api.erp.test/api/v1/tax-invoices/ti-1/report');
      assert.strictEqual(reqInit.method, 'POST');
      assert.strictEqual(reqInit.headers['If-Match'], '1');
      assert.ok(reqInit.headers['Idempotency-Key'].startsWith('tax-report-ti-1-'));
      assert.deepStrictEqual(JSON.parse(reqInit.body), {
        efakturNo: '010.000-26.00000001',
        ntte: 'NTTE1234567890ABCDEF',
      });
    } finally {
      global.chrome = prevChrome;
      global.fetch = prevFetch;
    }
  });

  test('service-worker launchOidcPkceFlow uses PKCE S256 and stores exchanged token in session', async () => {
    const prevChrome = global.chrome;
    const prevFetch = global.fetch;

    try {
      let launchedAuthUrl = '';
      const env = createMockChrome(
        {},
        {},
        {
          async launchWebAuthFlow({ url }) {
            launchedAuthUrl = url;
            const parsed = new URL(url);
            const state = parsed.searchParams.get('state');
            return `https://ext-id.chromiumapp.org/oidc-callback?code=auth-code-123&state=${encodeURIComponent(state)}`;
          },
        }
      );
      global.chrome = env.chrome;

      let tokenExchangeBody = '';
      global.fetch = async (url, init) => {
        assert.strictEqual(url, 'https://idp.sinergi.example.com/oauth2/token');
        tokenExchangeBody = init.body;
        return {
          ok: true,
          status: 200,
          async json() {
            return { access_token: 'pkce-session-token-xyz' };
          },
        };
      };

      const res = await serviceWorker.handleMessage({
        action: 'START_OIDC_PKCE',
        payload: {
          authUrl: 'https://idp.sinergi.example.com/oauth2/authorize',
          tokenUrl: 'https://idp.sinergi.example.com/oauth2/token',
          clientId: 'erp-chrome-companion',
        },
      });

      assert.strictEqual(res.success, true);
      assert.strictEqual(res.hasToken, true);
      assert.strictEqual(env.sessionStore.token, 'pkce-session-token-xyz');

      const authParsed = new URL(launchedAuthUrl);
      assert.strictEqual(authParsed.searchParams.get('code_challenge_method'), 'S256');
      assert.ok(authParsed.searchParams.get('code_challenge'));
      assert.ok(tokenExchangeBody.includes('grant_type=authorization_code'));
      assert.ok(tokenExchangeBody.includes('code_verifier='));
    } finally {
      global.chrome = prevChrome;
      global.fetch = prevFetch;
    }
  });
});

describe('Production Packager (scripts/package-prod.mjs)', () => {
  const scriptPath = path.resolve(__dirname, '../scripts/package-prod.mjs');

  test('package-prod.mjs --dry-run strips localhost permissions and validates MV3 defaults', () => {
    const stdout = execFileSync(process.execPath, [scriptPath, '--dry-run'], {
      encoding: 'utf8',
    });
    const result = JSON.parse(stdout);

    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.mode, 'dry-run');
    assert.strictEqual(result.manifestVersion, 3);
    assert.strictEqual(result.origin, 'https://erp.sinergi.example.com/*');
    assert.ok(
      result.strippedHostPermissions.includes('http://localhost:8000/*'),
      'Must strip http://localhost:8000/*'
    );
    assert.ok(
      result.strippedHostPermissions.includes('http://localhost:3000/*'),
      'Must strip http://localhost:3000/*'
    );
    assert.ok(
      result.hostPermissions.includes('https://erp.sinergi.example.com/*'),
      'Must include default production origin'
    );
    for (const perm of result.hostPermissions) {
      assert.ok(!perm.includes('localhost'), `Production host_permissions must not contain localhost: ${perm}`);
    }
  });

  test('package-prod.mjs --dry-run supports custom --origin normalization', () => {
    const stdout = execFileSync(
      process.execPath,
      [scriptPath, '--dry-run', '--origin', 'https://erp.corp.id'],
      { encoding: 'utf8' }
    );
    const result = JSON.parse(stdout);
    assert.strictEqual(result.valid, true);
    assert.strictEqual(result.origin, 'https://erp.corp.id/*');
    assert.ok(result.hostPermissions.includes('https://erp.corp.id/*'));
  });
});

