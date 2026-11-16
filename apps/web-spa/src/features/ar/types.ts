export interface CustomerInvoiceLineDto {
  id?: string;
  lineNo: number;
  accountId: string;
  itemId?: string | null;
  quantity?: string | null;
  unitPrice?: string | null;
  amount: string;
  description: string;
}

export interface CustomerInvoiceDto {
  id: string;
  invoiceNo: string;
  customerId: string;
  invoiceDate: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  outstandingAmount: string;
  status: string;
  rowVersion: number;
  lines?: CustomerInvoiceLineDto[];
}

export interface CreateCustomerInvoiceInput {
  invoiceNo: string;
  customerId: string;
  invoiceDate: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  taxAmount?: string;
  salesOrderRef?: string;
  lines: Array<{
    lineNo: number;
    accountId: string;
    amount: string;
    description: string;
  }>;
}

export interface ArPartyLookupDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  partyType: string;
  taxId?: string | null;
  isShared: boolean;
  rowVersion: number;
}

export interface ArAccountLookupDto {
  id: string;
  code: string;
  name: string;
  accountType: string;
  allowPosting?: boolean;
}

export interface ArCurrencyLookupDto {
  id: string;
  code: string;
  name: string;
  isActive: boolean;
}
