import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface ProjectDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  projectType: string;
  currencyCode: string;
  startDate?: string | null;
  endDate?: string | null;
  status: string;
  isSystem?: boolean;
  rowVersion: number;
}

export interface WbsElementDto {
  id: string;
  projectId: string;
  parentId?: string | null;
  code: string;
  name: string;
  levelNo: number;
  rowVersion: number;
}

export interface ProjectBillingDto {
  id: string;
  projectId: string;
  billingType: string;
  amount: string;
  arInvoiceRef?: string | null;
  status: string;
  rowVersion: number;
}

export function useProjectsQuery(orgId?: string, status?: string) {
  return useQuery<{ items: ProjectDto[]; page: PageMeta }>({
    queryKey: ['projects', orgId, status ?? 'ALL'],
    queryFn: async () => {
      const sp = new URLSearchParams({ page: '1', size: '50' });
      if (status && status !== 'ALL') sp.set('status', status);
      const raw = await apiClient<ListEnvelope<ProjectDto> | ProjectDto[]>('/projects', { query: sp });
      return unwrapListResponse<ProjectDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useProjectWbsQuery(orgId?: string, projectId?: string | null) {
  return useQuery<WbsElementDto[]>({
    queryKey: ['projects', 'wbs', orgId, projectId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<WbsElementDto> | WbsElementDto[]>(`/projects/${projectId}/wbs-elements`);
      return unwrapListResponse<WbsElementDto>(raw).items;
    },
    enabled: Boolean(orgId && projectId),
  });
}

export function useProjectBillingsQuery(orgId?: string) {
  return useQuery<{ items: ProjectBillingDto[]; page: PageMeta }>({
    queryKey: ['project-billings', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<ProjectBillingDto> | ProjectBillingDto[]>('/project-billings');
      return unwrapListResponse<ProjectBillingDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateProjectMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      code: string;
      name: string;
      projectType: string;
      currencyCode: string;
      startDate?: string;
      endDate?: string;
    }) =>
      apiClient<ProjectDto>('/projects', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['projects'] }),
  });
}

export function useTransitionProjectMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion }: { id: string; action: 'activate' | 'hold' | 'close'; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/projects/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['projects'] }),
  });
}

export function useApproveProjectBillingMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<{ id: string; status: string; arInvoiceRef: string }>(`/project-billings/${id}/approve`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['project-billings'] }),
  });
}
