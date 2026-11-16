import { useEffect, useMemo, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { AlertCircle, CheckCircle2, Plus, Scale, Trash2 } from 'lucide-react';
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox, type ComboboxOption } from '../../components/ui/Combobox';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { MoneyInput } from '../../components/ui/MoneyInput';
import { Select, type SelectOption } from '../../components/ui/Select';
import { ApiClientError, apiClient, applyApiErrorsToForm, unwrapListResponse } from '../../lib/api';
import { formatCurrency } from '../../lib/decimal';
import { useUiStore } from '../../store/ui-store';
import type { ListEnvelope } from '../../types';
import {
  computeFunctionalAmount,
  computeJournalTotals,
  generateDefaultJournalNo,
  journalFormSchema,
  type AccountLookupDto,
  type CurrencyLookupDto,
  type FiscalPeriodLookupDto,
  type JournalEntryDto,
  type JournalFormValues,
  type LedgerLookupDto,
} from './journals-types';

export interface JournalFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingEntry?: JournalEntryDto | null;
  onSaved?: (entry: JournalEntryDto) => void;
}

function getTodayIsoDate(): string {
  return new Date().toISOString().slice(0, 10);
}

function createEmptyLines(): JournalFormValues['lines'] {
  return [
    { accountId: '', debitAmount: '0.00', creditAmount: '0.00', description: '' },
    { accountId: '', debitAmount: '0.00', creditAmount: '0.00', description: '' },
  ];
}

export function JournalFormDialog({
  open,
  onOpenChange,
  editingEntry = null,
  onSaved,
}: JournalFormDialogProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const queryClient = useQueryClient();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const isEditing = Boolean(editingEntry);

  const ledgersQuery = useQuery({
    queryKey: ['ledgers', 'lookup'],
    enabled: open,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<LedgerLookupDto> | LedgerLookupDto[]>('/ledgers', {
        query: { page: 1, size: 100 },
      });
      return unwrapListResponse(res).items;
    },
  });

  const periodsQuery = useQuery({
    queryKey: ['fiscal-periods', 'lookup'],
    enabled: open,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<FiscalPeriodLookupDto> | FiscalPeriodLookupDto[]>(
        '/fiscal-periods',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const currenciesQuery = useQuery({
    queryKey: ['currencies', 'lookup'],
    enabled: open,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<CurrencyLookupDto> | CurrencyLookupDto[]>(
        '/currencies',
        { query: new URLSearchParams({ isActive: 'true', page: '1', size: '100' }) }
      );
      return unwrapListResponse(res).items;
    },
  });

  const accountsQuery = useQuery({
    queryKey: ['coa-accounts', 'postable'],
    enabled: open,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<AccountLookupDto> | AccountLookupDto[]>(
        '/accounts',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const ledgers = useMemo(() => ledgersQuery.data ?? [], [ledgersQuery.data]);
  const periods = useMemo(() => periodsQuery.data ?? [], [periodsQuery.data]);
  const currencies = useMemo(() => currenciesQuery.data ?? [], [currenciesQuery.data]);
  const accounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);

  const postableAccounts = useMemo(
    () => accounts.filter((acc) => acc.allowPosting),
    [accounts]
  );

  const ledgerOptions: ComboboxOption[] = useMemo(
    () =>
      ledgers.map((l) => ({
        value: l.id,
        label: `${l.code} (${l.currencyCode})`,
        subtitle: l.ledgerType,
        keywords: [l.code, l.ledgerType, l.currencyCode],
      })),
    [ledgers]
  );

  const periodOptions: ComboboxOption[] = useMemo(
    () =>
      periods.map((p) => ({
        value: p.id,
        label: `${isId ? 'Periode' : 'Period'} #${p.periodNo} (${p.startDate} → ${p.endDate})`,
        subtitle: p.status,
        keywords: [String(p.periodNo), p.startDate, p.endDate, p.status],
      })),
    [periods, isId]
  );

  const currencyOptions: SelectOption[] = useMemo(() => {
    if (currencies.length === 0) {
      return [{ value: editingEntry?.currencyCode || 'IDR', label: editingEntry?.currencyCode || 'IDR' }];
    }
    return currencies.map((c) => ({
      value: c.code,
      label: `${c.code} — ${c.name}`,
    }));
  }, [currencies, editingEntry?.currencyCode]);

  const accountOptions: ComboboxOption[] = useMemo(
    () =>
      postableAccounts.map((acc) => ({
        value: acc.id,
        label: `${acc.code} — ${acc.name}`,
        subtitle: acc.accountType,
        keywords: [acc.code, acc.name, acc.accountType],
      })),
    [postableAccounts]
  );

  const form = useForm<JournalFormValues>({
    resolver: zodResolver(journalFormSchema),
    mode: 'onChange',
    defaultValues: {
      journalNo: generateDefaultJournalNo(),
      financialOrganizationId: '',
      ledgerId: '',
      fiscalPeriodId: '',
      postingDate: getTodayIsoDate(),
      journalSource: 'Manual',
      currencyCode: 'IDR',
      exchangeRate: '1.0000',
      sourceDocumentRef: '',
      lines: createEmptyLines(),
    },
  });

  const { control, handleSubmit, register, reset, setValue, setError, formState } = form;
  const { fields, append, remove } = useFieldArray({ control, name: 'lines' });

  const watchedLines = useWatch({ control, name: 'lines' });
  const watchedCurrency = useWatch({ control, name: 'currencyCode' }) || 'IDR';
  const watchedExchangeRate = useWatch({ control, name: 'exchangeRate' }) || '1.0000';

  const activeCurrencyPrecision = useMemo(() => {
    const matched = currencies.find((c) => c.code === watchedCurrency);
    return matched ? Math.min(Math.max(matched.precision, 0), 4) : 2;
  }, [currencies, watchedCurrency]);

  const totals = useMemo(() => computeJournalTotals(watchedLines ?? []), [watchedLines]);

  useEffect(() => {
    if (!open) return;
    setSubmitError(null);

    if (editingEntry) {
      const mappedLines =
        editingEntry.lines && editingEntry.lines.length >= 2
          ? editingEntry.lines.map((l) => ({
              accountId: l.accountId,
              debitAmount: l.debitAmount || '0.00',
              creditAmount: l.creditAmount || '0.00',
              description: l.description ?? '',
            }))
          : createEmptyLines();

      reset({
        journalNo: editingEntry.journalNo,
        financialOrganizationId: editingEntry.financialOrganizationId,
        ledgerId: editingEntry.ledgerId,
        fiscalPeriodId: editingEntry.fiscalPeriodId,
        postingDate: editingEntry.postingDate.slice(0, 10),
        journalSource: editingEntry.journalSource || 'Manual',
        currencyCode: editingEntry.currencyCode || 'IDR',
        exchangeRate: editingEntry.exchangeRate || '1.0000',
        sourceDocumentRef: editingEntry.sourceDocumentRef ?? '',
        lines: mappedLines,
      });
    } else {
      const firstLedger = ledgers[0];
      const preferredPeriod =
        periods.find((p) => p.status.trim().toUpperCase() === 'OPEN') ?? periods[0];

      reset({
        journalNo: generateDefaultJournalNo(),
        financialOrganizationId: firstLedger?.financialOrganizationId ?? '',
        ledgerId: firstLedger?.id ?? '',
        fiscalPeriodId: preferredPeriod?.id ?? '',
        postingDate: getTodayIsoDate(),
        journalSource: 'Manual',
        currencyCode: firstLedger?.currencyCode ?? currencies[0]?.code ?? 'IDR',
        exchangeRate: '1.0000',
        sourceDocumentRef: '',
        lines: createEmptyLines(),
      });
    }
  }, [open, editingEntry, reset, ledgers, periods, currencies]);

  // Populate initial ledger/period asynchronously once lookups finish when creating
  useEffect(() => {
    if (!open || editingEntry) return;
    const currentLedger = form.getValues('ledgerId');
    if (!currentLedger && ledgers.length > 0) {
      const first = ledgers[0];
      setValue('ledgerId', first.id, { shouldValidate: true });
      setValue('financialOrganizationId', first.financialOrganizationId, { shouldValidate: true });
      setValue('currencyCode', first.currencyCode || 'IDR', { shouldValidate: true });
    }
    const currentPeriod = form.getValues('fiscalPeriodId');
    if (!currentPeriod && periods.length > 0) {
      const preferred =
        periods.find((p) => p.status.trim().toUpperCase() === 'OPEN') ?? periods[0];
      setValue('fiscalPeriodId', preferred.id, { shouldValidate: true });
    }
  }, [open, editingEntry, ledgers, periods, form, setValue]);

  const handleLedgerSelect = (nextLedgerId: string) => {
    setValue('ledgerId', nextLedgerId, { shouldValidate: true });
    const matched = ledgers.find((l) => l.id === nextLedgerId);
    if (matched) {
      setValue('financialOrganizationId', matched.financialOrganizationId, {
        shouldValidate: true,
      });
      if (!isEditing && matched.currencyCode) {
        setValue('currencyCode', matched.currencyCode, { shouldValidate: true });
      }
    }
  };

  const mutation = useMutation({
    mutationFn: async (values: JournalFormValues) => {
      const formattedLines = values.lines.map((line, index) => ({
        lineNo: index + 1,
        accountId: line.accountId,
        debitAmount: line.debitAmount,
        creditAmount: line.creditAmount,
        functionalAmount: computeFunctionalAmount(
          line.debitAmount,
          line.creditAmount,
          values.exchangeRate
        ),
        description: line.description?.trim() || undefined,
      }));

      if (editingEntry) {
        return apiClient<JournalEntryDto>(`/journal-entries/${editingEntry.id}`, {
          method: 'PUT',
          ifMatch: String(editingEntry.rowVersion),
          body: JSON.stringify({
            currencyCode: values.currencyCode,
            exchangeRate: values.exchangeRate,
            sourceDocumentRef: values.sourceDocumentRef?.trim() || undefined,
            lines: formattedLines,
          }),
        });
      }

      return apiClient<JournalEntryDto>('/journal-entries', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          journalNo: values.journalNo.trim(),
          financialOrganizationId: values.financialOrganizationId,
          ledgerId: values.ledgerId,
          fiscalPeriodId: values.fiscalPeriodId,
          postingDate: values.postingDate,
          journalSource: values.journalSource || 'Manual',
          currencyCode: values.currencyCode,
          exchangeRate: values.exchangeRate,
          sourceDocumentRef: values.sourceDocumentRef?.trim() || undefined,
          lines: formattedLines,
        }),
      });
    },
    onSuccess: (saved) => {
      void queryClient.invalidateQueries({ queryKey: ['journals'] });
      void queryClient.invalidateQueries({ queryKey: ['balances'] });
      void queryClient.invalidateQueries({ queryKey: ['gl-balances'] });
      void queryClient.invalidateQueries({ queryKey: ['dashboard-metrics'] });
      toast.success(
        isEditing
          ? isId
            ? `Draf jurnal ${saved.journalNo} diperbarui.`
            : `Journal draft ${saved.journalNo} updated.`
          : isId
            ? `Draf jurnal ${saved.journalNo} berhasil dibuat.`
            : `Journal draft ${saved.journalNo} created.`
      );
      onSaved?.(saved);
      onOpenChange(false);
    },
    onError: (err) => {
      const applied = applyApiErrorsToForm(err, setError);
      const msg =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : isId
              ? 'Gagal menyimpan entri jurnal.'
              : 'Failed to save journal entry.';
      setSubmitError(
        applied > 0
          ? `${msg} (${isId ? 'Periksa kolom yang ditandai' : 'Check highlighted fields'})`
          : msg
      );
      toast.error(msg);
    },
  });

  const missingLookups =
    !ledgersQuery.isLoading &&
    !periodsQuery.isLoading &&
    (ledgers.length === 0 || periods.length === 0);

  const linesRootError =
    typeof formState.errors.lines?.message === 'string'
      ? formState.errors.lines.message
      : typeof formState.errors.lines?.root?.message === 'string'
        ? formState.errors.lines.root.message
        : undefined;

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="2xl"
      title={
        isEditing
          ? `${isId ? 'Ubah Draf Jurnal' : 'Edit Journal Draft'} — ${editingEntry?.journalNo}`
          : isId
            ? 'Buat Entri Jurnal Double-Entry'
            : 'Create Double-Entry Journal'
      }
      description={
        isId
          ? 'Pilih Ledger dan Periode Fiskal aktif dari CoA Service. Semua nominal dihitung secara presisi desimal tetap.'
          : 'Select live Ledger and Fiscal Period from CoA Service. All amounts use fixed-point decimal arithmetic.'
      }
      footer={
        <div className="flex w-full flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-xs">
            <Scale
              className={`h-4 w-4 ${totals.canSubmit ? 'text-emerald-600' : 'text-rose-600'}`}
              aria-hidden="true"
            />
            <span className="text-slate-600 dark:text-slate-300">
              {isId ? 'Debit:' : 'Debit:'}{' '}
              <strong className="font-money text-slate-900 dark:text-slate-100">
                {formatCurrency(totals.totalDebit, watchedCurrency, locale)}
              </strong>
            </span>
            <span className="text-slate-400">·</span>
            <span className="text-slate-600 dark:text-slate-300">
              {isId ? 'Kredit:' : 'Credit:'}{' '}
              <strong className="font-money text-slate-900 dark:text-slate-100">
                {formatCurrency(totals.totalCredit, watchedCurrency, locale)}
              </strong>
            </span>
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-semibold ${
                totals.canSubmit
                  ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300'
                  : 'bg-rose-50 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300'
              }`}
            >
              {totals.canSubmit ? (
                <>
                  <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
                  {isId ? 'Seimbang' : 'Balanced'}
                </>
              ) : (
                <>
                  <AlertCircle className="h-3.5 w-3.5" aria-hidden="true" />
                  {isId ? `Selisih ${totals.difference}` : `Diff ${totals.difference}`}
                </>
              )}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={mutation.isPending}
            >
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button
              type="submit"
              form="journal-entry-form"
              variant="primary"
              isLoading={mutation.isPending}
              disabled={!totals.canSubmit || missingLookups}
            >
              {isEditing
                ? isId
                  ? 'Simpan Perubahan'
                  : 'Save Draft Changes'
                : isId
                  ? 'Simpan sebagai Draf'
                  : 'Save as Draft'}
            </Button>
          </div>
        </div>
      }
    >
      <form
        id="journal-entry-form"
        onSubmit={handleSubmit((vals) => {
          setSubmitError(null);
          mutation.mutate(vals);
        })}
        className="space-y-4"
        noValidate
      >
        {missingLookups ? (
          <div
            role="alert"
            className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-800/70 dark:bg-amber-950/50 dark:text-amber-300"
          >
            <span>
              {isId
                ? 'Ledger atau Periode Fiskal belum tersedia di backend. Buat terlebih dahulu sebelum mencatat jurnal.'
                : 'Ledgers or Fiscal Periods are not configured yet. Create them before recording journals.'}
            </span>
            <Link
              to={'/coa' as never}
              className="font-semibold underline hover:text-amber-950 dark:hover:text-amber-100"
            >
              {isId ? 'Buka Bagan Akun & Periode →' : 'Go to CoA & Fiscal Periods →'}
            </Link>
          </div>
        ) : null}

        {submitError ? (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{submitError}</span>
          </div>
        ) : null}

        {/* Header Fields */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <FormField
            label={isId ? 'Nomor Jurnal' : 'Journal No'}
            htmlFor="journal-no"
            required
            error={formState.errors.journalNo?.message}
          >
            <Input
              id="journal-no"
              disabled={isEditing}
              error={Boolean(formState.errors.journalNo)}
              {...register('journalNo')}
            />
          </FormField>

          <FormField
            label={isId ? 'Buku Besar (Ledger)' : 'Ledger'}
            htmlFor="journal-ledger"
            required
            error={formState.errors.ledgerId?.message || formState.errors.financialOrganizationId?.message}
          >
            <Controller
              control={control}
              name="ledgerId"
              render={({ field }) => (
                <Combobox
                  id="journal-ledger"
                  value={field.value}
                  onValueChange={handleLedgerSelect}
                  options={ledgerOptions}
                  disabled={isEditing}
                  error={Boolean(formState.errors.ledgerId)}
                  placeholder={isId ? 'Pilih Ledger...' : 'Select Ledger...'}
                />
              )}
            />
          </FormField>

          <FormField
            label={isId ? 'Periode Fiskal' : 'Fiscal Period'}
            htmlFor="journal-period"
            required
            error={formState.errors.fiscalPeriodId?.message}
          >
            <Controller
              control={control}
              name="fiscalPeriodId"
              render={({ field }) => (
                <Combobox
                  id="journal-period"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={periodOptions}
                  disabled={isEditing}
                  error={Boolean(formState.errors.fiscalPeriodId)}
                  placeholder={isId ? 'Pilih Periode...' : 'Select Period...'}
                />
              )}
            />
          </FormField>

          <FormField
            label={isId ? 'Tanggal Posting' : 'Posting Date'}
            htmlFor="journal-posting-date"
            required
            error={formState.errors.postingDate?.message}
          >
            <Input
              id="journal-posting-date"
              type="date"
              disabled={isEditing}
              error={Boolean(formState.errors.postingDate)}
              {...register('postingDate')}
            />
          </FormField>

          <FormField
            label={isId ? 'Mata Uang' : 'Currency'}
            htmlFor="journal-currency"
            required
            error={formState.errors.currencyCode?.message}
          >
            <Controller
              control={control}
              name="currencyCode"
              render={({ field }) => (
                <Select
                  id="journal-currency"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={currencyOptions}
                  error={Boolean(formState.errors.currencyCode)}
                />
              )}
            />
          </FormField>

          <FormField
            label={isId ? 'Kurs Valas' : 'Exchange Rate'}
            htmlFor="journal-exchange-rate"
            required
            error={formState.errors.exchangeRate?.message}
          >
            <Input
              id="journal-exchange-rate"
              numeric
              error={Boolean(formState.errors.exchangeRate)}
              {...register('exchangeRate')}
            />
          </FormField>
        </div>

        <FormField
          label={isId ? 'Referensi Dokumen Sumber / Keterangan' : 'Source Document Ref / Memo'}
          htmlFor="journal-source-ref"
          error={formState.errors.sourceDocumentRef?.message}
        >
          <Input
            id="journal-source-ref"
            placeholder={isId ? 'contoh: INV-2026-001 atau Penyesuaian Akrual' : 'e.g. INV-2026-001 or Monthly Accrual'}
            {...register('sourceDocumentRef')}
          />
        </FormField>

        {/* Dynamic Journal Lines Grid */}
        <section className="space-y-3 rounded-xl border border-border bg-surface-alt/70 p-3.5 dark:bg-slate-800/40">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <h3 className="text-xs font-bold text-slate-900 dark:text-slate-100">
                {isId ? 'Baris Entri Jurnal' : 'Journal Entry Lines'}
              </h3>
              <span className="inline-flex items-center rounded-full border border-border bg-surface px-2 py-0.5 text-[10.5px] font-semibold text-slate-600 dark:text-slate-300">
                {fields.length} {isId ? 'Baris' : 'Lines'}
              </span>
              {accountsQuery.isSuccess && postableAccounts.length === 0 ? (
                <p className="text-[11px] text-amber-600 dark:text-amber-400">
                  {isId
                    ? 'Tidak ada akun dengan allowPosting=true ditemukan di Bagan Akun.'
                    : 'No postable accounts (allowPosting=true) found in Chart of Accounts.'}
                </p>
              ) : null}
            </div>
            <Button
              type="button"
              variant="outline"
              size="xs"
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() =>
                append({ accountId: '', debitAmount: '0.00', creditAmount: '0.00', description: '' })
              }
            >
              {isId ? 'Tambah Baris' : 'Add Line'}
            </Button>
          </div>

          {linesRootError ? (
            <p role="alert" className="text-xs font-medium text-rose-600 dark:text-rose-400">
              {linesRootError}
            </p>
          ) : null}

          <div className="space-y-2">
            {fields.map((fieldItem, index) => {
              const lineErrors = formState.errors.lines?.[index];
              const lineVal = watchedLines?.[index];
              const lineFunctional = computeFunctionalAmount(
                lineVal?.debitAmount || '0',
                lineVal?.creditAmount || '0',
                watchedExchangeRate
              );
              return (
                <div
                  key={fieldItem.id}
                  className="grid grid-cols-1 gap-2.5 rounded-xl border border-border bg-surface p-3 shadow-2xs sm:grid-cols-12 sm:items-start"
                >
                  <div className="sm:col-span-4">
                    <FormField
                      label={index === 0 ? (isId ? 'Akun GL' : 'GL Account') : undefined}
                      htmlFor={`journal-line-account-${index}`}
                      error={lineErrors?.accountId?.message}
                    >
                      <Controller
                        control={control}
                        name={`lines.${index}.accountId`}
                        render={({ field }) => (
                          <Combobox
                            id={`journal-line-account-${index}`}
                            value={field.value}
                            onValueChange={field.onChange}
                            options={accountOptions}
                            error={Boolean(lineErrors?.accountId)}
                            placeholder={isId ? `Akun Baris #${index + 1}` : `Account #${index + 1}`}
                          />
                        )}
                      />
                    </FormField>
                  </div>

                  <div className="sm:col-span-3">
                    <FormField
                      label={index === 0 ? (isId ? 'Debit' : 'Debit') : undefined}
                      error={lineErrors?.debitAmount?.message}
                    >
                      <Controller
                        control={control}
                        name={`lines.${index}.debitAmount`}
                        render={({ field }) => (
                          <MoneyInput
                            aria-label={`${isId ? 'Debit Baris' : 'Line Debit'} ${index + 1}`}
                            currency={watchedCurrency}
                            precision={activeCurrencyPrecision}
                            value={field.value}
                            onValueChange={field.onChange}
                            error={Boolean(lineErrors?.debitAmount)}
                          />
                        )}
                      />
                    </FormField>
                  </div>

                  <div className="sm:col-span-3">
                    <FormField
                      label={index === 0 ? (isId ? 'Kredit' : 'Credit') : undefined}
                      error={lineErrors?.creditAmount?.message}
                    >
                      <Controller
                        control={control}
                        name={`lines.${index}.creditAmount`}
                        render={({ field }) => (
                          <MoneyInput
                            aria-label={`${isId ? 'Kredit Baris' : 'Line Credit'} ${index + 1}`}
                            currency={watchedCurrency}
                            precision={activeCurrencyPrecision}
                            value={field.value}
                            onValueChange={field.onChange}
                            error={Boolean(lineErrors?.creditAmount)}
                          />
                        )}
                      />
                    </FormField>
                  </div>

                  <div className="flex items-end gap-1.5 sm:col-span-2">
                    <div className="flex-1 min-w-0">
                      <FormField label={index === 0 ? (isId ? 'Catatan' : 'Memo') : undefined}>
                        <Input
                          aria-label={`${isId ? 'Catatan Baris' : 'Line Memo'} ${index + 1}`}
                          placeholder={isId ? 'Opsional' : 'Optional'}
                          {...register(`lines.${index}.description`)}
                        />
                      </FormField>
                      <p className="mt-1 text-right font-mono text-[10px] text-slate-400 dark:text-slate-500 truncate">
                        ≈ {formatCurrency(lineFunctional, 'IDR', locale)}
                      </p>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      disabled={fields.length <= 2}
                      aria-label={`${isId ? 'Hapus baris' : 'Remove line'} ${index + 1}`}
                      onClick={() => remove(index)}
                      className="text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50"
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        {/* Full-width Fixed-Point Validation Banner */}
        <div
          role="status"
          className={`flex flex-wrap items-center justify-between gap-2 rounded-xl border px-3.5 py-2.5 text-xs ${
            totals.canSubmit
              ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800/70 dark:bg-emerald-950/40 dark:text-emerald-200'
              : totals.isPositive
                ? 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-800/70 dark:bg-rose-950/40 dark:text-rose-200'
                : 'border-border bg-surface-alt text-slate-600 dark:text-slate-300'
          }`}
        >
          <div className="flex items-center gap-2">
            {totals.canSubmit ? (
              <CheckCircle2 className="h-4 w-4 text-emerald-600 dark:text-emerald-400 shrink-0" aria-hidden="true" />
            ) : (
              <AlertCircle
                className={`h-4 w-4 shrink-0 ${totals.isPositive ? 'text-rose-600 dark:text-rose-400' : 'text-slate-400'}`}
                aria-hidden="true"
              />
            )}
            <span className="font-medium">
              {totals.canSubmit
                ? isId
                  ? 'Validasi keseimbangan lolos: Total Debit = Total Kredit (selisih 0,00).'
                  : 'Balance check passed: Total Debit = Total Credit (difference 0.00).'
                : totals.isPositive
                  ? isId
                    ? `Entri belum seimbang (Selisih ${formatCurrency(totals.difference, watchedCurrency, locale)}).`
                    : `Entry out of balance (Diff ${formatCurrency(totals.difference, watchedCurrency, locale)}).`
                  : isId
                    ? 'Lengkapi nominal debit dan kredit yang seimbang pada minimal 2 baris.'
                    : 'Enter balanced debit and credit amounts across at least 2 lines.'}
            </span>
          </div>
          <span className="font-mono text-[11px] font-semibold">
            {totals.canSubmit ? 'Fixed-Point Verified' : 'ADR-014 Decimal'}
          </span>
        </div>
      </form>
    </AppDialog>
  );
}
