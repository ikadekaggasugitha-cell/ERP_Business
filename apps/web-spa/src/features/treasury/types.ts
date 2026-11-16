export interface BankAccountDto {
  id: string;
  organizationId?: string;
  code: string;
  bankName: string;
  accountNo: string;
  currencyCode: string;
  glAccountId: string;
  rowVersion: number;
}

export interface CreateBankAccountInput {
  code: string;
  bankName: string;
  accountNo: string;
  currencyCode: string;
  glAccountId: string;
}

export interface ReconcileMatchDto {
  statementLineId: string;
  paymentId: string;
  paymentNo: string;
  amount: string;
}

export interface ReconcileResultDto {
  bankAccountId: string;
  matchedCount: number;
  matches: ReconcileMatchDto[];
}

export interface PaymentAllocationDto {
  id?: string;
  targetDocumentRef: string;
  allocatedAmount: string;
  rowVersion?: number;
}

export interface PaymentDto {
  id: string;
  organizationId?: string;
  paymentNo: string;
  paymentType: 'Outgoing' | 'Incoming' | string;
  bankAccountId: string;
  partyId?: string | null;
  amount: string;
  paymentDate: string;
  status: string;
  externalRef?: string | null;
  correlationId?: string;
  rowVersion: number;
  allocations?: PaymentAllocationDto[];
}

export interface CreatePaymentInput {
  paymentNo: string;
  paymentType: 'Outgoing' | 'Incoming';
  bankAccountId: string;
  partyId?: string | null;
  amount: string;
  paymentDate: string;
  externalRef?: string | null;
  allocations?: Array<{
    targetDocumentRef: string;
    allocatedAmount: string;
  }>;
}

export interface UpdatePaymentInput {
  paymentType: 'Outgoing' | 'Incoming';
  bankAccountId: string;
  partyId?: string | null;
  amount: string;
  paymentDate: string;
  externalRef?: string | null;
  allocations: Array<{
    targetDocumentRef: string;
    allocatedAmount: string;
  }>;
}

export interface BankStatementLineDto {
  id: string;
  organizationId?: string;
  bankAccountId: string;
  statementDate: string;
  description: string;
  amount: string;
  externalRef?: string | null;
  matchedPaymentId?: string | null;
  isReconciled: boolean;
  rowVersion: number;
}

export interface CreateSingleStatementLineInput {
  bankAccountId: string;
  statementDate: string;
  description: string;
  amount: string;
  externalRef?: string;
}

export interface CreateBatchStatementLinesInput {
  bankAccountId: string;
  lines: Array<{
    statementDate: string;
    description: string;
    amount: string;
    externalRef?: string;
  }>;
}

export interface BatchStatementImportResultDto {
  importedCount: number;
  ids: string[];
  bankAccountId: string;
}

export interface TreasuryPartyLookupDto {
  id: string;
  code: string;
  name: string;
  partyType: string;
}

export interface TreasuryAccountLookupDto {
  id: string;
  code: string;
  name: string;
  accountType: string;
  allowPosting?: boolean;
}

export interface TreasuryCurrencyLookupDto {
  id: string;
  code: string;
  name: string;
}
