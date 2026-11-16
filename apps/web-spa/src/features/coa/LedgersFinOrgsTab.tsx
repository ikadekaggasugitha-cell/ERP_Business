import { useEffect, useMemo, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, Building2, Layers, Plus } from 'lucide-react';
import { Controller, useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox, type ComboboxOption } from '../../components/ui/Combobox';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { Select, type SelectOption } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { apiClient, applyApiErrorsToForm } from '../../lib/api';
import { useUiStore } from '../../store/ui-store';
import {
  financialOrgSchema,
  ledgerFormSchema,
  type ChartOfAccountsDto,
  type CoaCurrencyLookupDto,
  type FinancialOrganizationDto,
  type FinancialOrgFormValues,
  type FiscalCalendarDto,
  type LedgerDto,
  type LedgerFormValues,
  type OrgUnitLookupDto,
} from './coa-types';

export interface LedgersFinOrgsTabProps {
  financialOrgs: FinancialOrganizationDto[];
  ledgers: LedgerDto[];
  charts: ChartOfAccountsDto[];
  calendars: FiscalCalendarDto[];
  currencies: CoaCurrencyLookupDto[];
  orgUnits: OrgUnitLookupDto[];
  isLoading: boolean;
  error: unknown;
  onRetry: () => void;
  canWrite: boolean;
}

export function LedgersFinOrgsTab({
  financialOrgs,
  ledgers,
  charts,
  calendars,
  currencies,
  orgUnits,
  isLoading,
  error,
  onRetry,
  canWrite,
}: LedgersFinOrgsTabProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const queryClient = useQueryClient();

  const [finOrgDialogOpen, setFinOrgDialogOpen] = useState(false);
  const [ledgerDialogOpen, setLedgerDialogOpen] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const finOrgById = useMemo(() => {
    const m = new Map<string, FinancialOrganizationDto>();
    for (const f of financialOrgs) m.set(f.id, f);
    return m;
  }, [financialOrgs]);

  const chartById = useMemo(() => {
    const m = new Map<string, ChartOfAccountsDto>();
    for (const c of charts) m.set(c.id, c);
    return m;
  }, [charts]);

  const calendarById = useMemo(() => {
    const m = new Map<string, FiscalCalendarDto>();
    for (const c of calendars) m.set(c.id, c);
    return m;
  }, [calendars]);

  const orgUnitOptions: ComboboxOption[] = useMemo(
    () =>
      orgUnits.map((o) => ({
        value: o.id,
        label: `${o.code} — ${o.name}`,
        keywords: [o.code, o.name],
      })),
    [orgUnits]
  );

  const currencyOptions: SelectOption[] = useMemo(
    () =>
      currencies.length > 0
        ? currencies.map((c) => ({ value: c.code, label: `${c.code} — ${c.name}` }))
        : [{ value: 'IDR', label: 'IDR — Indonesian Rupiah' }],
    [currencies]
  );

  const finOrgForm = useForm<FinancialOrgFormValues>({
    resolver: zodResolver(financialOrgSchema),
    defaultValues: {
      code: '',
      name: '',
      organizationRefId: '',
      functionalCurrencyCode: 'IDR',
      chartOfAccountsId: '',
      fiscalCalendarId: '',
    },
  });

  const ledgerForm = useForm<LedgerFormValues>({
    resolver: zodResolver(ledgerFormSchema),
    defaultValues: {
      financialOrganizationId: '',
      code: '',
      ledgerType: 'Primary',
      currencyCode: 'IDR',
    },
  });

  useEffect(() => {
    if (!finOrgDialogOpen) return;
    setSubmitError(null);
    finOrgForm.reset({
      code: '',
      name: '',
      organizationRefId: orgUnits[0]?.id ?? '',
      functionalCurrencyCode: currencies[0]?.code ?? 'IDR',
      chartOfAccountsId: charts[0]?.id ?? '',
      fiscalCalendarId: calendars[0]?.id ?? '',
    });
  }, [finOrgDialogOpen, orgUnits, currencies, charts, calendars, finOrgForm]);

  useEffect(() => {
    if (!ledgerDialogOpen) return;
    setSubmitError(null);
    const defaultFinOrg = financialOrgs[0];
    ledgerForm.reset({
      financialOrganizationId: defaultFinOrg?.id ?? '',
      code: '',
      ledgerType: 'Primary',
      currencyCode: defaultFinOrg?.functionalCurrencyCode ?? currencies[0]?.code ?? 'IDR',
    });
  }, [ledgerDialogOpen, financialOrgs, currencies, ledgerForm]);

  const invalidateMaster = () => {
    void queryClient.invalidateQueries({ queryKey: ['coa-accounts'] });
    void queryClient.invalidateQueries({ queryKey: ['ledgers'] });
    void queryClient.invalidateQueries({ queryKey: ['fiscal-periods'] });
  };

  const createFinOrgMutation = useMutation({
    mutationFn: async (vals: FinancialOrgFormValues) =>
      apiClient<FinancialOrganizationDto>('/financial-organizations', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(vals),
      }),
    onSuccess: (created) => {
      invalidateMaster();
      setFinOrgDialogOpen(false);
      toast.success(
        isId
          ? `Entitas Keuangan ${created.code} berhasil dibuat.`
          : `Financial Org ${created.code} created.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, finOrgForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to create financial organization';
      setSubmitError(msg);
      toast.error(msg);
    },
  });

  const createLedgerMutation = useMutation({
    mutationFn: async (vals: LedgerFormValues) =>
      apiClient<LedgerDto>('/ledgers', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify(vals),
      }),
    onSuccess: (created) => {
      invalidateMaster();
      setLedgerDialogOpen(false);
      toast.success(
        isId ? `Ledger ${created.code} berhasil dibuat.` : `Ledger ${created.code} created.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, ledgerForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to create ledger';
      setSubmitError(msg);
      toast.error(msg);
    },
  });

  const finOrgColumns: ColumnDef<FinancialOrganizationDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Kode' : 'Code',
        width: '120px',
        cell: (r) => (
          <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
            {r.code}
          </span>
        ),
      },
      {
        header: isId ? 'Nama Entitas Keuangan' : 'Financial Org Name',
        width: '230px',
        cell: (r) => <span className="font-medium text-slate-800 dark:text-slate-200">{r.name}</span>,
      },
      {
        header: isId ? 'Mata Uang' : 'Functional CCY',
        width: '120px',
        cell: (r) => <span className="font-mono font-semibold">{r.functionalCurrencyCode}</span>,
      },
      {
        header: 'Chart of Accounts',
        width: '180px',
        cell: (r) => {
          const coa = chartById.get(r.chartOfAccountsId);
          return (
            <span className="text-xs text-slate-600 dark:text-slate-300">
              {coa ? `${coa.code} — ${coa.name}` : r.chartOfAccountsId.slice(0, 8)}
            </span>
          );
        },
      },
      {
        header: isId ? 'Kalender Fiskal' : 'Fiscal Calendar',
        width: '180px',
        cell: (r) => {
          const cal = calendarById.get(r.fiscalCalendarId);
          return (
            <span className="text-xs text-slate-600 dark:text-slate-300">
              {cal ? `${cal.code} (${cal.periodType})` : r.fiscalCalendarId.slice(0, 8)}
            </span>
          );
        },
      },
    ],
    [isId, chartById, calendarById]
  );

  const ledgerColumns: ColumnDef<LedgerDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Kode Ledger' : 'Ledger Code',
        width: '150px',
        cell: (r) => (
          <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
            {r.code}
          </span>
        ),
      },
      {
        header: isId ? 'Tipe Ledger' : 'Ledger Type',
        width: '150px',
        cell: (r) => <StatusBadge status="ACTIVE" label={r.ledgerType} tone="info" />,
      },
      {
        header: isId ? 'Entitas Keuangan' : 'Financial Org',
        width: '240px',
        cell: (r) => {
          const fin = finOrgById.get(r.financialOrganizationId);
          return (
            <span className="text-slate-800 dark:text-slate-200">
              {fin ? `${fin.code} — ${fin.name}` : r.financialOrganizationId.slice(0, 8)}
            </span>
          );
        },
      },
      {
        header: isId ? 'Mata Uang' : 'Currency',
        width: '110px',
        cell: (r) => <span className="font-mono font-semibold">{r.currencyCode}</span>,
      },
    ],
    [isId, finOrgById]
  );

  return (
    <div className="space-y-6">
      {/* Financial Organizations Section */}
      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-slate-100">
              <Building2 className="h-4 w-4 text-brand-600 dark:text-brand-400" aria-hidden="true" />
              {isId ? 'Entitas Keuangan (Financial Organizations)' : 'Financial Organizations'}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {isId
                ? 'Menghubungkan unit organisasi dengan mata uang fungsional, Bagan Akun, dan Kalender Fiskal.'
                : 'Links organization units with functional currency, Chart of Accounts, and Fiscal Calendar.'}
            </p>
          </div>
          {canWrite ? (
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => setFinOrgDialogOpen(true)}
            >
              {isId ? 'Tambah Entitas Keuangan' : 'New Financial Org'}
            </Button>
          ) : null}
        </div>

        <VirtualizedTable
          data={financialOrgs}
          columns={finOrgColumns}
          getRowId={(r) => r.id}
          tableHeight={260}
          isLoading={isLoading}
          error={error}
          onRetry={onRetry}
          emptyTitle={isId ? 'Belum Ada Entitas Keuangan' : 'No Financial Organizations'}
          emptyDescription={
            isId
              ? 'Buat entitas keuangan untuk mulai menyiapkan buku besar (Ledger).'
              : 'Create a financial organization before configuring accounting ledgers.'
          }
        />
      </section>

      {/* Ledgers Section */}
      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-slate-100">
              <Layers className="h-4 w-4 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
              {isId ? 'Buku Besar Akuntansi (Ledgers)' : 'Accounting Ledgers'}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {isId
                ? 'Ledger Primer, Sekunder, dan Konsolidasi yang dipilih pada saat pencatatan Jurnal GL.'
                : 'Primary, Secondary, and Consolidation ledgers used when recording GL journals.'}
            </p>
          </div>
          {canWrite ? (
            <Button
              variant="primary"
              size="sm"
              disabled={financialOrgs.length === 0}
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => setLedgerDialogOpen(true)}
            >
              {isId ? 'Tambah Ledger' : 'New Ledger'}
            </Button>
          ) : null}
        </div>

        <VirtualizedTable
          data={ledgers}
          columns={ledgerColumns}
          getRowId={(r) => r.id}
          tableHeight={280}
          isLoading={isLoading}
          error={error}
          onRetry={onRetry}
          emptyTitle={isId ? 'Belum Ada Buku Besar' : 'No Ledgers Found'}
          emptyDescription={
            isId
              ? 'Tambahkan minimal satu Ledger Primer untuk entitas keuangan Anda.'
              : 'Create at least one Primary ledger for your financial organization.'
          }
        />
      </section>

      {/* Create Financial Organization Dialog */}
      <AppDialog
        open={finOrgDialogOpen}
        onOpenChange={setFinOrgDialogOpen}
        size="lg"
        title={isId ? 'Buat Entitas Keuangan Baru' : 'Create Financial Organization'}
        footer={
          <>
            <Button variant="outline" onClick={() => setFinOrgDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button
              type="submit"
              form="finorg-create-form"
              isLoading={createFinOrgMutation.isPending}
            >
              {isId ? 'Simpan Entitas' : 'Save Financial Org'}
            </Button>
          </>
        }
      >
        <form
          id="finorg-create-form"
          onSubmit={finOrgForm.handleSubmit((vals) => {
            setSubmitError(null);
            createFinOrgMutation.mutate(vals);
          })}
          className="space-y-3"
          noValidate
        >
          {submitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{submitError}</span>
            </div>
          ) : null}

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <FormField
              label={isId ? 'Kode Entitas' : 'Code'}
              required
              error={finOrgForm.formState.errors.code?.message}
            >
              <Input placeholder="FIN-HLD-01" {...finOrgForm.register('code')} />
            </FormField>
            <FormField
              label={isId ? 'Nama Entitas Keuangan' : 'Name'}
              required
              error={finOrgForm.formState.errors.name?.message}
            >
              <Input placeholder="PT Sinergi Holding Accounting" {...finOrgForm.register('name')} />
            </FormField>
            <FormField
              label={isId ? 'Unit Organisasi' : 'Organization Unit'}
              required
              error={finOrgForm.formState.errors.organizationRefId?.message}
            >
              <Controller
                control={finOrgForm.control}
                name="organizationRefId"
                render={({ field }) => (
                  <Combobox
                    value={field.value}
                    onValueChange={field.onChange}
                    options={orgUnitOptions}
                    placeholder={isId ? 'Pilih unit organisasi...' : 'Select org unit...'}
                  />
                )}
              />
            </FormField>
            <FormField
              label={isId ? 'Mata Uang Fungsional' : 'Functional Currency'}
              required
              error={finOrgForm.formState.errors.functionalCurrencyCode?.message}
            >
              <Controller
                control={finOrgForm.control}
                name="functionalCurrencyCode"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={currencyOptions}
                  />
                )}
              />
            </FormField>
            <FormField
              label="Chart of Accounts"
              required
              error={finOrgForm.formState.errors.chartOfAccountsId?.message}
            >
              <Controller
                control={finOrgForm.control}
                name="chartOfAccountsId"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={charts.map((c) => ({ value: c.id, label: `${c.code} — ${c.name}` }))}
                    placeholder={isId ? 'Pilih CoA...' : 'Select CoA...'}
                  />
                )}
              />
            </FormField>
            <FormField
              label={isId ? 'Kalender Fiskal' : 'Fiscal Calendar'}
              required
              error={finOrgForm.formState.errors.fiscalCalendarId?.message}
            >
              <Controller
                control={finOrgForm.control}
                name="fiscalCalendarId"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={calendars.map((c) => ({ value: c.id, label: `${c.code} — ${c.name}` }))}
                    placeholder={isId ? 'Pilih Kalender...' : 'Select Calendar...'}
                  />
                )}
              />
            </FormField>
          </div>
        </form>
      </AppDialog>

      {/* Create Ledger Dialog */}
      <AppDialog
        open={ledgerDialogOpen}
        onOpenChange={setLedgerDialogOpen}
        size="md"
        title={isId ? 'Buat Buku Besar (Ledger)' : 'Create Accounting Ledger'}
        footer={
          <>
            <Button variant="outline" onClick={() => setLedgerDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button
              type="submit"
              form="ledger-create-form"
              isLoading={createLedgerMutation.isPending}
            >
              {isId ? 'Simpan Ledger' : 'Save Ledger'}
            </Button>
          </>
        }
      >
        <form
          id="ledger-create-form"
          onSubmit={ledgerForm.handleSubmit((vals) => {
            setSubmitError(null);
            createLedgerMutation.mutate(vals);
          })}
          className="space-y-3"
          noValidate
        >
          {submitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{submitError}</span>
            </div>
          ) : null}

          <FormField
            label={isId ? 'Entitas Keuangan' : 'Financial Organization'}
            required
            error={ledgerForm.formState.errors.financialOrganizationId?.message}
          >
            <Controller
              control={ledgerForm.control}
              name="financialOrganizationId"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  options={financialOrgs.map((f) => ({ value: f.id, label: `${f.code} — ${f.name}` }))}
                />
              )}
            />
          </FormField>

          <FormField
            label={isId ? 'Kode Ledger' : 'Ledger Code'}
            required
            error={ledgerForm.formState.errors.code?.message}
          >
            <Input placeholder="LEDGER-PRIMARY-IDR" {...ledgerForm.register('code')} />
          </FormField>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <FormField
              label={isId ? 'Tipe Ledger' : 'Ledger Type'}
              required
              error={ledgerForm.formState.errors.ledgerType?.message}
            >
              <Controller
                control={ledgerForm.control}
                name="ledgerType"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={[
                      { value: 'Primary', label: 'Primary' },
                      { value: 'Secondary', label: 'Secondary' },
                      { value: 'Consolidation', label: 'Consolidation' },
                    ]}
                  />
                )}
              />
            </FormField>

            <FormField
              label={isId ? 'Mata Uang' : 'Currency'}
              required
              error={ledgerForm.formState.errors.currencyCode?.message}
            >
              <Controller
                control={ledgerForm.control}
                name="currencyCode"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={currencyOptions}
                  />
                )}
              />
            </FormField>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
