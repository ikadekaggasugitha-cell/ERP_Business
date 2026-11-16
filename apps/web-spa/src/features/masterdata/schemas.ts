import { z } from 'zod';

export const partyFormSchema = z
  .object({
    code: z.string().trim().min(1, 'Party code is required').max(64, 'Max 64 characters'),
    name: z.string().trim().min(1, 'Party name is required').max(255, 'Max 255 characters'),
    partyType: z.enum(['customer', 'vendor', 'both']),
    taxId: z.string().trim().max(64, 'Max 64 characters').optional().default(''),
    isShared: z.boolean().default(false),
    includePrimaryAddress: z.boolean().default(false),
    addressType: z.enum(['billing', 'shipping', 'headquarters']).default('billing'),
    line1: z.string().trim().max(255).optional().default(''),
    city: z.string().trim().max(120).optional().default(''),
    postalCode: z.string().trim().max(32).optional().default(''),
    countryCode: z.string().trim().max(8).optional().default('ID'),
  })
  .superRefine((val, ctx) => {
    if (val.includePrimaryAddress && val.line1.trim().length === 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['line1'],
        message: 'Address line 1 is required when adding a primary address',
      });
    }
  });

export type PartyFormValues = z.infer<typeof partyFormSchema>;

export const itemFormSchema = z.object({
  code: z.string().trim().min(1, 'Item SKU code is required').max(64, 'Max 64 characters'),
  name: z.string().trim().min(1, 'Item name is required').max(255, 'Max 255 characters'),
  itemType: z.enum(['stock', 'service', 'asset']),
  baseUom: z
    .string()
    .trim()
    .min(1, 'Base UOM is required')
    .max(16, 'Max 16 characters')
    .transform((v) => v.toUpperCase()),
});

export type ItemFormValues = z.infer<typeof itemFormSchema>;
