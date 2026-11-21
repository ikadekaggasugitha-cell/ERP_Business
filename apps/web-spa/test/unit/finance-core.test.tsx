import * as React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nextProvider } from 'react-i18next';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { CoaWorkspace } from '../../src/features/coa/CoaWorkspace';
import { buildAccountHierarchyRows } from '../../src/features/coa/coa-types';
import { DashboardWorkspace } from '../../src/features/dashboard/DashboardWorkspace';
import {
  classifyAccountCategory,
  summarizeGlSnapshots,
  summarizeSalesAndInventory,
} from '../../src/features/dashboard/dashboard-utils';
import { JournalsWorkspace } from '../../src/features/journals/JournalsWorkspace';
import {
  computeFunctionalAmount,
  computeJournalTotals,
  journalFormSchema,
} from '../../src/features/journals/journals-types';
import { ReportsWorkspace } from '../../src/features/reports/ReportsWorkspace';
import * as exportLib from '../../src/lib/export';
import i18n from '../../src/lib/i18n';
import { useAuthStore } from '../../src/store/auth-store';
import { useUiStore } from '../../src/store/ui-store';
import type { Organization, User } from '../../src/types';

const mockNavigate = vi.fn();

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    to,
    children,
    className,
  }: {
    to: string;
    children: React.ReactNode;
    className?: string;
  }) => (
    <a href={typeof to === 'string' ? to : '/'} className={className}>
      {children}
    </a>
  ),
  useNavigate: () => mockNavigate,
}));

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>();
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div style={{ width: 640, height: 260 }} data-testid="recharts-container">
        {children}
      </div>
    ),
  };
});

const TEST_ORG: Organization = {
  id: '11111111-1111-1111-1111-111111111111',
  code: 'HLD-01',
  name: 'PT Sinergi Investasi Properti',
  type: 'HOLDING',
  parentId: null,
  functionalCurrency: 'IDR',
};

const FULL_FINANCE_USER: User = {
  id: 'usr-fin-01',
  username: 'finance.controller',
  email: 'controller@sinergi.co.id',
  fullName: 'Finance Controller',
  roles: ['FINANCE_CONTROLLER'],
  permissions: [
    'reporting.read',
    'gl:read',
    'gl:create',
    'gl:update',
    'gl:approve',
    'coa:read',
    'coa:write',
    'currency:read',
    'org:read',
    'workflow:read',
  ],
};

const READONLY_USER: User = {
  id: 'usr-ro-01',
  username: 'auditor.viewer',
  email: 'auditor@sinergi.co.id',
  fullName: 'Internal Auditor',
  roles: ['AUDITOR'],
  permissions: ['reporting.read', 'gl:read', 'coa:read', 'currency:read', 'workflow:read'],
};

function jsonResponse({ status = 200, body }: { status?: number; body?: unknown }): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function listEnvelope<T>(data: T[], totalElements = data.length) {
  return {
    data,
    page: {
      number: 1,
      size: Math.max(data.length, 25),
      totalElements,
      totalPages: totalElements > 0 ? 1 : 0,
    },
  };
}

function renderWithProviders(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>{ui}</I18nextProvider>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  mockNavigate.mockReset();
  useUiStore.setState({ locale: 'id', isRealtimeConnected: true });
  useAuthStore.setState({
    user: FULL_FINANCE_USER,
    token: 'test-token',
    activeOrg: TEST_ORG,
    organizations: [TEST_ORG],
    isAuthenticated: true,
    hasHydrated: true,
    isDemoSession: false,
  });

  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  });

  if (typeof Element !== 'undefined') {
    if (!Element.prototype.scrollIntoView) {
      Element.prototype.scrollIntoView = vi.fn();
    }
    if (!Element.prototype.hasPointerCapture) {
      Element.prototype.hasPointerCapture = vi.fn(() => false);
    }
    if (!Element.prototype.setPointerCapture) {
      Element.prototype.setPointerCapture = vi.fn();
    }
    if (!Element.prototype.releasePointerCapture) {
      Element.prototype.releasePointerCapture = vi.fn();
    }
  }
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Fixed-point Finance Core Logic', () => {
  it('computes exact multi-line decimal totals and functional amounts without IEEE-754 drift', () => {
    const unbalanced = computeJournalTotals([
      { debitAmount: '0.1', creditAmount: '0' },
      { debitAmount: '0.2', creditAmount: '0' },
      { debitAmount: '0', creditAmount: '0.29' },
    ]);
    expect(unbalanced.isBalanced).toBe(false);
    expect(unbalanced.canSubmit).toBe(false);
    expect(unbalanced.difference).toBe('0.01');

    const balanced = computeJournalTotals([
      { debitAmount: '1000000.10', creditAmount: '0' },
      { debitAmount: '250000.20', creditAmount: '0' },
      { debitAmount: '0', creditAmount: '1250000.30' },
    ]);
    expect(balanced.totalDebit).toBe('1250000.30');
    expect(balanced.totalCredit).toBe('1250000.30');
    expect(balanced.difference).toBe('0.00');
    expect(balanced.isBalanced).toBe(true);
    expect(balanced.canSubmit).toBe(true);

    expect(computeFunctionalAmount('100.25', '0', '15800.50')).toBe('1584000.1250');
  });

  it('enforces multi-line Zod double-entry and XOR debit/credit validation', () => {
    const invalidXor = journalFormSchema.safeParse({
      journalNo: 'JRN-2026-0001',
      financialOrganizationId: 'fin-1',
      ledgerId: 'led-1',
      fiscalPeriodId: 'per-1',
      postingDate: '2026-03-15',
      journalSource: 'Manual',
      currencyCode: 'IDR',
      exchangeRate: '1',
      lines: [
        { accountId: 'acc-1', debitAmount: '100.00', creditAmount: '100.00' },
        { accountId: 'acc-2', debitAmount: '0.00', creditAmount: '100.00' },
      ],
    });
    expect(invalidXor.success).toBe(false);

    const validThreeLines = journalFormSchema.safeParse({
      journalNo: 'JRN-2026-0002',
      financialOrganizationId: 'fin-1',
      ledgerId: 'led-1',
      fiscalPeriodId: 'per-1',
      postingDate: '2026-03-15',
      journalSource: 'Manual',
      currencyCode: 'USD',
      exchangeRate: '15850.25',
      lines: [
        { accountId: 'acc-1', debitAmount: '100.00', creditAmount: '0.00' },
        { accountId: 'acc-2', debitAmount: '50.00', creditAmount: '0.00' },
        { accountId: 'acc-3', debitAmount: '0.00', creditAmount: '150.00' },
      ],
    });
    expect(validThreeLines.success).toBe(true);
  });

  it('aggregates GL categories, sales, inventory, and CoA hierarchy cleanly', () => {
    expect(classifyAccountCategory('1101')).toBe('Asset');
    expect(classifyAccountCategory('2101')).toBe('Liability');
    expect(classifyAccountCategory('4001')).toBe('Revenue');
    expect(classifyAccountCategory('6100')).toBe('Expense');
    expect(classifyAccountCategory('ACCT:1101-uuid')).toBe('Other');
    expect(classifyAccountCategory('JV-170000000')).toBe('Other');

    const glSummary = summarizeGlSnapshots([
      {
        id: '1',
        fiscalPeriodId: 'per-1',
        accountCode: '1101',
        closingBalance: '100000000.50',
        refreshedAt: '2026-03-15T10:00:00Z',
      },
      {
        id: '2',
        fiscalPeriodId: 'per-1',
        accountCode: '4100',
        closingBalance: '50000000.00',
        refreshedAt: '2026-03-15T11:00:00Z',
      },
      {
        id: '3',
        fiscalPeriodId: 'per-1',
        accountCode: '5100',
        closingBalance: '20000000.25',
        refreshedAt: '2026-03-15T09:00:00Z',
      },
    ]);
    expect(glSummary.assetsTotal).toBe('100000000.50');
    expect(glSummary.netOperatingMargin).toBe('29999999.75');
    expect(glSummary.lastRefreshedAt).toBe('2026-03-15T11:00:00Z');

    const opsSummary = summarizeSalesAndInventory(
      [
        {
          id: 's1',
          organizationId: TEST_ORG.id,
          periodYear: 2026,
          periodMonth: 3,
          customerId: 'CUST-1',
          totalSalesAmount: '25000000.00',
          orderCount: 5,
          refreshedAt: '2026-03-15T12:00:00Z',
        },
      ],
      [
        {
          id: 'i1',
          organizationId: TEST_ORG.id,
          warehouseCode: 'WH-JKT',
          itemCode: 'ITEM-01',
          onHandQty: '100.00',
          availableQty: '85.50',
          refreshedAt: '2026-03-15T12:00:00Z',
        },
      ]
    );
    expect(opsSummary.totalSalesAmount).toBe('25000000.00');
    expect(opsSummary.totalReservedQty).toBe('14.50');

    const tree = buildAccountHierarchyRows([
      {
        id: 'leaf',
        chartOfAccountsId: 'coa-1',
        code: '1101',
        name: 'Kas Besar',
        accountType: 'Asset',
        parentId: 'root',
        isControl: false,
        isMonetary: true,
        allowPosting: true,
        rowVersion: 1,
      },
      {
        id: 'root',
        chartOfAccountsId: 'coa-1',
        code: '1000',
        name: 'Aset',
        accountType: 'Asset',
        parentId: null,
        isControl: true,
        isMonetary: false,
        allowPosting: false,
        rowVersion: 1,
      },
    ]);
    expect(tree.map((r) => [r.code, r.depth])).toEqual([
      ['1000', 0],
      ['1101', 1],
    ]);
  });
});

describe('DashboardWorkspace (/)', () => {
  it('renders live BI KPIs, pending workflow approvals, chart summary table toggle, and drills down without fake deltas', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/gl-balance-snapshots')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'snap-1',
              fiscalPeriodId: 'per-1',
              accountCode: '1101',
              closingBalance: '150000000.00',
              refreshedAt: '2026-03-15T08:00:00Z',
            },
            {
              id: 'snap-2',
              fiscalPeriodId: 'per-1',
              accountCode: '2101',
              closingBalance: '40000000.00',
              refreshedAt: '2026-03-15T08:00:00Z',
            },
            {
              id: 'snap-3',
              fiscalPeriodId: 'per-1',
              accountCode: '3101',
              closingBalance: '110000000.00',
              refreshedAt: '2026-03-15T08:00:00Z',
            },
          ]),
        });
      }
      if (url.includes('/sales-summaries')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'sale-1',
              organizationId: TEST_ORG.id,
              periodYear: 2026,
              periodMonth: 3,
              customerId: 'CUST-01',
              totalSalesAmount: '25500000.00',
              orderCount: 12,
              refreshedAt: '2026-03-15T08:30:00Z',
            },
          ]),
        });
      }
      if (url.includes('/inventory-positions')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'inv-1',
              organizationId: TEST_ORG.id,
              warehouseCode: 'WH-JKT',
              itemCode: 'SKU-A',
              onHandQty: '500.00',
              availableQty: '450.00',
              refreshedAt: '2026-03-15T08:30:00Z',
            },
          ]),
        });
      }
      if (url.includes('/journal-entries')) {
        return jsonResponse({
          body: listEnvelope(
            [
              {
                id: 'j1',
                journalNo: 'JRN-100',
                postingDate: '2026-03-15',
                currencyCode: 'IDR',
                totalDebit: '5000.00',
                status: 'Draft',
              },
            ],
            3
          ),
        });
      }
      if (url.includes('/workflow-tasks')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'wf-1',
              organizationId: TEST_ORG.id,
              instanceId: 'inst-1',
              assigneeId: 'finance.director',
              documentRef: 'AP-INV-2026-088',
              instanceStatus: 'PENDING',
              createdBy: 'ap.clerk',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/currencies')) {
        return jsonResponse({
          body: listEnvelope([
            { id: 'c1', code: 'IDR', name: 'Rupiah', precision: 2, isActive: true },
            { id: 'c2', code: 'USD', name: 'US Dollar', precision: 2, isActive: true },
          ]),
        });
      }
      return jsonResponse({ body: listEnvelope([]) });
    });
    vi.stubGlobal('fetch', fetchMock);

    const user = userEvent.setup();
    renderWithProviders(<DashboardWorkspace />);

    await waitFor(() => {
      expect(screen.getByText(/150\.000\.000,00/)).toBeInTheDocument();
    });

    expect(screen.getByText(/25\.500\.000,00/)).toBeInTheDocument();
    expect(screen.getByText(/450,00 Unit/)).toBeInTheDocument();
    expect(screen.getByText('AP-INV-2026-088')).toBeInTheDocument();
    expect(screen.getByText('A = L + E · Valid')).toBeInTheDocument();
    expect(screen.getByText('Aksi Cepat & Pintasan Modul')).toBeInTheDocument();
    expect(screen.getByText('Status Sinkronisasi & BI')).toBeInTheDocument();
    expect(screen.queryByText('+4.2%')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Ringkasan Tabel/i }));
    expect(
      screen.getByRole('table', { name: /Tabel Ringkasan Saldo GL per Kategori/i })
    ).toBeInTheDocument();

    await user.click(screen.getByText(/Total Aset Buku Besar/i));
    expect(mockNavigate).toHaveBeenCalledWith({ to: '/journals' });
  });

  it('shows explicit ErrorState instead of FALLBACK_KPIS when reporting endpoints fail', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/gl-balance-snapshots')) {
        return jsonResponse({
          status: 500,
          body: {
            error: {
              code: 'REPORTING_UNAVAILABLE',
              message: 'BigQuery snapshot store offline',
            },
          },
        });
      }
      return jsonResponse({ body: listEnvelope([]) });
    });
    vi.stubGlobal('fetch', fetchMock);

    renderWithProviders(<DashboardWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('Gagal Memuat Snapshot BI Dashboard')).toBeInTheDocument();
    });
    expect(screen.getByText('BigQuery snapshot store offline')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Coba Lagi/i })).toBeInTheDocument();
  });
});

describe('JournalsWorkspace (/journals)', () => {
  const ledgersFixture = [
    {
      id: 'led-primary-1',
      financialOrganizationId: 'fin-org-1',
      code: 'LEDGER-MAIN-IDR',
      ledgerType: 'Primary',
      currencyCode: 'IDR',
      rowVersion: 1,
    },
  ];

  const periodsFixture = [
    {
      id: 'period-mar-26',
      fiscalYearId: 'fy-2026',
      periodNo: 3,
      startDate: '2026-03-01',
      endDate: '2026-03-31',
      status: 'Open',
      rowVersion: 1,
    },
  ];

  const accountsFixture = [
    {
      id: 'acc-cash',
      chartOfAccountsId: 'coa-1',
      code: '1101',
      name: 'Kas Besar',
      accountType: 'Asset',
      allowPosting: true,
      isControl: false,
      isMonetary: true,
      rowVersion: 1,
    },
    {
      id: 'acc-expense',
      chartOfAccountsId: 'coa-1',
      code: '6101',
      name: 'Beban Operasional',
      accountType: 'Expense',
      allowPosting: true,
      isControl: false,
      isMonetary: false,
      rowVersion: 1,
    },
    {
      id: 'acc-rev',
      chartOfAccountsId: 'coa-1',
      code: '4101',
      name: 'Pendapatan Jasa',
      accountType: 'Revenue',
      allowPosting: true,
      isControl: false,
      isMonetary: false,
      rowVersion: 1,
    },
  ];

  const entriesFixture = [
    {
      id: 'jrn-draft-1',
      journalNo: 'JRN-202603-1001',
      financialOrganizationId: 'fin-org-1',
      ledgerId: 'led-primary-1',
      fiscalPeriodId: 'period-mar-26',
      postingDate: '2026-03-15',
      journalSource: 'Manual',
      currencyCode: 'IDR',
      exchangeRate: '1.0000',
      totalDebit: '500000.00',
      totalCredit: '500000.00',
      sourceDocumentRef: 'INV-001',
      status: 'Draft',
      rowVersion: 3,
      lines: [
        {
          id: 'l1',
          lineNo: 1,
          accountId: 'acc-cash',
          debitAmount: '500000.00',
          creditAmount: '0.00',
          functionalAmount: '500000.0000',
          description: 'Kas masuk',
        },
        {
          id: 'l2',
          lineNo: 2,
          accountId: 'acc-rev',
          debitAmount: '0.00',
          creditAmount: '500000.00',
          functionalAmount: '500000.0000',
          description: 'Pendapatan',
        },
      ],
    },
    {
      id: 'jrn-posted-1',
      journalNo: 'JRN-202603-0999',
      financialOrganizationId: 'fin-org-1',
      ledgerId: 'led-primary-1',
      fiscalPeriodId: 'period-mar-26',
      postingDate: '2026-03-14',
      journalSource: 'Manual',
      currencyCode: 'IDR',
      exchangeRate: '1.0000',
      totalDebit: '250000.00',
      totalCredit: '250000.00',
      sourceDocumentRef: 'EXP-099',
      status: 'Posted',
      rowVersion: 5,
      lines: [
        {
          id: 'l3',
          lineNo: 1,
          accountId: 'acc-expense',
          debitAmount: '250000.00',
          creditAmount: '0.00',
          functionalAmount: '250000.0000',
        },
        {
          id: 'l4',
          lineNo: 2,
          accountId: 'acc-cash',
          debitAmount: '0.00',
          creditAmount: '250000.00',
          functionalAmount: '250000.0000',
        },
      ],
    },
  ];

  function createJournalsFetchMock(options?: { failCreateWithFieldErrors?: boolean }) {
    return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = (init?.method || 'GET').toUpperCase();

      if (url.includes('/ledgers')) return jsonResponse({ body: listEnvelope(ledgersFixture) });
      if (url.includes('/fiscal-periods')) {
        return jsonResponse({ body: listEnvelope(periodsFixture) });
      }
      if (url.includes('/accounts')) return jsonResponse({ body: listEnvelope(accountsFixture) });
      if (url.includes('/currencies')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'cur-idr',
              code: 'IDR',
              name: 'Rupiah',
              precision: 2,
              isActive: true,
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/trial-balance')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'tb-1',
              ledgerId: 'led-primary-1',
              accountId: 'acc-cash',
              fiscalPeriodId: 'period-mar-26',
              openingBalance: '1000000.00',
              periodDebit: '500000.00',
              periodCredit: '500000.00',
              closingBalance: '1000000.00',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.endsWith('/journal-entries/jrn-draft-1/post') && method === 'POST') {
        return jsonResponse({
          body: { ...entriesFixture[0], status: 'Posted', rowVersion: 4 },
        });
      }
      if (url.endsWith('/journal-entries/jrn-posted-1/reverse') && method === 'POST') {
        return jsonResponse({
          body: {
            ...entriesFixture[1],
            id: 'jrn-rev-1',
            journalNo: 'JRN-202603-REV1',
            status: 'Posted',
            rowVersion: 1,
          },
        });
      }
      if (url.endsWith('/journal-entries/jrn-draft-1') && method === 'PUT') {
        return jsonResponse({
          body: { ...entriesFixture[0], sourceDocumentRef: 'INV-001-REV', rowVersion: 4 },
        });
      }
      if (url.endsWith('/journal-entries/jrn-draft-1') && method === 'DELETE') {
        return jsonResponse({ status: 204 });
      }
      if (url.endsWith('/journal-entries/jrn-draft-1') && method === 'GET') {
        return jsonResponse({ body: entriesFixture[0] });
      }
      if (url.endsWith('/journal-entries/jrn-posted-1') && method === 'GET') {
        return jsonResponse({ body: entriesFixture[1] });
      }
      if (url.includes('/journal-entries') && method === 'POST') {
        if (options?.failCreateWithFieldErrors) {
          return jsonResponse({
            status: 422,
            body: {
              error: {
                code: 'VALIDATION_ERROR',
                message: 'Validasi jurnal ditolak server',
                details: [
                  {
                    field: 'journalNo',
                    issue: 'Nomor jurnal sudah terdaftar',
                  },
                ],
              },
            },
          });
        }
        const parsed = JSON.parse(String(init?.body || '{}')) as { journalNo: string };
        return jsonResponse({
          status: 201,
          body: { ...entriesFixture[0], id: 'jrn-created', journalNo: parsed.journalNo },
        });
      }
      if (url.includes('/journal-entries')) {
        return jsonResponse({ body: listEnvelope(entriesFixture, 2) });
      }
      return jsonResponse({ body: listEnvelope([]) });
    });
  }

  it('creates balanced 3-line journal using live lookups and surfaces Trial Balance tab', async () => {
    const fetchMock = createJournalsFetchMock();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderWithProviders(<JournalsWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('JRN-202603-1001')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: /Buat Jurnal Baru/i }));
    const dialog = await screen.findByRole('dialog');

    // Add 3rd dynamic line
    await user.click(within(dialog).getByRole('button', { name: /Tambah Baris/i }));

    const line0Combo = dialog.querySelector('#journal-line-account-0') as HTMLElement;
    const line1Combo = dialog.querySelector('#journal-line-account-1') as HTMLElement;
    const line2Combo = dialog.querySelector('#journal-line-account-2') as HTMLElement;
    expect(line0Combo).toBeTruthy();
    expect(line1Combo).toBeTruthy();
    expect(line2Combo).toBeTruthy();

    await user.click(line0Combo);
    await user.click(await screen.findByRole('option', { name: /1101/i }));

    await user.click(line1Combo);
    await user.click(await screen.findByRole('option', { name: /6101/i }));

    await user.click(line2Combo);
    await user.click(await screen.findByRole('option', { name: /4101/i }));

    fireEvent.change(within(dialog).getByLabelText('Debit Baris 1'), { target: { value: '100.00' } });
    fireEvent.change(within(dialog).getByLabelText('Debit Baris 2'), { target: { value: '50.00' } });
    fireEvent.change(within(dialog).getByLabelText('Kredit Baris 3'), {
      target: { value: '120.00' },
    });

    const submitBtn = within(dialog).getByRole('button', { name: /Simpan sebagai Draf/i });
    expect(submitBtn).toBeDisabled();
    expect(within(dialog).getByText('ADR-014 Decimal')).toBeInTheDocument();

    fireEvent.change(within(dialog).getByLabelText('Kredit Baris 3'), {
      target: { value: '150.00' },
    });
    await waitFor(() => {
      expect(submitBtn).toBeEnabled();
      expect(within(dialog).getByText('Fixed-Point Verified')).toBeInTheDocument();
    });
    await user.click(submitBtn);

    await waitFor(() => {
      const postCall = fetchMock.mock.calls.find(
        ([u, init]) => String(u).endsWith('/journal-entries') && init?.method === 'POST'
      );
      expect(postCall).toBeDefined();
      const [, init] = postCall!;
      const headers = new Headers(init?.headers);
      expect(headers.get('Idempotency-Key')).toBeTruthy();
      const body = JSON.parse(String(init?.body)) as {
        ledgerId: string;
        fiscalPeriodId: string;
        lines: Array<{ functionalAmount: string }>;
      };
      expect(body.ledgerId).toBe('led-primary-1');
      expect(body.fiscalPeriodId).toBe('period-mar-26');
      expect(body.lines).toHaveLength(3);
      expect(body.lines[0].functionalAmount).toBe('100.0000');
    });

    await user.click(screen.getByRole('tab', { name: /Neraca Saldo/i }));
    await waitFor(() => {
      expect(screen.getByText('Kas Besar')).toBeInTheDocument();
    });
  });

  it('executes Draft Edit (PUT), Delete (DELETE), Post (POST), and Posted Reverse with If-Match headers', async () => {
    const fetchMock = createJournalsFetchMock();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderWithProviders(<JournalsWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('JRN-202603-1001')).toBeInTheDocument();
    });

    // Open Draft detail -> Edit Draft -> PUT with If-Match: "3"
    await user.click(screen.getByText('JRN-202603-1001'));
    await user.click(await screen.findByRole('button', { name: /Ubah Draf/i }));
    const editDialog = await screen.findByRole('dialog');
    const memoInput = within(editDialog).getByLabelText(/Referensi Dokumen Sumber/i);
    await user.clear(memoInput);
    await user.type(memoInput, 'INV-001-REV');
    await user.click(within(editDialog).getByRole('button', { name: /Simpan Perubahan/i }));

    await waitFor(() => {
      const putCall = fetchMock.mock.calls.find(
        ([u, init]) => String(u).endsWith('/journal-entries/jrn-draft-1') && init?.method === 'PUT'
      );
      expect(putCall).toBeDefined();
      expect(new Headers(putCall![1]?.headers).get('If-Match')).toBe('"3"');
    });

    // Re-open Draft detail -> verify Tri-Currency ADR-014 summary & Post with If-Match: "3"
    await user.click(screen.getByText('JRN-202603-1001'));
    expect(await screen.findByText('Kepatuhan Tri-Currency (ADR-014)')).toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: /Posting ke GL/i }));
    await waitFor(() => {
      const postCall = fetchMock.mock.calls.find(([u]) =>
        String(u).endsWith('/journal-entries/jrn-draft-1/post')
      );
      expect(postCall).toBeDefined();
      expect(new Headers(postCall![1]?.headers).get('If-Match')).toBe('"3"');
    });

    // Delete Draft flow inside drawer
    await user.click(screen.getByRole('button', { name: /Hapus Draf/i }));
    await user.click(await screen.findByRole('button', { name: /Ya, Hapus Draf/i }));
    await waitFor(() => {
      const delCall = fetchMock.mock.calls.find(
        ([u, init]) =>
          String(u).endsWith('/journal-entries/jrn-draft-1') && init?.method === 'DELETE'
      );
      expect(delCall).toBeDefined();
      expect(new Headers(delCall![1]?.headers).get('If-Match')).toBe('"3"');
    });

    // Open Posted row -> Reverse with reason & If-Match: "5"
    await user.click(screen.getByText('JRN-202603-0999'));
    await user.click(await screen.findByRole('button', { name: /Balik Jurnal/i }));
    await user.type(screen.getByLabelText(/Alasan pembalikan jurnal/i), 'Salah periode');
    await user.click(screen.getByRole('button', { name: /Konfirmasi Balik Jurnal/i }));

    await waitFor(() => {
      const revCall = fetchMock.mock.calls.find(([u]) =>
        String(u).endsWith('/journal-entries/jrn-posted-1/reverse')
      );
      expect(revCall).toBeDefined();
      expect(new Headers(revCall![1]?.headers).get('If-Match')).toBe('"5"');
      expect(String(revCall![1]?.body)).toContain('Salah periode');
    });
  });

  it('enforces RBAC for read-only users and surfaces API fieldErrors via extractApiFieldErrors', async () => {
    const fetchMock = createJournalsFetchMock({ failCreateWithFieldErrors: true });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    const { unmount } = renderWithProviders(<JournalsWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('JRN-202603-1001')).toBeInTheDocument();
    });

    // Trigger 422 API field error surfacing on Create
    await user.click(screen.getByRole('button', { name: /Buat Jurnal Baru/i }));
    const dialog = await screen.findByRole('dialog');
    const line0Combo = dialog.querySelector('#journal-line-account-0') as HTMLElement;
    const line1Combo = dialog.querySelector('#journal-line-account-1') as HTMLElement;
    await user.click(line0Combo);
    await user.click(await screen.findByRole('option', { name: /1101/i }));
    await user.click(line1Combo);
    await user.click(await screen.findByRole('option', { name: /4101/i }));
    fireEvent.change(within(dialog).getByLabelText('Debit Baris 1'), { target: { value: '100.00' } });
    fireEvent.change(within(dialog).getByLabelText('Kredit Baris 2'), {
      target: { value: '100.00' },
    });
    const submitBtn = within(dialog).getByRole('button', { name: /Simpan sebagai Draf/i });
    await waitFor(() => expect(submitBtn).toBeEnabled());
    await user.click(submitBtn);

    await waitFor(() => {
      expect(screen.getByText('Nomor jurnal sudah terdaftar')).toBeInTheDocument();
    });

    unmount();

    // Re-render as read-only user -> no create/post/delete actions
    useAuthStore.setState({ user: READONLY_USER });
    renderWithProviders(<JournalsWorkspace />);
    await waitFor(() => {
      expect(screen.getByText('JRN-202603-1001')).toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: /Buat Jurnal Baru/i })).not.toBeInTheDocument();
    await user.click(screen.getByText('JRN-202603-1001'));
    await screen.findByText('Kas masuk');
    expect(screen.queryByRole('button', { name: /Posting ke GL/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Hapus Draf/i })).not.toBeInTheDocument();
  });
});

describe('CoaWorkspace (/coa)', () => {
  function createCoaFetchMock() {
    return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = (init?.method || 'GET').toUpperCase();

      if (url.includes('/charts-of-accounts')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'coa-1',
              code: 'COA-ID',
              name: 'PSAK Standard',
              structure: '4-Digit',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/accounts') && method === 'POST') {
        const body = JSON.parse(String(init?.body)) as { code: string; name: string };
        return jsonResponse({
          status: 201,
          body: {
            id: 'acc-new',
            chartOfAccountsId: 'coa-1',
            code: body.code,
            name: body.name,
            accountType: 'Asset',
            isControl: false,
            isMonetary: true,
            allowPosting: true,
            rowVersion: 1,
          },
        });
      }
      if (url.includes('/accounts')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'acc-1000',
              chartOfAccountsId: 'coa-1',
              code: '1000',
              name: 'Aset',
              accountType: 'Asset',
              isControl: true,
              isMonetary: false,
              allowPosting: false,
              rowVersion: 1,
            },
            {
              id: 'acc-1101',
              chartOfAccountsId: 'coa-1',
              code: '1101',
              name: 'Kas Besar',
              accountType: 'Asset',
              parentId: 'acc-1000',
              isControl: false,
              isMonetary: true,
              allowPosting: true,
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/financial-organizations')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'fin-1',
              code: 'FIN-HLD',
              name: 'PT Sinergi Finance',
              organizationRefId: TEST_ORG.id,
              functionalCurrencyCode: 'IDR',
              chartOfAccountsId: 'coa-1',
              fiscalCalendarId: 'cal-1',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/ledgers')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'led-1',
              financialOrganizationId: 'fin-1',
              code: 'LEDGER-MAIN',
              ledgerType: 'Primary',
              currencyCode: 'IDR',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/fiscal-calendars')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'cal-1',
              code: 'CAL-STD',
              name: 'Kalender Jan-Des',
              periodType: 'Monthly',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.includes('/fiscal-years')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'fy-2026',
              fiscalCalendarId: 'cal-1',
              yearNo: 2026,
              name: 'FY 2026',
              startDate: '2026-01-01',
              endDate: '2026-12-31',
              status: 'Open',
              rowVersion: 1,
            },
          ]),
        });
      }
      if (url.endsWith('/fiscal-periods/per-1/close') && method === 'POST') {
        return jsonResponse({
          body: {
            id: 'per-1',
            fiscalYearId: 'fy-2026',
            periodNo: 1,
            startDate: '2026-01-01',
            endDate: '2026-01-31',
            status: 'Closed',
            rowVersion: 2,
          },
        });
      }
      if (url.endsWith('/fiscal-periods/per-2/reopen') && method === 'POST') {
        return jsonResponse({
          body: {
            id: 'per-2',
            fiscalYearId: 'fy-2026',
            periodNo: 2,
            startDate: '2026-02-01',
            endDate: '2026-02-28',
            status: 'Open',
            rowVersion: 3,
          },
        });
      }
      if (url.endsWith('/fiscal-periods/per-2/lock') && method === 'POST') {
        return jsonResponse({
          body: {
            id: 'per-2',
            fiscalYearId: 'fy-2026',
            periodNo: 2,
            startDate: '2026-02-01',
            endDate: '2026-02-28',
            status: 'Locked',
            rowVersion: 3,
          },
        });
      }
      if (url.includes('/fiscal-periods')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'per-1',
              fiscalYearId: 'fy-2026',
              periodNo: 1,
              startDate: '2026-01-01',
              endDate: '2026-01-31',
              status: 'Open',
              rowVersion: 1,
            },
            {
              id: 'per-2',
              fiscalYearId: 'fy-2026',
              periodNo: 2,
              startDate: '2026-02-01',
              endDate: '2026-02-28',
              status: 'Closed',
              rowVersion: 2,
            },
          ]),
        });
      }
      return jsonResponse({ body: listEnvelope([]) });
    });
  }

  it('renders account hierarchy, creates GL account, and runs fiscal period Close, Reopen & Lock with If-Match', async () => {
    const fetchMock = createCoaFetchMock();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    renderWithProviders(<CoaWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('Kas Besar')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: /Tambah Akun GL/i }));
    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByPlaceholderText('1101'), '1105');
    await user.type(within(dialog).getByPlaceholderText('Kas & Setara Kas'), 'Bank Mandiri');
    await user.click(within(dialog).getByRole('button', { name: /Simpan Akun/i }));

    await waitFor(() => {
      const postAcc = fetchMock.mock.calls.find(
        ([u, init]) => String(u).endsWith('/accounts') && init?.method === 'POST'
      );
      expect(postAcc).toBeDefined();
      expect(String(postAcc![1]?.body)).toContain('1105');
    });

    await user.click(screen.getByRole('tab', { name: /Ledger & Entitas Keuangan/i }));
    await waitFor(() => {
      expect(screen.getByText('LEDGER-MAIN')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('tab', { name: /Kalender, Tahun & Periode Fiskal/i }));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Tutup \(Close\)/i })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: /Tutup \(Close\)/i }));
    await waitFor(() => {
      const closeCall = fetchMock.mock.calls.find(([u]) =>
        String(u).endsWith('/fiscal-periods/per-1/close')
      );
      expect(closeCall).toBeDefined();
      expect(new Headers(closeCall![1]?.headers).get('If-Match')).toBe('"1"');
    });

    await user.click(screen.getByRole('button', { name: /Buka Ulang/i }));
    await waitFor(() => {
      const reopenCall = fetchMock.mock.calls.find(([u]) =>
        String(u).endsWith('/fiscal-periods/per-2/reopen')
      );
      expect(reopenCall).toBeDefined();
      expect(new Headers(reopenCall![1]?.headers).get('If-Match')).toBe('"2"');
    });

    await user.click(screen.getByRole('button', { name: 'Kunci' }));
    await user.click(await screen.findByRole('button', { name: /Ya, Kunci Permanen/i }));
    await waitFor(() => {
      const lockCall = fetchMock.mock.calls.find(([u]) =>
        String(u).endsWith('/fiscal-periods/per-2/lock')
      );
      expect(lockCall).toBeDefined();
      expect(new Headers(lockCall![1]?.headers).get('If-Match')).toBe('"2"');
    });
  });

  it('hides CoA and Fiscal Period mutation buttons when coa:write is absent', async () => {
    useAuthStore.setState({ user: READONLY_USER });
    vi.stubGlobal('fetch', createCoaFetchMock());
    const user = userEvent.setup();

    renderWithProviders(<CoaWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('Kas Besar')).toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: /Tambah Akun GL/i })).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /Kalender, Tahun & Periode Fiskal/i }));
    await waitFor(() => {
      expect(screen.getByText('#1')).toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: /Tutup \(Close\)/i })).not.toBeInTheDocument();
  });
});

describe('ReportsWorkspace (/reports)', () => {
  it('renders GL Snapshots, Sales Summaries, Inventory Positions, Trial Balance, and invokes exports', async () => {
    const csvSpy = vi.spyOn(exportLib, 'exportToCsv').mockImplementation(() => {});
    const jsonSpy = vi.spyOn(exportLib, 'exportToJson').mockImplementation(() => {});
    const printSpy = vi.spyOn(exportLib, 'triggerPrint').mockImplementation(() => {});

    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/gl-balance-snapshots')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'snap-1',
              fiscalPeriodId: 'per-1',
              accountCode: '1101',
              closingBalance: '80000000.00',
              refreshedAt: '2026-03-15T09:00:00Z',
            },
          ]),
        });
      }
      if (url.includes('/sales-summaries')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'sale-1',
              organizationId: TEST_ORG.id,
              periodYear: 2026,
              periodMonth: 3,
              customerId: 'CUST-88',
              totalSalesAmount: '12000000.00',
              orderCount: 4,
              refreshedAt: '2026-03-15T09:00:00Z',
            },
          ]),
        });
      }
      if (url.includes('/inventory-positions')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'inv-1',
              organizationId: TEST_ORG.id,
              warehouseCode: 'WH-SBY',
              itemCode: 'MAT-007',
              onHandQty: '200.00',
              availableQty: '175.00',
              refreshedAt: '2026-03-15T09:00:00Z',
            },
          ]),
        });
      }
      if (url.includes('/ledgers')) {
        return jsonResponse({
          body: listEnvelope([
            { id: 'led-1', code: 'MAIN-IDR', ledgerType: 'Primary', currencyCode: 'IDR' },
          ]),
        });
      }
      if (url.includes('/fiscal-periods')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'per-1',
              periodNo: 3,
              startDate: '2026-03-01',
              endDate: '2026-03-31',
              status: 'Open',
            },
          ]),
        });
      }
      if (url.includes('/accounts')) {
        return jsonResponse({
          body: listEnvelope([
            { id: 'acc-1', code: '1101', name: 'Kas Besar', accountType: 'Asset' },
          ]),
        });
      }
      if (url.includes('/trial-balance')) {
        return jsonResponse({
          body: listEnvelope([
            {
              id: 'tb-1',
              ledgerId: 'led-1',
              accountId: 'acc-1',
              fiscalPeriodId: 'per-1',
              openingBalance: '1000.00',
              periodDebit: '500.00',
              periodCredit: '500.00',
              closingBalance: '1000.00',
              rowVersion: 1,
            },
          ]),
        });
      }
      return jsonResponse({ body: listEnvelope([]) });
    });
    vi.stubGlobal('fetch', fetchMock);

    const user = userEvent.setup();
    renderWithProviders(<ReportsWorkspace />);

    await waitFor(() => {
      expect(screen.getByText('1101')).toBeInTheDocument();
    });

    await user.click(screen.getAllByRole('button', { name: /Ekspor CSV/i })[0]);
    await user.click(screen.getByRole('button', { name: /Ekspor JSON/i }));
    await user.click(screen.getByRole('button', { name: /Cetak/i }));
    expect(csvSpy).toHaveBeenCalledTimes(1);
    expect(jsonSpy).toHaveBeenCalledTimes(1);
    expect(printSpy).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('tab', { name: /Ringkasan Penjualan/i }));
    await waitFor(() => {
      expect(screen.getByText('CUST-88')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('tab', { name: /Posisi Persediaan/i }));
    await waitFor(() => {
      expect(screen.getByText('MAT-007')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('tab', { name: /Neraca Saldo/i }));
    await waitFor(() => {
      expect(screen.getByText('Kas Besar')).toBeInTheDocument();
    });

    csvSpy.mockRestore();
    jsonSpy.mockRestore();
    printSpy.mockRestore();
  });

  it('displays ErrorState when reporting API returns 503', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        jsonResponse({
          status: 503,
          body: { error: { code: 'BI_DOWN', message: 'Dataform offline' } },
        })
      )
    );

    renderWithProviders(<ReportsWorkspace />);

    await waitFor(() => {
      expect(screen.getAllByText('Dataform offline').length).toBeGreaterThanOrEqual(1);
    });
  });
});
