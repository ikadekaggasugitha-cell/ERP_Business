import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  CreateTaxCodeInput,
  CreateTaxInvoiceInput,
  ReportEfakturInput,
  TaxAccountLookupDto,
  TaxCodeDto,
  TaxInvoiceDto,
} from './types';

export interface PagedResult<T> {
  items: T[];
  page: PageMeta;
}

export const taxQueryKeys = {
  invoicesAll: ['tax-invoices'] as const,
  invoicesList: (params: { page: number; size: number; efakturStatus?: string }) =>
    ['tax-invoices', 'list', params] as const,
  invoiceDetail: (id: string) => ['tax-invoices', 'detail', id] as const,
  codesAll: ['tax-codes'] as const,
  codesList: (params: { page: number; size: number }) => ['tax-codes', 'list', params] as const,
  accounts: ['coa-accounts', 'lookup', 'all'] as const,
};

export async function fetchTaxInvoices(params: {
  page: number;
  size: number;
  efakturStatus?: string;
}): Promise<PagedResult<TaxInvoiceDto>> {
  const query = new URLSearchParams();
  query.set('page', String(params.page));
  query.set('size', String(params.size));
  if (params.efakturStatus && params.efakturStatus !== 'ALL') {
    query.set('efakturStatus', params.efakturStatus);
  }
  const raw = await apiClient<ListEnvelope<TaxInvoiceDto> | TaxInvoiceDto[]>('/tax-invoices', {
    query,
  });
  return unwrapListResponse<TaxInvoiceDto>(raw);
}

export async function fetchTaxInvoiceDetail(id: string): Promise<TaxInvoiceDto> {
  return apiClient<TaxInvoiceDto>(`/tax-invoices/${id}`);
}

export async function createTaxInvoice(input: CreateTaxInvoiceInput): Promise<TaxInvoiceDto> {
  return apiClient<TaxInvoiceDto>('/tax-invoices', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      sourceDocumentRef: input.sourceDocumentRef.trim(),
      taxCodeId: input.taxCodeId.trim(),
      baseAmount: input.baseAmount.trim(),
      taxAmount: input.taxAmount?.trim() ?? '',
      efakturNo: input.efakturNo?.trim() ?? '',
    }),
  });
}

export async function reportTaxInvoiceEfaktur(
  id: string,
  rowVersion: number,
  input: ReportEfakturInput,
): Promise<TaxInvoiceDto> {
  const payload: Record<string, string> = {};
  if (input.efakturNo?.trim()) {
    payload.efakturNo = input.efakturNo.trim();
  }
  if (input.ntte?.trim()) {
    payload.ntte = input.ntte.trim();
  }
  return apiClient<TaxInvoiceDto>(`/tax-invoices/${id}/report`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify(payload),
  });
}

export async function fetchTaxCodes(params: {
  page: number;
  size: number;
}): Promise<PagedResult<TaxCodeDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  const raw = await apiClient<ListEnvelope<TaxCodeDto> | TaxCodeDto[]>('/tax-codes', { query });
  return unwrapListResponse<TaxCodeDto>(raw);
}

export async function createTaxCode(input: CreateTaxCodeInput): Promise<TaxCodeDto> {
  return apiClient<TaxCodeDto>('/tax-codes', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      code: input.code.trim(),
      rate: input.rate.trim(),
      taxType: input.taxType.trim(),
      glAccountId: input.glAccountId?.trim() ?? '',
    }),
  });
}

export async function fetchTaxAccountsLookup(): Promise<TaxAccountLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<ListEnvelope<TaxAccountLookupDto> | TaxAccountLookupDto[]>(
    '/accounts',
    { query },
  );
  return unwrapListResponse<TaxAccountLookupDto>(raw).items;
}
