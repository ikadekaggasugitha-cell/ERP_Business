export type AppLocale = 'id' | 'en';
export type AppTheme = 'light' | 'dark';
export type PortalMode = 'backoffice' | 'admin' | 'reporting';

export interface ApiFieldError {
  field?: string;
  issue: string;
}

export interface ApiError {
  code: string;
  message: string;
  correlationId?: string;
  details?: ApiFieldError[];
}

export interface ApiErrorResponse {
  error: ApiError;
}

export interface PageMeta {
  number: number;
  size: number;
  totalElements: number;
  totalPages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  page: PageMeta;
}

export type ListEnvelope<T> = PaginatedResponse<T>;

export interface PageQueryParams {
  page?: number;
  size?: number;
  sort?: string;
  filters?: Record<string, string | number | boolean | null | undefined>;
}

export interface User {
  id: string;
  username: string;
  fullName: string;
  email: string;
  activeOrgId: string;
  adminTier?: number;
  roles: string[];
  permissions: string[];
}

export interface Organization {
  id: string;
  code: string;
  name: string;
  type:
    | 'HOLDING'
    | 'SUB_HOLDING'
    | 'COMPANY'
    | 'BRANCH'
    | 'WAREHOUSE'
    | 'DIVISION'
    | 'DEPARTMENT'
    | 'PLACE';
  parentId?: string | null;
  functionalCurrency: string;
  isActive?: boolean;
  rowVersion?: number;
}

export interface AuthSessionResponse {
  token?: string;
  streamTicket?: string;
  expiresIn?: number;
  user: User;
  activeOrg: Organization;
  organizations: Organization[];
}

export interface MoneyValue {
  amount: string;
  currency: string;
}

// --- GL & Accounting (SVC-001 GL, SVC-002 CoA) ---

export interface JournalLine {
  id?: string;
  journalEntryId?: string;
  lineNo: number;
  accountId: string;
  accountCode?: string;
  accountName?: string;
  debitAmount: string;
  creditAmount: string;
  functionalAmount: string;
  presentationAmount?: string;
  description?: string;
}

export interface JournalEntry {
  id: string;
  journalNo: string;
  financialOrgId: string;
  financialOrganizationId?: string;
  ledgerId?: string;
  fiscalPeriodId?: string;
  postingDate: string;
  journalSource: 'MANUAL' | 'SUB_LEDGER' | 'RECURRING' | 'REVALUATION' | string;
  currencyCode: string;
  exchangeRate: string;
  sourceDocumentRef?: string;
  status: 'DRAFT' | 'POSTED' | 'REVERSED' | 'Draft' | 'Posted' | 'Reversed';
  totalDebit: string;
  totalCredit: string;
  linesCount: number;
  rowVersion?: number;
  lines?: JournalLine[];
  createdAt?: string;
  updatedAt?: string;
}

export interface GlBalanceSnapshotRow {
  id: string;
  fiscalPeriodId: string;
  accountCode: string;
  closingBalance: string;
  refreshedAt: string;
}

export interface CoaAccount {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  accountType: 'ASSET' | 'LIABILITY' | 'EQUITY' | 'REVENUE' | 'EXPENSE' | string;
  normalBalance: 'DEBIT' | 'CREDIT';
  parentAccountId?: string | null;
  isPostable: boolean;
  isActive: boolean;
  currencyCode?: string;
  rowVersion?: number;
}

export interface FiscalPeriod {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  startDate: string;
  endDate: string;
  status: 'OPEN' | 'SOFT_CLOSED' | 'HARD_CLOSED' | string;
  rowVersion?: number;
}

export interface LedgerDefinition {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  functionalCurrency: string;
  isPrimary: boolean;
  isActive: boolean;
  rowVersion?: number;
}

// --- AP & AR (SVC-003, SVC-004) ---

export interface ApInvoiceLine {
  id?: string;
  lineNo: number;
  description: string;
  accountId: string;
  quantity: string;
  unitPrice: string;
  taxCode?: string;
  taxAmount?: string;
  lineTotal: string;
}

export interface ApInvoice {
  id: string;
  invoiceNo: string;
  vendorId: string;
  vendorName?: string;
  invoiceDate: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  outstandingAmount: string;
  status: string;
  rowVersion?: number;
  lines?: ApInvoiceLine[];
}

export interface ArInvoiceLine {
  id?: string;
  lineNo: number;
  description: string;
  accountId: string;
  quantity: string;
  unitPrice: string;
  taxCode?: string;
  taxAmount?: string;
  lineTotal: string;
}

export interface ArInvoice {
  id: string;
  invoiceNo: string;
  customerId: string;
  customerName?: string;
  invoiceDate: string;
  dueDate: string;
  currencyCode: string;
  totalAmount: string;
  outstandingAmount: string;
  status: string;
  rowVersion?: number;
  lines?: ArInvoiceLine[];
}

// --- Tax (SVC-005) ---

export interface TaxCodeItem {
  id: string;
  code: string;
  name: string;
  rate: string;
  taxType: 'PPN' | 'PPH21' | 'PPH23' | 'PPH4_2' | 'WITHHOLDING' | string;
  isInclusive?: boolean;
  isActive: boolean;
  rowVersion?: number;
}

export interface TaxInvoice {
  id: string;
  organizationId: string;
  taxInvoiceNo: string;
  nsfp?: string;
  counterpartyTaxId?: string;
  counterpartyName?: string;
  sourceDocType: string;
  sourceDocId: string;
  taxDate: string;
  taxBaseAmount: string;
  taxAmount: string;
  status: 'DRAFT' | 'ISSUED' | 'REPORTED' | 'VOIDED' | string;
  rowVersion?: number;
}

// --- Treasury & Cashbank (SVC-006) ---

export interface BankAccount {
  id: string;
  organizationId?: string;
  code: string;
  bankName: string;
  accountNo: string;
  currencyCode: string;
  glAccountId: string;
  balance?: string;
  isActive?: boolean;
  rowVersion?: number;
}

export interface CashbankTransaction {
  id: string;
  organizationId: string;
  transactionNo: string;
  bankAccountId: string;
  direction: 'INFLOW' | 'OUTFLOW' | 'TRANSFER';
  transactionDate: string;
  currencyCode: string;
  amount: string;
  counterpartyName?: string;
  referenceNo?: string;
  status: 'DRAFT' | 'SETTLED' | 'RECONCILED' | 'VOIDED' | string;
  rowVersion?: number;
}

export interface BankReconciliation {
  id: string;
  bankAccountId: string;
  statementDate: string;
  statementClosingBalance: string;
  ledgerClosingBalance: string;
  unreconciledDiff: string;
  status: 'OPEN' | 'MATCHED' | 'CLOSED' | string;
  rowVersion?: number;
}

// --- Currency & FX (SVC-007) ---

export interface CurrencyItem {
  id: string;
  code: string;
  numericCode?: string;
  name: string;
  precision: number;
  isActive: boolean;
  rowVersion?: number;
}

export interface ExchangeRateItem {
  id: string;
  fromCurrency: string;
  toCurrency: string;
  rateTypeCode?: string;
  rate: string;
  validFrom: string;
  validTo?: string;
  rowVersion?: number;
}

// --- Masterdata (SVC-014) ---

export interface BusinessPartner {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  partnerType: 'VENDOR' | 'CUSTOMER' | 'BOTH' | string;
  taxId?: string;
  email?: string;
  phone?: string;
  paymentTermsDays?: number;
  isActive: boolean;
  rowVersion?: number;
}

export interface UnitOfMeasure {
  id: string;
  code: string;
  name: string;
  category?: string;
  isActive: boolean;
}

// --- Supply Chain: Inventory, Procurement, Sales, Manufacturing (SVC-008..011) ---

export interface InventoryItem {
  id: string;
  organizationId?: string;
  code: string;
  name: string;
  itemType: string;
  baseUom: string;
  onHandQty?: string;
  availableQty?: string;
  reservedQty?: string;
  avgCost?: string;
  reorderPoint?: string;
  rowVersion?: number;
}

export interface Warehouse {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  location?: string;
  isActive: boolean;
}

export interface StockMovement {
  id: string;
  itemId: string;
  warehouseId: string;
  movementType: 'RECEIPT' | 'ISSUE' | 'RESERVE' | 'TRANSFER' | 'ADJUSTMENT' | string;
  quantity: string;
  unitCost?: string;
  referenceDoc?: string;
  occurredAt: string;
}

export interface PurchaseRequisition {
  id: string;
  prNo: string;
  requesterId?: string;
  requiredDate: string;
  status: 'DRAFT' | 'SUBMITTED' | 'APPROVED' | 'REJECTED' | 'CONVERTED' | string;
  estimatedTotal: string;
  currencyCode: string;
  rowVersion?: number;
}

export interface PurchaseOrder {
  id: string;
  poNo: string;
  vendorId: string;
  vendorName?: string;
  purchaseRequisitionId?: string;
  orderDate: string;
  currencyCode: string;
  totalAmount: string;
  status: string;
  rowVersion?: number;
}

export interface GoodsReceipt {
  id: string;
  grnNo: string;
  purchaseOrderId: string;
  warehouseId: string;
  receivedDate: string;
  status: string;
  rowVersion?: number;
}

export interface SalesOrder {
  id: string;
  soNo: string;
  customerId: string;
  customerName?: string;
  orderDate: string;
  deliveryDate?: string;
  currencyCode: string;
  totalAmount: string;
  status: 'DRAFT' | 'CONFIRMED' | 'FULFILLED' | 'INVOICED' | 'CANCELLED' | string;
  rowVersion?: number;
}

export interface BillOfMaterials {
  id: string;
  code: string;
  finishedItemId: string;
  finishedItemName?: string;
  outputQuantity: string;
  uom: string;
  isActive: boolean;
  rowVersion?: number;
}

export interface WorkOrder {
  id: string;
  woNo: string;
  bomId: string;
  itemId: string;
  itemName?: string;
  plannedQty: string;
  completedQty: string;
  scheduledStart: string;
  scheduledEnd?: string;
  status: 'PLANNED' | 'RELEASED' | 'IN_PROGRESS' | 'COMPLETED' | 'CLOSED' | string;
  rowVersion?: number;
}

// --- Human Capital & Payroll (SVC-012, SVC-013) ---

export interface Employee {
  id: string;
  organizationId: string;
  employeeNo: string;
  fullName: string;
  email?: string;
  departmentName?: string;
  positionTitle?: string;
  employmentType: 'PERMANENT' | 'CONTRACT' | 'PROBATION' | string;
  hireDate: string;
  baseSalary?: string;
  currencyCode?: string;
  status: 'ACTIVE' | 'ON_LEAVE' | 'TERMINATED' | string;
  rowVersion?: number;
}

export interface PayrollRun {
  id: string;
  organizationId: string;
  runCode: string;
  periodStart: string;
  periodEnd: string;
  paymentDate: string;
  currencyCode: string;
  employeeCount: number;
  grossTotal: string;
  deductionsTotal: string;
  netTotal: string;
  status: 'DRAFT' | 'CALCULATED' | 'APPROVED' | 'PAID' | string;
  rowVersion?: number;
}

// --- Platform, Governance, Workflow, IAM & Reporting (SVC-015..019) ---

export interface WorkflowApprovalTask {
  id: string;
  organizationId: string;
  documentType: string;
  documentId: string;
  documentNo: string;
  requesterName?: string;
  requestedAt: string;
  amount?: string;
  currencyCode?: string;
  currentStep: number;
  totalSteps: number;
  status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'CANCELLED' | string;
  decisionComment?: string;
  rowVersion?: number;
}

export interface RoleDefinition {
  id: string;
  code: string;
  name: string;
  description?: string;
  permissions: string[];
  isSystem?: boolean;
  rowVersion?: number;
}

export interface UserAccountRecord {
  id: string;
  username: string;
  fullName: string;
  email: string;
  roles: string[];
  organizationIds: string[];
  status: 'ACTIVE' | 'LOCKED' | 'DISABLED' | string;
  lastLoginAt?: string;
  rowVersion?: number;
}

export interface AuditEvent {
  id: string;
  occurredAt: string;
  organizationId: string;
  actorId?: string;
  actorName?: string;
  action: string;
  entityType: string;
  entityId?: string;
  correlationId: string;
  reason?: string;
  beforeValue?: Record<string, unknown> | null;
  afterValue?: Record<string, unknown> | null;
}

export interface FinancialReportSummary {
  reportType: 'TRIAL_BALANCE' | 'INCOME_STATEMENT' | 'BALANCE_SHEET' | 'CASH_FLOW' | string;
  organizationId: string;
  fiscalPeriodId: string;
  currencyCode: string;
  generatedAt: string;
  rows: Array<{
    accountCode: string;
    accountName: string;
    accountType?: string;
    openingBalance?: string;
    debitTotal?: string;
    creditTotal?: string;
    closingBalance: string;
  }>;
}

export interface RealtimeEvent<T = unknown> {
  eventId: string;
  eventType: string;
  topic?: string;
  aggregateType: string;
  aggregateId: string;
  organizationId: string;
  correlationId: string;
  timestamp: string;
  payload: T;
}
