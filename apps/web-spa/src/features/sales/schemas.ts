import { z } from 'zod';
import { Decimal, addDecimals, multiplyDecimals } from '../../lib/decimal';

const decimalRegex = /^\d+(\.\d{1,6})?$/;

function isPositiveDecimal(val: string): boolean {
  if (!decimalRegex.test(val.trim())) return false;
  try {
    return new Decimal(val.trim()).isPositive();
  } catch {
    return false;
  }
}

function isNonNegativeDecimal(val: string): boolean {
  if (!decimalRegex.test(val.trim())) return false;
  try {
    return !new Decimal(val.trim()).isNegative();
  } catch {
    return false;
  }
}

export function computeSalesOrderTotalDecimal(
  lines: Array<{ orderQty?: string; unitPrice?: string }>
): string {
  let acc = '0.00000000';
  for (const line of lines) {
    const qty = (line.orderQty ?? '').trim();
    const price = (line.unitPrice ?? '').trim();
    if (!qty || !price || !decimalRegex.test(qty) || !decimalRegex.test(price)) {
      continue;
    }
    const lineTotal = multiplyDecimals(qty, price, 8);
    acc = addDecimals(acc, lineTotal, 8);
  }
  const [intPart = '0', fracRaw = ''] = acc.split('.');
  const trimmedExtra = fracRaw.replace(/0+$/, '');
  const paddedFrac = trimmedExtra.length <= 4 ? trimmedExtra.padEnd(4, '0') : trimmedExtra;
  return `${intPart}.${paddedFrac}`;
}

export const salesOrderLineSchema = z.object({
  itemId: z.string().trim().min(1, 'Item is required'),
  orderQty: z.string().trim().refine(isPositiveDecimal, 'Order quantity must be > 0'),
  unitPrice: z.string().trim().refine(isNonNegativeDecimal, 'Unit price must be >= 0'),
});

export const salesOrderFormSchema = z.object({
  soNo: z.string().trim().min(1, 'Sales order number is required').max(64),
  customerId: z.string().trim().min(1, 'Customer is required'),
  orderDate: z.string().trim().regex(/^\d{4}-\d{2}-\d{2}$/, 'Order date must be YYYY-MM-DD'),
  currencyCode: z
    .string()
    .trim()
    .length(3, 'ISO 3-letter currency code required')
    .transform((v) => v.toUpperCase()),
  lines: z.array(salesOrderLineSchema).min(1, 'At least one SO line is required'),
});

export type SalesOrderFormValues = z.infer<typeof salesOrderFormSchema>;

export const confirmSalesOrderFormSchema = z.object({
  warehouseId: z.string().trim().optional().default(''),
});

export type ConfirmSalesOrderFormValues = z.infer<typeof confirmSalesOrderFormSchema>;

export const createDeliveryFormSchema = z.object({
  deliveryNo: z.string().trim().max(64).optional().default(''),
  warehouseId: z.string().trim().optional().default(''),
  deliveryDate: z
    .string()
    .trim()
    .regex(/^\d{4}-\d{2}-\d{2}$/, 'Delivery date must be YYYY-MM-DD'),
});

export type CreateDeliveryFormValues = z.infer<typeof createDeliveryFormSchema>;
