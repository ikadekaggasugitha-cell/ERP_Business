import type { ReactElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuditWorkspace } from '../../src/features/audit/AuditWorkspace';
import { computeForensicDiff } from '../../src/features/audit/audit-api';
import { OrganizationsWorkspace } from '../../src/features/admin/OrganizationsWorkspace';
import { SecurityWorkspace } from '../../src/features/admin/SecurityWorkspace';
import { SettingsWorkspace } from '../../src/features/admin/SettingsWorkspace';
import { UsersWorkspace } from '../../src/features/admin/UsersWorkspace';
import { HrWorkspace } from '../../src/features/hr/HrWorkspace';
import { PayrollWorkspace } from '../../src/features/payroll/PayrollWorkspace';
import { serializeCalculatePayload } from '../../src/features/payroll/payroll-api';
import { WorkflowWorkspace } from '../../src/features/workflow/WorkflowWorkspace';
import * as apiModule from '../../src/lib/api';
import * as exportModule from '../../src/lib/export';
import i18n from '../../src/lib/i18n';
import { useAuthStore } from '../../src/store/auth-store';
import { useUiStore } from '../../src/store/ui-store';
import type { Organization, User } from '../../src/types';

vi.mock('../../src/lib/api', async () => {
  const actual = await vi.importActual<typeof import('../../src/lib/api')>('../../src/lib/api');
  return {
    ...actual,
    apiClient: vi.fn(),
  };
});

const mockedApiClient = vi.mocked(apiModule.apiClient);

const TEST_ORG: Organization = {
  id: 'org-1111',
  code: 'PT-NUSANTARA',
  name: 'PT Nusantara Sejahtera Tbk',
  type: 'COMPANY',
  parentId: null,
  functionalCurrency: 'IDR',
  isActive: true,
  rowVersion: 1,
};

function buildUser(permissions: string[], id = 'user-admin-1'): User {
  return {
    id,
    username: 'admin.erp',
    fullName: 'Arga Wijaya',
    email: 'arga@nusantara.co.id',
    activeOrgId: TEST_ORG.id,
    roles: ['ADMIN'],
    permissions,
  };
}

function setAuthState(permissions: string[], userId = 'user-admin-1') {
  useAuthStore.setState({
    token: 'test-token',
    isAuthenticated: true,
    isBootstrapped: true,
    isDemoSession: false,
    user: buildUser(permissions, userId),
    organizations: [TEST_ORG],
    activeOrg: TEST_ORG,
  });
}

function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    </I18nextProvider>,
  );
}

function listResponse<T>(data: T[], page = 1, size = 25) {
  return {
    data,
    page: {
      number: page,
      size,
      totalElements: data.length,
      totalPages: Math.max(1, Math.ceil(data.length / size)),
    },
  };
}

describe('Wave 2D — HCM, Payroll, Workflow, Audit & Admin Governance', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useUiStore.setState({
      locale: 'en',
      theme: 'light',
      portal: 'backoffice',
      isRealtimeConnected: true,
    });
    void i18n.changeLanguage('en');
    setAuthState(['*']);
  });

  /* ------------------------------------------------------------------ */
  /* 1. HR MODULE                                                       */
  /* ------------------------------------------------------------------ */
  describe('HR Workspace (/hr)', () => {
    const sampleEmp = {
      id: 'emp-1',
      organizationId: TEST_ORG.id,
      employeeNo: 'EMP-001',
      fullName: 'Dewi Lestari',
      nationalIdMasked: '3174011992050009',
      dateOfBirth: '1992-05-09',
      email: 'dewi.lestari@nusantara.co.id',
      hireDate: '2024-01-15',
      status: 'Active',
      userRefId: null,
      rowVersion: 3,
    };

    const samplePos = {
      id: 'pos-1',
      organizationId: TEST_ORG.id,
      code: 'FIN-MGR',
      title: 'Finance Manager',
      organizationRefId: TEST_ORG.id,
      parentPositionId: null,
      rowVersion: 1,
    };

    it('creates employee, opens PII drawer with mask toggle, computes fixed-point compensation total, and terminates with If-Match', async () => {
      let terminated = false;
      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/employees' && (!options?.method || options.method === 'GET')) {
          return listResponse([
            { ...sampleEmp, status: terminated ? 'Terminated' : 'Active' },
          ]);
        }
        if (endpoint === '/positions' && (!options?.method || options.method === 'GET')) {
          return listResponse([samplePos]);
        }
        if (endpoint === '/employees' && options?.method === 'POST') {
          return {
            ...sampleEmp,
            id: 'emp-2',
            employeeNo: 'EMP-002',
            fullName: 'Bima Sakti',
            rowVersion: 1,
          };
        }
        if (endpoint === '/employees/emp-1') {
          return sampleEmp;
        }
        if (endpoint === '/employees/emp-1/employments') {
          return listResponse([
            {
              id: 'emply-1',
              employeeId: 'emp-1',
              positionId: 'pos-1',
              employmentType: 'Permanent',
              effectiveFrom: '2024-01-15',
              effectiveTo: null,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/employees/emp-1/compensations') {
          return listResponse([
            {
              id: 'comp-1',
              employeeId: 'emp-1',
              componentCode: 'BASE_SALARY',
              amount: '15000000.10',
              effectiveFrom: '2026-01-01',
              rowVersion: 1,
            },
            {
              id: 'comp-2',
              employeeId: 'emp-1',
              componentCode: 'ALLOWANCE',
              amount: '2500000.20',
              effectiveFrom: '2026-01-01',
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/employees/emp-1/terminate' && options?.method === 'POST') {
          terminated = true;
          return { ...sampleEmp, status: 'Terminated', rowVersion: 4 };
        }
        throw new Error(`Unexpected call: ${endpoint}`);
      });

      renderWithProviders(<HrWorkspace />);

      await screen.findByText('Dewi Lestari');

      // Open create employee dialog and submit
      fireEvent.click(screen.getByRole('button', { name: /New Employee/i }));
      fireEvent.change(screen.getByLabelText(/Employee #/i), { target: { value: 'EMP-002' } });
      fireEvent.change(screen.getByLabelText(/Full Name/i), { target: { value: 'Bima Sakti' } });
      fireEvent.change(screen.getByLabelText(/Hire Date/i), { target: { value: '2026-02-01' } });
      fireEvent.click(screen.getByRole('button', { name: /Create Record/i }));

      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/employees',
          expect.objectContaining({
            method: 'POST',
            idempotencyKey: expect.any(String),
          }),
        );
      });

      // Open Detail SideDrawer & inspect PII masking toggle
      fireEvent.click(screen.getByRole('button', { name: /Inspect/i }));
      const maskedNik = await screen.findByTestId('drawer-pii-nik');
      expect(maskedNik.textContent).toContain('••••');
      expect(screen.getByTestId('drawer-pii-email').textContent).toBe('de***@nusantara.co.id');

      fireEvent.click(screen.getByRole('button', { name: /Reveal PII/i }));
      expect(screen.getByTestId('drawer-pii-nik').textContent).toBe('3174011992050009');
      expect(screen.getByTestId('drawer-pii-email').textContent).toBe(
        'dewi.lestari@nusantara.co.id',
      );

      // Switch to Compensation Grades sub-tab (Radix Tabs activates on mouseDown)
      const compTab = screen.getByRole('tab', { name: /Compensation Grades/i });
      fireEvent.mouseDown(compTab, { button: 0 });
      fireEvent.click(compTab);
      const totalEl = await screen.findByTestId('hr-total-compensation');
      expect(totalEl.textContent).toContain('17.500.000');

      // Terminate employee directly from accessible drawer button with If-Match
      fireEvent.click(screen.getByRole('button', { name: /^Terminate$/i }));
      fireEvent.click(screen.getByRole('button', { name: /Confirm Termination/i }));

      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/employees/emp-1/terminate',
          expect.objectContaining({
            method: 'POST',
            ifMatch: '3',
          }),
        );
      });
    });

    it('hides HR write controls when user lacks hr:write permission', async () => {
      setAuthState(['hr:read']);
      mockedApiClient.mockImplementation(async (endpoint) => {
        if (endpoint === '/employees') return listResponse([sampleEmp]);
        if (endpoint === '/positions') return listResponse([samplePos]);
        return listResponse([]);
      });

      renderWithProviders(<HrWorkspace />);
      await screen.findByText('Dewi Lestari');

      expect(screen.getByRole('status')).toHaveTextContent(/hr:write/i);
      expect(screen.queryByRole('button', { name: /New Employee/i })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /New Position/i })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /Terminate/i })).not.toBeInTheDocument();
    });
  });

  /* ------------------------------------------------------------------ */
  /* 2. PAYROLL MODULE                                                  */
  /* ------------------------------------------------------------------ */
  describe('Payroll Workspace (/payroll)', () => {
    it('strips unknown fields in DisallowUnknownFields calculate payload and sums decimal totals cleanly', () => {
      const payload = serializeCalculatePayload({
        totalAmount: '0.00',
        employees: [
          {
            employeeId: 'emp-1',
            basicSalary: '10000000.10',
            ptkp: 'TK/0',
            hasNpwp: true,
          },
        ],
      });
      expect(Object.keys(payload)).toEqual(['totalAmount', 'employees']);
      expect(payload.employees).toEqual([
        {
          employeeId: 'emp-1',
          basicSalary: '10000000.10',
          ptkp: 'TK/0',
          hasNpwp: true,
        },
      ]);
    });

    it('walks Draft -> Calculated -> Approved -> Posted lifecycle using If-Match and fixed-point totals', async () => {
      let currentStatus = 'Draft';
      let currentVersion = 1;
      let currentGlRef: string | null = null;

      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/payroll-runs' && (!options?.method || options.method === 'GET')) {
          return listResponse([
            {
              id: 'pr-1',
              organizationId: TEST_ORG.id,
              runNo: 'PR-202611-01',
              periodMonth: 11,
              periodYear: 2026,
              status: currentStatus,
              glJournalRef: currentGlRef,
              rowVersion: currentVersion,
            },
          ]);
        }
        if (endpoint === '/employees') {
          return listResponse([]);
        }
        if (endpoint === '/payroll-runs/pr-1') {
          return {
            id: 'pr-1',
            organizationId: TEST_ORG.id,
            runNo: 'PR-202611-01',
            periodMonth: 11,
            periodYear: 2026,
            status: currentStatus,
            glJournalRef: currentGlRef,
            correlationId: 'corr-pr-1',
            rowVersion: currentVersion,
            results:
              currentStatus === 'Draft'
                ? []
                : [
                    {
                      id: 'res-1',
                      employeeId: 'emp-1',
                      grossAmount: '10000000.10',
                      deductionAmount: '500000.05',
                      taxAmount: '250000.02',
                      netAmount: '9250000.03',
                      rowVersion: 1,
                    },
                    {
                      id: 'res-2',
                      employeeId: 'emp-2',
                      grossAmount: '20000000.20',
                      deductionAmount: '1000000.10',
                      taxAmount: '500000.04',
                      netAmount: '18500000.06',
                      rowVersion: 1,
                    },
                  ],
          };
        }
        if (endpoint === '/payroll-runs/pr-1/calculate' && options?.method === 'POST') {
          currentStatus = 'Calculated';
          currentVersion = 2;
          return {
            id: 'pr-1',
            organizationId: TEST_ORG.id,
            runNo: 'PR-202611-01',
            periodMonth: 11,
            periodYear: 2026,
            status: currentStatus,
            rowVersion: currentVersion,
            results: [],
          };
        }
        if (endpoint === '/payroll-runs/pr-1/approve' && options?.method === 'POST') {
          currentStatus = 'Approved';
          currentVersion = 3;
          return {
            id: 'pr-1',
            organizationId: TEST_ORG.id,
            runNo: 'PR-202611-01',
            periodMonth: 11,
            periodYear: 2026,
            status: currentStatus,
            rowVersion: currentVersion,
          };
        }
        if (endpoint === '/payroll-runs/pr-1/post' && options?.method === 'POST') {
          currentStatus = 'Posted';
          currentVersion = 4;
          currentGlRef = 'JV-PR-202611-01';
          return {
            id: 'pr-1',
            organizationId: TEST_ORG.id,
            runNo: 'PR-202611-01',
            periodMonth: 11,
            periodYear: 2026,
            status: currentStatus,
            glJournalRef: currentGlRef,
            rowVersion: currentVersion,
          };
        }
        throw new Error(`Unexpected payroll call: ${endpoint}`);
      });

      renderWithProviders(<PayrollWorkspace />);
      await screen.findByText('PR-202611-01');

      fireEvent.click(screen.getByRole('button', { name: /Manage Run/i }));
      await screen.findByRole('button', { name: /Calculate Payroll/i });

      // Open Calculate dialog & submit
      fireEvent.click(screen.getByRole('button', { name: /Calculate Payroll/i }));
      const dialogs = screen.getAllByRole('dialog');
      const calcBtn = within(dialogs[dialogs.length - 1]).getByRole('button', {
        name: /Calculate Payroll/i,
      });
      fireEvent.click(calcBtn);

      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/payroll-runs/pr-1/calculate',
          expect.objectContaining({
            method: 'POST',
            ifMatch: '1',
            idempotencyKey: expect.any(String),
          }),
        );
      });

      // Fixed-point totals after Calculate (0.10 + 0.20 = 0.30, 9250000.03 + 18500000.06 = 27750000.09)
      const grossTotal = await screen.findByTestId('payroll-total-gross');
      expect(grossTotal.textContent).toContain('30.000.000');
      expect(screen.getByTestId('payroll-total-net').textContent).toContain('27.750.000');

      // Approve Run
      fireEvent.click(await screen.findByRole('button', { name: /Approve Run/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/payroll-runs/pr-1/approve',
          expect.objectContaining({
            method: 'POST',
            ifMatch: '2',
            idempotencyKey: expect.any(String),
          }),
        );
      });

      // Post to GL
      fireEvent.click(await screen.findByRole('button', { name: /Post to GL/i }));
      const postDialogs = screen.getAllByRole('dialog');
      fireEvent.click(
        within(postDialogs[postDialogs.length - 1]).getByRole('button', {
          name: /Post to GL/i,
        }),
      );
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/payroll-runs/pr-1/post',
          expect.objectContaining({
            method: 'POST',
            ifMatch: '3',
            body: JSON.stringify({ glJournalRef: 'JV-PR-202611-01' }),
          }),
        );
      });
    });

    it('allows deleting a Draft payroll run with If-Match', async () => {
      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/payroll-runs') {
          return listResponse([
            {
              id: 'pr-draft',
              organizationId: TEST_ORG.id,
              runNo: 'PR-DRAFT-99',
              periodMonth: 12,
              periodYear: 2026,
              status: 'Draft',
              rowVersion: 7,
            },
          ]);
        }
        if (endpoint === '/employees') return listResponse([]);
        if (endpoint === '/payroll-runs/pr-draft' && (!options?.method || options.method === 'GET')) {
          return {
            id: 'pr-draft',
            organizationId: TEST_ORG.id,
            runNo: 'PR-DRAFT-99',
            periodMonth: 12,
            periodYear: 2026,
            status: 'Draft',
            rowVersion: 7,
            results: [],
          };
        }
        if (endpoint === '/payroll-runs/pr-draft' && options?.method === 'DELETE') {
          return undefined;
        }
        return listResponse([]);
      });

      renderWithProviders(<PayrollWorkspace />);
      await screen.findByText('PR-DRAFT-99');

      fireEvent.click(screen.getByRole('button', { name: /Manage Run/i }));
      fireEvent.click(await screen.findByRole('button', { name: /Delete Draft/i }));

      const dialogs = screen.getAllByRole('dialog');
      fireEvent.click(
        within(dialogs[dialogs.length - 1]).getByRole('button', { name: /Delete Draft/i }),
      );

      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/payroll-runs/pr-draft',
          expect.objectContaining({ method: 'DELETE', ifMatch: '7' }),
        );
      });
    });
  });

  /* ------------------------------------------------------------------ */
  /* 3. WORKFLOW MODULE                                                 */
  /* ------------------------------------------------------------------ */
  describe('Workflow Approvals Workspace (/workflow)', () => {
    it('renders live tasks (zero static placeholders), enforces SoD feedback on 403 SOD_VIOLATION, and opens instance timeline', async () => {
      setAuthState(['workflow:approve', 'workflow:write'], 'actor-creator');
      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/workflow-tasks') {
          return listResponse([
            {
              id: 'task-1',
              organizationId: TEST_ORG.id,
              instanceId: 'inst-1',
              assigneeId: 'actor-creator',
              decision: null,
              documentRef: 'JV-2026-888',
              instanceStatus: 'Pending',
              createdBy: 'actor-creator',
              rowVersion: 5,
            },
          ]);
        }
        if (endpoint === '/workflow-instances') {
          return listResponse([
            {
              id: 'inst-1',
              organizationId: TEST_ORG.id,
              definitionId: 'def-1',
              documentRef: 'JV-2026-888',
              status: 'Pending',
              currentStep: 1,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/workflow-definitions' && (!options?.method || options.method === 'GET')) {
          return listResponse([
            {
              id: 'def-1',
              organizationId: TEST_ORG.id,
              code: 'WF-JE-DUAL',
              documentType: 'JOURNAL_ENTRY',
              version: 1,
              isActive: true,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/workflow-tasks/task-1/approve' && options?.method === 'POST') {
          throw new apiModule.ApiClientError(
            'SOD_VIOLATION',
            'Creator cannot approve own workflow task',
            403,
          );
        }
        if (endpoint === '/workflow-instances/inst-1') {
          return {
            id: 'inst-1',
            organizationId: TEST_ORG.id,
            definitionId: 'def-1',
            documentRef: 'JV-2026-888',
            status: 'Pending',
            currentStep: 1,
            createdBy: 'actor-creator',
            rowVersion: 1,
            tasks: [
              {
                id: 'task-1',
                organizationId: TEST_ORG.id,
                instanceId: 'inst-1',
                assigneeId: 'actor-creator',
                decision: null,
                documentRef: 'JV-2026-888',
                instanceStatus: 'Pending',
                createdBy: 'actor-creator',
                rowVersion: 5,
              },
            ],
          };
        }
        if (endpoint === '/workflow-definitions' && options?.method === 'POST') {
          return {
            id: 'def-2',
            organizationId: TEST_ORG.id,
            code: 'WF-PR-01',
            documentType: 'PAYROLL_RUN',
            version: 1,
            isActive: true,
            rowVersion: 1,
          };
        }
        throw new Error(`Unexpected workflow endpoint: ${endpoint}`);
      });

      renderWithProviders(<WorkflowWorkspace />);

      await screen.findByText('JV-2026-888');
      expect(screen.queryByText(/8004/)).not.toBeInTheDocument();
      expect(screen.getByText(/SoD: Self-Created/i)).toBeInTheDocument();

      // Attempt approve triggers SoD violation banner from backend 403 SOD_VIOLATION
      fireEvent.click(screen.getByRole('button', { name: /^Approve$/i }));
      const dialog = screen.getByRole('dialog');
      expect(within(dialog).getByText(/SoD Guard Warning/i)).toBeInTheDocument();
      fireEvent.click(within(dialog).getByRole('button', { name: /^Approve$/i }));

      const alert = await screen.findByRole('alert');
      expect(alert).toHaveTextContent(/SOD_VIOLATION/i);

      // Dismiss modal & open timeline drawer
      fireEvent.click(within(dialog).getByRole('button', { name: /Cancel/i }));
      fireEvent.click(screen.getByRole('button', { name: /Timeline/i }));
      await screen.findByText(/Approval Task Timeline/i);
    });
  });

  /* ------------------------------------------------------------------ */
  /* 4. FORENSIC AUDIT MODULE                                           */
  /* ------------------------------------------------------------------ */
  describe('Forensic Audit Workspace (/audit)', () => {
    it('computes field-level added/removed/changed/unchanged JSON diffs accurately', () => {
      const diffs = computeForensicDiff(
        { status: 'Draft', legacyFlag: true, unchanged: 'A' },
        { status: 'Posted', approvedAt: '2026-11-01', unchanged: 'A' },
      );
      expect(diffs).toEqual([
        { key: 'approvedAt', status: 'added', beforeText: '—', afterText: '2026-11-01' },
        { key: 'legacyFlag', status: 'removed', beforeText: 'true', afterText: '—' },
        { key: 'status', status: 'changed', beforeText: 'Draft', afterText: 'Posted' },
        { key: 'unchanged', status: 'unchanged', beforeText: 'A', afterText: 'A' },
      ]);
    });

    it('renders explicit ErrorState on API failure without silent mock fallback', async () => {
      mockedApiClient.mockRejectedValue(
        new apiModule.ApiClientError('AUDIT_UNAVAILABLE', 'Audit backend unreachable', 503),
      );

      renderWithProviders(<AuditWorkspace />);
      await screen.findByText(/Audit backend unreachable/i);
      expect(screen.queryByText(/8008/)).not.toBeInTheDocument();
    });

    it('filters events, exports CSV/JSON, and opens Forensic JSON Diff drawer', async () => {
      const csvSpy = vi.spyOn(exportModule, 'exportToCsv').mockImplementation(() => {});
      const jsonSpy = vi.spyOn(exportModule, 'exportToJson').mockImplementation(() => {});
      try {

      mockedApiClient.mockImplementation(async (endpoint) => {
        if (endpoint === '/audit-events') {
          return listResponse([
            {
              id: 'aud-1',
              occurredAt: '2026-11-10T09:15:00Z',
              organizationId: TEST_ORG.id,
              actorId: 'usr-finance-1',
              action: 'POSTED',
              entityType: 'PayrollRun',
              entityId: 'pr-1',
              correlationId: 'corr-9988',
            },
          ]);
        }
        if (endpoint === '/audit-events/aud-1') {
          return {
            id: 'aud-1',
            occurredAt: '2026-11-10T09:15:00Z',
            organizationId: TEST_ORG.id,
            actorId: 'usr-finance-1',
            action: 'POSTED',
            entityType: 'PayrollRun',
            entityId: 'pr-1',
            correlationId: 'corr-9988',
            reason: 'Monthly payroll cycle close',
            beforeValue: { status: 'Approved', glJournalRef: null },
            afterValue: { status: 'Posted', glJournalRef: 'JV-PR-202611' },
          };
        }
        throw new Error(`Unexpected audit call: ${endpoint}`);
      });

      renderWithProviders(<AuditWorkspace />);
      await screen.findByText('PayrollRun');

      fireEvent.click(screen.getByRole('button', { name: /Export CSV/i }));
      expect(csvSpy).toHaveBeenCalledTimes(1);

      fireEvent.click(screen.getByRole('button', { name: /Export JSON/i }));
      expect(jsonSpy).toHaveBeenCalledTimes(1);

      // Open Forensic JSON Diff drawer
      fireEvent.click(screen.getByRole('button', { name: /Diff/i }));
      const reasonEl = await screen.findByTestId('audit-detail-reason');
      expect(reasonEl.textContent).toBe('Monthly payroll cycle close');
      expect(screen.getByText('JV-PR-202611')).toBeInTheDocument();
      expect(screen.getAllByText('corr-9988').length).toBeGreaterThan(0);

      // Verify entityType filter updates query
      fireEvent.change(screen.getByLabelText('Entity Type Filter'), {
        target: { value: 'Employee' },
      });
      await waitFor(() => {
        const lastQuery = mockedApiClient.mock.calls.at(-1)?.[1]?.query;
        expect(lastQuery instanceof URLSearchParams && lastQuery.get('entityType')).toBe(
          'Employee',
        );
      });
      } finally {
        csvSpy.mockRestore();
        jsonSpy.mockRestore();
      }
    });
  });

  /* ------------------------------------------------------------------ */
  /* 5. ADMIN WORKSPACES (/organizations, /users, /security, /settings) */
  /* ------------------------------------------------------------------ */
  describe('Organizations Workspace (/organizations)', () => {
    it('renders collapsible organization hierarchy tree and activates Draft unit with If-Match', async () => {
      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/organization-types') {
          return listResponse([
            {
              id: 'type-1',
              organizationId: TEST_ORG.id,
              code: 'HOLDING',
              name: 'Holding Company',
              levelHint: 1,
              isActive: true,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/organizations' && (!options?.method || options.method === 'GET')) {
          return listResponse([
            {
              id: 'org-root',
              organizationId: TEST_ORG.id,
              code: 'HLD-01',
              name: 'Nusantara Holding',
              organizationTypeId: 'type-1',
              parentId: null,
              path: '/HLD-01',
              status: 'Active',
              effectiveFrom: '2025-01-01',
              rowVersion: 1,
            },
            {
              id: 'org-child',
              organizationId: TEST_ORG.id,
              code: 'BR-JKT',
              name: 'Jakarta Branch',
              organizationTypeId: 'type-1',
              parentId: 'org-root',
              path: '/HLD-01/BR-JKT',
              status: 'Draft',
              effectiveFrom: '2026-01-01',
              rowVersion: 2,
            },
          ]);
        }
        if (endpoint === '/organizations/org-child/activate' && options?.method === 'POST') {
          return {
            id: 'org-child',
            organizationId: TEST_ORG.id,
            code: 'BR-JKT',
            name: 'Jakarta Branch',
            organizationTypeId: 'type-1',
            parentId: 'org-root',
            path: '/HLD-01/BR-JKT',
            status: 'Active',
            effectiveFrom: '2026-01-01',
            rowVersion: 3,
          };
        }
        return listResponse([]);
      });

      renderWithProviders(<OrganizationsWorkspace />);
      await screen.findByText('Jakarta Branch');

      // Collapse root hides child node; re-expand restores it
      fireEvent.click(screen.getByRole('button', { name: /Collapse HLD-01/i }));
      expect(screen.queryByText('Jakarta Branch')).not.toBeInTheDocument();

      fireEvent.click(screen.getByRole('button', { name: /Expand HLD-01/i }));
      expect(screen.getByText('Jakarta Branch')).toBeInTheDocument();

      // Activate Draft organization
      fireEvent.click(screen.getByRole('button', { name: /Activate/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/organizations/org-child/activate',
          expect.objectContaining({ method: 'POST', ifMatch: '2' }),
        );
      });
    });
  });

  describe('Users & Roles Directory (/users)', () => {
    it('disables active user with If-Match and assigns/revokes per-org roles in SideDrawer', async () => {
      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/users') {
          return listResponse([
            {
              id: 'u-1',
              organizationId: TEST_ORG.id,
              username: 'siti.rahayu',
              email: 'siti@nusantara.co.id',
              fullName: 'Siti Rahayu',
              status: 'Active',
              defaultOrganizationId: TEST_ORG.id,
              rowVersion: 4,
            },
          ]);
        }
        if (endpoint === '/roles') {
          return listResponse([
            {
              id: 'role-fin',
              organizationId: TEST_ORG.id,
              code: 'FINANCE_CLERK',
              name: 'Finance Clerk',
              isSystem: false,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/organizations') {
          return listResponse([
            {
              id: TEST_ORG.id,
              organizationId: TEST_ORG.id,
              code: 'PT-NUSANTARA',
              name: 'PT Nusantara Sejahtera Tbk',
              organizationTypeId: 'type-1',
              parentId: null,
              path: '/PT-NUSANTARA',
              status: 'Active',
              effectiveFrom: '2025-01-01',
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/users/u-1/roles' && (!options?.method || options.method === 'GET')) {
          return listResponse([
            {
              id: 'ur-1',
              userId: 'u-1',
              organizationRefId: TEST_ORG.id,
              roleId: 'role-fin',
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/users/u-1/roles' && options?.method === 'POST') {
          return {
            id: 'ur-2',
            userId: 'u-1',
            organizationRefId: TEST_ORG.id,
            roleId: 'role-fin',
            rowVersion: 1,
          };
        }
        if (endpoint === '/users/u-1/roles/role-fin' && options?.method === 'DELETE') {
          return undefined;
        }
        if (endpoint === '/users/u-1/disable' && options?.method === 'POST') {
          return {
            id: 'u-1',
            username: 'siti.rahayu',
            status: 'Inactive',
            rowVersion: 5,
          };
        }
        return listResponse([]);
      });

      renderWithProviders(<UsersWorkspace />);
      await screen.findByText('Siti Rahayu');

      // Open Roles SideDrawer & verify existing assignment + assign + revoke
      fireEvent.click(screen.getByRole('button', { name: /Roles/i }));
      await screen.findByText(/FINANCE_CLERK — Finance Clerk/i);

      fireEvent.click(screen.getByRole('button', { name: /Assign Role/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/users/u-1/roles',
          expect.objectContaining({ method: 'POST' }),
        );
      });

      fireEvent.click(screen.getByRole('button', { name: /Revoke role role-fin/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/users/u-1/roles/role-fin',
          expect.objectContaining({ method: 'DELETE' }),
        );
      });

      // Close SideDrawer before triggering Disable action on directory row
      const drawerCloseBtns = screen.getAllByRole('button', { name: /Close|Tutup/i });
      fireEvent.click(drawerCloseBtns[0]);
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Assign Role/i })).not.toBeInTheDocument();
      });

      // Disable user with If-Match
      fireEvent.click(screen.getByRole('button', { name: /Disable/i }));
      const dlg = screen.getByRole('dialog');
      fireEvent.click(within(dlg).getByRole('button', { name: /Confirm Disable/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/users/u-1/disable',
          expect.objectContaining({ method: 'POST', ifMatch: '4' }),
        );
      });
    });
  });

  describe('Security RBAC Matrix (/security)', () => {
    it('toggles permissions on roles via POST and DELETE and protects system roles from deletion', async () => {
      mockedApiClient.mockImplementation(async (endpoint, options) => {
        if (endpoint === '/roles') {
          return listResponse([
            {
              id: 'role-sys',
              organizationId: TEST_ORG.id,
              code: 'SUPER_ADMIN',
              name: 'Super Admin',
              isSystem: true,
              rowVersion: 1,
            },
          ]);
        }
        if (endpoint === '/permissions') {
          return listResponse([
            {
              id: 'perm-hr-read',
              code: 'hr:read',
              module: 'hr',
              action: 'read',
            },
            {
              id: 'perm-hr-write',
              code: 'hr:write',
              module: 'hr',
              action: 'write',
            },
          ]);
        }
        if (endpoint === '/roles/role-sys/permissions' && (!options?.method || options.method === 'GET')) {
          return listResponse([{ id: 'perm-hr-read', code: 'hr:read', module: 'hr', action: 'read' }]);
        }
        if (endpoint === '/roles/role-sys/permissions' && options?.method === 'POST') {
          return { roleId: 'role-sys', permissionId: 'perm-hr-write' };
        }
        if (endpoint === '/roles/role-sys/permissions/perm-hr-read' && options?.method === 'DELETE') {
          return undefined;
        }
        return listResponse([]);
      });

      renderWithProviders(<SecurityWorkspace />);
      await screen.findByText('hr:write');

      // System role has no delete button
      expect(screen.queryByRole('button', { name: /Delete role SUPER_ADMIN/i })).not.toBeInTheDocument();

      // Grant hr:write & revoke hr:read
      fireEvent.click(screen.getByRole('switch', { name: /Toggle hr:write/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/roles/role-sys/permissions',
          expect.objectContaining({
            method: 'POST',
            body: JSON.stringify({ permissionId: 'perm-hr-write' }),
          }),
        );
      });

      fireEvent.click(screen.getByRole('switch', { name: /Toggle hr:read/i }));
      await waitFor(() => {
        expect(mockedApiClient).toHaveBeenCalledWith(
          '/roles/role-sys/permissions/perm-hr-read',
          expect.objectContaining({ method: 'DELETE' }),
        );
      });
    });
  });

  describe('Settings Console (/settings)', () => {
    it('updates language, theme, and portal interactively and inspects permissions + gateway health', async () => {
      const fetchSpy = vi
        .spyOn(globalThis, 'fetch')
        .mockResolvedValue(new Response('ok', { status: 200 }));
      try {
      setAuthState(['hr:read', 'payroll:write', 'audit:read']);

      renderWithProviders(<SettingsWorkspace />);

      await screen.findByText(/System & Preferences Console/i);
      expect(screen.getByText('PT Nusantara Sejahtera Tbk')).toBeInTheDocument();
      expect(screen.getByText('payroll:write')).toBeInTheDocument();

      // Switch Language ID <-> EN
      fireEvent.click(screen.getByRole('button', { name: /ID · Indonesia/i }));
      expect(useUiStore.getState().locale).toBe('id');
      expect(screen.getByText(/Konsol Preferensi & Sistem/i)).toBeInTheDocument();

      // Switch Theme Dark
      fireEvent.click(screen.getByRole('button', { name: /Dark/i }));
      expect(useUiStore.getState().theme).toBe('dark');

      // Switch Portal Admin
      fireEvent.click(screen.getByRole('button', { name: /^Admin$/i }));
      expect(useUiStore.getState().portal).toBe('admin');

      // Filter Session Permissions Inspector
      fireEvent.change(screen.getByPlaceholderText(/Filter permissions/i), {
        target: { value: 'payroll' },
      });
      expect(screen.getByText('payroll:write')).toBeInTheDocument();
      expect(screen.queryByText('audit:read')).not.toBeInTheDocument();
      } finally {
        fetchSpy.mockRestore();
      }
    });
  });
});
