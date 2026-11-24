import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface BudgetDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  financialOrganizationRef: string;
  fiscalYearRef: string;
  version: number;
  status: string;
  isActive: boolean;
  rowVersion: number;
}

export interface BudgetLineDto {
  id: string;
  budgetId: string;
  accountRef: string;
  periodNo: number;
  budgetAmount: string;
}

export interface BudgetPolicyDto {
  id: string;
  organizationId: string;
  accountRef?: string | null;
  transactionType?: string | null;
  controlType: string;
  isSystem?: boolean;
  rowVersion: number;
}

export interface BudgetCommitmentDto {
  id: string;
  budgetId: string;
  sourceDocumentRef: string;
  committedAmount: string;
  status: string;
  rowVersion: number;
}

export function useBudgetsQuery(orgId?: string, status?: string, page = 1) {
  return useQuery<{ items: BudgetDto[]; page: PageMeta }>({
    queryKey: ['budgets', orgId, status ?? 'ALL', page],
    queryFn: async () => {
      const sp = new URLSearchParams({ page: String(page), size: '25' });
      if (status && status !== 'ALL') sp.set('status', status);
      const raw = await apiClient<ListEnvelope<BudgetDto> | BudgetDto[]>('/budgets', { query: sp });
      return unwrapListResponse<BudgetDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useBudgetDetailQuery(orgId?: string, budgetId?: string | null) {
  return useQuery<{ budget: BudgetDto; lines: BudgetLineDto[] }>({
    queryKey: ['budgets', 'detail', orgId, budgetId],
    queryFn: () => apiClient<{ budget: BudgetDto; lines: BudgetLineDto[] }>(`/budgets/${budgetId}`),
    enabled: Boolean(orgId && budgetId),
  });
}

export function useBudgetPoliciesQuery(orgId?: string) {
  return useQuery<{ items: BudgetPolicyDto[]; page: PageMeta }>({
    queryKey: ['budget-policies', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<BudgetPolicyDto> | BudgetPolicyDto[]>('/budget-policies');
      return unwrapListResponse<BudgetPolicyDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useBudgetCommitmentsQuery(orgId?: string) {
  return useQuery<{ items: BudgetCommitmentDto[]; page: PageMeta }>({
    queryKey: ['budget-commitments', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<BudgetCommitmentDto> | BudgetCommitmentDto[]>('/budget-commitments');
      return unwrapListResponse<BudgetCommitmentDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateBudgetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      code: string;
      name: string;
      financialOrganizationRef: string;
      fiscalYearRef: string;
      version: number;
    }) =>
      apiClient<BudgetDto>('/budgets', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['budgets'] }),
  });
}

export function useTransitionBudgetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion }: { id: string; action: 'submit' | 'approve' | 'close'; rowVersion: number }) =>
      apiClient<{ id: string; status: string; rowVersion: number }>(`/budgets/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['budgets'] }),
  });
}

export function useCreatePolicyMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { transactionType?: string; controlType: string }) =>
      apiClient<BudgetPolicyDto>('/budget-policies', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['budget-policies'] }),
  });
}

export function useAddBudgetLineMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      budgetId,
      accountRef,
      periodNo,
      budgetAmount,
    }: {
      budgetId: string;
      accountRef: string;
      periodNo: number;
      budgetAmount: string;
    }) =>
      apiClient<BudgetLineDto>(`/budgets/${budgetId}/lines`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({ accountRef, periodNo, budgetAmount }),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['budgets'] }),
  });
}
