import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface ConsolidationGroupDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  parentGroupId?: string | null;
  presentationCurrencyCode: string;
  isSystem?: boolean;
  rowVersion: number;
}

export interface ConsolidationMemberDto {
  id: string;
  consolidationGroupId: string;
  companyRef: string;
  ownershipPercentage: string;
  consolidationMethod: string;
  rowVersion: number;
}

export interface EliminationRuleDto {
  id: string;
  consolidationGroupId: string;
  ruleType: string;
  config: Record<string, unknown>;
  isActive: boolean;
  rowVersion: number;
}

export interface ConsolidationRunDto {
  id: string;
  consolidationGroupId: string;
  fiscalPeriodRef: string;
  version: number;
  status: string;
  rowVersion: number;
}

export function useConsolidationGroupsQuery(orgId?: string) {
  return useQuery<{ items: ConsolidationGroupDto[]; page: PageMeta }>({
    queryKey: ['consolidation-groups', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ConsolidationGroupDto> | ConsolidationGroupDto[]>('/consolidation-groups');
      return unwrapListResponse<ConsolidationGroupDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useGroupMembersQuery(orgId?: string, groupId?: string | null) {
  return useQuery<ConsolidationMemberDto[]>({
    queryKey: ['consolidation-groups', 'members', orgId, groupId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ConsolidationMemberDto> | ConsolidationMemberDto[]>(
        `/consolidation-groups/${groupId}/members`
      );
      return unwrapListResponse<ConsolidationMemberDto>(raw).items;
    },
    enabled: Boolean(orgId && groupId),
  });
}

export function useEliminationRulesQuery(orgId?: string) {
  return useQuery<{ items: EliminationRuleDto[]; page: PageMeta }>({
    queryKey: ['elimination-rules', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<EliminationRuleDto> | EliminationRuleDto[]>('/elimination-rules');
      return unwrapListResponse<EliminationRuleDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useConsolidationRunsQuery(orgId?: string) {
  return useQuery<{ items: ConsolidationRunDto[]; page: PageMeta }>({
    queryKey: ['consolidation-runs', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ConsolidationRunDto> | ConsolidationRunDto[]>('/consolidation-runs');
      return unwrapListResponse<ConsolidationRunDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateConsolidationGroupMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { code: string; name: string; presentationCurrencyCode: string }) =>
      apiClient<ConsolidationGroupDto>('/consolidation-groups', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['consolidation-groups'] }),
  });
}

export function useCreateConsolidationRunMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { consolidationGroupId: string; fiscalPeriodRef: string; version: number }) =>
      apiClient<ConsolidationRunDto>('/consolidation-runs', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['consolidation-runs'] }),
  });
}

export function useFinalizeConsolidationRunMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/consolidation-runs/${id}/finalize`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['consolidation-runs'] }),
  });
}
