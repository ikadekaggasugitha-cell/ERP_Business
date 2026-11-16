import { z } from 'zod';
import { Decimal } from '../../lib/decimal';

const decimalRegex = /^-?\d+(\.\d{1,6})?$/;

function isPositiveDecimal(val: string): boolean {
  if (!decimalRegex.test(val.trim())) return false;
  try {
    return new Decimal(val.trim()).isPositive();
  } catch {
    return false;
  }
}

function isNonZeroDecimal(val: string): boolean {
  if (!decimalRegex.test(val.trim())) return false;
  try {
    return !new Decimal(val.trim()).isZero();
  } catch {
    return false;
  }
}

function isNonNegativeOptionalDecimal(val?: string): boolean {
  const trimmed = (val ?? '').trim();
  if (!trimmed) return true;
  if (!decimalRegex.test(trimmed)) return false;
  try {
    return !new Decimal(trimmed).isNegative();
  } catch {
    return false;
  }
}

export const warehouseFormSchema = z.object({
  code: z.string().trim().min(1, 'Warehouse code is required').max(64),
  name: z.string().trim().min(1, 'Warehouse name is required').max(255),
});

export type WarehouseFormValues = z.infer<typeof warehouseFormSchema>;

export const stockMovementFormSchema = z
  .object({
    mode: z.enum(['Receipt', 'Issue', 'Adjustment', 'Transfer']),
    movementDate: z.string().trim().optional().default(''),
    warehouseId: z.string().trim().min(1, 'Source / target warehouse is required'),
    destinationWarehouseId: z.string().trim().optional().default(''),
    itemId: z.string().trim().min(1, 'Item is required'),
    quantity: z.string().trim().min(1, 'Quantity is required'),
    unitCost: z.string().trim().optional().default(''),
    sourceDocumentRef: z.string().trim().max(128).optional().default(''),
  })
  .superRefine((val, ctx) => {
    if (val.mode === 'Adjustment') {
      if (!isNonZeroDecimal(val.quantity)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['quantity'],
          message: 'Adjustment quantity must be a non-zero decimal number',
        });
      }
    } else if (!isPositiveDecimal(val.quantity)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['quantity'],
        message: 'Quantity must be greater than 0',
      });
    }

    if (!isNonNegativeOptionalDecimal(val.unitCost)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['unitCost'],
        message: 'Unit cost cannot be negative',
      });
    }

    if (val.mode === 'Transfer') {
      if (!val.destinationWarehouseId) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['destinationWarehouseId'],
          message: 'Destination warehouse is required for transfer',
        });
      } else if (val.destinationWarehouseId === val.warehouseId) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['destinationWarehouseId'],
          message: 'Destination warehouse must differ from source warehouse',
        });
      }
    }
  });

export type StockMovementFormValues = z.infer<typeof stockMovementFormSchema>;

export const reservationFormSchema = z.object({
  warehouseId: z.string().trim().min(1, 'Warehouse is required'),
  itemId: z.string().trim().min(1, 'Item is required'),
  reservedQty: z
    .string()
    .trim()
    .refine(isPositiveDecimal, 'Reserved quantity must be greater than 0'),
  salesOrderRef: z.string().trim().max(128).optional().default(''),
});

export type ReservationFormValues = z.infer<typeof reservationFormSchema>;

export const issueReservationFormSchema = z.object({
  amount: z
    .string()
    .trim()
    .optional()
    .default('')
    .refine(
      (v) => v === '' || isPositiveDecimal(v),
      'Issue amount must be blank (full) or greater than 0'
    ),
});

export type IssueReservationFormValues = z.infer<typeof issueReservationFormSchema>;
