import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  BankAccountDto,
  BankStatementLineDto,
  BatchStatementImportResultDto,
  CreateBankAccountInput,
  CreateBatchStatementLinesInput,
  CreatePaymentInput,
  CreateSingleStatementLineInput,
  PaymentDto,
  ReconcileResultDto,
  TreasuryAccountLookupDto,
  TreasuryCurrencyLookupDto,
  TreasuryPartyLookupDto,
  UpdatePaymentInput,
} from './types';

export interface PagedResult<T> {
  items: T[];
  page: PageMeta;
}

export const treasuryQueryKeys = {
  bankAccountsAll: ['bank-accounts'] as const,
  bankAccountsList: (params: { page: number; size: number; currencyCode?: string }) =>
    ['bank-accounts', 'list', params] as const,
  bankAccountDetail: (id: string) => ['bank-accounts', 'detail', id] as const,
  paymentsAll: ['payments'] as const,
  paymentsList: (params: {
    page: number;
    size: number;
    status?: string;
    bankAccountId?: string;
  }) => ['payments', 'list', params] as const,
  paymentDetail: (id: string) => ['payments', 'detail', id] as const,
  statementLinesAll: ['bank-statement-lines'] as const,
  statementLinesList: (params: {
    page: number;
    size: number;
    bankAccountId?: string;
    isReconciled?: string;
  }) => ['bank-statement-lines', 'list', params] as const,
  parties: ['parties', 'lookup', 'all'] as const,
  accounts: ['coa-accounts', 'lookup', 'all'] as const,
  currencies: ['currencies', 'lookup', 'active'] as const,
};

export async function fetchBankAccounts(params: {
  page: number;
  size: number;
  currencyCode?: string;
}): Promise<PagedResult<BankAccountDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  if (params.currencyCode && params.currencyCode !== 'ALL') {
    query.set('currencyCode', params.currencyCode);
  }
  const raw = await apiClient<ListEnvelope<BankAccountDto> | BankAccountDto[]>('/bank-accounts', {
    query,
  });
  return unwrapListResponse<BankAccountDto>(raw);
}

export async function createBankAccount(
  input: CreateBankAccountInput,
): Promise<BankAccountDto> {
  return apiClient<BankAccountDto>('/bank-accounts', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify(input),
  });
}

export async function reconcileBankAccount(id: string): Promise<ReconcileResultDto> {
  return apiClient<ReconcileResultDto>(`/bank-accounts/${id}/reconcile`, {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function fetchPayments(params: {
  page: number;
  size: number;
  status?: string;
  bankAccountId?: string;
}): Promise<PagedResult<PaymentDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  if (params.bankAccountId && params.bankAccountId !== 'ALL') {
    query.set('bankAccountId', params.bankAccountId);
  }
  const raw = await apiClient<ListEnvelope<PaymentDto> | PaymentDto[]>('/payments', { query });
  return unwrapListResponse<PaymentDto>(raw);
}

export async function fetchPaymentDetail(id: string): Promise<PaymentDto> {
  return apiClient<PaymentDto>(`/payments/${id}`);
}

export async function createPayment(input: CreatePaymentInput): Promise<PaymentDto> {
  return apiClient<PaymentDto>('/payments', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      ...input,
      partyId: input.partyId || null,
      externalRef: input.externalRef || null,
      allocations: input.allocations ?? [],
    }),
  });
}

export async function updatePayment(
  id: string,
  rowVersion: number,
  input: UpdatePaymentInput,
): Promise<PaymentDto> {
  return apiClient<PaymentDto>(`/payments/${id}`, {
    method: 'PUT',
    ifMatch: String(rowVersion),
    body: JSON.stringify({
      ...input,
      partyId: input.partyId || null,
      externalRef: input.externalRef || null,
      allocations: input.allocations ?? [],
    }),
  });
}

export async function deletePayment(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/payments/${id}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function settlePayment(id: string, rowVersion: number): Promise<PaymentDto> {
  return apiClient<PaymentDto>(`/payments/${id}/settle`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function fetchBankStatementLines(params: {
  page: number;
  size: number;
  bankAccountId?: string;
  isReconciled?: string;
}): Promise<PagedResult<BankStatementLineDto>> {
  const query = new URLSearchParams({
    page: String(params.page),
    size: String(params.size),
  });
  if (params.bankAccountId && params.bankAccountId !== 'ALL') {
    query.set('bankAccountId', params.bankAccountId);
  }
  if (params.isReconciled && params.isReconciled !== 'ALL') {
    query.set('isReconciled', params.isReconciled);
  }
  const raw = await apiClient<ListEnvelope<BankStatementLineDto> | BankStatementLineDto[]>(
    '/bank-statement-lines',
    { query },
  );
  return unwrapListResponse<BankStatementLineDto>(raw);
}

export async function createSingleStatementLine(
  input: CreateSingleStatementLineInput,
): Promise<BankStatementLineDto> {
  return apiClient<BankStatementLineDto>('/bank-statement-lines', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      bankAccountId: input.bankAccountId,
      statementDate: input.statementDate,
      description: input.description,
      amount: input.amount,
      externalRef: input.externalRef?.trim() || undefined,
    }),
  });
}

export async function createBatchStatementLines(
  input: CreateBatchStatementLinesInput,
): Promise<BatchStatementImportResultDto> {
  return apiClient<BatchStatementImportResultDto>('/bank-statement-lines', {
    method: 'POST',
    idempotencyKey: crypto.randomUUID(),
    body: JSON.stringify({
      bankAccountId: input.bankAccountId,
      lines: input.lines.map((line) => ({
        statementDate: line.statementDate,
        description: line.description.trim(),
        amount: line.amount.trim(),
        externalRef: line.externalRef?.trim() || undefined,
      })),
    }),
  });
}

export async function fetchTreasuryPartiesLookup(): Promise<TreasuryPartyLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<ListEnvelope<TreasuryPartyLookupDto> | TreasuryPartyLookupDto[]>(
    '/parties',
    { query },
  );
  return unwrapListResponse<TreasuryPartyLookupDto>(raw).items;
}

export async function fetchTreasuryAccountsLookup(): Promise<TreasuryAccountLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100' });
  const raw = await apiClient<
    ListEnvelope<TreasuryAccountLookupDto> | TreasuryAccountLookupDto[]
  >('/accounts', { query });
  return unwrapListResponse<TreasuryAccountLookupDto>(raw).items;
}

export async function fetchTreasuryCurrenciesLookup(): Promise<TreasuryCurrencyLookupDto[]> {
  const query = new URLSearchParams({ page: '1', size: '100', isActive: 'true' });
  const raw = await apiClient<
    ListEnvelope<TreasuryCurrencyLookupDto> | TreasuryCurrencyLookupDto[]
  >('/currencies', { query });
  return unwrapListResponse<TreasuryCurrencyLookupDto>(raw).items;
}
