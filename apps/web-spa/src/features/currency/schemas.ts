import { z } from 'zod';
import { Decimal } from '../../lib/decimal';

const DECIMAL_STRING_REGEX = /^\d+(\.\d+)?$/;

function isPositiveDecimal(value: string): boolean {
  const trimmed = value.trim();
  if (!DECIMAL_STRING_REGEX.test(trimmed)) return false;
  return new Decimal(trimmed).isPositive();
}

export const currencyFormSchema = z.object({
  code: z
    .string()
    .trim()
    .length(3, 'ISO-4217 currency code must be exactly 3 letters')
    .transform((v) => v.toUpperCase()),
  numericCode: z
    .string()
    .trim()
    .max(3, 'Numeric code max 3 digits')
    .optional(),
  name: z.string().trim().min(2, 'Currency name is required').max(120),
  precision: z
    .number()
    .int()
    .min(0, 'Precision cannot be negative')
    .max(8, 'Max precision is 8'),
  isActive: z.boolean(),
});

export type CurrencyFormValues = z.infer<typeof currencyFormSchema>;

export const rateTypeFormSchema = z.object({
  code: z
    .string()
    .trim()
    .min(2, 'Rate type code is required')
    .max(24)
    .transform((v) => v.toUpperCase()),
  name: z.string().trim().min(2, 'Rate type name is required').max(120),
});

export type RateTypeFormValues = z.infer<typeof rateTypeFormSchema>;

export const exchangeRateFormSchema = z
  .object({
    fromCurrency: z
      .string()
      .trim()
      .length(3, '3-letter currency required')
      .transform((v) => v.toUpperCase()),
    toCurrency: z
      .string()
      .trim()
      .length(3, '3-letter currency required')
      .transform((v) => v.toUpperCase()),
    rateTypeId: z.string().trim().min(1, 'Rate type is required'),
    rate: z.string().trim().refine(isPositiveDecimal, 'FX rate must be greater than zero'),
    validFrom: z.string().trim().min(10, 'Valid-from date is required'),
    validTo: z.string().trim().optional(),
  })
  .superRefine((data, ctx) => {
    if (data.fromCurrency === data.toCurrency) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['toCurrency'],
        message: 'Target currency must differ from source currency',
      });
    }
    if (data.validTo && data.validTo < data.validFrom) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['validTo'],
        message: 'Valid-to date cannot precede valid-from date',
      });
    }
  });

export type ExchangeRateFormValues = z.infer<typeof exchangeRateFormSchema>;

export const convertFxFormSchema = z.object({
  fromCurrency: z
    .string()
    .trim()
    .length(3, '3-letter currency required')
    .transform((v) => v.toUpperCase()),
  toCurrency: z
    .string()
    .trim()
    .length(3, '3-letter currency required')
    .transform((v) => v.toUpperCase()),
  rateTypeCode: z.string().trim().min(1, 'Rate type required'),
  amount: z.string().trim().refine(isPositiveDecimal, 'Amount must be greater than zero'),
  onDate: z.string().trim().min(10, 'Valuation date required'),
  anchor: z.string().trim().optional(),
});

export type ConvertFxFormValues = z.infer<typeof convertFxFormSchema>;
