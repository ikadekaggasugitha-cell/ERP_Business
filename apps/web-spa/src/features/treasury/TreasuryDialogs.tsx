import { useEffect, useMemo, useState } from 'react';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2 } from 'lucide-react';
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
import {
  createBankAccount,
  createBatchStatementLines,
  createPayment,
  createSingleStatementLine,
  treasuryQueryKeys,
  updatePayment,
} from './api';
import {
  bankAccountFormSchema,
  paymentFormSchema,
  statementLinesFormSchema,
  type BankAccountFormValues,
  type PaymentFormValues,
  type StatementLinesFormValues,
} from './schemas';
import type {
  BankAccountDto,
  PaymentDto,
  TreasuryAccountLookupDto,
  TreasuryCurrencyLookupDto,
  TreasuryPartyLookupDto,
} from './types';

function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

const FALLBACK_CURRENCIES = ['IDR', 'USD', 'EUR', 'SGD'];

interface CreateBankAccountDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  accounts: TreasuryAccountLookupDto[];
  currencies: TreasuryCurrencyLookupDto[];
}

export function CreateBankAccountDialog({
  open,
  onOpenChange,
  accounts,
  currencies,
}: CreateBankAccountDialogProps) {
  const queryClient = useQueryClient();
  const [manualGlUuid, setManualGlUuid] = useState(false);

  const glOptions = useMemo(
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
        : FALLBACK_CURRENCIES.map((code) => ({ value: code, label: code })),
    [currencies],
  );

  const form = useForm<BankAccountFormValues>({
    resolver: zodResolver(bankAccountFormSchema),
    defaultValues: {
      code: '',
      bankName: '',
      accountNo: '',
      currencyCode: 'IDR',
      glAccountId: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    setManualGlUuid(glOptions.length === 0);
    form.reset({
      code: '',
      bankName: '',
      accountNo: '',
      currencyCode: 'IDR',
      glAccountId: '',
    });
  }, [open, form, glOptions.length]);

  const mutation = useMutation({
    mutationFn: (values: BankAccountFormValues) => createBankAccount(values),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.bankAccountsAll });
      toast.success(`Bank account ${created.code} (${created.bankName}) registered`);
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to register bank account';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="lg"
      title="Register Bank Account"
      description="Link a corporate cash/bank treasury account to its controlling General Ledger account."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="create-bank-account-form" isLoading={mutation.isPending}>
            Save Bank Account
          </Button>
        </>
      }
    >
      <form
        id="create-bank-account-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Bank Code"
            htmlFor="bank-code"
            required
            error={form.formState.errors.code?.message}
          >
            <Input
              id="bank-code"
              placeholder="BCA-IDR-OPS"
              className="uppercase font-mono"
              error={Boolean(form.formState.errors.code)}
              {...form.register('code')}
            />
          </FormField>

          <FormField
            label="Currency"
            htmlFor="bank-currency"
            required
            error={form.formState.errors.currencyCode?.message}
          >
            <Controller
              control={form.control}
              name="currencyCode"
              render={({ field }) => (
                <Select
                  id="bank-currency"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={currencyOptions}
                  error={Boolean(form.formState.errors.currencyCode)}
                />
              )}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Bank Name"
            htmlFor="bank-name"
            required
            error={form.formState.errors.bankName?.message}
          >
            <Input
              id="bank-name"
              placeholder="Bank Central Asia"
              error={Boolean(form.formState.errors.bankName)}
              {...form.register('bankName')}
            />
          </FormField>

          <FormField
            label="Account Number"
            htmlFor="bank-account-no"
            required
            error={form.formState.errors.accountNo?.message}
          >
            <Input
              id="bank-account-no"
              placeholder="5210987654"
              className="font-mono"
              error={Boolean(form.formState.errors.accountNo)}
              {...form.register('accountNo')}
            />
          </FormField>
        </div>

        <FormField
          label="Controlling GL Account"
          htmlFor="bank-gl-account"
          required
          error={form.formState.errors.glAccountId?.message}
          hint={
            glOptions.length === 0
              ? 'Enter valid CoA Cash/Bank GL Account UUID'
              : undefined
          }
        >
          <div className="space-y-1.5">
            {!manualGlUuid && glOptions.length > 0 ? (
              <Controller
                control={form.control}
                name="glAccountId"
                render={({ field }) => (
                  <Combobox
                    id="bank-gl-account"
                    value={field.value}
                    onValueChange={field.onChange}
                    options={glOptions}
                    placeholder="Select GL cash/bank account..."
                    searchPlaceholder="Search GL code or name..."
                    error={Boolean(form.formState.errors.glAccountId)}
                  />
                )}
              />
            ) : (
              <Input
                id="bank-gl-account"
                placeholder="00000000-0000-0000-0000-000000000000"
                className="font-mono text-xs"
                error={Boolean(form.formState.errors.glAccountId)}
                {...form.register('glAccountId')}
              />
            )}
            {glOptions.length > 0 && (
              <button
                type="button"
                className="text-xs font-medium text-brand-700 hover:underline"
                onClick={() => setManualGlUuid((prev) => !prev)}
              >
                {manualGlUuid ? 'Select from CoA' : 'Enter GL Account UUID manually'}
              </button>
            )}
          </div>
        </FormField>
      </form>
    </AppDialog>
  );
}

interface PaymentFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingPayment: PaymentDto | null;
  bankAccounts: BankAccountDto[];
  parties: TreasuryPartyLookupDto[];
}

export function PaymentFormDialog({
  open,
  onOpenChange,
  editingPayment,
  bankAccounts,
  parties,
}: PaymentFormDialogProps) {
  const queryClient = useQueryClient();
  const isEdit = Boolean(editingPayment);
  const [manualPartyUuid, setManualPartyUuid] = useState(false);

  const bankOptions = useMemo(
    () =>
      bankAccounts.map((b) => ({
        value: b.id,
        label: `${b.code} — ${b.bankName} (${b.currencyCode})`,
        subtitle: b.accountNo,
        keywords: [b.code, b.bankName, b.accountNo, b.currencyCode],
      })),
    [bankAccounts],
  );

  const partyOptions = useMemo(
    () =>
      parties.map((p) => ({
        value: p.id,
        label: p.name,
        subtitle: `${p.code} · ${p.partyType}`,
        keywords: [p.code, p.name, p.partyType],
      })),
    [parties],
  );

  const form = useForm<PaymentFormValues>({
    resolver: zodResolver(paymentFormSchema),
    defaultValues: {
      paymentNo: '',
      paymentType: 'Outgoing',
      bankAccountId: '',
      partyId: '',
      amount: '0',
      paymentDate: todayIso(),
      externalRef: '',
      allocations: [],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'allocations',
  });

  useEffect(() => {
    if (!open) return;
    setManualPartyUuid(partyOptions.length === 0);

    if (editingPayment) {
      const normalizedType =
        editingPayment.paymentType === 'Incoming' ? 'Incoming' : 'Outgoing';
      form.reset({
        paymentNo: editingPayment.paymentNo,
        paymentType: normalizedType,
        bankAccountId: editingPayment.bankAccountId,
        partyId: editingPayment.partyId ?? '',
        amount: editingPayment.amount,
        paymentDate: editingPayment.paymentDate.slice(0, 10),
        externalRef: editingPayment.externalRef ?? '',
        allocations:
          editingPayment.allocations?.map((alloc) => ({
            targetDocumentRef: alloc.targetDocumentRef,
            allocatedAmount: alloc.allocatedAmount,
          })) ?? [],
      });
    } else {
      form.reset({
        paymentNo: `PAY-${new Date().getFullYear()}-${String(Date.now()).slice(-4)}`,
        paymentType: 'Outgoing',
        bankAccountId: bankAccounts[0]?.id ?? '',
        partyId: '',
        amount: '0',
        paymentDate: todayIso(),
        externalRef: '',
        allocations: [],
      });
    }
  }, [open, editingPayment, bankAccounts, partyOptions.length, form]);

  const selectedBankId = form.watch('bankAccountId');
  const activeCurrency =
    bankAccounts.find((b) => b.id === selectedBankId)?.currencyCode || 'IDR';

  const mutation = useMutation({
    mutationFn: async (values: PaymentFormValues) => {
      if (editingPayment) {
        return updatePayment(editingPayment.id, editingPayment.rowVersion, {
          paymentType: values.paymentType,
          bankAccountId: values.bankAccountId,
          partyId: values.partyId?.trim() || null,
          amount: values.amount.trim(),
          paymentDate: values.paymentDate,
          externalRef: values.externalRef?.trim() || null,
          allocations: values.allocations,
        });
      }
      return createPayment({
        paymentNo: values.paymentNo.trim(),
        paymentType: values.paymentType,
        bankAccountId: values.bankAccountId,
        partyId: values.partyId?.trim() || null,
        amount: values.amount.trim(),
        paymentDate: values.paymentDate,
        externalRef: values.externalRef?.trim() || null,
        allocations: values.allocations,
      });
    },
    onSuccess: async (saved) => {
      await queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.paymentsAll });
      toast.success(
        isEdit
          ? `Payment ${saved.paymentNo} updated`
          : `Draft payment ${saved.paymentNo} created`,
      );
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to save payment';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="xl"
      title={isEdit ? `Edit Draft Payment · ${editingPayment?.paymentNo}` : 'New Cash / Bank Payment'}
      description="Record an outgoing vendor disbursement or incoming customer receipt."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="treasury-payment-form" isLoading={mutation.isPending}>
            {isEdit ? 'Save Payment Changes' : 'Create Draft Payment'}
          </Button>
        </>
      }
    >
      <form
        id="treasury-payment-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <FormField
            label="Payment No"
            htmlFor="pay-no"
            required
            error={form.formState.errors.paymentNo?.message}
          >
            <Input
              id="pay-no"
              disabled={isEdit}
              placeholder="PAY-2026-0001"
              className="font-mono"
              error={Boolean(form.formState.errors.paymentNo)}
              {...form.register('paymentNo')}
            />
          </FormField>

          <FormField
            label="Direction"
            htmlFor="pay-type"
            required
            error={form.formState.errors.paymentType?.message}
          >
            <Controller
              control={form.control}
              name="paymentType"
              render={({ field }) => (
                <Select
                  id="pay-type"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={[
                    { value: 'Outgoing', label: 'Outgoing (AP / Expense)' },
                    { value: 'Incoming', label: 'Incoming (AR / Receipt)' },
                  ]}
                  error={Boolean(form.formState.errors.paymentType)}
                />
              )}
            />
          </FormField>

          <FormField
            label="Payment Date"
            htmlFor="pay-date"
            required
            error={form.formState.errors.paymentDate?.message}
          >
            <Input
              id="pay-date"
              type="date"
              error={Boolean(form.formState.errors.paymentDate)}
              {...form.register('paymentDate')}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Bank Account"
            htmlFor="pay-bank-id"
            required
            error={form.formState.errors.bankAccountId?.message}
          >
            {bankOptions.length > 0 ? (
              <Controller
                control={form.control}
                name="bankAccountId"
                render={({ field }) => (
                  <Combobox
                    id="pay-bank-id"
                    value={field.value}
                    onValueChange={field.onChange}
                    options={bankOptions}
                    placeholder="Select bank account..."
                    searchPlaceholder="Search bank code or name..."
                    error={Boolean(form.formState.errors.bankAccountId)}
                  />
                )}
              />
            ) : (
              <Input
                id="pay-bank-id"
                placeholder="Bank Account UUID"
                className="font-mono text-xs"
                error={Boolean(form.formState.errors.bankAccountId)}
                {...form.register('bankAccountId')}
              />
            )}
          </FormField>

          <FormField
            label="Counterparty (Optional)"
            htmlFor="pay-party-id"
            error={form.formState.errors.partyId?.message}
          >
            <div className="space-y-1.5">
              {!manualPartyUuid && partyOptions.length > 0 ? (
                <Controller
                  control={form.control}
                  name="partyId"
                  render={({ field }) => (
                    <Combobox
                      id="pay-party-id"
                      value={field.value ?? ''}
                      onValueChange={field.onChange}
                      options={partyOptions}
                      allowClear
                      placeholder="Optional vendor or customer..."
                      searchPlaceholder="Search counterparty..."
                      error={Boolean(form.formState.errors.partyId)}
                    />
                  )}
                />
              ) : (
                <Input
                  id="pay-party-id"
                  placeholder="Optional Party UUID"
                  className="font-mono text-xs"
                  error={Boolean(form.formState.errors.partyId)}
                  {...form.register('partyId')}
                />
              )}
              {partyOptions.length > 0 && (
                <button
                  type="button"
                  className="text-xs font-medium text-brand-700 hover:underline"
                  onClick={() => setManualPartyUuid((prev) => !prev)}
                >
                  {manualPartyUuid ? 'Use party selector' : 'Enter Party UUID manually'}
                </button>
              )}
            </div>
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Payment Amount"
            htmlFor="pay-amount"
            required
            error={form.formState.errors.amount?.message}
          >
            <Controller
              control={form.control}
              name="amount"
              render={({ field }) => (
                <MoneyInput
                  id="pay-amount"
                  value={field.value}
                  onValueChange={field.onChange}
                  currency={activeCurrency}
                  error={Boolean(form.formState.errors.amount)}
                />
              )}
            />
          </FormField>

          <FormField
            label="External Reference / Wire ID"
            htmlFor="pay-ext-ref"
            error={form.formState.errors.externalRef?.message}
          >
            <Input
              id="pay-ext-ref"
              placeholder="TRF-BI-FAST-8821"
              error={Boolean(form.formState.errors.externalRef)}
              {...form.register('externalRef')}
            />
          </FormField>
        </div>

        <div className="rounded-xl border border-slate-200 bg-slate-50/70 p-4">
          <div className="mb-3 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-semibold text-slate-900">
                Document Allocations (Optional)
              </h3>
              <p className="text-xs text-slate-500">
                Link bill/invoice document numbers to settle alongside this payment.
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="xs"
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => append({ targetDocumentRef: '', allocatedAmount: '0' })}
            >
              Add Allocation
            </Button>
          </div>

          {fields.length > 0 ? (
            <div className="space-y-2.5">
              {fields.map((item, idx) => {
                const allocErr = form.formState.errors.allocations?.[idx];
                return (
                  <div
                    key={item.id}
                    className="grid grid-cols-1 gap-3 rounded-lg border border-slate-200 bg-white p-3 sm:grid-cols-12"
                  >
                    <div className="sm:col-span-6">
                      <FormField
                        label="Target Document Ref"
                        required
                        error={allocErr?.targetDocumentRef?.message}
                      >
                        <Input
                          placeholder="BILL-2026-0001 / INV-2026-0004"
                          error={Boolean(allocErr?.targetDocumentRef)}
                          {...form.register(`allocations.${idx}.targetDocumentRef`)}
                        />
                      </FormField>
                    </div>
                    <div className="sm:col-span-5">
                      <FormField
                        label="Allocated Amount"
                        required
                        error={allocErr?.allocatedAmount?.message}
                      >
                        <Controller
                          control={form.control}
                          name={`allocations.${idx}.allocatedAmount`}
                          render={({ field }) => (
                            <MoneyInput
                              value={field.value}
                              onValueChange={field.onChange}
                              currency={activeCurrency}
                              error={Boolean(allocErr?.allocatedAmount)}
                            />
                          )}
                        />
                      </FormField>
                    </div>
                    <div className="flex items-end justify-end sm:col-span-1">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove allocation ${idx + 1}`}
                        onClick={() => remove(idx)}
                      >
                        <Trash2 className="h-4 w-4 text-slate-400 hover:text-rose-600" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-xs text-slate-500">
              No document allocations added yet. You may allocate later or leave unallocated.
            </p>
          )}
        </div>
      </form>
    </AppDialog>
  );
}

interface StatementLinesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  bankAccounts: BankAccountDto[];
}

export function StatementLinesDialog({
  open,
  onOpenChange,
  bankAccounts,
}: StatementLinesDialogProps) {
  const queryClient = useQueryClient();
  const [batchMode, setBatchMode] = useState(false);

  const bankOptions = useMemo(
    () =>
      bankAccounts.map((b) => ({
        value: b.id,
        label: `${b.code} — ${b.bankName} (${b.currencyCode})`,
        subtitle: b.accountNo,
      })),
    [bankAccounts],
  );

  const form = useForm<StatementLinesFormValues>({
    resolver: zodResolver(statementLinesFormSchema),
    defaultValues: {
      bankAccountId: '',
      lines: [{ statementDate: todayIso(), description: '', amount: '0', externalRef: '' }],
    },
  });

  const { fields, append, remove, replace } = useFieldArray({
    control: form.control,
    name: 'lines',
  });

  useEffect(() => {
    if (!open) return;
    setBatchMode(false);
    form.reset({
      bankAccountId: bankAccounts[0]?.id ?? '',
      lines: [{ statementDate: todayIso(), description: '', amount: '0', externalRef: '' }],
    });
  }, [open, bankAccounts, form]);

  const handleToggleBatchMode = (nextBatch: boolean) => {
    setBatchMode(nextBatch);
    if (!nextBatch && fields.length > 1) {
      const first = form.getValues('lines.0');
      replace([first]);
    } else if (nextBatch && fields.length === 1) {
      append({ statementDate: todayIso(), description: '', amount: '0', externalRef: '' });
    }
  };

  const selectedBankId = form.watch('bankAccountId');
  const activeCurrency =
    bankAccounts.find((b) => b.id === selectedBankId)?.currencyCode || 'IDR';

  const mutation = useMutation({
    mutationFn: async (values: StatementLinesFormValues) => {
      if (!batchMode && values.lines.length === 1) {
        const line = values.lines[0];
        await createSingleStatementLine({
          bankAccountId: values.bankAccountId,
          statementDate: line.statementDate,
          description: line.description,
          amount: line.amount,
          externalRef: line.externalRef,
        });
        return { importedCount: 1 };
      }
      return createBatchStatementLines({
        bankAccountId: values.bankAccountId,
        lines: values.lines,
      });
    },
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.statementLinesAll });
      toast.success(
        result.importedCount === 1
          ? 'Bank statement line recorded'
          : `${result.importedCount} bank statement lines imported`,
      );
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to save statement lines';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="2xl"
      title={batchMode ? 'Batch Import Bank Statement Lines' : 'New Bank Statement Line'}
      description="Positive amounts represent inflows/credits; negative amounts represent outflows/debits."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="statement-lines-form" isLoading={mutation.isPending}>
            {batchMode ? `Import ${fields.length} Statement Lines` : 'Save Statement Line'}
          </Button>
        </>
      }
    >
      <form
        id="statement-lines-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="min-w-[260px] flex-1">
            <FormField
              label="Bank Account"
              htmlFor="stmt-bank-id"
              required
              error={form.formState.errors.bankAccountId?.message}
            >
              {bankOptions.length > 0 ? (
                <Controller
                  control={form.control}
                  name="bankAccountId"
                  render={({ field }) => (
                    <Combobox
                      id="stmt-bank-id"
                      value={field.value}
                      onValueChange={field.onChange}
                      options={bankOptions}
                      placeholder="Select bank account..."
                      error={Boolean(form.formState.errors.bankAccountId)}
                    />
                  )}
                />
              ) : (
                <Input
                  id="stmt-bank-id"
                  placeholder="Bank Account UUID"
                  className="font-mono text-xs"
                  error={Boolean(form.formState.errors.bankAccountId)}
                  {...form.register('bankAccountId')}
                />
              )}
            </FormField>
          </div>

          <div className="pb-2">
            <Switch
              checked={batchMode}
              onCheckedChange={handleToggleBatchMode}
              label="Batch multi-line mode"
            />
          </div>
        </div>

        <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-900">
              Statement Entries ({fields.length})
            </h3>
            {batchMode && (
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() =>
                  append({ statementDate: todayIso(), description: '', amount: '0', externalRef: '' })
                }
              >
                Add Row
              </Button>
            )}
          </div>

          {fields.map((item, idx) => {
            const lineErr = form.formState.errors.lines?.[idx];
            return (
              <div
                key={item.id}
                className="grid grid-cols-1 gap-3 rounded-lg border border-slate-200 bg-white p-3 lg:grid-cols-12"
              >
                <div className="lg:col-span-3">
                  <FormField label="Statement Date" required error={lineErr?.statementDate?.message}>
                    <Input
                      type="date"
                      error={Boolean(lineErr?.statementDate)}
                      {...form.register(`lines.${idx}.statementDate`)}
                    />
                  </FormField>
                </div>

                <div className="lg:col-span-4">
                  <FormField label="Description" required error={lineErr?.description?.message}>
                    <Input
                      placeholder="Bank transfer / BI-FAST narration"
                      error={Boolean(lineErr?.description)}
                      {...form.register(`lines.${idx}.description`)}
                    />
                  </FormField>
                </div>

                <div className="lg:col-span-2">
                  <FormField label="External Ref" error={lineErr?.externalRef?.message}>
                    <Input
                      placeholder="TRF-001"
                      className="font-mono text-xs"
                      error={Boolean(lineErr?.externalRef)}
                      {...form.register(`lines.${idx}.externalRef`)}
                    />
                  </FormField>
                </div>

                <div className={batchMode ? 'lg:col-span-2' : 'lg:col-span-3'}>
                  <FormField
                    label="Signed Amount"
                    required
                    error={lineErr?.amount?.message}
                  >
                    <Controller
                      control={form.control}
                      name={`lines.${idx}.amount`}
                      render={({ field }) => (
                        <MoneyInput
                          value={field.value}
                          onValueChange={field.onChange}
                          currency={activeCurrency}
                          allowNegative
                          error={Boolean(lineErr?.amount)}
                        />
                      )}
                    />
                  </FormField>
                </div>

                {batchMode && (
                  <div className="flex items-end justify-end lg:col-span-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      aria-label={`Remove statement line ${idx + 1}`}
                      disabled={fields.length <= 1}
                      onClick={() => remove(idx)}
                    >
                      <Trash2 className="h-4 w-4 text-slate-400 hover:text-rose-600" />
                    </Button>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </form>
    </AppDialog>
  );
}
