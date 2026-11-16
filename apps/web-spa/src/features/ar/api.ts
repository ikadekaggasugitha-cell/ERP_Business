import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  ArAccountLookupDto,
  ArCurrencyLookupDto,
  ArPartyLookupDto,
  CreateCustomerInvoiceInput,
  CustomerInvoiceDto,
} from './types';

export interface PagedResult<T> {
  items: T[];
  page: PageMeta;
}

export const arQueryKeys = {
  all: ['ar-invoices'] as const,
  list: (params: { page: number; size: number; status?: string }) =>
    ['ar-invoices', 'list', params] as const,
  detail: (id: string) => ['ar-invoices', 'detail', id] as const,
  customers: ['parties', 'lookup', 'all'] as const,
  accounts: ['coa-accounts', 'lookup', 'all'] as const,
  currencies: ['currencies', 'lookup', 'active'] as const,
};

export async function fetchCustomerInvoices(params: {
  page: number;
  size: number;
  status?: string;
}): Promise<PagedResult<CustomerInvoiceDto>> {
  const query = new URLSearchParams();
  query.set('page', String(params.page));
  query.set('size', String(params.size));
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  const raw = await apiClient<ListEnvelope<CustomerInvoiceDto> | CustomerInvoiceDto[]>(
    '/customer-invoices',
    { query },
  );
  return unwrapListResponse<CustomerInvoiceDto>(raw);
}

export async function fetchCustomerInvoiceDetail(id: string): Promise<CustomerInvoiceDto> {
  return apiClient<CustomerInvoiceDto>(`/customer-invoices/${id}`);
}

export async function createCustomerInvoice(
  input: CreateCustomerInvoiceInput,
): Promise<CustomerInvoiceDto> {
  return apiClient<CustomerInvoiceDto>('/customer-invoices', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      ...input,
      taxAmount: input.taxAmount?.trim() || undefined,
      salesOrderRef: input.salesOrderRef?.trim() || undefined,
    }),
  });
}

export async function markCustomerInvoicePaid(
  id: string,
  rowVersion: number,
): Promise<CustomerInvoiceDto> {
  return apiClient<CustomerInvoiceDto>(`/customer-invoices/${id}/pay`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function deleteCustomerInvoice(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/customer-invoices/${id}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function fetchArPartiesLookup(): Promise<ArPartyLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<ListEnvelope<ArPartyLookupDto> | ArPartyLookupDto[]>('/parties', {
    query,
  });
  return unwrapListResponse<ArPartyLookupDto>(raw).items;
}

export async function fetchArAccountsLookup(): Promise<ArAccountLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<ListEnvelope<ArAccountLookupDto> | ArAccountLookupDto[]>(
    '/accounts',
    { query },
  );
  return unwrapListResponse<ArAccountLookupDto>(raw).items;
}

export async function fetchArCurrenciesLookup(): Promise<ArCurrencyLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100', isActive: 'true' });
  const raw = await apiClient<ListEnvelope<ArCurrencyLookupDto> | ArCurrencyLookupDto[]>(
    '/currencies',
    { query },
  );
  return unwrapListResponse<ArCurrencyLookupDto>(raw).items;
}
