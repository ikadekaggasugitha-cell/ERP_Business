import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface ClosingTemplateDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  isSystem?: boolean;
  rowVersion: number;
}

export interface ClosingRunDto {
  id: string;
  organizationId: string;
  financialOrganizationRef: string;
  fiscalPeriodRef: string;
  runType: string;
  status: string;
  startedAt?: string | null;
  completedAt?: string | null;
  rowVersion: number;
}

export interface ClosingTaskDto {
  id: string;
  closingRunId: string;
  taskOrder: number;
  taskName: string;
  isMandatory: boolean;
  status: string;
  rowVersion: number;
}

export interface PeriodStatusDto {
  id: string;
  fiscalPeriodRef: string;
  module: string;
  status: string;
  rowVersion: number;
}

export interface PeriodReopeningDto {
  id: string;
  fiscalPeriodRef: string;
  reason: string;
  status: string;
  rowVersion: number;
}

export function useClosingTemplatesQuery(orgId?: string) {
  return useQuery<{ items: ClosingTemplateDto[]; page: PageMeta }>({
    queryKey: ['closing-templates', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ClosingTemplateDto> | ClosingTemplateDto[]>('/closing-templates');
      return unwrapListResponse<ClosingTemplateDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useClosingRunsQuery(orgId?: string) {
  return useQuery<{ items: ClosingRunDto[]; page: PageMeta }>({
    queryKey: ['closing-runs', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ClosingRunDto> | ClosingRunDto[]>('/closing-runs');
      return unwrapListResponse<ClosingRunDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useClosingRunDetailQuery(orgId?: string, runId?: string | null) {
  return useQuery<{ run: ClosingRunDto; tasks: ClosingTaskDto[] }>({
    queryKey: ['closing-runs', 'detail', orgId, runId],
    queryFn: () => apiClient<{ run: ClosingRunDto; tasks: ClosingTaskDto[] }>(`/closing-runs/${runId}`),
    enabled: Boolean(orgId && runId),
  });
}

export function usePeriodStatusesQuery(orgId?: string) {
  return useQuery<{ items: PeriodStatusDto[]; page: PageMeta }>({
    queryKey: ['period-statuses', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<PeriodStatusDto> | PeriodStatusDto[]>('/period-statuses');
      return unwrapListResponse<PeriodStatusDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function usePeriodReopeningsQuery(orgId?: string) {
  return useQuery<{ items: PeriodReopeningDto[]; page: PageMeta }>({
    queryKey: ['period-reopenings', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<PeriodReopeningDto> | PeriodReopeningDto[]>('/period-reopenings');
      return unwrapListResponse<PeriodReopeningDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateClosingRunMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      financialOrganizationRef: string;
      fiscalPeriodRef: string;
      runType: string;
      templateId?: string;
    }) =>
      apiClient<ClosingRunDto>('/closing-runs', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['closing-runs'] }),
  });
}

export function useCompleteClosingTaskMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/closing-tasks/${id}/complete`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['closing-runs'] }),
  });
}

export function useTransitionClosingRunMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion }: { id: string; action: 'close' | 'lock'; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/closing-runs/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['closing-runs'] });
      void qc.invalidateQueries({ queryKey: ['period-statuses'] });
    },
  });
}

export function useRequestReopeningMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { financialOrganizationRef: string; fiscalPeriodRef: string; reason: string }) =>
      apiClient<PeriodReopeningDto>('/period-reopenings', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['period-reopenings'] }),
  });
}

export function useApproveReopeningMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/period-reopenings/${id}/approve`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['period-reopenings'] });
      void qc.invalidateQueries({ queryKey: ['period-statuses'] });
    },
  });
}
