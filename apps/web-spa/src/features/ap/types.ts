export interface VendorInvoiceLineDto {
  id?: string;
  lineNo: number;
  accountId: string;
  itemId?: string | null;
  quantity?: string | null;
  unitPrice?: string | null;
  amount: string;
  description: string;
}

export interface VendorInvoiceDto {
  id: string;
  invoiceNo: string;
  vendorId: string;
  invoiceDate: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  outstandingAmount: string;
  status: string;
  rowVersion: number;
  goodsReceiptRef?: string | null;
  lines?: VendorInvoiceLineDto[];
}

export interface CreateVendorInvoiceInput {
  invoiceNo: string;
  vendorId: string;
  invoiceDate: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  goodsReceiptRef?: string;
  lines: Array<{
    lineNo: number;
    accountId: string;
    amount: string;
    description: string;
  }>;
}

export interface UpdateVendorInvoiceInput {
  vendorId: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  goodsReceiptRef?: string;
  lines: Array<{
    lineNo: number;
    accountId: string;
    amount: string;
    description: string;
  }>;
}

export interface PartyLookupDto {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  partyType: string;
  taxId?: string | null;
  isShared: boolean;
  rowVersion: number;
}

export interface CoaAccountLookupDto {
  id: string;
  chartOfAccountsId: string;
  code: string;
  name: string;
  accountType: string;
  accountClass?: string;
  isControl?: boolean;
  isMonetary?: boolean;
  allowPosting?: boolean;
  rowVersion: number;
}

export interface CurrencyLookupDto {
  id: string;
  code: string;
  name: string;
  precision: number;
  isActive: boolean;
}
