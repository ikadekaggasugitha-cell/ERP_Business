import { z } from 'zod';
import { addDecimals, Decimal, multiplyDecimals, subtractDecimals } from '../../lib/decimal';

export type JournalStatusFilter = 'ALL' | 'DRAFT' | 'POSTED' | 'REVERSED';

export interface JournalLineDto {
  id?: string;
  lineNo: number;
  accountId: string;
  debitAmount: string;
  creditAmount: string;
  functionalAmount: string;
  description?: string | null;
}

export interface JournalEntryDto {
  id: string;
  journalNo: string;
  financialOrganizationId: string;
  ledgerId: string;
  fiscalPeriodId: string;
  postingDate: string;
  journalSource: string;
  currencyCode: string;
  exchangeRate: string;
  totalDebit: string;
  totalCredit: string;
  sourceDocumentRef?: string | null;
  reversedEntryId?: string | null;
  status: 'Draft' | 'Posted' | 'Reversed' | string;
  rowVersion: number;
  lines?: JournalLineDto[];
}

export interface TrialBalanceRowDto {
  id: string;
  ledgerId: string;
  accountId: string;
  fiscalPeriodId: string;
  openingBalance: string;
  periodDebit: string;
  periodCredit: string;
  closingBalance: string;
  rowVersion: number;
}

export interface LedgerLookupDto {
  id: string;
  financialOrganizationId: string;
  code: string;
  ledgerType: string;
  currencyCode: string;
  rowVersion: number;
}

export interface FiscalPeriodLookupDto {
  id: string;
  fiscalYearId: string;
  periodNo: number;
  startDate: string;
  endDate: string;
  status: string;
  rowVersion: number;
}

export interface AccountLookupDto {
  id: string;
  chartOfAccountsId: string;
  code: string;
  name: string;
  accountType: string;
  allowPosting: boolean;
  isControl: boolean;
  isMonetary: boolean;
  rowVersion: number;
}

export interface CurrencyLookupDto {
  id: string;
  code: string;
  name: string;
  precision: number;
  isActive: boolean;
  rowVersion: number;
}

export interface JournalTotalsSummary {
  totalDebit: string;
  totalCredit: string;
  difference: string;
  isBalanced: boolean;
  isPositive: boolean;
  canSubmit: boolean;
}

export function isValidDecimalString(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return false;
  return /^-?\d+(\.\d+)?$/.test(trimmed);
}

export function isNonNegativeDecimal(value: string): boolean {
  if (!isValidDecimalString(value)) return false;
  return new Decimal(value.trim(), 4).gte('0');
}

export function computeJournalTotals(
  lines: Array<{ debitAmount?: string; creditAmount?: string }>
): JournalTotalsSummary {
  let totalDebit = '0.0000';
  let totalCredit = '0.0000';

  for (const line of lines) {
    const debitStr = isValidDecimalString(line.debitAmount ?? '') ? line.debitAmount!.trim() : '0';
    const creditStr = isValidDecimalString(line.creditAmount ?? '')
      ? line.creditAmount!.trim()
      : '0';
    totalDebit = addDecimals(totalDebit, debitStr, 4);
    totalCredit = addDecimals(totalCredit, creditStr, 4);
  }

  const debitDec = new Decimal(totalDebit, 4);
  const creditDec = new Decimal(totalCredit, 4);
  const difference = subtractDecimals(totalDebit, totalCredit, 2);
  const isBalanced = debitDec.equals(creditDec);
  const isPositive = debitDec.gt('0');

  return {
    totalDebit: addDecimals(totalDebit, '0', 2),
    totalCredit: addDecimals(totalCredit, '0', 2),
    difference,
    isBalanced,
    isPositive,
    canSubmit: lines.length >= 2 && isBalanced && isPositive,
  };
}

export function computeFunctionalAmount(
  debitAmount: string,
  creditAmount: string,
  exchangeRate: string
): string {
  const debitDec = new Decimal(isValidDecimalString(debitAmount) ? debitAmount : '0', 4);
  const creditDec = new Decimal(isValidDecimalString(creditAmount) ? creditAmount : '0', 4);
  const baseAmount = debitDec.gt('0') ? debitDec.toRawString() : creditDec.toRawString();
  const safeRate = isValidDecimalString(exchangeRate) ? exchangeRate.trim() : '1';
  return multiplyDecimals(baseAmount, safeRate, 4);
}

export function normalizeJournalStatus(status: string): 'DRAFT' | 'POSTED' | 'REVERSED' | string {
  return (status || '').trim().toUpperCase();
}

export function generateDefaultJournalNo(): string {
  const now = new Date();
  const ym = `${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}`;
  const suffix = String(Math.floor(1000 + Math.random() * 9000));
  return `JRN-${ym}-${suffix}`;
}

export const journalLineFormSchema = z
  .object({
    accountId: z.string().trim().min(1, 'Akun GL wajib dipilih / Account is required'),
    debitAmount: z
      .string()
      .trim()
      .refine(isNonNegativeDecimal, 'Nominal debit tidak valid / Invalid debit amount'),
    creditAmount: z
      .string()
      .trim()
      .refine(isNonNegativeDecimal, 'Nominal kredit tidak valid / Invalid credit amount'),
    description: z.string().trim().optional(),
  })
  .superRefine((line, ctx) => {
    const debit = new Decimal(line.debitAmount || '0', 4);
    const credit = new Decimal(line.creditAmount || '0', 4);
    const hasDebit = debit.gt('0');
    const hasCredit = credit.gt('0');

    if (hasDebit && hasCredit) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['creditAmount'],
        message: 'Isi salah satu Debit atau Kredit / Enter either Debit or Credit, not both',
      });
    } else if (!hasDebit && !hasCredit) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['debitAmount'],
        message: 'Nominal baris harus > 0 / Line amount must be greater than zero',
      });
    }
  });

export const journalFormSchema = z
  .object({
    journalNo: z
      .string()
      .trim()
      .min(2, 'Nomor jurnal minimal 2 karakter / Journal No min 2 chars'),
    financialOrganizationId: z
      .string()
      .trim()
      .min(1, 'Entitas keuangan wajib terisi / Financial organization is required'),
    ledgerId: z.string().trim().min(1, 'Buku besar (Ledger) wajib dipilih / Ledger is required'),
    fiscalPeriodId: z
      .string()
      .trim()
      .min(1, 'Periode fiskal wajib dipilih / Fiscal period is required'),
    postingDate: z
      .string()
      .trim()
      .regex(/^\d{4}-\d{2}-\d{2}$/, 'Format tanggal YYYY-MM-DD wajib diisi'),
    journalSource: z.string().trim().min(1),
    currencyCode: z.string().trim().min(3, 'Mata uang wajib dipilih / Currency is required'),
    exchangeRate: z
      .string()
      .trim()
      .refine(
        (val) => isValidDecimalString(val) && new Decimal(val, 6).gt('0'),
        'Kurs harus lebih dari 0 / Exchange rate must be > 0'
      ),
    sourceDocumentRef: z.string().trim().optional(),
    lines: z
      .array(journalLineFormSchema)
      .min(2, 'Jurnal double-entry minimal memiliki 2 baris / At least 2 lines required'),
  })
  .superRefine((data, ctx) => {
    const totals = computeJournalTotals(data.lines);
    if (!totals.isPositive) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['lines'],
        message: 'Total debit dan kredit harus lebih besar dari 0 / Totals must exceed zero',
      });
    } else if (!totals.isBalanced) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['lines'],
        message: `Jurnal tidak seimbang (Selisih: ${totals.difference}) / Unbalanced entry (Diff: ${totals.difference})`,
      });
    }
  });

export type JournalFormValues = z.infer<typeof journalFormSchema>;
