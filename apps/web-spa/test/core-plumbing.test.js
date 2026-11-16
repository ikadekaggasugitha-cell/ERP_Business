import test from 'node:test';
import assert from 'node:assert/strict';
import {
  Decimal,
  addDecimals,
  divideDecimals,
  formatCurrency,
  formatNumber,
  formatRate,
} from '../src/lib/decimal.ts';
import {
  KNOWN_SSE_EVENT_TYPES,
  resolveInvalidatedQueryKeys,
} from '../src/lib/realtime-matcher.ts';
import {
  PORTAL_MODULES,
  filterNavByPortalAndPermission,
  hasUserPermission,
} from '../src/lib/permissions.ts';
import { emitErpEvent, onErpEvent } from '../src/lib/event-bus.ts';
import { buildCsvString, escapeCsvCell } from '../src/lib/export.ts';

test('Decimal comparisons, division, and edge cases preserve BigInt accuracy', () => {
  assert.equal(divideDecimals('100', '3', 4), '33.3333');
  assert.equal(divideDecimals('10', '6', 2), '1.67');
  assert.equal(divideDecimals('-100', '4', 2), '-25.00');
  assert.equal(divideDecimals('0', '50', 2), '0.00');
  assert.throws(() => divideDecimals('10', '0', 2), /Division by zero/);

  const a = new Decimal('1250.50');
  const b = new Decimal('1250.5000');
  const c = new Decimal('2000.00');

  assert.equal(a.equals(b), true);
  assert.equal(a.compareTo(b), 0);
  assert.equal(c.gt(a), true);
  assert.equal(c.gte(b), true);
  assert.equal(a.lt(c), true);
  assert.equal(a.lte(b), true);
  assert.equal(a.isPositive(), true);
  assert.equal(new Decimal('-0.00').isZero(), true);
  assert.equal(new Decimal('-0.00').isNegative(), false);
  assert.equal(a.negate().toString(2), '-1250.50');

  // Scientific notation string/number handling without float drift or digit mangling
  assert.equal(addDecimals('1e-7', '0.0000002', 7), '0.0000003');
  assert.equal(addDecimals('1.5e2', '25', 2), '175.00');
  assert.equal(new Decimal('1e-4').equals('0.0001'), true);
  assert.equal(formatRate(1e-4, 'id'), '0,0001');
  assert.equal(formatRate(2.5e3, 'en'), '2,500');
});

test('Multi-locale currency, number, and rate formatting work in ID and EN without floats', () => {
  assert.equal(formatCurrency('12450800000.55', 'IDR', 'id'), 'Rp\u00a012.450.800.000,55');
  assert.equal(formatCurrency('12450800000.55', 'IDR', 'en'), 'Rp\u00a012,450,800,000.55');
  assert.equal(formatCurrency('-500.5', 'USD', 'en-US'), '-USD\u00a0500.50');
  assert.equal(formatNumber('9876543.21', 2, 'en'), '9,876,543.21');
  assert.equal(formatRate('16250.5000', 'en'), '16,250.5000');
});

test('Realtime matcher covers all 29 canonical DomainTopics from api-gateway hub.go', () => {
  const canonicalTopics = [
    'erp.gl.journal.posted.v1',
    'erp.gl.journal.reversed.v1',
    'erp.coa.account.upserted.v1',
    'erp.coa.period.updated.v1',
    'erp.ap.invoice.recorded.v1',
    'erp.ar.invoice.issued.v1',
    'erp.tax.invoice.issued.v1',
    'erp.tax.efaktur.reported.v1',
    'erp.treasury.payment.settled.v1',
    'erp.cashbank.payment.settled.v1',
    'erp.hr.employee.updated.v1',
    'erp.payroll.run.completed.v1',
    'erp.procurement.po.approved.v1',
    'erp.procurement.goods.received.v1',
    'erp.inventory.stock.reserved.v1',
    'erp.inventory.goods.issued.v1',
    'erp.inventory.reservation.failed.v1',
    'erp.sales.order.confirmed.v1',
    'erp.manufacturing.workorder.released.v1',
    'erp.manufacturing.workorder.completed.v1',
    'erp.masterdata.party.upserted.v1',
    'erp.org.unit.upserted.v1',
    'erp.security.role.upserted.v1',
    'erp.user.account.upserted.v1',
    'erp.workflow.approval.requested.v1',
    'erp.workflow.approval.completed.v1',
    'erp.currency.created.v1',
    'erp.currency.rate.created.v1',
    'erp.currency.rate.updated.v1',
  ];

  for (const topic of canonicalTopics) {
    assert.ok(
      KNOWN_SSE_EVENT_TYPES.includes(topic),
      `Missing canonical SSE topic in KNOWN_SSE_EVENT_TYPES: ${topic}`
    );
  }

  assert.deepEqual(
    resolveInvalidatedQueryKeys('erp.coa.account.upserted.v1').map((k) => k[0]).sort(),
    ['audit-events', 'balances', 'coa-accounts', 'dashboard-metrics', 'fiscal-periods', 'ledgers']
  );
  assert.deepEqual(
    resolveInvalidatedQueryKeys('erp.tax.efaktur.reported.v1').map((k) => k[0]).sort(),
    ['audit-events', 'balances', 'dashboard-metrics', 'tax-codes', 'tax-invoices']
  );
  assert.deepEqual(
    resolveInvalidatedQueryKeys('erp.payroll.run.completed.v1').map((k) => k[0]).sort(),
    ['audit-events', 'bank-accounts', 'dashboard-metrics', 'journals', 'payroll-runs']
  );
  assert.deepEqual(
    resolveInvalidatedQueryKeys('erp.security.role.upserted.v1').map((k) => k[0]).sort(),
    ['audit-events', 'roles', 'users']
  );
});

test('RBAC permission matcher handles dot/colon equivalence, wildcards, and denies unauthorized writes', () => {
  const apReader = { roles: ['VIEWER'], permissions: ['ap:read', 'gl.read'] };
  assert.equal(hasUserPermission(apReader, 'ap:read'), true);
  assert.equal(hasUserPermission(apReader, 'ap.read'), true);
  assert.equal(hasUserPermission(apReader, 'gl:read'), true);
  assert.equal(hasUserPermission(apReader, 'ap:write'), false);
  assert.equal(hasUserPermission(apReader, 'gl.create'), false);

  const glWildcard = { roles: ['ACCOUNTANT'], permissions: ['gl.*'] };
  assert.equal(hasUserPermission(glWildcard, 'gl:read'), true);
  assert.equal(hasUserPermission(glWildcard, 'gl.post'), true);
  assert.equal(hasUserPermission(glWildcard, 'ap:read'), false);

  // Prevent write -> approve privilege escalation
  const accountant = { roles: ['ACCOUNTANT'], permissions: ['workflow:write', 'gl.update', 'gl.write'] };
  assert.equal(hasUserPermission(accountant, 'workflow:write'), true);
  assert.equal(hasUserPermission(accountant, 'workflow:approve'), false);
  assert.equal(hasUserPermission(accountant, 'gl.approve'), false);

  const approver = { roles: ['FINANCE_MANAGER'], permissions: ['workflow:approve', 'gl.approve'] };
  assert.equal(hasUserPermission(approver, 'workflow:approve'), true);
  assert.equal(hasUserPermission(approver, 'gl.approve'), true);

  const superAdmin = { roles: ['SYSTEM_ADMIN'], permissions: [] };
  assert.equal(hasUserPermission(superAdmin, 'security:write'), true);
  assert.equal(hasUserPermission(null, 'gl:read'), false);

  const navAdmin = filterNavByPortalAndPermission('admin', superAdmin);
  assert.equal(navAdmin.some((n) => n.id === 'organizations'), true);
  assert.equal(navAdmin.some((n) => n.id === 'security'), true);

  const navViewerBackoffice = filterNavByPortalAndPermission('backoffice', apReader);
  assert.equal(navViewerBackoffice.some((n) => n.id === 'ap'), true);
  assert.equal(navViewerBackoffice.some((n) => n.id === 'payroll'), false);
  assert.ok(PORTAL_MODULES.length >= 20);
});

test('Host MFE event bus emits and cleans up typed listeners', () => {
  let received = '';
  const off = onErpEvent('erp:locale-changed', (detail) => {
    received = detail.locale;
  });

  emitErpEvent('erp:locale-changed', { locale: 'en' });
  assert.equal(received, 'en');

  off();
  emitErpEvent('erp:locale-changed', { locale: 'id' });
  assert.equal(received, 'en');
});

test('CSV export neutralizes formula injection while keeping valid negative ledger decimals', () => {
  assert.equal(escapeCsvCell('=CMD("calc")'), '"\'=CMD(""calc"")"');
  assert.equal(escapeCsvCell('-1+1'), '"\'-1+1"');
  assert.equal(escapeCsvCell('-12500.50'), '"-12500.50"');
  assert.equal(escapeCsvCell(null), '""');

  const csv = buildCsvString(
    [{ code: 'JV-01', net: '-12500.50', note: '=SUM(A1:A2)' }],
    [
      { header: 'Code', accessor: (r) => r.code },
      { header: 'Net', accessor: (r) => r.net },
      { header: 'Note', accessor: (r) => r.note },
    ]
  );
  assert.ok(csv.startsWith('\uFEFF"Code","Net","Note"\r\n'));
  assert.ok(csv.includes('"-12500.50","\'=SUM(A1:A2)"'));
});
