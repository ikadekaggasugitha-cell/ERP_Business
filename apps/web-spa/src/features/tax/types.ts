export interface TaxCodeDto {
  id: string;
  code: string;
  rate: string;
  taxType: string;
  glAccountId?: string;
  rowVersion: number;
}

export interface CreateTaxCodeInput {
  code: string;
  rate: string;
  taxType: string;
  glAccountId?: string;
}

export interface TaxInvoiceDto {
  id: string;
  taxCodeId: string;
  sourceDocumentRef: string;
  baseAmount: string;
  taxAmount: string;
  efakturNo?: string;
  efakturStatus: string;
  rowVersion: number;
}

export interface CreateTaxInvoiceInput {
  sourceDocumentRef: string;
  taxCodeId: string;
  baseAmount: string;
  taxAmount?: string;
  efakturNo?: string;
}

export interface ReportEfakturInput {
  efakturNo?: string;
  ntte?: string;
}

export interface TaxAccountLookupDto {
  id: string;
  code: string;
  name: string;
  accountType: string;
  allowPosting?: boolean;
}
