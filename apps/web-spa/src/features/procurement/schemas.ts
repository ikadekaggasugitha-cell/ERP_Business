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

/**
 * Computes exact sum(orderQty * unitPrice) using BigInt fixed-point math
 * matching Go's big.Rat exact comparison without IEEE-754 float error.
 */
export function computeOrderTotalDecimal(
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

export const purchaseOrderLineSchema = z.object({
  itemId: z.string().trim().min(1, 'Item is required'),
  orderQty: z.string().trim().refine(isPositiveDecimal, 'Quantity must be > 0'),
  unitPrice: z.string().trim().refine(isNonNegativeDecimal, 'Unit price must be >= 0'),
});

export const purchaseOrderFormSchema = z.object({
  poNo: z.string().trim().min(1, 'PO number is required').max(64),
  vendorId: z.string().trim().min(1, 'Vendor is required'),
  purchaseRequisitionId: z.string().trim().optional().default(''),
  orderDate: z.string().trim().regex(/^\d{4}-\d{2}-\d{2}$/, 'Order date must be YYYY-MM-DD'),
  currencyCode: z
    .string()
    .trim()
    .length(3, 'ISO 3-letter currency code required')
    .transform((v) => v.toUpperCase()),
  lines: z.array(purchaseOrderLineSchema).min(1, 'At least one PO line is required'),
});

export type PurchaseOrderFormValues = z.infer<typeof purchaseOrderFormSchema>;

export const receivePurchaseOrderSchema = z.object({
  grNo: z.string().trim().max(64).optional().default(''),
  warehouseId: z.string().trim().min(1, 'Receiving warehouse is required'),
  receiptDate: z
    .string()
    .trim()
    .regex(/^\d{4}-\d{2}-\d{2}$/, 'Receipt date must be YYYY-MM-DD'),
});

export type ReceivePurchaseOrderFormValues = z.infer<typeof receivePurchaseOrderSchema>;

export const purchaseRequisitionLineSchema = z.object({
  itemId: z.string().trim().min(1, 'Item is required'),
  requestQty: z.string().trim().refine(isPositiveDecimal, 'Request quantity must be > 0'),
  neededDate: z.string().trim().optional().default(''),
});

export const purchaseRequisitionFormSchema = z.object({
  prNo: z.string().trim().min(1, 'PR number is required').max(64),
  neededDate: z.string().trim().optional().default(''),
  workflowRef: z.string().trim().max(128).optional().default(''),
  lines: z.array(purchaseRequisitionLineSchema).min(1, 'At least one PR line is required'),
});

export type PurchaseRequisitionFormValues = z.infer<typeof purchaseRequisitionFormSchema>;
