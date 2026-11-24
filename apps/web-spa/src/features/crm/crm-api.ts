import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface LeadDto {
  id: string;
  code: string;
  name: string;
  source?: string | null;
  status: string;
  estimatedValue?: string | null;
  customerRef?: string | null;
  rowVersion: number;
}

export interface OpportunityDto {
  id: string;
  code: string;
  leadId?: string | null;
  customerRef?: string | null;
  stage: string;
  probabilityPct?: string | null;
  estimatedValue?: string | null;
  expectedCloseDate?: string | null;
  status: string;
  rowVersion: number;
}

export interface CustomerSegmentDto {
  id: string;
  code: string;
  name: string;
  criteria: Record<string, unknown>;
  isSystem?: boolean;
  rowVersion: number;
}

export interface SupportCaseDto {
  id: string;
  caseNo: string;
  subject: string;
  status: string;
  resolution?: string | null;
  rowVersion: number;
}

export function useLeadsQuery(orgId?: string, status?: string) {
  return useQuery<{ items: LeadDto[]; page: PageMeta }>({
    queryKey: ['crm-leads', orgId, status ?? 'ALL'],
    queryFn: async () => {
      const sp = new URLSearchParams({ page: '1', size: '50' });
      if (status && status !== 'ALL') sp.set('status', status);
      const raw = await apiClient<ListEnvelope<LeadDto> | LeadDto[]>('/leads', { query: sp });
      return unwrapListResponse<LeadDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useOpportunitiesQuery(orgId?: string) {
  return useQuery<{ items: OpportunityDto[]; page: PageMeta }>({
    queryKey: ['crm-opportunities', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<OpportunityDto> | OpportunityDto[]>('/opportunities');
      return unwrapListResponse<OpportunityDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCustomerSegmentsQuery(orgId?: string) {
  return useQuery<{ items: CustomerSegmentDto[]; page: PageMeta }>({
    queryKey: ['crm-segments', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<CustomerSegmentDto> | CustomerSegmentDto[]>('/customer-segments');
      return unwrapListResponse<CustomerSegmentDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useSupportCasesQuery(orgId?: string) {
  return useQuery<{ items: SupportCaseDto[]; page: PageMeta }>({
    queryKey: ['crm-support-cases', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<SupportCaseDto> | SupportCaseDto[]>('/support-cases');
      return unwrapListResponse<SupportCaseDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateLeadMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { code: string; name: string; source?: string; estimatedValue?: string }) =>
      apiClient<LeadDto>('/leads', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['crm-leads'] }),
  });
}

export function useTransitionLeadMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion }: { id: string; action: 'qualify' | 'convert' | 'lose'; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/leads/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['crm-leads'] });
      void qc.invalidateQueries({ queryKey: ['crm-opportunities'] });
    },
  });
}

export function useCloseOpportunityMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion, currencyCode }: { id: string; action: 'win' | 'lose'; rowVersion: number; currencyCode?: string }) =>
      apiClient<{ id: string; status: string }>(`/opportunities/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(action === 'win' ? { currencyCode: currencyCode ?? 'IDR' } : {}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['crm-opportunities'] }),
  });
}

export function useTransitionSupportCaseMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion, resolution }: { id: string; action: 'escalate' | 'resolve' | 'close'; rowVersion: number; resolution?: string }) =>
      apiClient<{ id: string; status: string }>(`/support-cases/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(action === 'resolve' ? { resolution: resolution ?? 'Resolved by support team' } : {}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['crm-support-cases'] }),
  });
}
