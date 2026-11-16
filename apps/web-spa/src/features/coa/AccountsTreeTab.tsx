import { useEffect, useMemo, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  AlertCircle,
  Edit3,
  FileSpreadsheet,
  FolderTree,
  Plus,
  Trash2,
} from 'lucide-react';
import { Controller, useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox, type ComboboxOption } from '../../components/ui/Combobox';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { Select, type SelectOption } from '../../components/ui/Select';
import { StatusBadge, type StatusTone } from '../../components/ui/StatusBadge';
import { Switch } from '../../components/ui/Switch';
import { ApiClientError, apiClient, applyApiErrorsToForm } from '../../lib/api';
import { exportToCsv } from '../../lib/export';
import { useUiStore } from '../../store/ui-store';
import {
  ACCOUNT_TYPES,
  accountFormSchema,
  buildAccountHierarchyRows,
  chartOfAccountsSchema,
  type AccountDto,
  type AccountFormValues,
  type AccountTreeRow,
  type ChartOfAccountsDto,
  type ChartOfAccountsFormValues,
} from './coa-types';

export interface AccountsTreeTabProps {
  charts: ChartOfAccountsDto[];
  accounts: AccountDto[];
  isLoading: boolean;
  error: unknown;
  onRetry: () => void;
  canWrite: boolean;
}

const ACCOUNT_TYPE_TONE: Record<string, StatusTone> = {
  Asset: 'info',
  Liability: 'warning',
  Equity: 'brand',
  Revenue: 'success',
  Expense: 'danger',
};

export function AccountsTreeTab({
  charts,
  accounts,
  isLoading,
  error,
  onRetry,
  canWrite,
}: AccountsTreeTabProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const queryClient = useQueryClient();

  const [selectedChartId, setSelectedChartId] = useState<string>('ALL');
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('ALL');
  const [postableOnly, setPostableOnly] = useState(false);

  const [coaDialogOpen, setCoaDialogOpen] = useState(false);
  const [editingCoa, setEditingCoa] = useState<ChartOfAccountsDto | null>(null);
  const [accountDialogOpen, setAccountDialogOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<AccountDto | null>(null);
  const [deletingAccount, setDeletingAccount] = useState<AccountDto | null>(null);
  const [formSubmitError, setFormSubmitError] = useState<string | null>(null);

  useEffect(() => {
    if (selectedChartId !== 'ALL' && charts.some((c) => c.id === selectedChartId)) return;
    if (charts.length === 1) {
      setSelectedChartId(charts[0].id);
    }
  }, [charts, selectedChartId]);

  const chartOptions: SelectOption[] = useMemo(
    () => [
      { value: 'ALL', label: isId ? 'Semua Bagan Akun (CoA)' : 'All Charts of Accounts' },
      ...charts.map((c) => ({ value: c.id, label: `${c.code} — ${c.name}` })),
    ],
    [charts, isId]
  );

  const typeFilterOptions: SelectOption[] = useMemo(
    () => [
      { value: 'ALL', label: isId ? 'Semua Tipe Akun' : 'All Account Types' },
      ...ACCOUNT_TYPES.map((t) => ({ value: t, label: t })),
    ],
    [isId]
  );

  const chartForm = useForm<ChartOfAccountsFormValues>({
    resolver: zodResolver(chartOfAccountsSchema),
    defaultValues: { code: '', name: '', structure: '4-Digit' },
  });

  const accountForm = useForm<AccountFormValues>({
    resolver: zodResolver(accountFormSchema),
    defaultValues: {
      chartOfAccountsId: '',
      code: '',
      name: '',
      accountType: 'Asset',
      accountClass: '',
      parentId: '',
      isControl: false,
      isMonetary: false,
      allowPosting: true,
    },
  });

  useEffect(() => {
    if (!coaDialogOpen) return;
    setFormSubmitError(null);
    if (editingCoa) {
      chartForm.reset({
        code: editingCoa.code,
        name: editingCoa.name,
        structure: editingCoa.structure ?? '',
      });
    } else {
      chartForm.reset({ code: '', name: '', structure: '4-Digit' });
    }
  }, [coaDialogOpen, editingCoa, chartForm]);

  useEffect(() => {
    if (!accountDialogOpen) return;
    setFormSubmitError(null);
    if (editingAccount) {
      accountForm.reset({
        chartOfAccountsId: editingAccount.chartOfAccountsId,
        code: editingAccount.code,
        name: editingAccount.name,
        accountType: ACCOUNT_TYPES.includes(editingAccount.accountType as AccountFormValues['accountType'])
          ? (editingAccount.accountType as AccountFormValues['accountType'])
          : 'Asset',
        accountClass: editingAccount.accountClass ?? '',
        parentId: editingAccount.parentId ?? '',
        isControl: Boolean(editingAccount.isControl),
        isMonetary: Boolean(editingAccount.isMonetary),
        allowPosting: Boolean(editingAccount.allowPosting),
      });
    } else {
      const defaultChartId =
        selectedChartId !== 'ALL' ? selectedChartId : (charts[0]?.id ?? '');
      accountForm.reset({
        chartOfAccountsId: defaultChartId,
        code: '',
        name: '',
        accountType: 'Asset',
        accountClass: '',
        parentId: '',
        isControl: false,
        isMonetary: false,
        allowPosting: true,
      });
    }
  }, [accountDialogOpen, editingAccount, selectedChartId, charts, accountForm]);

  const invalidateCoa = () => {
    void queryClient.invalidateQueries({ queryKey: ['coa-accounts'] });
    void queryClient.invalidateQueries({ queryKey: ['dashboard-metrics'] });
  };

  const coaMutation = useMutation({
    mutationFn: async (vals: ChartOfAccountsFormValues) => {
      if (editingCoa) {
        return apiClient<ChartOfAccountsDto>(`/charts-of-accounts/${editingCoa.id}`, {
          method: 'PUT',
          ifMatch: String(editingCoa.rowVersion),
          body: JSON.stringify({
            name: vals.name.trim(),
            structure: vals.structure?.trim() || undefined,
          }),
        });
      }
      return apiClient<ChartOfAccountsDto>('/charts-of-accounts', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: vals.code.trim(),
          name: vals.name.trim(),
          structure: vals.structure?.trim() || undefined,
        }),
      });
    },
    onSuccess: (saved) => {
      invalidateCoa();
      setCoaDialogOpen(false);
      setSelectedChartId(saved.id);
      toast.success(
        isId ? `Bagan Akun ${saved.code} berhasil disimpan.` : `Chart of Accounts ${saved.code} saved.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, chartForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to save Chart of Accounts';
      setFormSubmitError(msg);
      toast.error(msg);
    },
  });

  const accountMutation = useMutation({
    mutationFn: async (vals: AccountFormValues) => {
      const payload = {
        name: vals.name.trim(),
        accountType: vals.accountType,
        accountClass: vals.accountClass?.trim() || undefined,
        parentId: vals.parentId?.trim() || undefined,
        isControl: vals.isControl,
        isMonetary: vals.isMonetary,
        allowPosting: vals.allowPosting,
      };
      if (editingAccount) {
        return apiClient<AccountDto>(`/accounts/${editingAccount.id}`, {
          method: 'PUT',
          ifMatch: String(editingAccount.rowVersion),
          body: JSON.stringify(payload),
        });
      }
      return apiClient<AccountDto>('/accounts', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          ...payload,
          chartOfAccountsId: vals.chartOfAccountsId,
          code: vals.code.trim(),
        }),
      });
    },
    onSuccess: (saved) => {
      invalidateCoa();
      setAccountDialogOpen(false);
      toast.success(
        isId ? `Akun ${saved.code} — ${saved.name} disimpan.` : `Account ${saved.code} saved.`
      );
    },
    onError: (err) => {
      applyApiErrorsToForm(err, accountForm.setError);
      const msg = err instanceof Error ? err.message : 'Failed to save account';
      setFormSubmitError(msg);
      toast.error(msg);
    },
  });

  const deleteAccountMutation = useMutation({
    mutationFn: async (target: AccountDto) => {
      await apiClient<unknown>(`/accounts/${target.id}`, {
        method: 'DELETE',
        ifMatch: String(target.rowVersion),
      });
    },
    onSuccess: () => {
      invalidateCoa();
      setDeletingAccount(null);
      toast.success(isId ? 'Akun berhasil dihapus.' : 'Account deleted.');
    },
    onError: (err) => {
      const msg =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : 'Failed to delete account';
      toast.error(msg);
    },
  });

  const scopedAccounts = useMemo(() => {
    const byChart =
      selectedChartId === 'ALL'
        ? accounts
        : accounts.filter((a) => a.chartOfAccountsId === selectedChartId);
    const treeRows = buildAccountHierarchyRows(byChart);
    const term = search.trim().toLowerCase();

    return treeRows.filter((row) => {
      if (typeFilter !== 'ALL' && row.accountType !== typeFilter) return false;
      if (postableOnly && !row.allowPosting) return false;
      if (!term) return true;
      return (
        row.code.toLowerCase().includes(term) ||
        row.name.toLowerCase().includes(term) ||
        (row.accountClass ?? '').toLowerCase().includes(term)
      );
    });
  }, [accounts, selectedChartId, search, typeFilter, postableOnly]);

  const parentAccountOptions: ComboboxOption[] = useMemo(() => {
    const activeChart = accountForm.watch('chartOfAccountsId');
    return accounts
      .filter(
        (acc) =>
          (!activeChart || acc.chartOfAccountsId === activeChart) &&
          (!editingAccount || acc.id !== editingAccount.id)
      )
      .map((acc) => ({
        value: acc.id,
        label: `${acc.code} — ${acc.name}`,
        subtitle: acc.accountType,
        keywords: [acc.code, acc.name],
      }));
  }, [accounts, accountForm, editingAccount]);

  const columns: ColumnDef<AccountTreeRow>[] = useMemo(
    () => [
      {
        header: isId ? 'Hierarki & Kode Akun' : 'Account Hierarchy & Code',
        width: '230px',
        cell: (row) => (
          <div
            className="flex items-center gap-2"
            style={{ paddingLeft: `${row.depth * 18}px` }}
          >
            {row.hasChildren ? (
              <FolderTree className="h-4 w-4 shrink-0 text-brand-600 dark:text-brand-400" aria-hidden="true" />
            ) : (
              <FileSpreadsheet className="h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
            )}
            <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
              {row.code}
            </span>
          </div>
        ),
      },
      {
        header: isId ? 'Nama Akun' : 'Account Name',
        width: '260px',
        cell: (row) => (
          <div>
            <span className={row.isControl ? 'font-semibold text-slate-900 dark:text-slate-100' : 'text-slate-800 dark:text-slate-200'}>
              {row.name}
            </span>
            {row.accountClass ? (
              <p className="text-[11px] text-slate-500 dark:text-slate-400">{row.accountClass}</p>
            ) : null}
          </div>
        ),
      },
      {
        header: isId ? 'Tipe Akun' : 'Account Type',
        width: '130px',
        cell: (row) => (
          <StatusBadge
            status={row.accountType}
            label={row.accountType}
            tone={ACCOUNT_TYPE_TONE[row.accountType] ?? 'neutral'}
          />
        ),
      },
      {
        header: isId ? 'Kontrol & Posting' : 'Flags & Posting',
        width: '220px',
        cell: (row) => (
          <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
            <span
              className={`rounded-md px-1.5 py-0.5 font-medium ${
                row.allowPosting
                  ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300'
                  : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'
              }`}
            >
              {row.allowPosting ? (isId ? 'Bisa Posting' : 'Postable') : isId ? 'Header' : 'Summary'}
            </span>
            {row.isControl ? (
              <span className="rounded-md bg-brand-50 px-1.5 py-0.5 font-medium text-brand-700 dark:bg-brand-950/60 dark:text-brand-300">
                Control
              </span>
            ) : null}
            {row.isMonetary ? (
              <span className="rounded-md bg-amber-50 px-1.5 py-0.5 font-medium text-amber-800 dark:bg-amber-950/60 dark:text-amber-300">
                Monetary
              </span>
            ) : null}
          </div>
        ),
      },
      {
        header: isId ? 'Aksi' : 'Actions',
        align: 'right',
        width: '120px',
        cell: (row) =>
          canWrite ? (
            <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
              <Button
                variant="ghost"
                size="icon"
                aria-label={`${isId ? 'Ubah akun' : 'Edit account'} ${row.code}`}
                onClick={() => {
                  setEditingAccount(row);
                  setAccountDialogOpen(true);
                }}
              >
                <Edit3 className="h-3.5 w-3.5" aria-hidden="true" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                aria-label={`${isId ? 'Hapus akun' : 'Delete account'} ${row.code}`}
                className="text-slate-400 hover:text-rose-600"
                onClick={() => setDeletingAccount(row)}
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
              </Button>
            </div>
          ) : (
            <span className="text-xs text-slate-400">—</span>
          ),
      },
    ],
    [canWrite, isId]
  );

  const selectedChart = charts.find((c) => c.id === selectedChartId) ?? null;

  return (
    <div className="space-y-4">
      {/* CoA Switcher & Creation Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
        <div className="flex flex-wrap items-center gap-3">
          <div className="w-72">
            <Select
              aria-label={isId ? 'Pilih Bagan Akun' : 'Select Chart of Accounts'}
              value={selectedChartId}
              onValueChange={setSelectedChartId}
              options={chartOptions}
            />
          </div>
          {canWrite && selectedChart ? (
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Edit3 className="h-3.5 w-3.5" />}
              onClick={() => {
                setEditingCoa(selectedChart);
                setCoaDialogOpen(true);
              }}
            >
              {isId ? 'Ubah CoA' : 'Edit CoA'}
            </Button>
          ) : null}
          {canWrite ? (
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => {
                setEditingCoa(null);
                setCoaDialogOpen(true);
              }}
            >
              {isId ? 'Bagan Akun Baru' : 'New Chart of Accounts'}
            </Button>
          ) : null}
        </div>

        {canWrite ? (
          <Button
            variant="primary"
            size="sm"
            disabled={charts.length === 0}
            leftIcon={<Plus className="h-4 w-4" />}
            onClick={() => {
              setEditingAccount(null);
              setAccountDialogOpen(true);
            }}
          >
            {isId ? 'Tambah Akun GL' : 'Add GL Account'}
          </Button>
        ) : null}
      </div>

      {/* Inline Delete Account Confirmation */}
      {deletingAccount ? (
        <div
          role="alert"
          className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs dark:border-rose-800/70 dark:bg-rose-950/50"
        >
          <span className="font-semibold text-rose-800 dark:text-rose-200">
            {isId
              ? `Hapus akun ${deletingAccount.code} — ${deletingAccount.name}?`
              : `Delete account ${deletingAccount.code} — ${deletingAccount.name}?`}
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="danger"
              size="xs"
              isLoading={deleteAccountMutation.isPending}
              onClick={() => deleteAccountMutation.mutate(deletingAccount)}
            >
              {isId ? 'Konfirmasi Hapus' : 'Confirm Delete'}
            </Button>
            <Button variant="outline" size="xs" onClick={() => setDeletingAccount(null)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
          </div>
        </div>
      ) : null}

      <VirtualizedTable
        data={scopedAccounts}
        columns={columns}
        getRowId={(row) => row.id}
        isLoading={isLoading}
        error={error}
        onRetry={onRetry}
        emptyTitle={isId ? 'Belum Ada Akun GL' : 'No Accounts Found'}
        emptyDescription={
          isId
            ? 'Tambahkan akun baru atau ubah filter pencarian & tipe akun.'
            : 'Add a new GL account or reset your search and account type filters.'
        }
        onResetFilters={() => {
          setSearch('');
          setTypeFilter('ALL');
          setPostableOnly(false);
        }}
        toolbar={{
          searchValue: search,
          onSearchChange: setSearch,
          searchPlaceholder: isId ? 'Cari kode atau nama akun...' : 'Search account code or name...',
          filters: (
            <div className="flex flex-wrap items-center gap-3">
              <div className="w-48">
                <Select
                  aria-label={isId ? 'Filter Tipe Akun' : 'Filter Account Type'}
                  value={typeFilter}
                  onValueChange={setTypeFilter}
                  options={typeFilterOptions}
                />
              </div>
              <label className="inline-flex cursor-pointer items-center gap-2 text-xs font-medium text-slate-700 dark:text-slate-300">
                <Switch
                  checked={postableOnly}
                  onCheckedChange={setPostableOnly}
                  label={isId ? 'Hanya akun bisa posting' : 'Postable accounts only'}
                />
                <span>{isId ? 'Hanya Bisa Posting' : 'Postable Only'}</span>
              </label>
            </div>
          ),
          onExportCsv: () =>
            exportToCsv(
              scopedAccounts,
              [
                { header: 'Code', accessor: (r) => r.code },
                { header: 'Name', accessor: (r) => r.name },
                { header: 'Type', accessor: (r) => r.accountType },
                { header: 'Class', accessor: (r) => r.accountClass ?? '' },
                { header: 'Control', accessor: (r) => r.isControl },
                { header: 'Monetary', accessor: (r) => r.isMonetary },
                { header: 'Allow Posting', accessor: (r) => r.allowPosting },
              ],
              'chart-of-accounts.csv'
            ),
        }}
      />

      {/* Chart of Accounts Dialog */}
      <AppDialog
        open={coaDialogOpen}
        onOpenChange={setCoaDialogOpen}
        size="md"
        title={
          editingCoa
            ? `${isId ? 'Ubah Bagan Akun' : 'Edit Chart of Accounts'} (${editingCoa.code})`
            : isId
              ? 'Buat Bagan Akun (CoA)'
              : 'Create Chart of Accounts'
        }
        footer={
          <>
            <Button variant="outline" onClick={() => setCoaDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button type="submit" form="coa-master-form" isLoading={coaMutation.isPending}>
              {isId ? 'Simpan CoA' : 'Save CoA'}
            </Button>
          </>
        }
      >
        <form
          id="coa-master-form"
          onSubmit={chartForm.handleSubmit((vals) => {
            setFormSubmitError(null);
            coaMutation.mutate(vals);
          })}
          className="space-y-3"
          noValidate
        >
          {formSubmitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{formSubmitError}</span>
            </div>
          ) : null}
          <FormField
            label={isId ? 'Kode Bagan Akun' : 'CoA Code'}
            required
            error={chartForm.formState.errors.code?.message}
          >
            <Input disabled={Boolean(editingCoa)} {...chartForm.register('code')} />
          </FormField>
          <FormField
            label={isId ? 'Nama Bagan Akun' : 'CoA Name'}
            required
            error={chartForm.formState.errors.name?.message}
          >
            <Input {...chartForm.register('name')} />
          </FormField>
          <FormField
            label={isId ? 'Struktur Kode' : 'Code Structure'}
            error={chartForm.formState.errors.structure?.message}
          >
            <Input placeholder="4-Digit / PSAK" {...chartForm.register('structure')} />
          </FormField>
        </form>
      </AppDialog>

      {/* Account Create / Edit Dialog */}
      <AppDialog
        open={accountDialogOpen}
        onOpenChange={setAccountDialogOpen}
        size="lg"
        title={
          editingAccount
            ? `${isId ? 'Ubah Akun GL' : 'Edit GL Account'} (${editingAccount.code})`
            : isId
              ? 'Tambah Akun GL Baru'
              : 'Create GL Account'
        }
        footer={
          <>
            <Button variant="outline" onClick={() => setAccountDialogOpen(false)}>
              {isId ? 'Batal' : 'Cancel'}
            </Button>
            <Button type="submit" form="coa-account-form" isLoading={accountMutation.isPending}>
              {isId ? 'Simpan Akun' : 'Save Account'}
            </Button>
          </>
        }
      >
        <form
          id="coa-account-form"
          onSubmit={accountForm.handleSubmit((vals) => {
            setFormSubmitError(null);
            accountMutation.mutate(vals);
          })}
          className="space-y-4"
          noValidate
        >
          {formSubmitError ? (
            <div role="alert" className="flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300">
              <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{formSubmitError}</span>
            </div>
          ) : null}

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <FormField
              label={isId ? 'Bagan Akun (CoA)' : 'Chart of Accounts'}
              required
              error={accountForm.formState.errors.chartOfAccountsId?.message}
            >
              <Controller
                control={accountForm.control}
                name="chartOfAccountsId"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={Boolean(editingAccount)}
                    options={charts.map((c) => ({ value: c.id, label: `${c.code} — ${c.name}` }))}
                  />
                )}
              />
            </FormField>

            <FormField
              label={isId ? 'Kode Akun' : 'Account Code'}
              required
              error={accountForm.formState.errors.code?.message}
            >
              <Input
                disabled={Boolean(editingAccount)}
                placeholder="1101"
                {...accountForm.register('code')}
              />
            </FormField>

            <FormField
              label={isId ? 'Nama Akun' : 'Account Name'}
              required
              error={accountForm.formState.errors.name?.message}
            >
              <Input placeholder="Kas & Setara Kas" {...accountForm.register('name')} />
            </FormField>

            <FormField
              label={isId ? 'Tipe Akun' : 'Account Type'}
              required
              error={accountForm.formState.errors.accountType?.message}
            >
              <Controller
                control={accountForm.control}
                name="accountType"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={ACCOUNT_TYPES.map((t) => ({ value: t, label: t }))}
                  />
                )}
              />
            </FormField>

            <FormField
              label={isId ? 'Akun Induk (Opsional)' : 'Parent Account (Optional)'}
              error={accountForm.formState.errors.parentId?.message}
            >
              <Controller
                control={accountForm.control}
                name="parentId"
                render={({ field }) => (
                  <Combobox
                    value={field.value}
                    onValueChange={field.onChange}
                    options={parentAccountOptions}
                    allowClear
                    placeholder={isId ? 'Tanpa induk (Root)...' : 'No parent (Root)...'}
                  />
                )}
              />
            </FormField>

            <FormField
              label={isId ? 'Kelas Akun' : 'Account Class'}
              error={accountForm.formState.errors.accountClass?.message}
            >
              <Input placeholder="Current Asset / Operating" {...accountForm.register('accountClass')} />
            </FormField>
          </div>

          <div className="grid grid-cols-1 gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-3 sm:grid-cols-3 dark:border-slate-800 dark:bg-slate-900/60">
            <label className="flex cursor-pointer items-center justify-between gap-2 text-xs font-medium text-slate-700 dark:text-slate-200">
              <span>{isId ? 'Izinkan Posting' : 'Allow Posting'}</span>
              <Controller
                control={accountForm.control}
                name="allowPosting"
                render={({ field }) => (
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                )}
              />
            </label>
            <label className="flex cursor-pointer items-center justify-between gap-2 text-xs font-medium text-slate-700 dark:text-slate-200">
              <span>{isId ? 'Akun Kontrol' : 'Control Account'}</span>
              <Controller
                control={accountForm.control}
                name="isControl"
                render={({ field }) => (
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                )}
              />
            </label>
            <label className="flex cursor-pointer items-center justify-between gap-2 text-xs font-medium text-slate-700 dark:text-slate-200">
              <span>{isId ? 'Akun Moneter' : 'Monetary Account'}</span>
              <Controller
                control={accountForm.control}
                name="isMonetary"
                render={({ field }) => (
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                )}
              />
            </label>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
