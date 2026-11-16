import { z } from 'zod';
import { Decimal } from '../../lib/decimal';

const UUID_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const DECIMAL_STRING_REGEX = /^\d+(\.\d+)?$/;

function isPositiveDecimal(value: string): boolean {
  const trimmed = value.trim();
  if (!DECIMAL_STRING_REGEX.test(trimmed)) return false;
  return new Decimal(trimmed).isPositive();
}

function isNonNegativeDecimal(value: string): boolean {
  const trimmed = value.trim();
  if (!DECIMAL_STRING_REGEX.test(trimmed)) return false;
  const parsed = new Decimal(trimmed);
  return parsed.isPositive() || parsed.isZero();
}

export const taxCodeFormSchema = z.object({
  code: z
    .string()
    .trim()
    .min(2, 'Tax code is required')
    .max(32, 'Max 32 characters')
    .transform((v) => v.toUpperCase()),
  rate: z
    .string()
    .trim()
    .refine(isNonNegativeDecimal, 'Rate (%) must be zero or positive'),
  taxType: z.string().trim().min(1, 'Tax type is required'),
  glAccountId: z
    .string()
    .trim()
    .optional()
    .refine((v) => !v || UUID_REGEX.test(v), 'GL Account must be a valid UUID'),
});

export type TaxCodeFormValues = z.infer<typeof taxCodeFormSchema>;

export const taxInvoiceFormSchema = z.object({
  sourceDocumentRef: z
    .string()
    .trim()
    .min(1, 'Source document reference is required')
    .max(120),
  taxCodeId: z.string().trim().regex(UUID_REGEX, 'Select a valid Tax Code'),
  baseAmount: z
    .string()
    .trim()
    .refine(isPositiveDecimal, 'DPP / Base amount must be greater than zero'),
  taxAmount: z
    .string()
    .trim()
    .optional()
    .refine(
      (v) => !v || isNonNegativeDecimal(v),
      'Tax amount must be zero or positive',
    ),
  efakturNo: z.string().trim().max(40).optional(),
});

export type TaxInvoiceFormValues = z.infer<typeof taxInvoiceFormSchema>;

export const reportEfakturSchema = z.object({
  efakturNo: z.string().trim().max(40).optional(),
  ntte: z.string().trim().max(64).optional(),
});

export type ReportEfakturFormValues = z.infer<typeof reportEfakturSchema>;

export function computeTaxPreview(baseAmount: string, ratePercent: string): string {
  const base = new Decimal(DECIMAL_STRING_REGEX.test(baseAmount.trim()) ? baseAmount : '0');
  const rate = new Decimal(DECIMAL_STRING_REGEX.test(ratePercent.trim()) ? ratePercent : '0');
  return base.mul(rate, 4).div(new Decimal('100'), 2).toString(2);
}
