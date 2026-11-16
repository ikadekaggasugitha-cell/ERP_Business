import { useEffect, useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { MoneyInput } from '../../components/ui/MoneyInput';
import { Select } from '../../components/ui/Select';
import { Switch } from '../../components/ui/Switch';
import { ApiClientError, applyApiErrorsToForm } from '../../lib/api';
import { formatCurrency } from '../../lib/utils';
import { useUiStore } from '../../store/ui-store';
import {
  createTaxCode,
  createTaxInvoice,
  reportTaxInvoiceEfaktur,
  taxQueryKeys,
} from './api';
import {
  computeTaxPreview,
  reportEfakturSchema,
  taxCodeFormSchema,
  taxInvoiceFormSchema,
  type ReportEfakturFormValues,
  type TaxCodeFormValues,
  type TaxInvoiceFormValues,
} from './schemas';
import type { TaxAccountLookupDto, TaxCodeDto, TaxInvoiceDto } from './types';

interface CreateTaxInvoiceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  taxCodes: TaxCodeDto[];
}

export function CreateTaxInvoiceDialog({
  open,
  onOpenChange,
  taxCodes,
}: CreateTaxInvoiceDialogProps) {
  const queryClient = useQueryClient();
  const locale = useUiStore((s) => s.locale);
  const [manualCodeUuid, setManualCodeUuid] = useState(false);
  const [overrideTaxAmount, setOverrideTaxAmount] = useState(false);

  const taxCodeOptions = useMemo(
    () =>
      taxCodes.map((code) => ({
        value: code.id,
        label: `${code.code} (${code.rate}%)`,
        subtitle: code.taxType,
        keywords: [code.code, code.taxType, code.rate],
      })),
    [taxCodes],
  );

  const taxCodeById = useMemo(
    () => new Map(taxCodes.map((code) => [code.id, code])),
    [taxCodes],
  );

  const form = useForm<TaxInvoiceFormValues>({
    resolver: zodResolver(taxInvoiceFormSchema),
    defaultValues: {
      sourceDocumentRef: '',
      taxCodeId: '',
      baseAmount: '0',
      taxAmount: '',
      efakturNo: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    setManualCodeUuid(taxCodes.length === 0);
    setOverrideTaxAmount(false);
    form.reset({
      sourceDocumentRef: '',
      taxCodeId: taxCodes[0]?.id ?? '',
      baseAmount: '0',
      taxAmount: '',
      efakturNo: '',
    });
  }, [open, taxCodes, form]);

  const watchedBase = form.watch('baseAmount');
  const watchedTaxCodeId = form.watch('taxCodeId');
  const selectedCode = taxCodeById.get(watchedTaxCodeId);
  const previewTax = useMemo(
    () => computeTaxPreview(watchedBase, selectedCode?.rate ?? '0'),
    [watchedBase, selectedCode],
  );

  const mutation = useMutation({
    mutationFn: (values: TaxInvoiceFormValues) =>
      createTaxInvoice({
        sourceDocumentRef: values.sourceDocumentRef,
        taxCodeId: values.taxCodeId,
        baseAmount: values.baseAmount,
        taxAmount: overrideTaxAmount ? values.taxAmount : undefined,
        efakturNo: values.efakturNo,
      }),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: taxQueryKeys.invoicesAll });
      toast.success(`Tax invoice for ${created.sourceDocumentRef} created`);
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to create tax invoice';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="lg"
      title="New e-Faktur / Tax Invoice"
      description="Register PPN or PPh tax base and tax calculation against a source document."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="create-tax-invoice-form" isLoading={mutation.isPending}>
            Create Tax Invoice
          </Button>
        </>
      }
    >
      <form
        id="create-tax-invoice-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <FormField
          label="Source Document Ref"
          htmlFor="tax-source-ref"
          required
          error={form.formState.errors.sourceDocumentRef?.message}
        >
          <Input
            id="tax-source-ref"
            placeholder="INV-2026-0001 or BILL-2026-0014"
            error={Boolean(form.formState.errors.sourceDocumentRef)}
            {...form.register('sourceDocumentRef')}
          />
        </FormField>

        <FormField
          label="Tax Code"
          htmlFor="tax-code-select"
          required
          error={form.formState.errors.taxCodeId?.message}
          hint={
            taxCodes.length === 0
              ? 'Create a Tax Code first or enter UUID manually'
              : undefined
          }
        >
          <div className="space-y-1.5">
            {!manualCodeUuid && taxCodeOptions.length > 0 ? (
              <Controller
                control={form.control}
                name="taxCodeId"
                render={({ field }) => (
                  <Combobox
                    id="tax-code-select"
                    value={field.value}
                    onValueChange={field.onChange}
                    options={taxCodeOptions}
                    placeholder="Select tax code..."
                    searchPlaceholder="Search tax code or type..."
                    error={Boolean(form.formState.errors.taxCodeId)}
                  />
                )}
              />
            ) : (
              <Input
                id="tax-code-select"
                placeholder="Tax Code UUID"
                className="font-mono text-xs"
                error={Boolean(form.formState.errors.taxCodeId)}
                {...form.register('taxCodeId')}
              />
            )}
            {taxCodes.length > 0 && (
              <button
                type="button"
                className="text-xs font-medium text-brand-700 hover:underline"
                onClick={() => setManualCodeUuid((prev) => !prev)}
              >
                {manualCodeUuid ? 'Use tax code list' : 'Enter Tax Code UUID manually'}
              </button>
            )}
          </div>
        </FormField>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="DPP / Base Amount"
            htmlFor="tax-base-amount"
            required
            error={form.formState.errors.baseAmount?.message}
          >
            <Controller
              control={form.control}
              name="baseAmount"
              render={({ field }) => (
                <MoneyInput
                  id="tax-base-amount"
                  value={field.value}
                  onValueChange={field.onChange}
                  currency="IDR"
                  error={Boolean(form.formState.errors.baseAmount)}
                />
              )}
            />
          </FormField>

          <FormField
            label="e-Faktur NSFP (Optional)"
            htmlFor="tax-efaktur-no"
            error={form.formState.errors.efakturNo?.message}
          >
            <Input
              id="tax-efaktur-no"
              placeholder="010.002-26.00000001"
              className="font-mono text-xs"
              error={Boolean(form.formState.errors.efakturNo)}
              {...form.register('efakturNo')}
            />
          </FormField>
        </div>

        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p className="text-xs font-semibold text-slate-700">
                Estimated Tax ({selectedCode ? `${selectedCode.rate}%` : '—'})
              </p>
              <p className="font-mono text-sm font-bold text-slate-900 tabular-nums">
                {formatCurrency(previewTax, 'IDR', locale === 'id' ? 'id-ID' : 'en-US')}
              </p>
            </div>
            <Switch
              checked={overrideTaxAmount}
              onCheckedChange={setOverrideTaxAmount}
              label="Override calculated tax"
            />
          </div>

          {overrideTaxAmount && (
            <FormField
              label="Custom Tax Amount"
              htmlFor="tax-custom-amount"
              error={form.formState.errors.taxAmount?.message}
            >
              <Controller
                control={form.control}
                name="taxAmount"
                render={({ field }) => (
                  <MoneyInput
                    id="tax-custom-amount"
                    value={field.value || '0'}
                    onValueChange={field.onChange}
                    currency="IDR"
                    error={Boolean(form.formState.errors.taxAmount)}
                  />
                )}
              />
            </FormField>
          )}
        </div>
      </form>
    </AppDialog>
  );
}

interface CreateTaxCodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  accounts: TaxAccountLookupDto[];
}

export function CreateTaxCodeDialog({
  open,
  onOpenChange,
  accounts,
}: CreateTaxCodeDialogProps) {
  const queryClient = useQueryClient();
  const [manualGlUuid, setManualGlUuid] = useState(false);

  const accountOptions = useMemo(
    () =>
      accounts
        .filter((a) => a.allowPosting !== false)
        .map((a) => ({
          value: a.id,
          label: `${a.code} — ${a.name}`,
          subtitle: a.accountType,
          keywords: [a.code, a.name, a.accountType],
        })),
    [accounts],
  );

  const form = useForm<TaxCodeFormValues>({
    resolver: zodResolver(taxCodeFormSchema),
    defaultValues: {
      code: '',
      rate: '11.0000',
      taxType: 'VAT',
      glAccountId: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    setManualGlUuid(accountOptions.length === 0);
    form.reset({
      code: '',
      rate: '11.0000',
      taxType: 'VAT',
      glAccountId: '',
    });
  }, [open, form, accountOptions.length]);

  const mutation = useMutation({
    mutationFn: (values: TaxCodeFormValues) =>
      createTaxCode({
        code: values.code,
        rate: values.rate,
        taxType: values.taxType,
        glAccountId: values.glAccountId,
      }),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: taxQueryKeys.codesAll });
      toast.success(`Tax code ${created.code} (${created.rate}%) created`);
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to create tax code';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="md"
      title="New Tax Code"
      description="Configure PPN/VAT or PPh withholding tax rule."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="create-tax-code-form" isLoading={mutation.isPending}>
            Save Tax Code
          </Button>
        </>
      }
    >
      <form
        id="create-tax-code-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Tax Code"
            htmlFor="tax-code-input"
            required
            error={form.formState.errors.code?.message}
          >
            <Input
              id="tax-code-input"
              placeholder="PPN-11"
              className="uppercase font-mono"
              error={Boolean(form.formState.errors.code)}
              {...form.register('code')}
            />
          </FormField>

          <FormField
            label="Tax Rate (%)"
            htmlFor="tax-rate-input"
            required
            error={form.formState.errors.rate?.message}
          >
            <Input
              id="tax-rate-input"
              numeric
              placeholder="11.0000"
              error={Boolean(form.formState.errors.rate)}
              {...form.register('rate')}
            />
          </FormField>
        </div>

        <FormField
          label="Tax Type"
          htmlFor="tax-type-select"
          required
          error={form.formState.errors.taxType?.message}
        >
          <Controller
            control={form.control}
            name="taxType"
            render={({ field }) => (
              <Select
                id="tax-type-select"
                value={field.value}
                onValueChange={field.onChange}
                options={[
                  { value: 'VAT', label: 'VAT / PPN (Output/Input)' },
                  { value: 'PPN', label: 'PPN (Indonesian VAT)' },
                  { value: 'PPH21', label: 'PPh Pasal 21' },
                  { value: 'PPH23', label: 'PPh Pasal 23' },
                  { value: 'PPH4_2', label: 'PPh Pasal 4(2) Final' },
                  { value: 'WITHHOLDING', label: 'General Withholding Tax' },
                ]}
                error={Boolean(form.formState.errors.taxType)}
              />
            )}
          />
        </FormField>

        <FormField
          label="GL Tax Account (Optional)"
          htmlFor="tax-gl-account"
          error={form.formState.errors.glAccountId?.message}
        >
          <div className="space-y-1.5">
            {!manualGlUuid && accountOptions.length > 0 ? (
              <Controller
                control={form.control}
                name="glAccountId"
                render={({ field }) => (
                  <Combobox
                    id="tax-gl-account"
                    value={field.value ?? ''}
                    onValueChange={field.onChange}
                    options={accountOptions}
                    allowClear
                    placeholder="Optional GL tax liability/asset account..."
                    searchPlaceholder="Search account code or name..."
                    error={Boolean(form.formState.errors.glAccountId)}
                  />
                )}
              />
            ) : (
              <Input
                id="tax-gl-account"
                placeholder="Optional GL Account UUID"
                className="font-mono text-xs"
                error={Boolean(form.formState.errors.glAccountId)}
                {...form.register('glAccountId')}
              />
            )}
            {accountOptions.length > 0 && (
              <button
                type="button"
                className="text-xs font-medium text-brand-700 hover:underline"
                onClick={() => setManualGlUuid((prev) => !prev)}
              >
                {manualGlUuid ? 'Select from CoA' : 'Enter UUID manually'}
              </button>
            )}
          </div>
        </FormField>
      </form>
    </AppDialog>
  );
}

interface ReportEfakturDialogProps {
  invoice: TaxInvoiceDto | null;
  onOpenChange: (open: boolean) => void;
}

export function ReportEfakturDialog({ invoice, onOpenChange }: ReportEfakturDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm<ReportEfakturFormValues>({
    resolver: zodResolver(reportEfakturSchema),
    defaultValues: {
      efakturNo: '',
      ntte: '',
    },
  });

  useEffect(() => {
    if (!invoice) return;
    form.reset({
      efakturNo: invoice.efakturNo ?? '',
      ntte: '',
    });
  }, [invoice, form]);

  const mutation = useMutation({
    mutationFn: async (values: ReportEfakturFormValues) => {
      if (!invoice) {
        throw new Error('No tax invoice selected');
      }
      return reportTaxInvoiceEfaktur(invoice.id, invoice.rowVersion, values);
    },
    onSuccess: async (reported) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: taxQueryKeys.invoicesAll }),
        queryClient.invalidateQueries({ queryKey: taxQueryKeys.invoiceDetail(reported.id) }),
      ]);
      toast.success(`e-Faktur ${reported.efakturNo || reported.sourceDocumentRef} reported to DJP`);
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Failed to report e-Faktur to DJP';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={Boolean(invoice)}
      onOpenChange={onOpenChange}
      size="md"
      title={`Report to DJP · ${invoice?.sourceDocumentRef ?? ''}`}
      description="Submit e-Faktur NSFP and optional DJP NTTE receipt to finalize tax reporting."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="report-efaktur-form" isLoading={mutation.isPending}>
            Confirm Report to DJP
          </Button>
        </>
      }
    >
      <form
        id="report-efaktur-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <FormField
          label="e-Faktur Number (NSFP)"
          htmlFor="report-efaktur-no"
          error={form.formState.errors.efakturNo?.message}
          description="Leave empty to retain existing assigned e-Faktur number."
        >
          <Input
            id="report-efaktur-no"
            placeholder="010.002-26.00000001"
            className="font-mono text-xs"
            {...form.register('efakturNo')}
          />
        </FormField>

        <FormField
          label="NTTE Receipt Reference (Optional)"
          htmlFor="report-efaktur-ntte"
          error={form.formState.errors.ntte?.message}
        >
          <Input
            id="report-efaktur-ntte"
            placeholder="NTTE-2026-XXXX"
            className="font-mono text-xs"
            {...form.register('ntte')}
          />
        </FormField>
      </form>
    </AppDialog>
  );
}
