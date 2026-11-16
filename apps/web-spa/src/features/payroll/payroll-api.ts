import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface PayrollResultLine {
  id: string;
  componentCode: string;
  amount: string;
}

export interface PayrollStatutoryContribution {
  id: string;
  contributionType: string;
  employeeAmount: string;
  employerAmount: string;
}

export interface PayrollResult {
  id: string;
  employeeId: string;
  grossAmount: string;
  deductionAmount: string;
  taxAmount: string;
  netAmount: string;
  rowVersion: number;
  lines?: PayrollResultLine[];
  statutoryContributions?: PayrollStatutoryContribution[];
}

export interface PayrollRun {
  id: string;
  organizationId: string;
  runNo: string;
  periodMonth: number;
  periodYear: number;
  status: string;
  glJournalRef?: string | null;
  rowVersion: number;
}

export interface PayrollRunDetail extends PayrollRun {
  correlationId?: string | null;
  results?: PayrollResult[];
}

export interface PayrollRunListParams {
  page?: number;
  size?: number;
  status?: string;
}

export interface CalculateEmployeeInput {
  employeeId: string;
  basicSalary?: string;
  allowance?: string;
  bonus?: string;
  otherDeductions?: string;
  ptkp?: string;
  hasNpwp?: boolean;
}

export interface CalculatePayrollPayload {
  totalAmount?: string;
  employees?: CalculateEmployeeInput[];
}

export const PTKP_OPTIONS = [
  { value: 'TK/0', label: 'TK/0 — Single, 0 dependents' },
  { value: 'TK/1', label: 'TK/1 — Single, 1 dependent' },
  { value: 'TK/2', label: 'TK/2 — Single, 2 dependents' },
  { value: 'TK/3', label: 'TK/3 — Single, 3 dependents' },
  { value: 'K/0', label: 'K/0 — Married, 0 dependents' },
  { value: 'K/1', label: 'K/1 — Married, 1 dependent' },
  { value: 'K/2', label: 'K/2 — Married, 2 dependents' },
  { value: 'K/3', label: 'K/3 — Married, 3 dependents' },
] as const;

export const createPayrollRunSchema = z.object({
  runNo: z.string().trim().min(1, 'Run number is required').max(64),
  periodMonth: z.number().int().min(1, 'Month 1-12').max(12, 'Month 1-12'),
  periodYear: z.number().int().min(2000, 'Year >= 2000').max(2100, 'Year <= 2100'),
});

export type CreatePayrollRunValues = z.infer<typeof createPayrollRunSchema>;

export const payrollKeys = {
  all: ['payroll'] as const,
  runs: (orgId: string | undefined, params: PayrollRunListParams) =>
    ['payroll', 'runs', orgId, params] as const,
  detail: (orgId: string | undefined, runId: string | null) =>
    ['payroll', 'detail', orgId, runId] as const,
};

function buildQuery(params: Record<string, string | number | undefined>): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') {
      sp.set(k, String(v));
    }
  }
  return sp;
}

export function usePayrollRunsQuery(orgId: string | undefined, params: PayrollRunListParams) {
  return useQuery<{ items: PayrollRun[]; page: PageMeta }>({
    queryKey: payrollKeys.runs(orgId, params),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<PayrollRun> | PayrollRun[]>('/payroll-runs', {
        query: buildQuery({
          page: params.page ?? 1,
          size: params.size ?? 25,
          status: params.status && params.status !== 'ALL' ? params.status : undefined,
        }),
      });
      return unwrapListResponse<PayrollRun>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function usePayrollRunDetailQuery(orgId: string | undefined, runId: string | null) {
  return useQuery<PayrollRunDetail>({
    queryKey: payrollKeys.detail(orgId, runId),
    queryFn: () => apiClient<PayrollRunDetail>(`/payroll-runs/${runId}`),
    enabled: Boolean(orgId && runId),
  });
}

export function useCreatePayrollRunMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreatePayrollRunValues) =>
      apiClient<PayrollRunDetail>('/payroll-runs', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          runNo: values.runNo.trim(),
          periodMonth: values.periodMonth,
          periodYear: values.periodYear,
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: payrollKeys.all });
    },
  });
}

/**
 * Strictly builds payload with only fields allowed by Go's DisallowUnknownFields().
 */
export function serializeCalculatePayload(payload: CalculatePayrollPayload): Record<string, unknown> {
  const clean: Record<string, unknown> = {};
  if (payload.totalAmount && payload.totalAmount.trim() !== '') {
    clean.totalAmount = payload.totalAmount.trim();
  }
  if (payload.employees && payload.employees.length > 0) {
    clean.employees = payload.employees.map((emp) => {
      const item: Record<string, unknown> = {
        employeeId: emp.employeeId.trim(),
      };
      if (emp.basicSalary !== undefined && emp.basicSalary !== '') {
        item.basicSalary = emp.basicSalary.trim();
      }
      if (emp.allowance !== undefined && emp.allowance !== '') {
        item.allowance = emp.allowance.trim();
      }
      if (emp.bonus !== undefined && emp.bonus !== '') {
        item.bonus = emp.bonus.trim();
      }
      if (emp.otherDeductions !== undefined && emp.otherDeductions !== '') {
        item.otherDeductions = emp.otherDeductions.trim();
      }
      if (emp.ptkp) {
        item.ptkp = emp.ptkp;
      }
      if (typeof emp.hasNpwp === 'boolean') {
        item.hasNpwp = emp.hasNpwp;
      }
      return item;
    });
  }
  return clean;
}

export function useCalculatePayrollMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      payload,
    }: {
      id: string;
      rowVersion: number;
      payload: CalculatePayrollPayload;
    }) => {
      const bodyObj = serializeCalculatePayload(payload);
      return apiClient<PayrollRunDetail>(`/payroll-runs/${id}/calculate`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(bodyObj),
      });
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: payrollKeys.all });
    },
  });
}

export function useApprovePayrollMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<PayrollRunDetail>(`/payroll-runs/${id}/approve`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: payrollKeys.all });
    },
  });
}

export function usePostPayrollMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      glJournalRef,
    }: {
      id: string;
      rowVersion: number;
      glJournalRef?: string;
    }) => {
      const cleanBody: Record<string, string> = {};
      if (glJournalRef && glJournalRef.trim() !== '') {
        cleanBody.glJournalRef = glJournalRef.trim();
      }
      return apiClient<PayrollRunDetail>(`/payroll-runs/${id}/post`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(cleanBody),
      });
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: payrollKeys.all });
    },
  });
}

export function useDeletePayrollRunMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<void>(`/payroll-runs/${id}`, {
        method: 'DELETE',
        ifMatch: String(rowVersion),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: payrollKeys.all });
    },
  });
}
