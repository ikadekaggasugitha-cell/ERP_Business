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

export const apInvoiceLineSchema = z.object({
  accountId: z.string().trim().regex(UUID_REGEX, 'Valid GL Account UUID is required'),
  amount: z
    .string()
    .trim()
    .refine(isPositiveDecimal, 'Line amount must be greater than zero'),
  description: z.string().trim().min(1, 'Line description is required'),
});

export const apInvoiceFormSchema = z
  .object({
    invoiceNo: z.string().trim().min(1, 'Invoice number is required').max(64),
    vendorId: z.string().trim().regex(UUID_REGEX, 'Valid Vendor UUID is required'),
    invoiceDate: z.string().trim().min(10, 'Invoice date is required'),
    dueDate: z.string().trim().min(10, 'Due date is required'),
    currencyCode: z
      .string()
      .trim()
      .length(3, 'Currency must be a 3-letter ISO code')
      .transform((val) => val.toUpperCase()),
    totalAmount: z
      .string()
      .trim()
      .refine(isPositiveDecimal, 'Total amount must be greater than zero'),
    goodsReceiptRef: z.string().trim().max(120).optional(),
    lines: z.array(apInvoiceLineSchema).min(1, 'At least one expense/asset line is required'),
  })
  .superRefine((data, ctx) => {
    if (data.dueDate < data.invoiceDate) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['dueDate'],
        message: 'Due date cannot be earlier than invoice date',
      });
    }
    const total = new Decimal(data.totalAmount || '0');
    const lineSum = Decimal.sum(
      data.lines.map((line) => (isNonNegativeDecimal(line.amount) ? line.amount : '0')),
      4,
    );
    if (!total.equals(lineSum)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['totalAmount'],
        message: `Line sum (${lineSum.toString(2)}) must match invoice total (${total.toString(2)})`,
      });
    }
  });

export type ApInvoiceFormValues = z.infer<typeof apInvoiceFormSchema>;

export function computeLinesTotal(lines: Array<{ amount: string }>): string {
  return Decimal.sum(
    lines.map((line) => {
      const trimmed = line.amount?.trim() || '0';
      return isNonNegativeDecimal(trimmed) ? trimmed : '0';
    }),
    4,
  ).toRawString();
}
