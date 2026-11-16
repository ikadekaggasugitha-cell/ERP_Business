import { useEffect, useMemo, useState } from 'react';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Sparkles, Trash2 } from 'lucide-react';
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
import { arQueryKeys, createCustomerInvoice } from './api';
import {
  arInvoiceFormSchema,
  computeArLinesTotal,
  type ArInvoiceFormValues,
} from './schemas';
import type { ArAccountLookupDto, ArCurrencyLookupDto, ArPartyLookupDto } from './types';

interface ArInvoiceFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customers: ArPartyLookupDto[];
  accounts: ArAccountLookupDto[];
  currencies: ArCurrencyLookupDto[];
}

function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

function addDaysIso(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() + days);
  return date.toISOString().slice(0, 10);
}

const DEFAULT_CURRENCIES = ['IDR', 'USD', 'EUR', 'SGD'];

export function ArInvoiceFormDialog({
  open,
  onOpenChange,
  customers,
  accounts,
  currencies,
}: ArInvoiceFormDialogProps) {
  const queryClient = useQueryClient();
  const locale = useUiStore((s) => s.locale);
  const [manualCustomerUuid, setManualCustomerUuid] = useState(false);
  const [manualAccountLines, setManualAccountLines] = useState<Record<number, boolean>>({});

  const customerOptions = useMemo(
    () =>
      customers
        .filter((p) => p.partyType === 'customer' || p.partyType === 'both' || !p.partyType)
        .map((p) => ({
          value: p.id,
          label: p.name,
          subtitle: `${p.code}${p.taxId ? ` · NPWP ${p.taxId}` : ''}`,
          keywords: [p.code, p.name, p.taxId ?? ''],
        })),
    [customers],
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

  const currencyOptions = useMemo(
    () =>
      currencies.length > 0
        ? currencies.map((c) => ({ value: c.code, label: `${c.code} — ${c.name}` }))
        : DEFAULT_CURRENCIES.map((code) => ({ value: code, label: code })),
    [currencies],
  );

  const form = useForm<ArInvoiceFormValues>({
    resolver: zodResolver(arInvoiceFormSchema),
    defaultValues: {
      invoiceNo: '',
      customerId: '',
      invoiceDate: todayIso(),
      dueDate: addDaysIso(30),
      currencyCode: 'IDR',
      totalAmount: '0',
      taxAmount: '0',
      salesOrderRef: '',
      lines: [{ accountId: '', amount: '0', description: '' }],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'lines',
  });

  useEffect(() => {
    if (!open) return;
    setManualCustomerUuid(customerOptions.length === 0);
    setManualAccountLines({});
    form.reset({
      invoiceNo: `INV-${new Date().getFullYear()}-${String(Date.now()).slice(-4)}`,
      customerId: '',
      invoiceDate: todayIso(),
      dueDate: addDaysIso(30),
      currencyCode: 'IDR',
      totalAmount: '0',
      taxAmount: '0',
      salesOrderRef: '',
      lines: [{ accountId: '', amount: '0', description: '' }],
    });
  }, [open, form, customerOptions.length]);

  const watchedLines = form.watch('lines');
  const watchedTotal = form.watch('totalAmount');
  const watchedCurrency = form.watch('currencyCode') || 'IDR';
  const computedLinesSum = useMemo(() => computeArLinesTotal(watchedLines || []), [watchedLines]);

  const linesBalanced = useMemo(() => {
    return new Decimal(watchedTotal || '0').equals(new Decimal(computedLinesSum || '0'));
  }, [watchedTotal, computedLinesSum]);

  const mutation = useMutation({
    mutationFn: (values: ArInvoiceFormValues) =>
      createCustomerInvoice({
        invoiceNo: values.invoiceNo.trim(),
        customerId: values.customerId.trim(),
        invoiceDate: values.invoiceDate,
        dueDate: values.dueDate,
        currencyCode: values.currencyCode,
        totalAmount: values.totalAmount.trim(),
        taxAmount: values.taxAmount?.trim() || undefined,
        salesOrderRef: values.salesOrderRef?.trim() || undefined,
        lines: values.lines.map((line, index) => ({
          lineNo: index + 1,
          accountId: line.accountId.trim(),
          amount: line.amount.trim(),
          description: line.description.trim(),
        })),
      }),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: arQueryKeys.all });
      toast.success(`Customer invoice ${created.invoiceNo} issued`);
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const message =
        err instanceof ApiClientError ? err.message : 'Unable to issue customer invoice';
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
      title="Issue Customer Invoice"
      description="Create an Accounts Receivable billing document and emit receivable posting."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="ar-invoice-form" isLoading={mutation.isPending}>
            Issue Customer Invoice
          </Button>
        </>
      }
    >
      <form
        id="ar-invoice-form"
        className="space-y-5"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <FormField
            label="Invoice Number"
            htmlFor="ar-invoice-no"
            required
            error={form.formState.errors.invoiceNo?.message}
          >
            <Input
              id="ar-invoice-no"
              placeholder="INV-2026-0001"
              error={Boolean(form.formState.errors.invoiceNo)}
              {...form.register('invoiceNo')}
            />
          </FormField>

          <FormField
            label="Invoice Date"
            htmlFor="ar-invoice-date"
            required
            error={form.formState.errors.invoiceDate?.message}
          >
            <Input
              id="ar-invoice-date"
              type="date"
              error={Boolean(form.formState.errors.invoiceDate)}
              {...form.register('invoiceDate')}
            />
          </FormField>

          <FormField
            label="Due Date"
            htmlFor="ar-due-date"
            required
            error={form.formState.errors.dueDate?.message}
          >
            <Input
              id="ar-due-date"
              type="date"
              error={Boolean(form.formState.errors.dueDate)}
              {...form.register('dueDate')}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-12">
          <div className="lg:col-span-6">
            <FormField
              label="Customer"
              htmlFor="ar-customer-id"
              required
              error={form.formState.errors.customerId?.message}
              hint={
                customerOptions.length === 0
                  ? 'Enter Customer UUID (Masterdata lookup unavailable or empty)'
                  : undefined
              }
            >
              <div className="space-y-1.5">
                {!manualCustomerUuid && customerOptions.length > 0 ? (
                  <Controller
                    control={form.control}
                    name="customerId"
                    render={({ field }) => (
                      <Combobox
                        id="ar-customer-id"
                        value={field.value}
                        onValueChange={field.onChange}
                        options={customerOptions}
                        placeholder="Select customer..."
                        searchPlaceholder="Search customer name, code, NPWP..."
                        error={Boolean(form.formState.errors.customerId)}
                      />
                    )}
                  />
                ) : (
                  <Input
                    id="ar-customer-id"
                    placeholder="00000000-0000-0000-0000-000000000000"
                    className="font-mono text-xs"
                    error={Boolean(form.formState.errors.customerId)}
                    {...form.register('customerId')}
                  />
                )}
                {customerOptions.length > 0 && (
                  <button
                    type="button"
                    className="text-xs font-medium text-brand-700 hover:underline"
                    onClick={() => setManualCustomerUuid((prev) => !prev)}
                  >
                    {manualCustomerUuid
                      ? 'Use customer directory selector'
                      : 'Enter Customer UUID manually'}
                  </button>
                )}
              </div>
            </FormField>
          </div>

          <div className="lg:col-span-3">
            <FormField
              label="Currency"
              htmlFor="ar-currency-code"
              required
              error={form.formState.errors.currencyCode?.message}
            >
              <Controller
                control={form.control}
                name="currencyCode"
                render={({ field }) => (
                  <Select
                    id="ar-currency-code"
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
              label="Sales Order Ref"
              htmlFor="ar-so-ref"
              error={form.formState.errors.salesOrderRef?.message}
            >
              <Input
                id="ar-so-ref"
                placeholder="SO-2026-088"
                error={Boolean(form.formState.errors.salesOrderRef)}
                {...form.register('salesOrderRef')}
              />
            </FormField>
          </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-slate-50/70 p-4">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-semibold text-slate-900">Revenue Lines</h3>
              <p className="text-xs text-slate-500">
                Revenue lines must sum to Invoice Total Amount.
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
                Lines:{' '}
                {formatCurrency(
                  computedLinesSum,
                  watchedCurrency,
                  locale === 'id' ? 'id-ID' : 'en-US',
                )}
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
                                placeholder="Choose revenue GL account..."
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
                    <FormField label="Amount" required error={lineErrors?.amount?.message}>
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
                    <FormField label="Description" required error={lineErrors?.description?.message}>
                      <Input
                        placeholder="Revenue item description"
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
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Included Tax Amount (Optional)"
            htmlFor="ar-tax-amount"
            error={form.formState.errors.taxAmount?.message}
          >
            <Controller
              control={form.control}
              name="taxAmount"
              render={({ field }) => (
                <MoneyInput
                  id="ar-tax-amount"
                  value={field.value ?? '0'}
                  onValueChange={field.onChange}
                  currency={watchedCurrency}
                  error={Boolean(form.formState.errors.taxAmount)}
                />
              )}
            />
          </FormField>

          <FormField
            label="Total Invoice Amount"
            htmlFor="ar-total-amount"
            required
            error={form.formState.errors.totalAmount?.message}
          >
            <Controller
              control={form.control}
              name="totalAmount"
              render={({ field }) => (
                <MoneyInput
                  id="ar-total-amount"
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
