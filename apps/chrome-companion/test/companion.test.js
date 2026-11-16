const { test, describe } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');

const sandbox = require('../sandbox/sandbox.js');
const bankExtractor = require('../content/bank-extractor.js');
const taxHelper = require('../content/tax-helper.js');

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

