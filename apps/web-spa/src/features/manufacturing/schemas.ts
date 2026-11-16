import { z } from 'zod';
import { Decimal } from '../../lib/decimal';

const decimalRegex = /^\d+(\.\d{1,6})?$/;

function isPositiveDecimal(val: string): boolean {
  if (!decimalRegex.test(val.trim())) return false;
  try {
    return new Decimal(val.trim()).isPositive();
  } catch {
    return false;
  }
}

export const bomLineSchema = z.object({
  componentItemId: z.string().trim().min(1, 'Component item is required'),
  quantity: z.string().trim().refine(isPositiveDecimal, 'Component quantity must be > 0'),
  uom: z
    .string()
    .trim()
    .min(1, 'UOM is required')
    .max(16)
    .transform((v) => v.toUpperCase()),
});

export const bomFormSchema = z.object({
  code: z.string().trim().min(1, 'BOM code is required').max(64),
  productItemId: z.string().trim().min(1, 'Finished product item is required'),
  version: z.coerce.number().int().min(1, 'Version must be >= 1').default(1),
  lines: z.array(bomLineSchema).min(1, 'At least one BOM component line is required'),
});

export type BomFormValues = z.infer<typeof bomFormSchema>;

export const workOrderFormSchema = z.object({
  woNo: z.string().trim().min(1, 'Work order number is required').max(64),
  productItemId: z.string().trim().min(1, 'Product item is required'),
  bomId: z.string().trim().min(1, 'Bill of Materials is required'),
  plannedQty: z.string().trim().refine(isPositiveDecimal, 'Planned quantity must be > 0'),
  salesOrderRef: z.string().trim().max(128).optional().default(''),
});

export type WorkOrderFormValues = z.infer<typeof workOrderFormSchema>;

export const completeWorkOrderFormSchema = z.object({
  producedQty: z.string().trim().refine(isPositiveDecimal, 'Produced quantity must be > 0'),
});

export type CompleteWorkOrderFormValues = z.infer<typeof completeWorkOrderFormSchema>;
