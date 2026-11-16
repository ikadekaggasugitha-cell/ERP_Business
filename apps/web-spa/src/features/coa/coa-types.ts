import { z } from 'zod';

export type AccountTypeValue = 'Asset' | 'Liability' | 'Equity' | 'Revenue' | 'Expense';

export const ACCOUNT_TYPES: AccountTypeValue[] = [
  'Asset',
  'Liability',
  'Equity',
  'Revenue',
  'Expense',
];

export interface ChartOfAccountsDto {
  id: string;
  code: string;
  name: string;
  structure?: string;
  rowVersion: number;
}

export interface AccountDto {
  id: string;
  chartOfAccountsId: string;
  code: string;
  name: string;
  accountType: AccountTypeValue | string;
  accountClass?: string;
  parentId?: string | null;
  isControl: boolean;
  isMonetary: boolean;
  allowPosting: boolean;
  rowVersion: number;
}

export interface AccountTreeRow extends AccountDto {
  depth: number;
  hasChildren: boolean;
}

export interface FinancialOrganizationDto {
  id: string;
  code: string;
  name: string;
  organizationRefId: string;
  functionalCurrencyCode: string;
  chartOfAccountsId: string;
  fiscalCalendarId: string;
  rowVersion: number;
}

export interface LedgerDto {
  id: string;
  financialOrganizationId: string;
  code: string;
  ledgerType: string;
  currencyCode: string;
  rowVersion: number;
}

export interface FiscalCalendarDto {
  id: string;
  code: string;
  name: string;
  periodType: string;
  rowVersion: number;
}

export interface FiscalYearDto {
  id: string;
  fiscalCalendarId: string;
  yearNo: number;
  name: string;
  startDate: string;
  endDate: string;
  status: string;
  rowVersion: number;
}

export interface FiscalPeriodDto {
  id: string;
  fiscalYearId: string;
  periodNo: number;
  startDate: string;
  endDate: string;
  status: 'Open' | 'Closed' | 'Locked' | string;
  rowVersion: number;
}

export interface OrgUnitLookupDto {
  id: string;
  code: string;
  name: string;
}

export interface CoaCurrencyLookupDto {
  id: string;
  code: string;
  name: string;
  isActive: boolean;
}

function inferCodeDepth(code: string, isControl: boolean): number {
  const clean = code.trim();
  if (!clean) return 0;
  const segmented = clean.split(/[-.]/).filter(Boolean);
  if (segmented.length > 1) {
    return Math.min(segmented.length - 1, 4);
  }
  if (/^\d{4,}$/.test(clean)) {
    if (clean.endsWith('000')) return 0;
    if (clean.endsWith('00')) return 1;
    if (clean.endsWith('0')) return 2;
    return isControl ? 1 : 3;
  }
  return isControl ? 0 : 1;
}

export function buildAccountHierarchyRows(accounts: AccountDto[]): AccountTreeRow[] {
  const sorted = [...accounts].sort((a, b) => a.code.localeCompare(b.code, undefined, { numeric: true }));
  const byId = new Map<string, AccountDto>();
  const childrenMap = new Map<string, AccountDto[]>();

  for (const acc of sorted) {
    byId.set(acc.id, acc);
  }

  let hasExplicitParents = false;
  for (const acc of sorted) {
    if (acc.parentId && byId.has(acc.parentId)) {
      hasExplicitParents = true;
      const list = childrenMap.get(acc.parentId) ?? [];
      list.push(acc);
      childrenMap.set(acc.parentId, list);
    }
  }

  if (!hasExplicitParents) {
    return sorted.map((acc) => ({
      ...acc,
      depth: inferCodeDepth(acc.code, acc.isControl),
      hasChildren: acc.isControl,
    }));
  }

  const result: AccountTreeRow[] = [];
  const visited = new Set<string>();

  const walk = (node: AccountDto, depth: number) => {
    if (visited.has(node.id)) return;
    visited.add(node.id);
    const kids = childrenMap.get(node.id) ?? [];
    result.push({
      ...node,
      depth,
      hasChildren: kids.length > 0 || node.isControl,
    });
    for (const child of kids) {
      walk(child, Math.min(depth + 1, 5));
    }
  };

  for (const acc of sorted) {
    if (!acc.parentId || !byId.has(acc.parentId)) {
      walk(acc, 0);
    }
  }

  for (const acc of sorted) {
    if (!visited.has(acc.id)) {
      walk(acc, 0);
    }
  }

  return result;
}

export const chartOfAccountsSchema = z.object({
  code: z.string().trim().min(2, 'Kode CoA minimal 2 karakter / Code min 2 chars'),
  name: z.string().trim().min(2, 'Nama CoA wajib diisi / Name is required'),
  structure: z.string().trim().optional(),
});
export type ChartOfAccountsFormValues = z.infer<typeof chartOfAccountsSchema>;

export const accountFormSchema = z.object({
  chartOfAccountsId: z.string().trim().min(1, 'Chart of Accounts wajib dipilih'),
  code: z.string().trim().min(2, 'Kode akun wajib diisi / Account code is required'),
  name: z.string().trim().min(2, 'Nama akun wajib diisi / Account name is required'),
  accountType: z.enum(['Asset', 'Liability', 'Equity', 'Revenue', 'Expense']),
  accountClass: z.string().trim().optional(),
  parentId: z.string().trim().optional(),
  isControl: z.boolean(),
  isMonetary: z.boolean(),
  allowPosting: z.boolean(),
});
export type AccountFormValues = z.infer<typeof accountFormSchema>;

export const financialOrgSchema = z.object({
  code: z.string().trim().min(2, 'Kode wajib diisi'),
  name: z.string().trim().min(2, 'Nama wajib diisi'),
  organizationRefId: z.string().trim().min(1, 'Unit Organisasi wajib dipilih'),
  functionalCurrencyCode: z.string().trim().min(3, 'Mata uang fungsional wajib dipilih'),
  chartOfAccountsId: z.string().trim().min(1, 'Chart of Accounts wajib dipilih'),
  fiscalCalendarId: z.string().trim().min(1, 'Kalender Fiskal wajib dipilih'),
});
export type FinancialOrgFormValues = z.infer<typeof financialOrgSchema>;

export const ledgerFormSchema = z.object({
  financialOrganizationId: z.string().trim().min(1, 'Entitas keuangan wajib dipilih'),
  code: z.string().trim().min(2, 'Kode Ledger wajib diisi'),
  ledgerType: z.string().trim().min(2, 'Tipe Ledger wajib dipilih'),
  currencyCode: z.string().trim().min(3, 'Mata uang wajib dipilih'),
});
export type LedgerFormValues = z.infer<typeof ledgerFormSchema>;

export const fiscalCalendarSchema = z.object({
  code: z.string().trim().min(2, 'Kode kalender wajib diisi'),
  name: z.string().trim().min(2, 'Nama kalender wajib diisi'),
  periodType: z.enum(['Monthly', 'Quarterly']),
});
export type FiscalCalendarFormValues = z.infer<typeof fiscalCalendarSchema>;

export const fiscalYearSchema = z.object({
  fiscalCalendarId: z.string().trim().min(1, 'Kalender fiskal wajib dipilih'),
  yearNo: z.coerce.number().int().min(1900).max(2100),
  name: z.string().trim().min(2, 'Nama tahun fiskal wajib diisi'),
  startDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Tanggal mulai YYYY-MM-DD wajib diisi'),
  endDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Tanggal selesai YYYY-MM-DD wajib diisi'),
});
export type FiscalYearFormValues = z.infer<typeof fiscalYearSchema>;

export const fiscalPeriodSchema = z.object({
  fiscalYearId: z.string().trim().min(1, 'Tahun fiskal wajib dipilih'),
  periodNo: z.coerce.number().int().min(1).max(24),
  startDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Tanggal mulai YYYY-MM-DD wajib diisi'),
  endDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Tanggal selesai YYYY-MM-DD wajib diisi'),
});
export type FiscalPeriodFormValues = z.infer<typeof fiscalPeriodSchema>;
