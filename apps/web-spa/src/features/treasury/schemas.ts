import { z } from 'zod';
import { Decimal } from '../../lib/decimal';

const UUID_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const DECIMAL_STRING_REGEX = /^\d+(\.\d+)?$/;
const SIGNED_DECIMAL_REGEX = /^-?\d+(\.\d+)?$/;

function isPositiveDecimal(value: string): boolean {
  const trimmed = value.trim();
  if (!DECIMAL_STRING_REGEX.test(trimmed)) return false;
  return new Decimal(trimmed).isPositive();
}

function isNonZeroDecimal(value: string): boolean {
  const trimmed = value.trim();
  if (!SIGNED_DECIMAL_REGEX.test(trimmed)) return false;
  return !new Decimal(trimmed).isZero();
}

export const bankAccountFormSchema = z.object({
  code: z
    .string()
    .trim()
    .min(2, 'Account code is required')
    .max(32)
    .transform((v) => v.toUpperCase()),
  bankName: z.string().trim().min(2, 'Bank name is required').max(120),
  accountNo: z.string().trim().min(3, 'Bank account number is required').max(64),
  currencyCode: z
    .string()
    .trim()
    .length(3, '3-letter currency code required')
    .transform((v) => v.toUpperCase()),
  glAccountId: z.string().trim().regex(UUID_REGEX, 'Valid GL Account UUID is required'),
});

export type BankAccountFormValues = z.infer<typeof bankAccountFormSchema>;

export const paymentAllocationSchema = z.object({
  targetDocumentRef: z.string().trim().min(1, 'Document reference required').max(120),
  allocatedAmount: z
    .string()
    .trim()
    .refine(isPositiveDecimal, 'Allocation amount must be positive'),
});

export const paymentFormSchema = z
  .object({
    paymentNo: z.string().trim().min(1, 'Payment number is required').max(64),
    paymentType: z.enum(['Outgoing', 'Incoming']),
    bankAccountId: z.string().trim().regex(UUID_REGEX, 'Select a valid bank account'),
    partyId: z
      .string()
      .trim()
      .optional()
      .refine((v) => !v || UUID_REGEX.test(v), 'Party must be a valid UUID'),
    amount: z.string().trim().refine(isPositiveDecimal, 'Amount must be greater than zero'),
    paymentDate: z.string().trim().min(10, 'Payment date is required'),
    externalRef: z.string().trim().max(120).optional(),
    allocations: z.array(paymentAllocationSchema),
  })
  .superRefine((data, ctx) => {
    if (data.allocations.length > 0) {
      const payAmount = new Decimal(data.amount || '0');
      const allocSum = Decimal.sum(
        data.allocations.map((a) => a.allocatedAmount || '0'),
        4,
      );
      if (allocSum.gt(payAmount)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['amount'],
          message: `Allocated sum (${allocSum.toString(2)}) cannot exceed payment amount (${payAmount.toString(2)})`,
        });
      }
    }
  });

export type PaymentFormValues = z.infer<typeof paymentFormSchema>;

export const statementLineEntrySchema = z.object({
  statementDate: z.string().trim().min(10, 'Date is required'),
  description: z.string().trim().min(1, 'Description is required').max(255),
  amount: z
    .string()
    .trim()
    .refine(isNonZeroDecimal, 'Amount must be non-zero (+ credit / - debit)'),
  externalRef: z.string().trim().max(120).optional(),
});

export const statementLinesFormSchema = z.object({
  bankAccountId: z.string().trim().regex(UUID_REGEX, 'Select a valid bank account'),
  lines: z.array(statementLineEntrySchema).min(1, 'At least one statement line is required'),
});

export type StatementLinesFormValues = z.infer<typeof statementLinesFormSchema>;
