import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface CostCenterDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  parentId?: string | null;
  status: string;
  rowVersion: number;
}

export interface ProfitCenterDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  parentId?: string | null;
  status: string;
  rowVersion: number;
}

export interface AllocationTargetInput {
  costCenterId: string;
  weightPct: string;
}

export interface AllocationRuleDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  basis: string;
  isActive: boolean;
  targets?: AllocationTargetInput[];
  rowVersion: number;
}

export function useCostCentersQuery(orgId?: string) {
  return useQuery<{ items: CostCenterDto[]; page: PageMeta }>({
    queryKey: ['cost-centers', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<CostCenterDto> | CostCenterDto[]>('/cost-centers', {
        query: new URLSearchParams({ page: '1', size: '100' }),
      });
      return unwrapListResponse<CostCenterDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useProfitCentersQuery(orgId?: string) {
  return useQuery<{ items: ProfitCenterDto[]; page: PageMeta }>({
    queryKey: ['profit-centers', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ProfitCenterDto> | ProfitCenterDto[]>('/profit-centers', {
        query: new URLSearchParams({ page: '1', size: '100' }),
      });
      return unwrapListResponse<ProfitCenterDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useAllocationRulesQuery(orgId?: string) {
  return useQuery<{ items: AllocationRuleDto[]; page: PageMeta }>({
    queryKey: ['allocation-rules', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<AllocationRuleDto> | AllocationRuleDto[]>('/allocation-rules', {
        query: new URLSearchParams({ page: '1', size: '100' }),
      });
      return unwrapListResponse<AllocationRuleDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateCostCenterMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { code: string; name: string; parentId?: string }) =>
      apiClient<CostCenterDto>('/cost-centers', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['cost-centers'] }),
  });
}

export function useCreateProfitCenterMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { code: string; name: string; parentId?: string }) =>
      apiClient<ProfitCenterDto>('/profit-centers', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['profit-centers'] }),
  });
}

export function useCreateAllocationRuleMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { code: string; name: string; basis: string; targets: AllocationTargetInput[] }) =>
      apiClient<AllocationRuleDto>('/allocation-rules', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['allocation-rules'] }),
  });
}

export function useExecuteAllocationRuleMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, amount, periodDate }: { id: string; amount: string; periodDate: string }) =>
      apiClient<{ ruleId: string; totalAmount: string }>(`/allocation-rules/${id}/execute`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({ amount, periodDate }),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['allocation-rules'] }),
  });
}
