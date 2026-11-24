import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface AssetCategoryDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  defaultDepreciationMethod: string;
  defaultUsefulLifeMonths?: number | null;
  isActive: boolean;
  isSystem: boolean;
  rowVersion: number;
}

export interface FixedAssetDto {
  id: string;
  organizationId: string;
  assetNo: string;
  name: string;
  assetCategoryId: string;
  financialOrganizationRef: string;
  acquisitionDate: string;
  acquisitionCost: string;
  salvageValue: string;
  usefulLifeMonths: number;
  depreciationMethod: string;
  accumulatedDepreciation: string;
  netBookValue: string;
  status: string;
  assetAccountId: string;
  accumDepreciationAccountId: string;
  depreciationExpenseAccountId: string;
  costCenterRef?: string | null;
  rowVersion: number;
}

export interface DepreciationEntryDto {
  id: string;
  fixedAssetId: string;
  fiscalPeriodRef: string;
  runDate: string;
  depreciationAmount: string;
  accumulatedAfter: string;
  nbvAfter: string;
  status: string;
}

export interface AssetTransactionDto {
  id: string;
  fixedAssetId: string;
  transactionType: string;
  transactionDate: string;
  amount: string;
  proceedsAmount?: string | null;
  gainLossAmount?: string | null;
}

export const fixedAssetKeys = {
  all: ['fixed-assets'] as const,
  categories: (orgId?: string) => ['asset-categories', orgId] as const,
  assets: (orgId?: string, status?: string, page = 1) =>
    ['fixed-assets', orgId, status ?? 'ALL', page] as const,
  depreciations: (orgId?: string, assetId?: string | null) =>
    ['fixed-assets', 'depreciations', orgId, assetId] as const,
  transactions: (orgId?: string, assetId?: string | null) =>
    ['fixed-assets', 'transactions', orgId, assetId] as const,
};

export function useAssetCategoriesQuery(orgId?: string) {
  return useQuery<{ items: AssetCategoryDto[]; page: PageMeta }>({
    queryKey: fixedAssetKeys.categories(orgId),
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<AssetCategoryDto> | AssetCategoryDto[]>('/asset-categories', {
        query: new URLSearchParams({ page: '1', size: '100' }),
      });
      return unwrapListResponse<AssetCategoryDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useFixedAssetsQuery(orgId?: string, status?: string, page = 1, size = 25) {
  return useQuery<{ items: FixedAssetDto[]; page: PageMeta }>({
    queryKey: fixedAssetKeys.assets(orgId, status, page),
    queryFn: async () => {
      const sp = new URLSearchParams({ page: String(page), size: String(size) });
      if (status && status !== 'ALL') sp.set('status', status);
      const raw = await apiClient<ListEnvelope<FixedAssetDto> | FixedAssetDto[]>('/fixed-assets', { query: sp });
      return unwrapListResponse<FixedAssetDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useAssetDepreciationsQuery(orgId?: string, assetId?: string | null) {
  return useQuery<DepreciationEntryDto[]>({
    queryKey: fixedAssetKeys.depreciations(orgId, assetId),
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<DepreciationEntryDto> | DepreciationEntryDto[]>(
        `/fixed-assets/${assetId}/depreciation-entries`
      );
      return unwrapListResponse<DepreciationEntryDto>(raw).items;
    },
    enabled: Boolean(orgId && assetId),
  });
}

export function useAssetTransactionsQuery(orgId?: string, assetId?: string | null) {
  return useQuery<AssetTransactionDto[]>({
    queryKey: fixedAssetKeys.transactions(orgId, assetId),
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<AssetTransactionDto> | AssetTransactionDto[]>(
        `/fixed-assets/${assetId}/transactions`
      );
      return unwrapListResponse<AssetTransactionDto>(raw).items;
    },
    enabled: Boolean(orgId && assetId),
  });
}

export function useCreateCategoryMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      code: string;
      name: string;
      defaultDepreciationMethod: string;
      defaultUsefulLifeMonths?: number;
    }) =>
      apiClient<AssetCategoryDto>('/asset-categories', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['asset-categories'] });
    },
  });
}

export function useCreateFixedAssetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      assetNo: string;
      name: string;
      assetCategoryId: string;
      financialOrganizationRef: string;
      acquisitionDate: string;
      acquisitionCost: string;
      salvageValue: string;
      usefulLifeMonths: number;
      depreciationMethod: string;
      assetAccountId: string;
      accumDepreciationAccountId: string;
      depreciationExpenseAccountId: string;
    }) =>
      apiClient<FixedAssetDto>('/fixed-assets', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: fixedAssetKeys.all });
    },
  });
}

export function useCapitalizeAssetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion, transactionDate }: { id: string; rowVersion: number; transactionDate: string }) =>
      apiClient<{ id: string; status: string; rowVersion: number }>(`/fixed-assets/${id}/capitalize`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({ transactionDate }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: fixedAssetKeys.all });
    },
  });
}

export function useDepreciateAssetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      fiscalPeriodRef,
      runDate,
    }: {
      id: string;
      rowVersion: number;
      fiscalPeriodRef: string;
      runDate: string;
    }) =>
      apiClient<{ id: string; depreciationAmount: string; rowVersion: number }>(`/fixed-assets/${id}/depreciate`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({ fiscalPeriodRef, runDate }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: fixedAssetKeys.all });
    },
  });
}

export function useDisposeAssetMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      transactionDate,
      proceedsAmount,
    }: {
      id: string;
      rowVersion: number;
      transactionDate: string;
      proceedsAmount: string;
    }) =>
      apiClient<{ id: string; status: string; rowVersion: number }>(`/fixed-assets/${id}/dispose`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({ transactionDate, proceedsAmount }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: fixedAssetKeys.all });
    },
  });
}
