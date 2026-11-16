import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  ConvertCurrencyInput,
  ConvertCurrencyResultDto,
  CreateCurrencyInput,
  CreateExchangeRateInput,
  CreateExchangeRateTypeInput,
  CurrencyDto,
  ExchangeRateDto,
  ExchangeRateTypeDto,
  UpdateCurrencyInput,
  UpdateExchangeRateInput,
} from './types';

export interface PagedResult<T> {
  items: T[];
  page: PageMeta;
}

export const currencyQueryKeys = {
  currenciesAll: ['currencies'] as const,
  currenciesList: (params: { page: number; size: number; isActive?: string }) =>
    ['currencies', 'list', params] as const,
  rateTypesAll: ['exchange-rate-types'] as const,
  rateTypesList: (params: { page: number; size: number }) =>
    ['exchange-rate-types', 'list', params] as const,
  ratesAll: ['exchange-rates'] as const,
  ratesList: (params: {
    page: number;
    size: number;
    from?: string;
    to?: string;
    type?: string;
    onDate?: string;
  }) => ['exchange-rates', 'list', params] as const,
};

export async function fetchCurrencies(params: {
  page: number;
  size: number;
  isActive?: string;
}): Promise<PagedResult<CurrencyDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  if (params.isActive && params.isActive !== 'ALL') {
    query.set('isActive', params.isActive);
  }
  const raw = await apiClient<ListEnvelope<CurrencyDto> | CurrencyDto[]>('/currencies', { query });
  return unwrapListResponse<CurrencyDto>(raw);
}

export async function createCurrency(input: CreateCurrencyInput): Promise<CurrencyDto> {
  return apiClient<CurrencyDto>('/currencies', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      code: input.code.trim().toUpperCase(),
      numericCode: input.numericCode?.trim() || undefined,
      name: input.name.trim(),
      precision: input.precision,
      isActive: input.isActive,
    }),
  });
}

export async function updateCurrency(
  id: string,
  rowVersion: number,
  input: UpdateCurrencyInput,
): Promise<CurrencyDto> {
  return apiClient<CurrencyDto>(`/currencies/${id}`, {
    method: 'PUT',
    ifMatch: String(rowVersion),
    body: JSON.stringify({
      code: input.code?.trim().toUpperCase() || undefined,
      numericCode: input.numericCode?.trim() || undefined,
      name: input.name.trim(),
      precision: input.precision,
      isActive: input.isActive,
    }),
  });
}

export async function deleteCurrency(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/currencies/${id}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function fetchExchangeRateTypes(params: {
  page: number;
  size: number;
}): Promise<PagedResult<ExchangeRateTypeDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  const raw = await apiClient<ListEnvelope<ExchangeRateTypeDto> | ExchangeRateTypeDto[]>(
    '/exchange-rate-types',
    { query },
  );
  return unwrapListResponse<ExchangeRateTypeDto>(raw);
}

export async function createExchangeRateType(
  input: CreateExchangeRateTypeInput,
): Promise<ExchangeRateTypeDto> {
  return apiClient<ExchangeRateTypeDto>('/exchange-rate-types', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      code: input.code.trim().toUpperCase(),
      name: input.name.trim(),
    }),
  });
}

export async function fetchExchangeRates(params: {
  page: number;
  size: number;
  from?: string;
  to?: string;
  type?: string;
  onDate?: string;
}): Promise<PagedResult<ExchangeRateDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  if (params.from && params.from !== 'ALL') {
    query.set('from', params.from);
  }
  if (params.to && params.to !== 'ALL') {
    query.set('to', params.to);
  }
  if (params.type && params.type !== 'ALL') {
    query.set('type', params.type);
  }
  if (params.onDate?.trim()) {
    query.set('onDate', params.onDate.trim());
  }
  const raw = await apiClient<ListEnvelope<ExchangeRateDto> | ExchangeRateDto[]>(
    '/exchange-rates',
    { query },
  );
  return unwrapListResponse<ExchangeRateDto>(raw);
}

export async function createExchangeRate(
  input: CreateExchangeRateInput,
): Promise<ExchangeRateDto> {
  return apiClient<ExchangeRateDto>('/exchange-rates', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      fromCurrency: input.fromCurrency.trim().toUpperCase(),
      toCurrency: input.toCurrency.trim().toUpperCase(),
      rateTypeId: input.rateTypeId.trim(),
      rate: input.rate.trim(),
      validFrom: input.validFrom,
      validTo: input.validTo?.trim() || undefined,
    }),
  });
}

export async function updateExchangeRate(
  id: string,
  rowVersion: number,
  input: UpdateExchangeRateInput,
): Promise<ExchangeRateDto> {
  return apiClient<ExchangeRateDto>(`/exchange-rates/${id}`, {
    method: 'PUT',
    ifMatch: String(rowVersion),
    body: JSON.stringify({
      rate: input.rate.trim(),
      validTo: input.validTo?.trim() || undefined,
    }),
  });
}

export async function deleteExchangeRate(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/exchange-rates/${id}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function convertCurrency(
  input: ConvertCurrencyInput,
): Promise<ConvertCurrencyResultDto> {
  return apiClient<ConvertCurrencyResultDto>('/exchange-rates/convert', {
    method: 'POST',
    body: JSON.stringify({
      fromCurrency: input.fromCurrency.trim().toUpperCase(),
      toCurrency: input.toCurrency.trim().toUpperCase(),
      rateTypeCode: input.rateTypeCode?.trim().toUpperCase() || undefined,
      rateTypeId: input.rateTypeId?.trim() || undefined,
      amount: input.amount.trim(),
      onDate: input.onDate?.trim() || undefined,
      anchor: input.anchor?.trim().toUpperCase() || undefined,
    }),
  });
}
