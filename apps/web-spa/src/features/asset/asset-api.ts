import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface OperationalAssetDto {
  id: string;
  organizationId: string;
  assetCode: string;
  name: string;
  category: string;
  specification: Record<string, unknown>;
  acquisitionDate?: string | null;
  organizationRef: string;
  barcode?: string | null;
  status: string;
  isSystem?: boolean;
  rowVersion: number;
}

export interface AssetMovementDto {
  id: string;
  assetId: string;
  fromOrganizationRef?: string | null;
  toOrganizationRef: string;
  status: string;
  movedAt?: string | null;
  rowVersion: number;
}

export interface AssetVerificationDto {
  id: string;
  assetId: string;
  verificationDate: string;
  systemStatus?: string | null;
  physicalStatus?: string | null;
  discrepancyNote?: string | null;
}

export function useOperationalAssetsQuery(orgId?: string, status?: string) {
  return useQuery<{ items: OperationalAssetDto[]; page: PageMeta }>({
    queryKey: ['operational-assets', orgId, status ?? 'ALL'],
    queryFn: async () => {
      const sp = new URLSearchParams({ page: '1', size: '50' });
      if (status && status !== 'ALL') sp.set('status', status);
      const raw = await apiClient<ListEnvelope<OperationalAssetDto> | OperationalAssetDto[]>('/assets', { query: sp });
      return unwrapListResponse<OperationalAssetDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useAssetMovementsQuery(orgId?: string) {
  return useQuery<{ items: AssetMovementDto[]; page: PageMeta }>({
    queryKey: ['asset-movements', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<AssetMovementDto> | AssetMovementDto[]>('/asset-movements');
      return unwrapListResponse<AssetMovementDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useAssetVerificationsQuery(orgId?: string) {
  return useQuery<{ items: AssetVerificationDto[]; page: PageMeta }>({
    queryKey: ['asset-verifications', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<AssetVerificationDto> | AssetVerificationDto[]>('/asset-verifications');
      return unwrapListResponse<AssetVerificationDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateOperationalAssetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      assetCode: string;
      name: string;
      category: string;
      barcode?: string;
      status: string;
    }) =>
      apiClient<OperationalAssetDto>('/assets', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['operational-assets'] }),
  });
}

export function useTransitionMovementMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, rowVersion }: { id: string; action: 'approve' | 'complete'; rowVersion: number }) =>
      apiClient<{ id: string; status: string }>(`/asset-movements/${id}/${action}`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({}),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['asset-movements'] });
      void qc.invalidateQueries({ queryKey: ['operational-assets'] });
    },
  });
}
