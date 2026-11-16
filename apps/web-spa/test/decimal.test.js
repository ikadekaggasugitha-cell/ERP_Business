import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  Decimal,
  addDecimals,
  subtractDecimals,
  multiplyDecimals,
  formatCurrency,
  formatRate,
  formatDate,
} from '../src/lib/decimal.ts';
import {
  extractEventEnvelope,
  resolveInvalidatedQueryKeys,
} from '../src/lib/realtime-matcher.ts';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC_DIR = path.resolve(__dirname, '../src');


describe('SPA Decimal String Arithmetic (Fixed-Point Accounting)', () => {
  test('multiplyDecimals: preserves exact precision without IEEE-754 float drift', () => {
    // In IEEE 754: 0.1 * 0.2 = 0.020000000000000004
    // In our fixed-point arithmetic: 0.0200
    const res1 = multiplyDecimals('0.1', '0.2', 4);
    assert.strictEqual(res1, '0.0200');

    // Multi-billion ledger amounts multiplied by forex exchange rate
    // e.g. 100,000,000.00 IDR * 16250.2550 USD/IDR rate
    const res2 = multiplyDecimals('100000000.00', '16250.2550', 2);
    assert.strictEqual(res2, '1625025500000.00');

    // Exact decimal: 1234.56 * 1.5 = 1851.84
    const res3 = multiplyDecimals('1234.56', '1.5', 2);
    assert.strictEqual(res3, '1851.84');
  });

  test('addDecimals & subtractDecimals: exact BigInt addition/subtraction including negatives and empty inputs', () => {
    assert.strictEqual(addDecimals('14250000000.0000', '8500000000.0000', 4), '22750000000.0000');
    assert.strictEqual(addDecimals('0.1', '0.2', 4), '0.3000');
    assert.strictEqual(subtractDecimals('1000.0000', '250.2500', 4), '749.7500');
    assert.strictEqual(subtractDecimals('250.2500', '1000.0000', 4), '-749.7500');
    assert.strictEqual(addDecimals('', '50.5', 2), '50.50');
    assert.strictEqual(addDecimals('', '', 2), '0.00');
  });

  test('Decimal class: immutable fixed-point KPI aggregation for GL balance snapshots', () => {
    const snapshots = [
      { accountCode: '1101', closingBalance: '14250000000.0000' },
      { accountCode: '1102', closingBalance: '10500000000.0000' },
      { accountCode: '4101', closingBalance: '-12800000000.0000' },
      { accountCode: '4102', closingBalance: '5600000000.0000' },
    ];

    let cashTotal = new Decimal('0', 4);
    let revTotal = new Decimal('0', 4);
    for (const row of snapshots) {
      const bal = new Decimal(row.closingBalance, 4);
      if (row.accountCode.startsWith('11')) {
        cashTotal = cashTotal.add(bal);
      } else if (row.accountCode.startsWith('4')) {
        revTotal = revTotal.add(bal.isNegative() ? bal.abs() : bal);
      }
    }

    assert.strictEqual(cashTotal.toString(2), '24750000000.00');
    assert.strictEqual(revTotal.toString(2), '18400000000.00');
    assert.strictEqual(Decimal.sum(['10.25', '20.75', '-5.00'], 2).toString(2), '26.00');
  });

  test('formatCurrency: formats large enterprise integers with Indonesian locale without float loss', () => {
    const formatted1 = formatCurrency('12450800000.00', 'IDR');
    assert.ok(formatted1.includes('12.450.800.000,00'));
    assert.ok(formatted1.includes('Rp'));

    const formatted2 = formatCurrency('500.50', 'USD');
    assert.ok(formatted2.includes('500,50'));
    assert.ok(formatted2.includes('USD'));

    const formattedNeg = formatCurrency('-1500.00', 'IDR');
    assert.ok(formattedNeg.startsWith('-'));
  });

  test('formatRate & formatDate: formats exchange rates and ISO dates safely', () => {
    assert.strictEqual(formatRate('16250.5000'), '16.250,5000');
    assert.strictEqual(formatRate('1.0000'), '1,0000');
    assert.strictEqual(formatDate(''), '-');
    assert.ok(formatDate('2026-03-15').includes('2026'));
  });
});

describe('SSE Real-Time Event Normalization & Query Key Invalidation', () => {
  test('extractEventEnvelope: normalizes eventType, topic, or SSE event.type', () => {
    const e1 = extractEventEnvelope(
      JSON.stringify({ eventType: 'erp.gl.journal.posted.v1', payload: { journalNo: 'JE-001' } }),
      'message'
    );
    assert.strictEqual(e1.eventKey, 'erp.gl.journal.posted.v1');
    assert.strictEqual(e1.payload.journalNo, 'JE-001');

    const e2 = extractEventEnvelope(
      JSON.stringify({ topic: 'gl.journal.posted', data: { journalNo: 'JE-002' } }),
      'message'
    );
    assert.strictEqual(e2.eventKey, 'gl.journal.posted');
    assert.strictEqual(e2.payload.journalNo, 'JE-002');

    const e3 = extractEventEnvelope('invalid-json', 'workflow.approval.completed');
    assert.strictEqual(e3.eventKey, 'workflow.approval.completed');
  });

  test('resolveInvalidatedQueryKeys: matches both erp.<domain>.* and <domain>.* across all 11 query keys', () => {
    const flatten = (keys) => keys.map((k) => k[0]);

    for (const evt of ['erp.gl.journal.posted.v1', 'gl.journal.posted']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('journals'));
      assert.ok(keys.includes('balances'));
      assert.ok(keys.includes('audit-events'));
    }

    for (const evt of ['erp.workflow.approval.completed.v1', 'workflow.approval.requested']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('approvals'));
    }

    for (const evt of ['erp.ap.invoice.recorded.v1', 'ap.invoice.recorded']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('ap-invoices'));
    }

    for (const evt of ['erp.ar.invoice.issued.v1', 'ar.invoice.issued']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('ar-invoices'));
    }

    for (const evt of ['erp.treasury.payment.settled.v1', 'treasury.payment.settled', 'cashbank.payment.settled']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('bank-accounts'));
    }

    for (const evt of ['erp.inventory.goods.issued.v1', 'inventory.stock.reserved', 'masterdata.item.upserted']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('inventory-items'));
    }

    for (const evt of ['erp.procurement.goods.received.v1', 'procurement.po.approved']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('purchase-orders'));
    }

    for (const evt of ['erp.currency.rate.updated.v1', 'currency.exchange_rate.updated']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('currencies'));
      assert.ok(keys.includes('exchange-rates'));
    }

    for (const evt of ['erp.audit.event.recorded.v1', 'audit.event.recorded']) {
      const keys = flatten(resolveInvalidatedQueryKeys(evt));
      assert.ok(keys.includes('audit-events'));
    }
  });
});

describe('Web SPA Contract & Route Integrity Verification', () => {
  test('login.tsx DEMO_PERSONAS includes all 34 colon-separated backend permissions + gl.* + reporting.read', () => {
    const loginSrc = fs.readFileSync(path.join(SRC_DIR, 'routes/_public/login.tsx'), 'utf8');
    const requiredPerms = [
      'gl.*',
      'reporting.read',
      'ap:read',
      'ap:write',
      'ar:read',
      'ar:write',
      'coa:read',
      'coa:write',
      'tax:read',
      'tax:write',
      'currency:read',
      'currency:write',
      'treasury:read',
      'treasury:write',
      'workflow:read',
      'workflow:write',
      'workflow:approve',
      'inventory:read',
      'inventory:write',
      'procurement:read',
      'procurement:write',
      'sales:read',
      'sales:write',
      'manufacturing:read',
      'manufacturing:write',
      'hr:read',
      'hr:write',
      'payroll:read',
      'payroll:write',
      'masterdata:read',
      'masterdata:write',
      'org:read',
      'org:write',
      'user:read',
      'user:write',
      'security:read',
      'security:write',
      'audit:read',
    ];
    assert.ok(loginSrc.includes('DEMO_PERSONAS'), 'DEMO_PERSONAS must be defined in login.tsx');
    for (const perm of requiredPerms) {
      assert.ok(loginSrc.includes(`'${perm}'`), `Missing permission '${perm}' in login.tsx`);
    }
  });

  test('journals.tsx aligns createMutation and postMutation with gl-service schema and If-Match', () => {
    const journalsSrc = fs.readFileSync(path.join(SRC_DIR, 'routes/_auth/journals.tsx'), 'utf8');
    assert.ok(journalsSrc.includes('financialOrganizationId: activeOrgId'));
    assert.ok(journalsSrc.includes("ledgerId: '00000000-0000-0000-0000-000000000001'"));
    assert.ok(journalsSrc.includes("fiscalPeriodId: '00000000-0000-0000-0000-000000000010'"));
    assert.ok(journalsSrc.includes('sourceDocumentRef: data.description || undefined'));
    assert.ok(journalsSrc.includes("'If-Match': String(rowVersion ?? 1)"));
    assert.ok(!journalsSrc.includes('approvedBy'), 'postMutation must not send unknown body fields');
  });

  test('all 6 domain routes wire live useQuery + VirtualizedTable', () => {
    const routes = ['ap.tsx', 'ar.tsx', 'treasury.tsx', 'inventory.tsx', 'procurement.tsx', 'audit.tsx'];
    for (const routeFile of routes) {
      const content = fs.readFileSync(path.join(SRC_DIR, 'routes/_auth', routeFile), 'utf8');
      assert.ok(content.includes('useQuery'), `${routeFile} must use useQuery`);
      assert.ok(content.includes('VirtualizedTable'), `${routeFile} must render VirtualizedTable`);
    }
  });

  test('zero unused imports across all files in apps/web-spa/src/', () => {
    function walk(dir) {
      let results = [];
      for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          results = results.concat(walk(full));
        } else if (
          (entry.name.endsWith('.ts') || entry.name.endsWith('.tsx')) &&
          entry.name !== 'routeTree.gen.ts'
        ) {
          results.push(full);
        }
      }
      return results;
    }

    const files = walk(SRC_DIR);
    for (const file of files) {
      const src = fs.readFileSync(file, 'utf8');
      const importRegex = /import\s+(?:type\s+)?([\s\S]*?)\s+from\s+['"][^'"]+['"];?/g;
      const bodyWithoutImports = src.replace(importRegex, '');
      let match;
      while ((match = importRegex.exec(src)) !== null) {
        const clause = match[1].trim();
        const identifiers = [];
        const braceMatch = clause.match(/\{([\s\S]*?)\}/);
        if (braceMatch) {
          const parts = braceMatch[1].split(',').map((s) => s.trim()).filter(Boolean);
          for (const p of parts) {
            const cleaned = p.replace(/^type\s+/, '').trim();
            const aliasParts = cleaned.split(/\s+as\s+/);
            identifiers.push(aliasParts[aliasParts.length - 1].trim());
          }
        }
        const beforeBrace = clause.replace(/\{[\s\S]*?\}/, '').replace(/,/g, '').trim();
        if (beforeBrace && !beforeBrace.startsWith('*')) {
          identifiers.push(beforeBrace);
        }
        for (const id of identifiers) {
          const idRegex = new RegExp(`\\b${id}\\b`);
          assert.ok(
            idRegex.test(bodyWithoutImports),
            `Unused import '${id}' in ${path.relative(SRC_DIR, file)}`
          );
        }
      }
    }
  });
});



