import { useEffect, useMemo, useState } from 'react';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Sparkles } from 'lucide-react';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { MoneyInput } from '../../components/ui/MoneyInput';
import { Select } from '../../components/ui/Select';
import { ApiClientError, applyApiErrorsToForm } from '../../lib/api';
import { Decimal } from '../../lib/decimal';
import { formatCurrency } from '../../lib/utils';
import { useUiStore } from '../../store/ui-store';
import { apQueryKeys, createVendorInvoice, updateVendorInvoice } from './api';
import {
  apInvoiceFormSchema,
  computeLinesTotal,
  type ApInvoiceFormValues,
} from './schemas';
import type {
  CoaAccountLookupDto,
  CurrencyLookupDto,
  PartyLookupDto,
  VendorInvoiceDto,
} from './types';

interface ApInvoiceFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingInvoice: VendorInvoiceDto | null;
  vendors: PartyLookupDto[];
  accounts: CoaAccountLookupDto[];
  currencies: CurrencyLookupDto[];
}

function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

function addDaysIso(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() + days);
  return date.toISOString().slice(0, 10);
}

const DEFAULT_CURRENCY_CODES = ['IDR', 'USD', 'EUR', 'SGD'];

export function ApInvoiceFormDialog({
  open,
  onOpenChange,
  editingInvoice,
  vendors,
  accounts,
  currencies,
}: ApInvoiceFormDialogProps) {
  const queryClient = useQueryClient();
  const locale = useUiStore((s) => s.locale);
  const isEdit = Boolean(editingInvoice);
  const [manualVendorUuid, setManualVendorUuid] = useState(false);
  const [manualAccountLines, setManualAccountLines] = useState<Record<number, boolean>>({});

  const vendorOptions = useMemo(
    () =>
      vendors
        .filter((p) => p.partyType === 'vendor' || p.partyType === 'both' || !p.partyType)
        .map((p) => ({
          value: p.id,
          label: p.name,
          subtitle: `${p.code}${p.taxId ? ` · NPWP ${p.taxId}` : ''}`,
          keywords: [p.code, p.name, p.taxId ?? ''],
        })),
    [vendors],
  );

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

  const currencyOptions = useMemo(() => {
    const base =
      currencies.length > 0
        ? currencies.map((c) => ({
            value: c.code,
            label: `${c.code} — ${c.name}`,
          }))
        : DEFAULT_CURRENCY_CODES.map((code) => ({ value: code, label: code }));
    return base;
  }, [currencies]);

  const form = useForm<ApInvoiceFormValues>({
    resolver: zodResolver(apInvoiceFormSchema),
    defaultValues: {
      invoiceNo: '',
      vendorId: '',
      invoiceDate: todayIso(),
      dueDate: addDaysIso(30),
      currencyCode: 'IDR',
      totalAmount: '0',
      goodsReceiptRef: '',
      lines: [{ accountId: '', amount: '0', description: '' }],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'lines',
  });

  useEffect(() => {
    if (!open) return;
    setManualVendorUuid(vendorOptions.length === 0);
    setManualAccountLines({});

    if (editingInvoice) {
      form.reset({
        invoiceNo: editingInvoice.invoiceNo,
        vendorId: editingInvoice.vendorId,
        invoiceDate: editingInvoice.invoiceDate.slice(0, 10),
        dueDate: editingInvoice.dueDate.slice(0, 10),
        currencyCode: editingInvoice.currencyCode || 'IDR',
        totalAmount: editingInvoice.totalAmount,
        goodsReceiptRef: editingInvoice.goodsReceiptRef ?? '',
        lines:
          editingInvoice.lines && editingInvoice.lines.length > 0
            ? editingInvoice.lines.map((line) => ({
                accountId: line.accountId,
                amount: line.amount,
                description: line.description,
              }))
            : [{ accountId: '', amount: editingInvoice.totalAmount, description: 'Vendor payable' }],
      });
    } else {
      form.reset({
        invoiceNo: `BILL-${new Date().getFullYear()}-${String(Date.now()).slice(-4)}`,
        vendorId: '',
        invoiceDate: todayIso(),
        dueDate: addDaysIso(30),
        currencyCode: 'IDR',
        totalAmount: '0',
        goodsReceiptRef: '',
        lines: [{ accountId: '', amount: '0', description: '' }],
      });
    }
  }, [open, editingInvoice, form, vendorOptions.length]);

  const watchedLines = form.watch('lines');
  const watchedTotal = form.watch('totalAmount');
  const watchedCurrency = form.watch('currencyCode') || 'IDR';
  const computedLinesSum = computeLinesTotal(watchedLines || []);

  // `form.watch('lines')` returns the live (mutated in place) array, so a useMemo
  // keyed on its reference never invalidates — compute per render instead.
  const linesBalanced = new Decimal(watchedTotal || '0').equals(
    new Decimal(computedLinesSum || '0'),
  );

  const mutation = useMutation({
    mutationFn: async (values: ApInvoiceFormValues) => {
      const normalizedLines = values.lines.map((line, index) => ({
        lineNo: index + 1,
        accountId: line.accountId.trim(),
        amount: line.amount.trim(),
        description: line.description.trim(),
      }));

      if (editingInvoice) {
        return updateVendorInvoice(editingInvoice.id, editingInvoice.rowVersion, {
          vendorId: values.vendorId.trim(),
          dueDate: values.dueDate,
          currencyCode: values.currencyCode,
          totalAmount: values.totalAmount.trim(),
          goodsReceiptRef: values.goodsReceiptRef?.trim() || undefined,
          lines: normalizedLines,
        });
      }

      return createVendorInvoice({
        invoiceNo: values.invoiceNo.trim(),
        vendorId: values.vendorId.trim(),
        invoiceDate: values.invoiceDate,
        dueDate: values.dueDate,
        currencyCode: values.currencyCode,
        totalAmount: values.totalAmount.trim(),
        goodsReceiptRef: values.goodsReceiptRef?.trim() || undefined,
        lines: normalizedLines,
      });
    },
    onSuccess: async (saved) => {
      await queryClient.invalidateQueries({ queryKey: apQueryKeys.all });
      toast.success(
        isEdit
          ? `Vendor invoice ${saved.invoiceNo} updated`
          : `Vendor invoice ${saved.invoiceNo} created`,
      );
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const message =
        err instanceof ApiClientError ? err.message : 'Unable to save vendor invoice';
      toast.error(message);
    },
  });

  const handleMatchLineSum = () => {
    form.setValue('totalAmount', computedLinesSum, { shouldValidate: true, shouldDirty: true });
  };

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="2xl"
      title={isEdit ? `Edit Vendor Invoice · ${editingInvoice?.invoiceNo}` : 'New Vendor Invoice'}
      description="Enter supplier bill details and balanced GL expense/asset distribution lines."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            form="ap-invoice-form"
            isLoading={mutation.isPending}
          >
            {isEdit ? 'Save Changes' : 'Create Draft Bill'}
          </Button>
        </>
      }
    >
      <form
        id="ap-invoice-form"
        className="space-y-5"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <FormField
            label="Invoice Number"
            htmlFor="ap-invoice-no"
            required
            error={form.formState.errors.invoiceNo?.message}
          >
            <Input
              id="ap-invoice-no"
              placeholder="BILL-2026-0001"
              disabled={isEdit}
              error={Boolean(form.formState.errors.invoiceNo)}
              {...form.register('invoiceNo')}
            />
          </FormField>

          <FormField
            label="Invoice Date"
            htmlFor="ap-invoice-date"
            required
            error={form.formState.errors.invoiceDate?.message}
          >
            <Input
              id="ap-invoice-date"
              type="date"
              disabled={isEdit}
              error={Boolean(form.formState.errors.invoiceDate)}
              {...form.register('invoiceDate')}
            />
          </FormField>

          <FormField
            label="Due Date"
            htmlFor="ap-due-date"
            required
            error={form.formState.errors.dueDate?.message}
          >
            <Input
              id="ap-due-date"
              type="date"
              error={Boolean(form.formState.errors.dueDate)}
              {...form.register('dueDate')}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-12">
          <div className="lg:col-span-6">
            <FormField
              label="Vendor"
              htmlFor="ap-vendor-id"
              required
              error={form.formState.errors.vendorId?.message}
              hint={
                vendorOptions.length === 0
                  ? 'Enter Vendor UUID (Masterdata lookup unavailable or empty)'
                  : undefined
              }
            >
              <div className="space-y-1.5">
                {!manualVendorUuid && vendorOptions.length > 0 ? (
                  <Controller
                    control={form.control}
                    name="vendorId"
                    render={({ field }) => (
                      <Combobox
                        id="ap-vendor-id"
                        value={field.value}
                        onValueChange={field.onChange}
                        options={vendorOptions}
                        placeholder="Select vendor..."
                        searchPlaceholder="Search vendor name, code, NPWP..."
                        error={Boolean(form.formState.errors.vendorId)}
                      />
                    )}
                  />
                ) : (
                  <Input
                    id="ap-vendor-id"
                    placeholder="00000000-0000-0000-0000-000000000000"
                    className="font-mono text-xs"
                    error={Boolean(form.formState.errors.vendorId)}
                    {...form.register('vendorId')}
                  />
                )}
                {vendorOptions.length > 0 && (
                  <button
                    type="button"
                    className="text-xs font-medium text-brand-700 hover:underline"
                    onClick={() => setManualVendorUuid((prev) => !prev)}
                  >
                    {manualVendorUuid
                      ? 'Use vendor directory selector'
                      : 'Enter Vendor UUID manually'}
                  </button>
                )}
              </div>
            </FormField>
          </div>

          <div className="lg:col-span-3">
            <FormField
              label="Currency"
              htmlFor="ap-currency-code"
              required
              error={form.formState.errors.currencyCode?.message}
            >
              <Controller
                control={form.control}
                name="currencyCode"
                render={({ field }) => (
                  <Select
                    id="ap-currency-code"
                    value={field.value}
                    onValueChange={field.onChange}
                    options={currencyOptions}
                    error={Boolean(form.formState.errors.currencyCode)}
                  />
                )}
              />
            </FormField>
          </div>

          <div className="lg:col-span-3">
            <FormField
              label="Goods Receipt Ref"
              htmlFor="ap-gr-ref"
              error={form.formState.errors.goodsReceiptRef?.message}
            >
              <Input
                id="ap-gr-ref"
                placeholder="GR-2026-014"
                error={Boolean(form.formState.errors.goodsReceiptRef)}
                {...form.register('goodsReceiptRef')}
              />
            </FormField>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-slate-50/70 p-4">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-semibold text-slate-900">Distribution Lines</h3>
              <p className="text-xs text-slate-500">
                Allocate bill amounts across posting GL accounts. Line sum must match Total Amount.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <span
                className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium tabular-nums ${
                  linesBalanced
                    ? 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200'
                    : 'bg-amber-50 text-amber-800 ring-1 ring-amber-200'
                }`}
              >
                Lines: {formatCurrency(computedLinesSum, watchedCurrency, locale === 'id' ? 'id-ID' : 'en-US')}
              </span>
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Sparkles className="h-3 w-3" />}
                onClick={handleMatchLineSum}
              >
                Match Line Sum
              </Button>
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() => append({ accountId: '', amount: '0', description: '' })}
              >
                Add Line
              </Button>
            </div>
          </div>

          <div className="space-y-3">
            {fields.map((fieldItem, idx) => {
              const useManualAcc = manualAccountLines[idx] ?? accountOptions.length === 0;
              const lineErrors = form.formState.errors.lines?.[idx];
              return (
                <div
                  key={fieldItem.id}
                  className="grid grid-cols-1 gap-3 rounded-lg border border-slate-200 bg-white p-3 lg:grid-cols-12"
                >
                  <div className="lg:col-span-5">
                    <FormField
                      label={`Line ${idx + 1} Account`}
                      required
                      error={lineErrors?.accountId?.message}
                    >
                      <div className="space-y-1">
                        {!useManualAcc ? (
                          <Controller
                            control={form.control}
                            name={`lines.${idx}.accountId`}
                            render={({ field }) => (
                              <Combobox
                                value={field.value}
                                onValueChange={field.onChange}
                                options={accountOptions}
                                placeholder="Choose GL account..."
                                searchPlaceholder="Search account code or name..."
                                error={Boolean(lineErrors?.accountId)}
                              />
                            )}
                          />
                        ) : (
                          <Input
                            placeholder="Account UUID"
                            className="font-mono text-xs"
                            error={Boolean(lineErrors?.accountId)}
                            {...form.register(`lines.${idx}.accountId`)}
                          />
                        )}
                        {accountOptions.length > 0 && (
                          <button
                            type="button"
                            className="text-[11px] font-medium text-brand-700 hover:underline"
                            onClick={() =>
                              setManualAccountLines((prev) => ({ ...prev, [idx]: !useManualAcc }))
                            }
                          >
                            {useManualAcc ? 'Select from CoA' : 'Enter Account UUID'}
                          </button>
                        )}
                      </div>
                    </FormField>
                  </div>

                  <div className="lg:col-span-3">
                    <FormField
                      label="Amount"
                      required
                      error={lineErrors?.amount?.message}
                    >
                      <Controller
                        control={form.control}
                        name={`lines.${idx}.amount`}
                        render={({ field }) => (
                          <MoneyInput
                            value={field.value}
                            onValueChange={field.onChange}
                            currency={watchedCurrency}
                            error={Boolean(lineErrors?.amount)}
                          />
                        )}
                      />
                    </FormField>
                  </div>

                  <div className="lg:col-span-3">
                    <FormField
                      label="Description"
                      required
                      error={lineErrors?.description?.message}
                    >
                      <Input
                        placeholder="Expense description"
                        error={Boolean(lineErrors?.description)}
                        {...form.register(`lines.${idx}.description`)}
                      />
                    </FormField>
                  </div>

                  <div className="flex items-end justify-end lg:col-span-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      aria-label={`Remove line ${idx + 1}`}
                      disabled={fields.length <= 1}
                      onClick={() => remove(idx)}
                    >
                      <Trash2 className="h-4 w-4 text-slate-400 hover:text-rose-600" />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>

          {form.formState.errors.lines?.root?.message && (
            <p className="mt-2 text-xs text-rose-600" role="alert">
              {form.formState.errors.lines.root.message}
            </p>
          )}
        </div>

        <div className="grid grid-cols-1 items-end gap-4 sm:grid-cols-2">
          <div className="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-xs text-slate-600">
            All totals use exact fixed-point string arithmetic (`decimal.ts`) matching Go backend
            4-decimal validation.
          </div>
          <FormField
            label="Total Invoice Amount"
            htmlFor="ap-total-amount"
            required
            error={form.formState.errors.totalAmount?.message}
          >
            <Controller
              control={form.control}
              name="totalAmount"
              render={({ field }) => (
                <MoneyInput
                  id="ap-total-amount"
                  value={field.value}
                  onValueChange={field.onChange}
                  currency={watchedCurrency}
                  error={Boolean(form.formState.errors.totalAmount)}
                />
              )}
            />
          </FormField>
        </div>
      </form>
    </AppDialog>
  );
}
