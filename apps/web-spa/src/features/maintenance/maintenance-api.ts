import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface MaintenanceScheduleDto {
  id: string;
  assetRef: string;
  scheduleType: string;
  intervalValue: number;
  intervalUnit: string;
  nextDueDate?: string | null;
  isActive: boolean;
  isSystem?: boolean;
  rowVersion: number;
}

export interface MaintenanceWorkOrderDto {
  id: string;
  woNo: string;
  assetRef: string;
  maintenanceType: string;
  status: string;
  scheduledDate?: string | null;
  rowVersion: number;
}

export function useMaintenanceSchedulesQuery(orgId?: string) {
  return useQuery<{ items: MaintenanceScheduleDto[]; page: PageMeta }>({
    queryKey: ['maintenance-schedules', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<MaintenanceScheduleDto> | MaintenanceScheduleDto[]>('/maintenance-schedules');
      return unwrapListResponse<MaintenanceScheduleDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useMaintenanceWorkOrdersQuery(orgId?: string, status?: string) {
  return useQuery<{ items: MaintenanceWorkOrderDto[]; page: PageMeta }>({
    queryKey: ['maintenance-work-orders', orgId, status ?? 'ALL'],
    queryFn: async () => {
      const sp = new URLSearchParams({ page: '1', size: '50' });
      if (status && status !== 'ALL') sp.set('status', status);
      const raw = await apiClient<ListEnvelope<MaintenanceWorkOrderDto> | MaintenanceWorkOrderDto[]>('/maintenance-work-orders', {
        query: sp,
      });
      return unwrapListResponse<MaintenanceWorkOrderDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateMaintenanceWorkOrderMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { woNo: string; assetRef: string; maintenanceType: string; scheduledDate?: string }) =>
      apiClient<MaintenanceWorkOrderDto>('/maintenance-work-orders', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['maintenance-work-orders'] }),
  });
}

export function useTransitionWorkOrderMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      action,
      rowVersion,
      body,
    }: {
      id: string;
      action: 'assign' | 'start' | 'complete' | 'close';
      rowVersion: number;
      body?: Record<string, unknown>;
    }) =>
      apiClient<{ id: string; status: string; totalCost?: string }>(`/maintenance-work-orders/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(body ?? {}),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['maintenance-work-orders'] });
      void qc.invalidateQueries({ queryKey: ['maintenance-schedules'] });
    },
  });
}
