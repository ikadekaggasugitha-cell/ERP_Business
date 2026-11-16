import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  CoaAccountLookupDto,
  CreateVendorInvoiceInput,
  CurrencyLookupDto,
  PartyLookupDto,
  UpdateVendorInvoiceInput,
  VendorInvoiceDto,
} from './types';

export interface PagedResult<T> {
  items: T[];
  page: PageMeta;
}

export const apQueryKeys = {
  all: ['ap-invoices'] as const,
  list: (params: { page: number; size: number; status?: string }) =>
    ['ap-invoices', 'list', params] as const,
  detail: (id: string) => ['ap-invoices', 'detail', id] as const,
  vendors: ['parties', 'lookup', 'all'] as const,
  accounts: ['coa-accounts', 'lookup', 'all'] as const,
  currencies: ['currencies', 'lookup', 'active'] as const,
};

export async function fetchVendorInvoices(params: {
  page: number;
  size: number;
  status?: string;
}): Promise<PagedResult<VendorInvoiceDto>> {
  const query = new URLSearchParams();
  query.set('page', String(params.page));
  query.set('size', String(params.size));
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  const raw = await apiClient<ListEnvelope<VendorInvoiceDto> | VendorInvoiceDto[]>(
    '/vendor-invoices',
    { query },
  );
  return unwrapListResponse<VendorInvoiceDto>(raw);
}

export async function fetchVendorInvoiceDetail(id: string): Promise<VendorInvoiceDto> {
  return apiClient<VendorInvoiceDto>(`/vendor-invoices/${id}`);
}

export async function createVendorInvoice(
  input: CreateVendorInvoiceInput,
): Promise<VendorInvoiceDto> {
  return apiClient<VendorInvoiceDto>('/vendor-invoices', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      ...input,
      goodsReceiptRef: input.goodsReceiptRef?.trim() || undefined,
    }),
  });
}

export async function updateVendorInvoice(
  id: string,
  rowVersion: number,
  input: UpdateVendorInvoiceInput,
): Promise<VendorInvoiceDto> {
  return apiClient<VendorInvoiceDto>(`/vendor-invoices/${id}`, {
    method: 'PUT',
    ifMatch: String(rowVersion),
    body: JSON.stringify({
      ...input,
      goodsReceiptRef: input.goodsReceiptRef?.trim() || undefined,
    }),
  });
}

export async function deleteVendorInvoice(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/vendor-invoices/${id}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function recordVendorInvoice(
  id: string,
  rowVersion: number,
): Promise<VendorInvoiceDto> {
  return apiClient<VendorInvoiceDto>(`/vendor-invoices/${id}/record`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function fetchPartiesLookup(): Promise<PartyLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<ListEnvelope<PartyLookupDto> | PartyLookupDto[]>('/parties', {
    query,
  });
  return unwrapListResponse<PartyLookupDto>(raw).items;
}

export async function fetchCoaAccountsLookup(): Promise<CoaAccountLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<ListEnvelope<CoaAccountLookupDto> | CoaAccountLookupDto[]>(
    '/accounts',
    { query },
  );
  return unwrapListResponse<CoaAccountLookupDto>(raw).items;
}

export async function fetchCurrenciesLookup(): Promise<CurrencyLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100', isActive: 'true' });
  const raw = await apiClient<ListEnvelope<CurrencyLookupDto> | CurrencyLookupDto[]>(
    '/currencies',
    { query },
  );
  return unwrapListResponse<CurrencyLookupDto>(raw).items;
}
