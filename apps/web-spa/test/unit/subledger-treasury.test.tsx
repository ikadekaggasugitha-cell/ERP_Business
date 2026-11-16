import '../../src/lib/i18n';
import type { ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApWorkspace } from '../../src/features/ap/ApWorkspace';
import { ArWorkspace } from '../../src/features/ar/ArWorkspace';
import { CurrencyWorkspace } from '../../src/features/currency/CurrencyWorkspace';
import { TaxWorkspace } from '../../src/features/tax/TaxWorkspace';
import { TreasuryWorkspace } from '../../src/features/treasury/TreasuryWorkspace';
import { ApiClientError } from '../../src/lib/api';
import { DEFAULT_HOLDING_ORG, useAuthStore } from '../../src/store/auth-store';
import type { User } from '../../src/types';

const apiClientMock = vi.fn();

vi.mock('../../src/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../src/lib/api')>();
  return {
    ...actual,
    apiClient: (...args: unknown[]) => apiClientMock(...args),
  };
});

const FULL_USER: User = {
  id: '00000000-0000-0000-0000-000000000099',
  email: 'finance.controller@sinergi.co.id',
  name: 'Finance Controller',
  role: 'FINANCE_CONTROLLER',
  permissions: [
    'ap:read',
    'ap:write',
    'ar:read',
    'ar:write',
    'tax:read',
    'tax:write',
    'treasury:read',
    'treasury:write',
    'currency:read',
    'currency:write',
    'masterdata:read',
    'coa:read',
  ],
  organizationIds: [DEFAULT_HOLDING_ORG.id],
};

const READONLY_USER: User = {
  ...FULL_USER,
  id: '00000000-0000-0000-0000-000000000098',
  role: 'AUDITOR',
  permissions: ['ap:read', 'ar:read', 'tax:read', 'treasury:read', 'currency:read'],
};

const VENDOR_ID = '11111111-1111-4111-8111-111111111111';
const CUSTOMER_ID = '22222222-2222-4222-8222-222222222222';
const GL_EXPENSE_ID = '33333333-3333-4333-8333-333333333333';
const GL_CASH_ID = '44444444-4444-4444-8444-444444444444';
const TAX_CODE_ID = '55555555-5555-4555-8555-555555555555';
const BANK_ACCOUNT_ID = '66666666-6666-4666-8666-666666666666';

function createWrapper() {
  const client = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

function emptyPage<T>(items: T[]) {
  return {
    data: items,
    page: {
      number: 1,
      size: 25,
      totalElements: items.length,
      totalPages: 1,
    },
  };
}

beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
    configurable: true,
    value: 600,
  });
  Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
    configurable: true,
    value: 1000,
  });
  HTMLElement.prototype.getBoundingClientRect = () =>
    ({
      width: 1000,
      height: 600,
      top: 0,
      left: 0,
      bottom: 600,
      right: 1000,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }) as DOMRect;

  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  }
});

beforeEach(() => {
  apiClientMock.mockReset();
  useAuthStore.getState().setAuth(FULL_USER, 'test-token', DEFAULT_HOLDING_ORG);
});

describe('Wave 2B: AP Workspace', () => {
  it('surfaces explicit ErrorState with Correlation ID (zero silent mocks) and recovers on retry', async () => {
    let failFirst = true;
    apiClientMock.mockImplementation(async (endpoint: string) => {
      if (endpoint === '/vendor-invoices') {
        if (failFirst) {
          throw new ApiClientError(
            'SERVICE_UNAVAILABLE',
            'AP backend unreachable',
            503,
            'corr-ap-503',
          );
        }
        return emptyPage([]);
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <ApWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText(/AP backend unreachable/i)).toBeInTheDocument();
    expect(screen.getByText(/corr-ap-503/i)).toBeInTheDocument();
    expect(screen.queryByText(/AP_MOCK/i)).not.toBeInTheDocument();

    failFirst = false;
    await userEvent.click(screen.getByRole('button', { name: /retry|coba lagi/i }));
    expect(await screen.findByText(/No vendor invoices found/i)).toBeInTheDocument();
  });

  it('enriches vendor display from /parties, opens drawer, and records Draft bill with If-Match + Idempotency-Key', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(
      async (endpoint: string, options?: { method?: string; ifMatch?: string; idempotencyKey?: string }) => {
        if (endpoint === '/parties') {
          return emptyPage([
            {
              id: VENDOR_ID,
              organizationId: DEFAULT_HOLDING_ORG.id,
              code: 'VND-001',
              name: 'PT Supplier Prima',
              partyType: 'vendor',
              taxId: '01.234.567.8-901.000',
              isShared: false,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/accounts') {
          return emptyPage([
            {
              id: GL_EXPENSE_ID,
              chartOfAccountsId: 'coa-1',
              code: '5101',
              name: 'Cloud Hosting Expense',
              accountType: 'EXPENSE',
              allowPosting: true,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/currencies') {
          return emptyPage([{ id: 'cur-1', code: 'IDR', name: 'Indonesian Rupiah', precision: 2, isActive: true }]);
        }
        if (endpoint === '/vendor-invoices') {
          return emptyPage([
            {
              id: 'ap-inv-1',
              invoiceNo: 'BILL-2026-0001',
              vendorId: VENDOR_ID,
              invoiceDate: '2026-11-01',
              dueDate: '2026-12-01',
              currencyCode: 'IDR',
              totalAmount: '2500000.0000',
              outstandingAmount: '2500000.0000',
              status: 'Draft',
              rowVersion: 3,
            },
          ]);
        }
        if (endpoint === '/vendor-invoices/ap-inv-1' && (!options?.method || options.method === 'GET')) {
          return {
            id: 'ap-inv-1',
            invoiceNo: 'BILL-2026-0001',
            vendorId: VENDOR_ID,
            invoiceDate: '2026-11-01',
            dueDate: '2026-12-01',
            currencyCode: 'IDR',
            totalAmount: '2500000.0000',
            outstandingAmount: '2500000.0000',
            status: 'Draft',
            rowVersion: 3,
            lines: [
              {
                id: 'line-1',
                lineNo: 1,
                accountId: GL_EXPENSE_ID,
                amount: '2500000.0000',
                description: 'November Kubernetes cluster',
              },
            ],
          };
        }
        if (endpoint === '/vendor-invoices/ap-inv-1/record' && options?.method === 'POST') {
          return {
            id: 'ap-inv-1',
            invoiceNo: 'BILL-2026-0001',
            vendorId: VENDOR_ID,
            invoiceDate: '2026-11-01',
            dueDate: '2026-12-01',
            currencyCode: 'IDR',
            totalAmount: '2500000.0000',
            outstandingAmount: '2500000.0000',
            status: 'Recorded',
            rowVersion: 4,
          };
        }
        return emptyPage([]);
      },
    );

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <ApWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('BILL-2026-0001')).toBeInTheDocument();
    expect(await screen.findByText('PT Supplier Prima')).toBeInTheDocument();
    expect(screen.getByText('SVC-003 · AP')).toBeInTheDocument();

    await user.click(screen.getByText('BILL-2026-0001'));

    expect(await screen.findByText('November Kubernetes cluster')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Edit Draft/i })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Record \/ Post/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/vendor-invoices/ap-inv-1/record',
        expect.objectContaining({
          method: 'POST',
          ifMatch: '3',
          idempotencyKey: expect.any(String),
        }),
      );
    });
  });

  it('hides AP write actions when user lacks ap:write permission', async () => {
    useAuthStore.getState().setAuth(READONLY_USER, 'test-token', DEFAULT_HOLDING_ORG);
    apiClientMock.mockImplementation(async (endpoint: string) => {
      if (endpoint === '/vendor-invoices') {
        return emptyPage([
          {
            id: 'ap-inv-2',
            invoiceNo: 'BILL-2026-0002',
            vendorId: VENDOR_ID,
            invoiceDate: '2026-11-01',
            dueDate: '2026-12-01',
            currencyCode: 'IDR',
            totalAmount: '100000.0000',
            outstandingAmount: '100000.0000',
            status: 'Draft',
            rowVersion: 1,
          },
        ]);
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <ApWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('BILL-2026-0002')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /New Vendor Invoice/i })).not.toBeInTheDocument();

    await userEvent.click(screen.getByText('BILL-2026-0002'));
    expect(screen.queryByRole('button', { name: /Record \/ Post/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Edit Draft/i })).not.toBeInTheDocument();
  });
});

describe('Wave 2B: AR Workspace', () => {
  it('supports Customer lookup enrichment and Mark Paid (POST /pay) with If-Match without Edit action', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(
      async (endpoint: string, options?: { method?: string }) => {
        if (endpoint === '/parties') {
          return emptyPage([
            {
              id: CUSTOMER_ID,
              organizationId: DEFAULT_HOLDING_ORG.id,
              code: 'CUST-009',
              name: 'PT Nusantara Retail',
              partyType: 'customer',
              isShared: false,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/customer-invoices') {
          return emptyPage([
            {
              id: 'ar-inv-1',
              invoiceNo: 'INV-2026-0042',
              customerId: CUSTOMER_ID,
              invoiceDate: '2026-11-02',
              dueDate: '2026-12-02',
              currencyCode: 'IDR',
              totalAmount: '7500000.0000',
              outstandingAmount: '7500000.0000',
              status: 'Issued',
              rowVersion: 2,
            },
          ]);
        }
        if (endpoint === '/customer-invoices/ar-inv-1' && (!options?.method || options.method === 'GET')) {
          return {
            id: 'ar-inv-1',
            invoiceNo: 'INV-2026-0042',
            customerId: CUSTOMER_ID,
            invoiceDate: '2026-11-02',
            dueDate: '2026-12-02',
            currencyCode: 'IDR',
            totalAmount: '7500000.0000',
            outstandingAmount: '7500000.0000',
            status: 'Issued',
            rowVersion: 2,
            lines: [
              {
                id: 'ar-line-1',
                lineNo: 1,
                accountId: GL_EXPENSE_ID,
                amount: '7500000.0000',
                description: 'Enterprise SaaS License',
              },
            ],
          };
        }
        if (endpoint === '/customer-invoices/ar-inv-1/pay' && options?.method === 'POST') {
          return {
            id: 'ar-inv-1',
            invoiceNo: 'INV-2026-0042',
            customerId: CUSTOMER_ID,
            invoiceDate: '2026-11-02',
            dueDate: '2026-12-02',
            currencyCode: 'IDR',
            totalAmount: '7500000.0000',
            outstandingAmount: '0.0000',
            status: 'Paid',
            rowVersion: 3,
          };
        }
        return emptyPage([]);
      },
    );

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <ArWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('INV-2026-0042')).toBeInTheDocument();
    expect(await screen.findByText('PT Nusantara Retail')).toBeInTheDocument();

    await user.click(screen.getByText('INV-2026-0042'));
    expect(await screen.findByText('Enterprise SaaS License')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Edit/i })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Mark Paid/i }));
    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/customer-invoices/ar-inv-1/pay',
        expect.objectContaining({
          method: 'POST',
          ifMatch: '2',
          idempotencyKey: expect.any(String),
        }),
      );
    });
  });
});

describe('Wave 2B: Tax & e-Faktur Workspace', () => {
  it('lists Tax Invoices and Tax Codes and submits Report to DJP with If-Match', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(async (endpoint: string, options?: { method?: string }) => {
      if (endpoint === '/tax-codes' && (!options?.method || options.method === 'GET')) {
        return emptyPage([
          {
            id: TAX_CODE_ID,
            code: 'PPN-11',
            rate: '11.0000',
            taxType: 'VAT',
            glAccountId: GL_EXPENSE_ID,
            rowVersion: 1,
          },
        ]);
      }
      if (endpoint === '/tax-codes' && options?.method === 'POST') {
        return {
          id: 'tax-code-new',
          code: 'PPH23-2',
          rate: '2.0000',
          taxType: 'PPH23',
          rowVersion: 1,
        };
      }
      if (endpoint === '/tax-invoices') {
        return emptyPage([
          {
            id: 'tx-inv-1',
            taxCodeId: TAX_CODE_ID,
            sourceDocumentRef: 'INV-2026-0042',
            baseAmount: '10000000.0000',
            taxAmount: '1100000.0000',
            efakturNo: '010.002-26.00000101',
            efakturStatus: 'Draft',
            rowVersion: 1,
          },
        ]);
      }
      if (endpoint === '/tax-invoices/tx-inv-1' && (!options?.method || options.method === 'GET')) {
        return {
          id: 'tx-inv-1',
          taxCodeId: TAX_CODE_ID,
          sourceDocumentRef: 'INV-2026-0042',
          baseAmount: '10000000.0000',
          taxAmount: '1100000.0000',
          efakturNo: '010.002-26.00000101',
          efakturStatus: 'Draft',
          rowVersion: 1,
        };
      }
      if (endpoint === '/tax-invoices/tx-inv-1/report' && options?.method === 'POST') {
        return {
          id: 'tx-inv-1',
          taxCodeId: TAX_CODE_ID,
          sourceDocumentRef: 'INV-2026-0042',
          baseAmount: '10000000.0000',
          taxAmount: '1100000.0000',
          efakturNo: '010.002-26.00000101',
          efakturStatus: 'Reported',
          rowVersion: 2,
        };
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <TaxWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('010.002-26.00000101')).toBeInTheDocument();

    await user.click(screen.getByText('INV-2026-0042'));
    await user.click(await screen.findByRole('button', { name: /Report to DJP/i }));

    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText(/NTTE Receipt Reference/i), 'NTTE-DJP-881');
    await user.click(within(dialog).getByRole('button', { name: /Confirm Report to DJP/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/tax-invoices/tx-inv-1/report',
        expect.objectContaining({
          method: 'POST',
          ifMatch: '1',
          idempotencyKey: expect.any(String),
          body: JSON.stringify({
            efakturNo: '010.002-26.00000101',
            ntte: 'NTTE-DJP-881',
          }),
        }),
      );
    });

    // Close Tax Invoice SideDrawer before switching tabs
    await user.keyboard('{Escape}');
    await user.click(screen.getByRole('tab', { name: /Tax Codes/i }));
    expect(await screen.findByText('PPN-11')).toBeInTheDocument();
  });
});

describe('Wave 2B: Treasury Cash & Bank Workspace', () => {
  it('supports Bank Account Reconcile, Payment Settle, and Batch Bank Statement Lines import', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(async (endpoint: string, options?: { method?: string }) => {
      if (endpoint === '/bank-accounts') {
        return emptyPage([
          {
            id: BANK_ACCOUNT_ID,
            organizationId: DEFAULT_HOLDING_ORG.id,
            code: 'BCA-IDR',
            bankName: 'Bank Central Asia',
            accountNo: '5210008899',
            currencyCode: 'IDR',
            glAccountId: GL_CASH_ID,
            rowVersion: 1,
          },
        ]);
      }
      if (endpoint === `/bank-accounts/${BANK_ACCOUNT_ID}/reconcile` && options?.method === 'POST') {
        return {
          bankAccountId: BANK_ACCOUNT_ID,
          matchedCount: 1,
          matches: [
            {
              statementLineId: 'stmt-1',
              paymentId: 'pay-1',
              paymentNo: 'PAY-2026-0001',
              amount: '5000000.0000',
            },
          ],
        };
      }
      if (endpoint === '/payments') {
        return emptyPage([
          {
            id: 'pay-1',
            paymentNo: 'PAY-2026-0001',
            paymentType: 'Outgoing',
            bankAccountId: BANK_ACCOUNT_ID,
            partyId: VENDOR_ID,
            amount: '5000000.0000',
            paymentDate: '2026-11-05',
            status: 'Draft',
            rowVersion: 1,
          },
        ]);
      }
      if (endpoint === '/payments/pay-1' && (!options?.method || options.method === 'GET')) {
        return {
          id: 'pay-1',
          paymentNo: 'PAY-2026-0001',
          paymentType: 'Outgoing',
          bankAccountId: BANK_ACCOUNT_ID,
          partyId: VENDOR_ID,
          amount: '5000000.0000',
          paymentDate: '2026-11-05',
          status: 'Draft',
          rowVersion: 1,
          allocations: [],
        };
      }
      if (endpoint === '/payments/pay-1/settle' && options?.method === 'POST') {
        return {
          id: 'pay-1',
          paymentNo: 'PAY-2026-0001',
          paymentType: 'Outgoing',
          bankAccountId: BANK_ACCOUNT_ID,
          amount: '5000000.0000',
          paymentDate: '2026-11-05',
          status: 'Settled',
          rowVersion: 2,
        };
      }
      if (endpoint === '/bank-statement-lines' && options?.method === 'POST') {
        return {
          importedCount: 2,
          ids: ['s1', 's2'],
          bankAccountId: BANK_ACCOUNT_ID,
        };
      }
      if (endpoint === '/bank-statement-lines') {
        return emptyPage([
          {
            id: 'stmt-1',
            bankAccountId: BANK_ACCOUNT_ID,
            statementDate: '2026-11-05',
            description: 'BI-FAST Vendor Settlement',
            amount: '-5000000.0000',
            externalRef: 'TRF-001',
            isReconciled: false,
            rowVersion: 1,
          },
        ]);
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <TreasuryWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('BCA-IDR')).toBeInTheDocument();

    // 1. Bank Account auto-reconcile
    await user.click(screen.getByText('BCA-IDR'));
    await user.click(await screen.findByRole('button', { name: /Run Auto-Reconcile/i }));
    expect(await screen.findByText(/Reconciliation Summary: 1 match\(es\) found/i)).toBeInTheDocument();

    // Close Bank drawer cleanly
    await user.keyboard('{Escape}');

    // 2. Payments tab -> Settle payment
    await user.click(screen.getByRole('tab', { name: /Payments/i }));
    expect(await screen.findByText('PAY-2026-0001')).toBeInTheDocument();
    await user.click(screen.getByText('PAY-2026-0001'));
    await user.click(await screen.findByRole('button', { name: /Settle Payment/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/payments/pay-1/settle',
        expect.objectContaining({
          method: 'POST',
          ifMatch: '1',
          idempotencyKey: expect.any(String),
        }),
      );
    });

    await user.keyboard('{Escape}');

    // 3. Batch Bank Statement Lines dialog
    await user.click(screen.getByRole('button', { name: /Add Statement Lines/i }));
    const stmtDialog = await screen.findByRole('dialog');
    await user.click(within(stmtDialog).getByRole('switch'));

    const descInputs = within(stmtDialog).getAllByPlaceholderText(/Bank transfer/i);
    expect(descInputs.length).toBe(2);
    await user.type(descInputs[0], 'Incoming Wire Client A');
    await user.type(descInputs[1], 'Vendor Settlement Out');

    const amountInputs = within(stmtDialog).getAllByPlaceholderText('0.00');
    fireEvent.change(amountInputs[0], { target: { value: '1250000' } });
    fireEvent.change(amountInputs[1], { target: { value: '-500000' } });

    await user.click(within(stmtDialog).getByRole('button', { name: /Import 2 Statement Lines/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/bank-statement-lines',
        expect.objectContaining({
          method: 'POST',
          idempotencyKey: expect.any(String),
        }),
      );
    });
  });
});

describe('Wave 2B: Currency & FX Triangulation Workspace', () => {
  it('displays system-seeded ISO currencies, exchange rates, and executes FX triangulation conversion', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(async (endpoint: string, options?: { method?: string }) => {
      if (endpoint === '/currencies') {
        return emptyPage([
          { id: 'c-idr', code: 'IDR', numericCode: '360', name: 'Indonesian Rupiah', precision: 2, isActive: true, rowVersion: 1 },
          { id: 'c-usd', code: 'USD', numericCode: '840', name: 'US Dollar', precision: 2, isActive: true, rowVersion: 1 },
          { id: 'c-sgd', code: 'SGD', numericCode: '702', name: 'Singapore Dollar', precision: 2, isActive: true, rowVersion: 1 },
        ]);
      }
      if (endpoint === '/exchange-rate-types') {
        return emptyPage([
          { id: 'rt-spot', code: 'SPOT', name: 'Spot Market Rate', rowVersion: 1 },
          { id: 'rt-close', code: 'CLOSING', name: 'Period Closing Rate', rowVersion: 1 },
          { id: 'rt-avg', code: 'AVERAGE', name: 'Period Average Rate', rowVersion: 1 },
        ]);
      }
      if (endpoint === '/exchange-rates') {
        return emptyPage([
          {
            id: 'fx-1',
            fromCurrency: 'USD',
            toCurrency: 'IDR',
            rateTypeId: 'rt-spot',
            rateTypeCode: 'SPOT',
            rateTypeName: 'Spot Market Rate',
            rate: '15850.000000',
            validFrom: '2026-11-01',
            rowVersion: 1,
          },
        ]);
      }
      if (endpoint === '/exchange-rates/convert' && options?.method === 'POST') {
        return {
          fromCurrency: 'SGD',
          toCurrency: 'EUR',
          rateTypeId: 'rt-spot',
          originalAmount: '1000',
          convertedAmount: '692.450000',
          rate: '0.692450',
          rateSource: 'TRIANGULATED',
          anchorCurrency: 'USD',
          effectiveDate: '2026-11-14',
        };
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <CurrencyWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('Indonesian Rupiah')).toBeInTheDocument();
    expect(screen.getByText('SVC-007 · FX')).toBeInTheDocument();

    // Run FX Triangulation Converter
    await user.click(screen.getByRole('button', { name: /Calculate Conversion/i }));

    expect(await screen.findByText(/TRIANGULATED · Anchor USD/i)).toBeInTheDocument();
    expect(screen.getByText(/692\.450000/)).toBeInTheDocument();

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/exchange-rates/convert',
        expect.objectContaining({
          method: 'POST',
        }),
      );
    });

    // Switch to Exchange Rates & Rate Types tab
    await user.click(screen.getByRole('tab', { name: /Exchange Rates & Rate Types/i }));
    expect(await screen.findByText('USD → IDR')).toBeInTheDocument();
  });

  it('surfaces explicit ErrorState on FX triangulation failure and creates custom currency', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(async (endpoint: string, options?: { method?: string }) => {
      if (endpoint === '/currencies' && options?.method === 'POST') {
        return {
          id: 'c-thb',
          code: 'THB',
          numericCode: '764',
          name: 'Thai Baht',
          precision: 2,
          isActive: true,
          rowVersion: 1,
        };
      }
      if (endpoint === '/currencies') {
        return emptyPage([
          { id: 'c-idr', code: 'IDR', name: 'Indonesian Rupiah', precision: 2, isActive: true, rowVersion: 1 },
          { id: 'c-usd', code: 'USD', name: 'US Dollar', precision: 2, isActive: true, rowVersion: 1 },
        ]);
      }
      if (endpoint === '/exchange-rates/convert' && options?.method === 'POST') {
        throw new ApiClientError('FX_RATE_NOT_FOUND', 'No SPOT quote for pair on selected date', 404, 'corr-fx-404');
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <CurrencyWorkspace />
      </Wrapper>,
    );

    await user.click(screen.getByRole('button', { name: /Calculate Conversion/i }));
    expect(await screen.findByText(/No SPOT quote for pair on selected date/i)).toBeInTheDocument();
    expect(screen.getByText(/corr-fx-404/i)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Add Currency/i }));
    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText(/ISO Code/i), 'THB');
    await user.type(within(dialog).getByLabelText(/Numeric Code/i), '764');
    await user.type(within(dialog).getByLabelText(/Currency Name/i), 'Thai Baht');
    await user.click(within(dialog).getByRole('button', { name: /Create Currency/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/currencies',
        expect.objectContaining({
          method: 'POST',
          idempotencyKey: expect.any(String),
          body: JSON.stringify({
            code: 'THB',
            numericCode: '764',
            name: 'Thai Baht',
            precision: 2,
            isActive: true,
          }),
        }),
      );
    });
  });
});

describe('Wave 2B: AP Form Validation, Create, Edit & Delete Draft', () => {
  it('validates line sum balance, syncs with Match Line Sum, creates Draft bill, edits, and deletes', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(async (endpoint: string, options?: { method?: string }) => {
      if (endpoint === '/vendor-invoices' && options?.method === 'POST') {
        return {
          id: 'ap-created-1',
          invoiceNo: 'BILL-NEW-01',
          vendorId: VENDOR_ID,
          invoiceDate: '2026-11-10',
          dueDate: '2026-12-10',
          currencyCode: 'IDR',
          totalAmount: '1500000',
          outstandingAmount: '1500000',
          status: 'Draft',
          rowVersion: 1,
        };
      }
      if (endpoint === '/vendor-invoices/ap-draft-1' && options?.method === 'PUT') {
        return {
          id: 'ap-draft-1',
          invoiceNo: 'BILL-DRAFT-1',
          vendorId: VENDOR_ID,
          invoiceDate: '2026-11-01',
          dueDate: '2026-12-05',
          currencyCode: 'IDR',
          totalAmount: '1500000.0000',
          outstandingAmount: '1500000.0000',
          status: 'Draft',
          rowVersion: 2,
        };
      }
      if (endpoint === '/vendor-invoices/ap-draft-1' && options?.method === 'DELETE') {
        return {};
      }
      if (endpoint === '/vendor-invoices/ap-draft-1') {
        return {
          id: 'ap-draft-1',
          invoiceNo: 'BILL-DRAFT-1',
          vendorId: VENDOR_ID,
          invoiceDate: '2026-11-01',
          dueDate: '2026-12-01',
          currencyCode: 'IDR',
          totalAmount: '1500000.0000',
          outstandingAmount: '1500000.0000',
          status: 'Draft',
          rowVersion: 1,
          lines: [
            {
              id: 'l-1',
              lineNo: 1,
              accountId: GL_EXPENSE_ID,
              amount: '1500000.0000',
              description: 'Initial consulting fee',
            },
          ],
        };
      }
      if (endpoint === '/vendor-invoices') {
        return emptyPage([
          {
            id: 'ap-draft-1',
            invoiceNo: 'BILL-DRAFT-1',
            vendorId: VENDOR_ID,
            invoiceDate: '2026-11-01',
            dueDate: '2026-12-01',
            currencyCode: 'IDR',
            totalAmount: '1500000.0000',
            outstandingAmount: '1500000.0000',
            status: 'Draft',
            rowVersion: 1,
          },
        ]);
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <ApWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText('BILL-DRAFT-1')).toBeInTheDocument();

    // Create Draft Bill with decimal balance validation
    await user.click(screen.getByRole('button', { name: /New Vendor Invoice/i }));
    const createDialog = await screen.findByRole('dialog');

    fireEvent.change(within(createDialog).getByLabelText(/Invoice Number/i), {
      target: { value: 'BILL-NEW-01' },
    });
    await user.type(within(createDialog).getByLabelText(/Vendor/i), VENDOR_ID);
    await user.type(within(createDialog).getByPlaceholderText('Account UUID'), GL_EXPENSE_ID);
    const moneyInputs = within(createDialog).getAllByPlaceholderText('0.00');
    fireEvent.change(moneyInputs[0], { target: { value: '1500000' } });
    await user.type(within(createDialog).getByPlaceholderText('Expense description'), 'Cloud infra');

    // Submit with mismatched totalAmount (0 vs 1500000) -> blocked client-side
    await user.click(within(createDialog).getByRole('button', { name: /Create Draft Bill/i }));
    expect(await within(createDialog).findByText(/must match invoice total/i)).toBeInTheDocument();

    // Fix with Match Line Sum -> submits cleanly
    await user.click(within(createDialog).getByRole('button', { name: /Match Line Sum/i }));
    await user.click(within(createDialog).getByRole('button', { name: /Create Draft Bill/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/vendor-invoices',
        expect.objectContaining({
          method: 'POST',
          idempotencyKey: expect.any(String),
        }),
      );
    });

    // Open Draft -> Edit Draft -> PUT with If-Match
    await user.click(await screen.findByText('BILL-DRAFT-1'));
    await user.click(await screen.findByRole('button', { name: /Edit Draft/i }));
    const editDialog = await screen.findByRole('dialog');
    await user.click(within(editDialog).getByRole('button', { name: /Save Changes/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/vendor-invoices/ap-draft-1',
        expect.objectContaining({
          method: 'PUT',
          ifMatch: '1',
        }),
      );
    });

    // Open Draft -> Delete -> Confirm Delete -> DELETE with If-Match
    await user.click(await screen.findByText('BILL-DRAFT-1'));
    await user.click(await screen.findByRole('button', { name: /^Delete$/i }));
    await user.click(await screen.findByRole('button', { name: /Confirm Delete/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/vendor-invoices/ap-draft-1',
        expect.objectContaining({
          method: 'DELETE',
          ifMatch: '1',
        }),
      );
    });
  });
});

describe('Wave 2B: Tax Invoice Decimal Preview & Code Creation', () => {
  it('computes fixed-point tax preview and creates new tax code', async () => {
    const user = userEvent.setup();
    apiClientMock.mockImplementation(async (endpoint: string, options?: { method?: string }) => {
      if (endpoint === '/tax-codes' && options?.method === 'POST') {
        return {
          id: 'tax-new-2',
          code: 'PPH23-2',
          rate: '2.0000',
          taxType: 'PPH23',
          rowVersion: 1,
        };
      }
      if (endpoint === '/tax-codes') {
        return emptyPage([
          {
            id: TAX_CODE_ID,
            code: 'PPN-11',
            rate: '11.0000',
            taxType: 'VAT',
            rowVersion: 1,
          },
        ]);
      }
      if (endpoint === '/tax-invoices') {
        return emptyPage([]);
      }
      return emptyPage([]);
    });

    const Wrapper = createWrapper();
    render(
      <Wrapper>
        <TaxWorkspace />
      </Wrapper>,
    );

    expect(await screen.findByText(/No tax invoices recorded/i)).toBeInTheDocument();

    // Open Create Tax Invoice dialog -> verify live decimal preview
    await user.click(screen.getByRole('button', { name: /New Tax Invoice/i }));
    const invDialog = await screen.findByRole('dialog');
    const baseInput = within(invDialog).getByLabelText(/DPP \/ Base Amount/i);
    fireEvent.change(baseInput, { target: { value: '10000000' } });
    expect(await within(invDialog).findByText(/1\.100\.000,00/)).toBeInTheDocument();

    await user.keyboard('{Escape}');

    // Open Create Tax Code dialog -> submit PPH23-2
    await user.click(screen.getByRole('button', { name: /New Tax Code/i }));
    const codeDialog = await screen.findByRole('dialog');
    await user.type(within(codeDialog).getByLabelText(/Tax Code/i), 'PPH23-2');
    fireEvent.change(within(codeDialog).getByLabelText(/Tax Rate/i), {
      target: { value: '2.0000' },
    });
    await user.click(within(codeDialog).getByRole('button', { name: /Save Tax Code/i }));

    await waitFor(() => {
      expect(apiClientMock).toHaveBeenCalledWith(
        '/tax-codes',
        expect.objectContaining({
          method: 'POST',
          idempotencyKey: expect.any(String),
        }),
      );
    });
  });
});

