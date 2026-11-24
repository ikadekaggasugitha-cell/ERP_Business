import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface LanguageDto {
  id: string;
  code: string;
  name: string;
  isActive: boolean;
  isFallback: boolean;
  isSystem: boolean;
  rowVersion: number;
}

export interface TranslationDto {
  id: string;
  languageCode: string;
  resourceKey: string;
  resourceValue: string;
  module?: string | null;
  isSystem: boolean;
  rowVersion: number;
}

export interface LocaleFormatDto {
  id: string;
  localeCode: string;
  dateFormat: string;
  timeFormat?: string;
  decimalSeparator: string;
  thousandSeparator: string;
  isSystem: boolean;
  rowVersion: number;
}

export function useLanguagesQuery(orgId?: string) {
  return useQuery<{ items: LanguageDto[]; page: PageMeta }>({
    queryKey: ['localization-languages', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<LanguageDto> | LanguageDto[]>('/languages');
      return unwrapListResponse<LanguageDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useTranslationsQuery(orgId?: string, languageCode?: string) {
  return useQuery<{ items: TranslationDto[]; page: PageMeta }>({
    queryKey: ['localization-translations', orgId, languageCode ?? 'ALL'],
    queryFn: async () => {
      const sp = new URLSearchParams({ page: '1', size: '100' });
      if (languageCode && languageCode !== 'ALL') sp.set('languageCode', languageCode);
      const raw = await apiClient<ListEnvelope<TranslationDto> | TranslationDto[]>('/translations', { query: sp });
      return unwrapListResponse<TranslationDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useLocaleFormatsQuery(orgId?: string) {
  return useQuery<{ items: LocaleFormatDto[]; page: PageMeta }>({
    queryKey: ['localization-formats', orgId],
    queryFn: async () => {
      const raw = await apiClient<ListEnvelope<LocaleFormatDto> | LocaleFormatDto[]>('/locale-formats');
      return unwrapListResponse<LocaleFormatDto>(raw);
    },
    enabled: Boolean(orgId),
  });
}

export function useCreateLanguageMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { code: string; name: string; isActive?: boolean; isFallback?: boolean }) =>
      apiClient<LanguageDto>('/languages', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['localization-languages'] }),
  });
}

export function useCreateTranslationMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { languageCode: string; resourceKey: string; resourceValue: string; module?: string }) =>
      apiClient<TranslationDto>('/translations', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(payload),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['localization-translations'] }),
  });
}
