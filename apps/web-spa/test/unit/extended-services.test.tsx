import * as React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nextProvider } from 'react-i18next';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AssetsWorkspace } from '../../src/features/asset/AssetsWorkspace';
import { BudgetsWorkspace } from '../../src/features/budget/BudgetsWorkspace';
import { ConsolidationWorkspace } from '../../src/features/consolidation/ConsolidationWorkspace';
import { CrmWorkspace } from '../../src/features/crm/CrmWorkspace';
import { DimensionsWorkspace } from '../../src/features/dimension/DimensionsWorkspace';
import { FinancialCloseWorkspace } from '../../src/features/financialclose/FinancialCloseWorkspace';
import { FixedAssetsWorkspace } from '../../src/features/fixedasset/FixedAssetsWorkspace';
import { LocalizationWorkspace } from '../../src/features/localization/LocalizationWorkspace';
import { MaintenanceWorkspace } from '../../src/features/maintenance/MaintenanceWorkspace';
import { ProjectsWorkspace } from '../../src/features/project/ProjectsWorkspace';
import { ReportsWorkspace } from '../../src/features/reports/ReportsWorkspace';
import { Route as DeleteAccountRoute } from '../../src/routes/_public/delete-account';
import { Route as PrivacyRoute } from '../../src/routes/_public/privacy';
import { Route as TermsRoute } from '../../src/routes/_public/terms';
import i18n from '../../src/lib/i18n';
import { useAuthStore } from '../../src/store/auth-store';
import { useUiStore } from '../../src/store/ui-store';
import type { Organization, User } from '../../src/types';

const mockNavigate = vi.fn();

vi.mock('@tanstack/react-router', async () => {
  return {
    createFileRoute: () => (opts: { component: React.ComponentType }) => ({
      options: opts,
    }),
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
  };
});

vi.mock('recharts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('recharts')>();
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div style={{ width: 640, height: 260 }}>{children}</div>
    ),
  };
});

const HOLDING_ORG: Organization = {
  id: '11111111-1111-1111-1111-111111111111',
  code: 'HLD-01',
  name: 'PT Sinergi Investasi Properti',
  type: 'HOLDING',
  parentId: null,
  functionalCurrency: 'IDR',
};

const HOLDING_ADMIN_USER: User = {
  id: 'usr-ho-1',
  username: 'admin.ho',
  email: 'admin.ho@sinergi.example.com',
  fullName: 'Head Office Admin',
  adminTier: 1,
  roles: ['HOLDING_ADMIN'],
  permissions: [
    'fixedasset:read',
    'fixedasset:write',
    'dimension:read',
    'dimension:write',
    'budget:read',
    'budget:write',
    'budget:approve',
    'financialclose:read',
    'financialclose:write',
    'financialclose:approve',
    'consolidation:read',
    'consolidation:write',
    'project:read',
    'project:write',
    'asset:read',
    'asset:write',
    'maintenance:read',
    'maintenance:write',
    'crm:read',
    'crm:write',
    'localization:read',
    'localization:write',
    'reporting.read',
    'gl:read',
  ],
};

const SUBSIDIARY_ADMIN_USER: User = {
  ...HOLDING_ADMIN_USER,
  id: 'usr-sub-1',
  username: 'admin.properti',
  fullName: 'Subsidiary Admin',
  adminTier: 2,
  roles: ['SUBSIDIARY_ADMIN'],
};

function makeJsonResp(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

function renderWithProviders(ui: React.ReactElement) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <I18nextProvider i18n={i18n}>{ui}</I18nextProvider>
    </QueryClientProvider>
  );
}

describe('Extended Services Workspaces, Financial Statements & Legal Trio', () => {
  beforeEach(() => {
    useUiStore.setState({ locale: 'en' });
    useAuthStore.getState().setAuth(HOLDING_ADMIN_USER, 'live-test-token', HOLDING_ORG, [HOLDING_ORG]);

    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url;

      if (url.includes('/financial-statements/income-statement')) {
        return makeJsonResp({
          statementType: 'INCOME_STATEMENT',
          currency: 'IDR',
          period: '2026-Q4',
          sections: [
            {
              code: 'REVENUE',
              name: 'Operating Revenue',
              items: [{ accountCode: '4000', accountName: 'Property Rental Revenue', amount: '500000000.0000' }],
              subtotal: '500000000.0000',
            },
          ],
          totals: { netIncome: '350000000.0000' },
        });
      }
      if (url.includes('/financial-statements/balance-sheet')) {
        return makeJsonResp({
          statementType: 'BALANCE_SHEET',
          currency: 'IDR',
          asOfDate: '2026-11-24',
          sections: [
            {
              code: 'ASSETS',
              name: 'Assets',
              items: [{ accountCode: '1100', accountName: 'Cash & Bank', amount: '1000000000.0000' }],
              subtotal: '1000000000.0000',
            },
          ],
          totals: { totalAssets: '1000000000.0000', totalLiabilitiesAndEquity: '1000000000.0000' },
          isBalanced: true,
        });
      }
      if (url.includes('/financial-statements/cash-flow')) {
        return makeJsonResp({
          statementType: 'CASH_FLOW',
          currency: 'IDR',
          period: '2026-Q4',
          sections: [
            {
              code: 'OPERATING',
              name: 'Operating Activities',
              items: [{ accountCode: '1100', accountName: 'Cash Collected', amount: '250000000.0000' }],
              subtotal: '250000000.0000',
            },
          ],
          totals: { netCashChange: '250000000.0000' },
        });
      }

      return makeJsonResp({
        data: [],
        page: { number: 1, size: 25, totalElements: 0, totalPages: 1 },
      });
    }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders all 10 extended service workspaces without crashing', async () => {
    const workspaces: Array<{ name: string; element: React.ReactElement; heading: RegExp }> = [
      { name: 'FixedAssets', element: <FixedAssetsWorkspace />, heading: /Fixed Assets & Depreciation Register/i },
      { name: 'Dimensions', element: <DimensionsWorkspace />, heading: /Analytical Dimensions & Cost Allocation/i },
      { name: 'Budgets', element: <BudgetsWorkspace />, heading: /Budgeting, Policies & Commitments/i },
      { name: 'FinancialClose', element: <FinancialCloseWorkspace />, heading: /Financial Close & Period Governance/i },
      { name: 'Consolidation', element: <ConsolidationWorkspace />, heading: /Multi-Entity Group Consolidation/i },
      { name: 'Projects', element: <ProjectsWorkspace />, heading: /Project Management, WBS & Progress Billing/i },
      { name: 'Assets', element: <AssetsWorkspace />, heading: /Operational Asset Registry/i },
      { name: 'Maintenance', element: <MaintenanceWorkspace />, heading: /Maintenance Management \(CMMS\)/i },
      { name: 'Crm', element: <CrmWorkspace />, heading: /CRM Leads, Opportunities & Support Cases/i },
      { name: 'Localization', element: <LocalizationWorkspace />, heading: /Languages, Translations & Locale Formats/i },
    ];

    for (const ws of workspaces) {
      const { unmount } = renderWithProviders(ws.element);
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: ws.heading })).toBeInTheDocument();
      });
      unmount();
    }
  });

  it('enforces Head Office tier write gate in ConsolidationWorkspace for SUBSIDIARY_ADMIN vs HOLDING_ADMIN', async () => {
    useAuthStore.getState().setAuth(SUBSIDIARY_ADMIN_USER, 'live-test-token', HOLDING_ORG, [HOLDING_ORG]);
    const { unmount } = renderWithProviders(<ConsolidationWorkspace />);
    expect(screen.getByText(/Head Office Tier Guard/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /New Group/i })).not.toBeInTheDocument();
    unmount();

    useAuthStore.getState().setAuth(HOLDING_ADMIN_USER, 'live-test-token', HOLDING_ORG, [HOLDING_ORG]);
    renderWithProviders(<ConsolidationWorkspace />);
    expect(screen.queryByText(/Head Office Tier Guard/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /New Group/i })).toBeInTheDocument();
  });

  it('renders Income Statement, Balance Sheet (Balanced badge), and Cash Flow tabs in ReportsWorkspace', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ReportsWorkspace />);

    await user.click(screen.getByRole('tab', { name: /Income Statement \(P&L\)/i }));
    await waitFor(() => {
      expect(screen.getByText('Operating Revenue')).toBeInTheDocument();
      expect(screen.getByText('Property Rental Revenue')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('tab', { name: /Balance Sheet/i }));
    await waitFor(() => {
      expect(screen.getByText('Balanced')).toBeInTheDocument();
      expect(screen.getByText('Cash & Bank')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('tab', { name: /Cash Flow/i }));
    await waitFor(() => {
      expect(screen.getByText('Operating Activities')).toBeInTheDocument();
      expect(screen.getByText('Cash Collected')).toBeInTheDocument();
    });
  });

  it('renders the Legal Trio (/privacy, /terms, /delete-account)', () => {
    const PrivacyComponent = PrivacyRoute.options.component as React.ComponentType;
    const TermsComponent = TermsRoute.options.component as React.ComponentType;
    const DeleteAccountComponent = DeleteAccountRoute.options.component as React.ComponentType;

    const r1 = renderWithProviders(<PrivacyComponent />);
    expect(screen.getByRole('heading', { name: /Enterprise ERP Platform Privacy Policy/i })).toBeInTheDocument();
    r1.unmount();

    const r2 = renderWithProviders(<TermsComponent />);
    expect(screen.getByRole('heading', { name: /ERP Platform Terms of Service/i })).toBeInTheDocument();
    r2.unmount();

    const r3 = renderWithProviders(<DeleteAccountComponent />);
    expect(screen.getByRole('heading', { name: /Account & Personal Data Deletion Request/i })).toBeInTheDocument();
    r3.unmount();
  });
});
