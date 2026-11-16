import { useEffect, useMemo, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, CalendarDays, Lock, LockOpen, Plus, ShieldAlert } from 'lucide-react';
import { Controller, useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { Select, type SelectOption } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { ApiClientError, apiClient, applyApiErrorsToForm, unwrapListResponse } from '../../lib/api';
import { formatDate } from '../../lib/decimal';
import { useUiStore } from '../../store/ui-store';
import type { ListEnvelope } from '../../types';
import {
  fiscalCalendarSchema,
  fiscalPeriodSchema,
  fiscalYearSchema,
  type FiscalCalendarDto,
  type FiscalCalendarFormValues,
  type FiscalPeriodDto,
  type FiscalPeriodFormValues,
  type FiscalYearDto,
  type FiscalYearFormValues,
} from './coa-types';

export interface FiscalPeriodsTabProps {
  calendars: FiscalCalendarDto[];
  fiscalYears: FiscalYearDto[];
  isLoading: boolean;
  error: unknown;
  onRetry: () => void;
  canWrite: boolean;
}

export function FiscalPeriodsTab({
  calendars,
  fiscalYears,
  isLoading,
  error,
  onRetry,
  canWrite,
}: FiscalPeriodsTabProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const queryClient = useQueryClient();

  const [selectedYearId, setSelectedYearId] = useState<string>('ALL');
  const [calendarDialogOpen, setCalendarDialogOpen] = useState(false);
  const [yearDialogOpen, setYearDialogOpen] = useState(false);
  const [periodDialogOpen, setPeriodDialogOpen] = useState(false);
  const [confirmLockPeriod, setConfirmLockPeriod] = useState<FiscalPeriodDto | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    if (selectedYearId !== 'ALL' && fiscalYears.some((y) => y.id === selectedYearId)) return;
    if (fiscalYears.length > 0 && selectedYearId === 'ALL') {
      setSelectedYearId(fiscalYears[0].id);
    }
  }, [fiscalYears, selectedYearId]);

  const periodsQuery = useQuery({
    queryKey: ['fiscal-periods', 'list', selectedYearId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<FiscalPeriodDto> | FiscalPeriodDto[]>(
        '/fiscal-periods',
        {
          query:
            selectedYearId && selectedYearId !== 'ALL'
              ? new URLSearchParams({ fiscalYearId: selectedYearId, page: '1', size: '100' })
              : { page: 1, size: 100 },
        }
      );
      return unwrapListResponse(res).items;
    },
  });

  const periods = useMemo(() => periodsQuery.data ?? [], [periodsQuery.data]);

  const calendarById = useMemo(() => {
    const m = new Map<string, FiscalCalendarDto>();
    for (const c of calendars) m.set(c.id, c);
    return m;
  }, [calendars]);

  const yearById = useMemo(() => {
    const m = new Map<string, FiscalYearDto>();
    for (const y of fiscalYears) m.set(y.id, y);
    return m;
  }, [fiscalYears]);

  const yearFilterOptions: SelectOption[] = useMemo(
    () => [
      { value: 'ALL', label: isId ? 'Semua Tahun Fiskal' : 'All Fiscal Years' },
      ...fiscalYears.map((y) => ({
        value: y.id,
        label: `${y.yearNo} — ${y.name}`,
      })),
    ],
    [fiscalYears, isId]
  );

  const calendarForm = useForm<FiscalCalendarFormValues>({
    resolver: zodResolver(fiscalCalendarSchema),
    defaultValues: { code: '', name: '', periodType: 'Monthly' },
  });

  const yearForm = useForm<FiscalYearFormValues>({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: {
      fiscalCalendarId: '',
      yearNo: new Date().getFullYear(),
      name: `FY ${new Date().getFullYear()}`,
      startDate: `${new Date().getFullYear()}-01-01`,
      endDate: `${new Date().getFullYear()}-12-31`,
    },
  });

  const periodForm = useForm<FiscalPeriodFormValues>({
    resolver: zodResolver(fiscalPeriodSchema),
    defaultValues: {
      fiscalYearId: '',
      periodNo: 1,
      startDate: `${new Date().getFullYear()}-01-01`,
      endDate: `${new Date().getFullYear()}-01-31`,
    },
  });

  useEffect(() => {
    if (!calendarDialogOpen) return;
    setSubmitError(null);
    calendarForm.reset({ code: '', name: '', periodType: 'Monthly' });
  }, [calendarDialogOpen, calendarForm]);

  useEffect(() => {
    if (!yearDialogOpen) return;
    setSubmitError(null);
    const yr = new Date().getFullYear();
    yearForm.reset({
      fiscalCalendarId: calendars[0]?.id ?? '',
      yearNo: yr,
      name: `FY ${yr}`,
      startDate: `${yr}-01-01`,
      endDate: `${yr}-12-31`,
    });
  }, [yearDialogOpen, calendars, yearForm]);

  useEffect(() => {
    if (!periodDialogOpen) return;
    setSubmitError(null);
    const activeYear =
      (selectedYearId !== 'ALL' ? yearById.get(selectedYearId) : undefined) ?? fiscalYears[0];
    const nextPeriodNo = periods.length + 1;
    periodForm.reset({
      fiscalYearId: activeYear?.id ?? '',
      periodNo: nextPeriodNo,
      startDate: activeYear?.startDate ?? `${new Date().getFullYear()}-01-01`,
      endDate: activeYear?.endDate ?? `${new Date().getFullYear()}-01-31`,
    });
  }, [periodDialogOpen, selectedYearId, yearById, fiscalYears, periods.length, periodForm]);

  const invalidateFiscal = () => {
    void queryClient.invalidateQueries({ queryKey: ['fiscal-periods'] });
    void queryClient.invalidateQueries({ queryKey: ['coa-accounts'] });
  };

  const createCalendarMutation = useMutation({
    mutationFn: async (vals: FiscalCalendarFormValues) =>
      apiClient<FiscalCalendarDto>('/fiscal-calendars', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(vals),
      }),
    onSuccess: (created) => {
      invalidateFiscal();
      setCalendarDialogOpen(false);
      toast.success(
        isId ? `Kalender Fiskal ${created.code} dibuat.` : `Fiscal Calendar ${created.code} created.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, calendarForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to create fiscal calendar';
      setSubmitError(msg);
      toast.error(msg);
    },
  });

  const createYearMutation = useMutation({
    mutationFn: async (vals: FiscalYearFormValues) =>
      apiClient<FiscalYearDto>('/fiscal-years', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(vals),
      }),
    onSuccess: (created) => {
      invalidateFiscal();
      setYearDialogOpen(false);
      setSelectedYearId(created.id);
      toast.success(
        isId ? `Tahun Fiskal ${created.name} dibuat.` : `Fiscal Year ${created.name} created.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, yearForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to create fiscal year';
      setSubmitError(msg);
      toast.error(msg);
    },
  });

  const createPeriodMutation = useMutation({
    mutationFn: async (vals: FiscalPeriodFormValues) =>
      apiClient<FiscalPeriodDto>('/fiscal-periods', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(vals),
      }),
    onSuccess: (created) => {
      invalidateFiscal();
      setPeriodDialogOpen(false);
      toast.success(
        isId ? `Periode #${created.periodNo} dibuat.` : `Fiscal Period #${created.periodNo} created.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, periodForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to create fiscal period';
      setSubmitError(msg);
      toast.error(msg);
    },
  });

  const transitionMutation = useMutation({
    mutationFn: async ({
      period,
      action,
    }: {
      period: FiscalPeriodDto;
      action: 'close' | 'reopen' | 'lock';
    }) =>
      apiClient<FiscalPeriodDto>(`/fiscal-periods/${period.id}/${action}`, {
        method: 'POST',
        ifMatch: String(period.rowVersion),
      }),
    onSuccess: (updated, vars) => {
      invalidateFiscal();
      setConfirmLockPeriod(null);
      const label =
        vars.action === 'close'
          ? isId
            ? 'ditutup'
            : 'closed'
          : vars.action === 'reopen'
            ? isId
              ? 'dibuka kembali'
              : 'reopened'
            : isId
              ? 'dikunci permanen'
              : 'locked permanently';
      toast.success(
        isId
          ? `Periode #${updated.periodNo} berhasil ${label}.`
          : `Period #${updated.periodNo} ${label}.`
      );
    },
    onError: (err) => {
      const msg =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : 'Failed to update fiscal period status';
      toast.error(msg);
    },
  });

  const periodColumns: ColumnDef<FiscalPeriodDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Periode' : 'Period',
        width: '120px',
        cell: (row) => (
          <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
            #{row.periodNo}
          </span>
        ),
      },
      {
        header: isId ? 'Tahun Fiskal' : 'Fiscal Year',
        width: '180px',
        cell: (row) => {
          const yr = yearById.get(row.fiscalYearId);
          return (
            <span className="font-medium text-slate-800 dark:text-slate-200">
              {yr ? `${yr.yearNo} — ${yr.name}` : row.fiscalYearId.slice(0, 8)}
            </span>
          );
        },
      },
      {
        header: isId ? 'Tanggal Mulai' : 'Start Date',
        width: '140px',
        cell: (row) => <span className="font-mono text-xs">{formatDate(row.startDate, locale)}</span>,
      },
      {
        header: isId ? 'Tanggal Selesai' : 'End Date',
        width: '140px',
        cell: (row) => <span className="font-mono text-xs">{formatDate(row.endDate, locale)}</span>,
      },
      {
        header: 'Status',
        width: '130px',
        cell: (row) => {
          const norm = row.status.trim().toUpperCase();
          if (norm === 'CLOSED') {
            return <StatusBadge status="CLOSED" label={isId ? 'Ditutup' : 'Closed'} tone="warning" />;
          }
          if (norm === 'LOCKED') {
            return <StatusBadge status="LOCKED" label={isId ? 'Terkunci' : 'Locked'} tone="danger" />;
          }
          return <StatusBadge status="OPEN" label={isId ? 'Terbuka' : 'Open'} tone="info" />;
        },
      },
      {
        header: isId ? 'Kontrol Periode' : 'Period Actions',
        align: 'right',
        width: '220px',
        cell: (row) => {
          const norm = row.status.trim().toUpperCase();
          if (!canWrite || norm === 'LOCKED') {
            return (
              <span className="text-xs text-slate-400">
                {norm === 'LOCKED' ? (isId ? 'Final / Terkunci' : 'Permanently Locked') : '—'}
              </span>
            );
          }

          return (
            <div className="flex items-center justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
              {norm === 'OPEN' ? (
                <Button
                  variant="outline"
                  size="xs"
                  isLoading={transitionMutation.isPending}
                  onClick={() => transitionMutation.mutate({ period: row, action: 'close' })}
                >
                  {isId ? 'Tutup (Close)' : 'Close'}
                </Button>
              ) : null}

              {norm === 'CLOSED' ? (
                <>
                  <Button
                    variant="outline"
                    size="xs"
                    leftIcon={<LockOpen className="h-3 w-3" />}
                    isLoading={transitionMutation.isPending}
                    onClick={() => transitionMutation.mutate({ period: row, action: 'reopen' })}
                  >
                    {isId ? 'Buka Ulang' : 'Reopen'}
                  </Button>
                  <Button
                    variant="danger"
                    size="xs"
                    leftIcon={<Lock className="h-3 w-3" />}
                    disabled={transitionMutation.isPending}
                    onClick={() => setConfirmLockPeriod(row)}
                  >
                    {isId ? 'Kunci' : 'Lock'}
                  </Button>
                </>
              ) : null}
            </div>
          );
        },
      },
    ],
    [canWrite, isId, locale, transitionMutation, yearById]
  );

  return (
    <div className="space-y-6">
      {/* Calendars & Fiscal Years Overview Strip */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* Calendars Card */}
        <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center justify-between gap-2">
            <div>
              <h2 className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
                <CalendarDays className="h-4 w-4 text-brand-600 dark:text-brand-400" aria-hidden="true" />
                {isId ? 'Kalender Fiskal' : 'Fiscal Calendars'}
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {calendars.length} {isId ? 'kalender terdaftar' : 'registered calendars'}
              </p>
            </div>
            {canWrite ? (
              <Button
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() => setCalendarDialogOpen(true)}
              >
                {isId ? 'Kalender Baru' : 'New Calendar'}
              </Button>
            ) : null}
          </div>

          <ul className="mt-3 divide-y divide-slate-100 text-xs dark:divide-slate-800">
            {calendars.length === 0 ? (
              <li className="py-4 text-center text-slate-500 dark:text-slate-400">
                {isId ? 'Belum ada kalender fiskal.' : 'No fiscal calendars configured.'}
              </li>
            ) : (
              calendars.map((c) => (
                <li key={c.id} className="flex items-center justify-between py-2">
                  <div>
                    <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
                      {c.code}
                    </span>{' '}
                    <span className="text-slate-600 dark:text-slate-300">— {c.name}</span>
                  </div>
                  <StatusBadge status="ACTIVE" label={c.periodType} tone="neutral" dot={false} />
                </li>
              ))
            )}
          </ul>
        </section>

        {/* Fiscal Years Card */}
        <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center justify-between gap-2">
            <div>
              <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                {isId ? 'Tahun Fiskal' : 'Fiscal Years'}
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {fiscalYears.length} {isId ? 'tahun buku aktif' : 'configured fiscal years'}
              </p>
            </div>
            {canWrite ? (
              <Button
                variant="outline"
                size="xs"
                disabled={calendars.length === 0}
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() => setYearDialogOpen(true)}
              >
                {isId ? 'Tahun Fiskal Baru' : 'New Fiscal Year'}
              </Button>
            ) : null}
          </div>

          <ul className="mt-3 divide-y divide-slate-100 text-xs dark:divide-slate-800">
            {fiscalYears.length === 0 ? (
              <li className="py-4 text-center text-slate-500 dark:text-slate-400">
                {isId ? 'Belum ada tahun fiskal.' : 'No fiscal years configured.'}
              </li>
            ) : (
              fiscalYears.map((y) => (
                <li key={y.id} className="flex items-center justify-between gap-2 py-2">
                  <div>
                    <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
                      {y.yearNo}
                    </span>{' '}
                    <span className="text-slate-700 dark:text-slate-300">{y.name}</span>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400">
                      {calendarById.get(y.fiscalCalendarId)?.code ?? ''} · {y.startDate} → {y.endDate}
                    </p>
                  </div>
                  <StatusBadge status={y.status.toUpperCase()} />
                </li>
              ))
            )}
          </ul>
        </section>
      </div>

      {/* Permanent Lock Confirmation Banner */}
      {confirmLockPeriod ? (
        <div
          role="alert"
          className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs dark:border-rose-800/70 dark:bg-rose-950/50"
        >
          <div className="flex items-center gap-2 text-rose-800 dark:text-rose-200">
            <ShieldAlert className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="font-semibold">
              {isId
                ? `Kunci permanen Periode #${confirmLockPeriod.periodNo}? Periode yang terkunci tidak dapat dibuka ulang.`
                : `Permanently lock Period #${confirmLockPeriod.periodNo}? Locked periods cannot be reopened.`}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="danger"
              size="xs"
              isLoading={transitionMutation.isPending}
              onClick={() =>
                transitionMutation.mutate({ period: confirmLockPeriod, action: 'lock' })
              }
            >
              {isId ? 'Ya, Kunci Permanen' : 'Yes, Lock Period'}
            </Button>
            <Button variant="outline" size="xs" onClick={() => setConfirmLockPeriod(null)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
          </div>
        </div>
      ) : null}

      {/* Fiscal Periods Table */}
      <VirtualizedTable
        data={periods}
        columns={periodColumns}
        getRowId={(r) => r.id}
        isLoading={isLoading || periodsQuery.isLoading}
        error={error ?? periodsQuery.error}
        onRetry={() => {
          onRetry();
          void periodsQuery.refetch();
        }}
        emptyTitle={isId ? 'Belum Ada Periode Fiskal' : 'No Fiscal Periods Found'}
        emptyDescription={
          isId
            ? 'Tambahkan periode akuntansi bulanan atau kuartalan untuk tahun fiskal terpilih.'
            : 'Create monthly or quarterly accounting periods under your fiscal year.'
        }
        toolbar={{
          filters: (
            <div className="w-64">
              <Select
                aria-label={isId ? 'Filter Tahun Fiskal' : 'Filter Fiscal Year'}
                value={selectedYearId}
                onValueChange={setSelectedYearId}
                options={yearFilterOptions}
              />
            </div>
          ),
          actions: canWrite ? (
            <Button
              variant="primary"
              size="sm"
              disabled={fiscalYears.length === 0}
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => setPeriodDialogOpen(true)}
            >
              {isId ? 'Tambah Periode' : 'New Period'}
            </Button>
          ) : null,
        }}
      />

      {/* Dialog: Create Fiscal Calendar */}
      <AppDialog
        open={calendarDialogOpen}
        onOpenChange={setCalendarDialogOpen}
        size="sm"
        title={isId ? 'Buat Kalender Fiskal' : 'Create Fiscal Calendar'}
        footer={
          <>
            <Button variant="outline" onClick={() => setCalendarDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button
              type="submit"
              form="calendar-create-form"
              isLoading={createCalendarMutation.isPending}
            >
              {isId ? 'Simpan Kalender' : 'Save Calendar'}
            </Button>
          </>
        }
      >
        <form
          id="calendar-create-form"
          onSubmit={calendarForm.handleSubmit((vals) => {
            setSubmitError(null);
            createCalendarMutation.mutate(vals);
          })}
          className="space-y-3"
          noValidate
        >
          {submitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{submitError}</span>
            </div>
          ) : null}
          <FormField label={isId ? 'Kode' : 'Code'} required error={calendarForm.formState.errors.code?.message}>
            <Input placeholder="CAL-STD" {...calendarForm.register('code')} />
          </FormField>
          <FormField label={isId ? 'Nama' : 'Name'} required error={calendarForm.formState.errors.name?.message}>
            <Input placeholder="Kalender Standar Jan-Des" {...calendarForm.register('name')} />
          </FormField>
          <FormField
            label={isId ? 'Frekuensi Periode' : 'Period Type'}
            required
            error={calendarForm.formState.errors.periodType?.message}
          >
            <Controller
              control={calendarForm.control}
              name="periodType"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  options={[
                    { value: 'Monthly', label: isId ? 'Bulanan (Monthly)' : 'Monthly' },
                    { value: 'Quarterly', label: isId ? 'Kuartalan (Quarterly)' : 'Quarterly' },
                  ]}
                />
              )}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Dialog: Create Fiscal Year */}
      <AppDialog
        open={yearDialogOpen}
        onOpenChange={setYearDialogOpen}
        size="md"
        title={isId ? 'Buat Tahun Fiskal' : 'Create Fiscal Year'}
        footer={
          <>
            <Button variant="outline" onClick={() => setYearDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button type="submit" form="year-create-form" isLoading={createYearMutation.isPending}>
              {isId ? 'Simpan Tahun Fiskal' : 'Save Fiscal Year'}
            </Button>
          </>
        }
      >
        <form
          id="year-create-form"
          onSubmit={yearForm.handleSubmit((vals) => {
            setSubmitError(null);
            createYearMutation.mutate(vals);
          })}
          className="space-y-3"
          noValidate
        >
          {submitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{submitError}</span>
            </div>
          ) : null}
          <FormField
            label={isId ? 'Kalender Fiskal' : 'Fiscal Calendar'}
            required
            error={yearForm.formState.errors.fiscalCalendarId?.message}
          >
            <Controller
              control={yearForm.control}
              name="fiscalCalendarId"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  options={calendars.map((c) => ({ value: c.id, label: `${c.code} — ${c.name}` }))}
                />
              )}
            />
          </FormField>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <FormField label={isId ? 'Tahun' : 'Year No'} required error={yearForm.formState.errors.yearNo?.message}>
              <Input type="number" {...yearForm.register('yearNo')} />
            </FormField>
            <FormField label={isId ? 'Nama' : 'Name'} required error={yearForm.formState.errors.name?.message}>
              <Input {...yearForm.register('name')} />
            </FormField>
            <FormField label={isId ? 'Tanggal Mulai' : 'Start Date'} required error={yearForm.formState.errors.startDate?.message}>
              <Input type="date" {...yearForm.register('startDate')} />
            </FormField>
            <FormField label={isId ? 'Tanggal Selesai' : 'End Date'} required error={yearForm.formState.errors.endDate?.message}>
              <Input type="date" {...yearForm.register('endDate')} />
            </FormField>
          </div>
        </form>
      </AppDialog>

      {/* Dialog: Create Fiscal Period */}
      <AppDialog
        open={periodDialogOpen}
        onOpenChange={setPeriodDialogOpen}
        size="md"
        title={isId ? 'Buat Periode Fiskal Baru' : 'Create Fiscal Period'}
        footer={
          <>
            <Button variant="outline" onClick={() => setPeriodDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button type="submit" form="period-create-form" isLoading={createPeriodMutation.isPending}>
              {isId ? 'Simpan Periode' : 'Save Period'}
            </Button>
          </>
        }
      >
        <form
          id="period-create-form"
          onSubmit={periodForm.handleSubmit((vals) => {
            setSubmitError(null);
            createPeriodMutation.mutate(vals);
          })}
          className="space-y-3"
          noValidate
        >
          {submitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{submitError}</span>
            </div>
          ) : null}
          <FormField
            label={isId ? 'Tahun Fiskal' : 'Fiscal Year'}
            required
            error={periodForm.formState.errors.fiscalYearId?.message}
          >
            <Controller
              control={periodForm.control}
              name="fiscalYearId"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  options={fiscalYears.map((y) => ({ value: y.id, label: `${y.yearNo} — ${y.name}` }))}
                />
              )}
            />
          </FormField>
          <FormField label={isId ? 'Nomor Periode' : 'Period No'} required error={periodForm.formState.errors.periodNo?.message}>
            <Input type="number" min={1} max={24} {...periodForm.register('periodNo')} />
          </FormField>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <FormField label={isId ? 'Tanggal Mulai' : 'Start Date'} required error={periodForm.formState.errors.startDate?.message}>
              <Input type="date" {...periodForm.register('startDate')} />
            </FormField>
            <FormField label={isId ? 'Tanggal Selesai' : 'End Date'} required error={periodForm.formState.errors.endDate?.message}>
              <Input type="date" {...periodForm.register('endDate')} />
            </FormField>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
