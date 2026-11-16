import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { BookOpen, CheckCircle2, Download, FileSpreadsheet, Plus, Scale } from 'lucide-react';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { Button } from '../../components/ui/Button';
import { EmptyState } from '../../components/ui/EmptyState';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select, type SelectOption } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { apiClient, unwrapListResponse } from '../../lib/api';
import { addDecimals, Decimal, formatCurrency, formatDate, formatRate } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { ListEnvelope } from '../../types';
import { JournalDetailDrawer } from './JournalDetailDrawer';
import { JournalFormDialog } from './JournalFormDialog';
import {
  normalizeJournalStatus,
  type AccountLookupDto,
  type FiscalPeriodLookupDto,
  type JournalEntryDto,
  type JournalStatusFilter,
  type LedgerLookupDto,
  type TrialBalanceRowDto,
} from './journals-types';

export interface JournalsWorkspaceProps {
  search?: string;
  status?: JournalStatusFilter;
  page?: number;
  onSearchStateChange?: (next: {
    search?: string;
    status?: JournalStatusFilter;
    page?: number;
  }) => void;
}

interface EnrichedTrialBalanceRow extends TrialBalanceRowDto {
  accountCode: string;
  accountName: string;
  accountType: string;
}

export function JournalsWorkspace({
  search: controlledSearch,
  status: controlledStatus,
  page: controlledPage,
  onSearchStateChange,
}: JournalsWorkspaceProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const user = useAuthStore((s) => s.user);
  const orgId = activeOrg?.id ?? 'default';
  const baseCurrency = activeOrg?.functionalCurrency || 'IDR';

  const [localSearch, setLocalSearch] = useState(controlledSearch ?? '');
  const [localStatus, setLocalStatus] = useState<JournalStatusFilter>(controlledStatus ?? 'ALL');
  const [localPage, setLocalPage] = useState(controlledPage ?? 1);
  const [activeTab, setActiveTab] = useState<'entries' | 'trial-balance'>('entries');

  const search = controlledSearch ?? localSearch;
  const statusFilter = controlledStatus ?? localStatus;
  const page = controlledPage ?? localPage;

  const [formOpen, setFormOpen] = useState(false);
  const [editingEntry, setEditingEntry] = useState<JournalEntryDto | null>(null);
  const [drawerEntryId, setDrawerEntryId] = useState<string | null>(null);
  const [selectedTbLedgerId, setSelectedTbLedgerId] = useState('');
  const [selectedTbPeriodId, setSelectedTbPeriodId] = useState('');

  const canCreate = hasUserPermission(user, ['gl:create', 'gl.create', 'gl.*']);

  const updateFilterState = (next: {
    search?: string;
    status?: JournalStatusFilter;
    page?: number;
  }) => {
    if (next.search !== undefined) setLocalSearch(next.search);
    if (next.status !== undefined) setLocalStatus(next.status);
    if (next.page !== undefined) setLocalPage(next.page);
    onSearchStateChange?.(next);
  };

  const journalsQuery = useQuery({
    queryKey: ['journals', 'list', orgId, page],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<JournalEntryDto> | JournalEntryDto[]>(
        '/journal-entries',
        { query: { page, size: 25 } }
      );
      return unwrapListResponse(res);
    },
  });

  const ledgersQuery = useQuery({
    queryKey: ['ledgers', 'lookup', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<LedgerLookupDto> | LedgerLookupDto[]>('/ledgers', {
        query: { page: 1, size: 100 },
      });
      return unwrapListResponse(res).items;
    },
  });

  const periodsQuery = useQuery({
    queryKey: ['fiscal-periods', 'lookup', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<FiscalPeriodLookupDto> | FiscalPeriodLookupDto[]>(
        '/fiscal-periods',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const accountsQuery = useQuery({
    queryKey: ['coa-accounts', 'lookup', orgId],
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
  const accounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);

  useEffect(() => {
    if (!selectedTbLedgerId && ledgers.length > 0) {
      setSelectedTbLedgerId(ledgers[0].id);
    }
  }, [selectedTbLedgerId, ledgers]);

  useEffect(() => {
    if (!selectedTbPeriodId && periods.length > 0) {
      const preferred =
        periods.find((p) => p.status.trim().toUpperCase() === 'OPEN') ?? periods[0];
      setSelectedTbPeriodId(preferred.id);
    }
  }, [selectedTbPeriodId, periods]);

  const accountsById = useMemo(() => {
    const map = new Map<string, AccountLookupDto>();
    for (const acc of accounts) {
      map.set(acc.id, acc);
    }
    return map;
  }, [accounts]);

  const ledgersById = useMemo(() => {
    const map = new Map<string, LedgerLookupDto>();
    for (const l of ledgers) {
      map.set(l.id, l);
    }
    return map;
  }, [ledgers]);

  const periodsById = useMemo(() => {
    const map = new Map<string, FiscalPeriodLookupDto>();
    for (const p of periods) {
      map.set(p.id, p);
    }
    return map;
  }, [periods]);

  const trialBalanceQuery = useQuery({
    queryKey: ['gl-balances', 'trial-balance', orgId, selectedTbLedgerId, selectedTbPeriodId],
    enabled: activeTab === 'trial-balance' && Boolean(selectedTbLedgerId && selectedTbPeriodId),
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<TrialBalanceRowDto> | TrialBalanceRowDto[]>(
        '/trial-balance',
        {
          query: new URLSearchParams({
            ledgerId: selectedTbLedgerId,
            fiscalPeriodId: selectedTbPeriodId,
            page: '1',
            size: '100',
          }),
        }
      );
      return unwrapListResponse(res);
    },
  });

  const allEntries = useMemo(() => journalsQuery.data?.items ?? [], [journalsQuery.data?.items]);

  const filteredEntries = useMemo(() => {
    const term = search.trim().toLowerCase();
    return allEntries.filter((row) => {
      const normStatus = normalizeJournalStatus(row.status);
      if (statusFilter !== 'ALL' && normStatus !== statusFilter) {
        return false;
      }
      if (!term) return true;
      const ledgerCode = ledgersById.get(row.ledgerId)?.code ?? '';
      return (
        row.journalNo.toLowerCase().includes(term) ||
        (row.sourceDocumentRef ?? '').toLowerCase().includes(term) ||
        row.journalSource.toLowerCase().includes(term) ||
        ledgerCode.toLowerCase().includes(term)
      );
    });
  }, [allEntries, search, statusFilter, ledgersById]);

  const kpiStats = useMemo(() => {
    let draftCount = 0;
    let postedCount = 0;
    let totalPostedDebit = '0.00';

    for (const entry of allEntries) {
      const s = normalizeJournalStatus(entry.status);
      if (s === 'DRAFT') draftCount += 1;
      if (s === 'POSTED') {
        postedCount += 1;
        totalPostedDebit = addDecimals(totalPostedDebit, entry.totalDebit || '0', 2);
      }
    }

    return {
      totalElements: journalsQuery.data?.page.totalElements ?? allEntries.length,
      draftCount,
      postedCount,
      totalPostedDebit,
    };
  }, [allEntries, journalsQuery.data?.page.totalElements]);

  const enrichedTrialBalanceRows: EnrichedTrialBalanceRow[] = useMemo(() => {
    const rows = trialBalanceQuery.data?.items ?? [];
    return rows.map((row) => {
      const acc = accountsById.get(row.accountId);
      return {
        ...row,
        accountCode: acc?.code ?? row.accountId.slice(0, 8),
        accountName: acc?.name ?? (isId ? 'Akun Tidak Ditemukan' : 'Unknown Account'),
        accountType: acc?.accountType ?? '—',
      };
    });
  }, [trialBalanceQuery.data?.items, accountsById, isId]);

  const trialBalanceTotals = useMemo(() => {
    let periodDebit = '0.00';
    let periodCredit = '0.00';
    let closingBalance = '0.00';

    for (const row of enrichedTrialBalanceRows) {
      periodDebit = addDecimals(periodDebit, row.periodDebit || '0', 2);
      periodCredit = addDecimals(periodCredit, row.periodCredit || '0', 2);
      closingBalance = addDecimals(closingBalance, row.closingBalance || '0', 2);
    }

    const isBalanced = new Decimal(periodDebit, 2).equals(new Decimal(periodCredit, 2));
    return { periodDebit, periodCredit, closingBalance, isBalanced };
  }, [enrichedTrialBalanceRows]);

  const statusOptions: SelectOption[] = useMemo(
    () => [
      { value: 'ALL', label: isId ? 'Semua Status' : 'All Statuses' },
      { value: 'DRAFT', label: isId ? 'Draf (DRAFT)' : 'Draft' },
      { value: 'POSTED', label: isId ? 'Diposting (POSTED)' : 'Posted' },
      { value: 'REVERSED', label: isId ? 'Dibalik (REVERSED)' : 'Reversed' },
    ],
    [isId]
  );

  const ledgerOptions: SelectOption[] = useMemo(
    () =>
      ledgers.map((l) => ({
        value: l.id,
        label: `${l.code} (${l.currencyCode})`,
      })),
    [ledgers]
  );

  const periodOptions: SelectOption[] = useMemo(
    () =>
      periods.map((p) => ({
        value: p.id,
        label: `${isId ? 'Periode' : 'Period'} #${p.periodNo} (${p.startDate} → ${p.endDate})`,
      })),
    [periods, isId]
  );

  const journalColumns: ColumnDef<JournalEntryDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Nomor Jurnal' : 'Journal No',
        sortKey: 'journalNo',
        width: '155px',
        cell: (row) => (
          <div>
            <span className="font-mono font-bold text-brand-700 dark:text-brand-400">
              {row.journalNo}
            </span>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
              {ledgersById.get(row.ledgerId)?.code ?? row.journalSource}
            </p>
          </div>
        ),
      },
      {
        header: isId ? 'Tgl Posting' : 'Posting Date',
        sortKey: 'postingDate',
        width: '120px',
        cell: (row) => (
          <span className="font-mono text-xs text-slate-700 dark:text-slate-300">
            {formatDate(row.postingDate, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Referensi / Memo' : 'Source Ref / Memo',
        width: '220px',
        cell: (row) => (
          <span className="truncate text-slate-700 dark:text-slate-300">
            {row.sourceDocumentRef || row.journalSource}
          </span>
        ),
      },
      {
        header: isId ? 'Valas & Kurs' : 'CCY & Rate',
        width: '125px',
        cell: (row) => (
          <span className="font-mono text-xs font-semibold text-brand-700 dark:text-brand-400">
            {row.currencyCode} @ {formatRate(row.exchangeRate, locale)}
          </span>
        ),
      },
      {
        header: 'Total Debit',
        align: 'right',
        width: '160px',
        cell: (row) => (
          <span className="font-money font-semibold text-slate-900 dark:text-slate-100">
            {formatCurrency(row.totalDebit, row.currencyCode, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Total Kredit' : 'Total Credit',
        align: 'right',
        width: '160px',
        cell: (row) => (
          <span className="font-money font-semibold text-slate-900 dark:text-slate-100">
            {formatCurrency(row.totalCredit, row.currencyCode, locale)}
          </span>
        ),
      },
      {
        header: 'Status',
        width: '130px',
        cell: (row) => <StatusBadge status={normalizeJournalStatus(row.status)} />,
      },
    ],
    [isId, locale, ledgersById]
  );

  const trialBalanceColumns: ColumnDef<EnrichedTrialBalanceRow>[] = useMemo(
    () => [
      {
        header: isId ? 'Kode Akun' : 'Account Code',
        width: '130px',
        cell: (row) => (
          <span className="font-mono font-semibold text-slate-900 dark:text-slate-100">
            {row.accountCode}
          </span>
        ),
      },
      {
        header: isId ? 'Nama Akun' : 'Account Name',
        width: '240px',
        cell: (row) => (
          <div>
            <span className="font-medium text-slate-800 dark:text-slate-200">
              {row.accountName}
            </span>
            <p className="text-[11px] text-slate-500 dark:text-slate-400">{row.accountType}</p>
          </div>
        ),
      },
      {
        header: isId ? 'Saldo Awal' : 'Opening Balance',
        align: 'right',
        width: '160px',
        cell: (row) => (
          <span className="font-money text-slate-700 dark:text-slate-300">
            {formatCurrency(row.openingBalance, baseCurrency, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Mutasi Debit' : 'Period Debit',
        align: 'right',
        width: '160px',
        cell: (row) => (
          <span className="font-money font-medium text-slate-900 dark:text-slate-100">
            {formatCurrency(row.periodDebit, baseCurrency, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Mutasi Kredit' : 'Period Credit',
        align: 'right',
        width: '160px',
        cell: (row) => (
          <span className="font-money font-medium text-slate-900 dark:text-slate-100">
            {formatCurrency(row.periodCredit, baseCurrency, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Saldo Akhir' : 'Closing Balance',
        align: 'right',
        width: '160px',
        cell: (row) => (
          <span className="font-money font-bold text-slate-900 dark:text-slate-100">
            {formatCurrency(row.closingBalance, baseCurrency, locale)}
          </span>
        ),
      },
    ],
    [isId, baseCurrency, locale]
  );

  const handleExportJournalsCsv = () => {
    exportToCsv(
      filteredEntries,
      [
        { header: 'Journal No', accessor: (r) => r.journalNo },
        { header: 'Posting Date', accessor: (r) => r.postingDate },
        { header: 'Ledger', accessor: (r) => ledgersById.get(r.ledgerId)?.code ?? r.ledgerId },
        { header: 'Currency', accessor: (r) => r.currencyCode },
        { header: 'Exchange Rate', accessor: (r) => r.exchangeRate },
        { header: 'Total Debit', accessor: (r) => r.totalDebit },
        { header: 'Total Credit', accessor: (r) => r.totalCredit },
        { header: 'Status', accessor: (r) => r.status },
        { header: 'Source Ref', accessor: (r) => r.sourceDocumentRef ?? '' },
      ],
      'journal-entries.csv'
    );
  };

  const handleExportTrialBalanceCsv = () => {
    exportToCsv(
      enrichedTrialBalanceRows,
      [
        { header: 'Account Code', accessor: (r) => r.accountCode },
        { header: 'Account Name', accessor: (r) => r.accountName },
        { header: 'Account Type', accessor: (r) => r.accountType },
        { header: 'Opening Balance', accessor: (r) => r.openingBalance },
        { header: 'Period Debit', accessor: (r) => r.periodDebit },
        { header: 'Period Credit', accessor: (r) => r.periodCredit },
        { header: 'Closing Balance', accessor: (r) => r.closingBalance },
      ],
      'trial-balance.csv'
    );
  };

  return (
    <div className="space-y-5">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <Breadcrumbs
            items={[{ label: isId ? 'Buku Besar & Jurnal' : 'General Ledger & Journals' }]}
          />
          <h1 className="mt-1.5 text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {isId ? 'Buku Besar & Jurnal Double-Entry' : 'General Ledger & Double-Entry Journals'}
          </h1>
          <p className="mt-1 text-xs sm:text-sm text-slate-500 dark:text-slate-400">
            {isId
              ? 'Pencatatan entri jurnal multi-baris, validasi keseimbangan desimal tetap, posting, pembalikan, dan neraca saldo.'
              : 'Multi-line double-entry journals with fixed-point balance checks, posting, reversal, and trial balance.'}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<Download className="h-3.5 w-3.5" />}
            onClick={handleExportJournalsCsv}
          >
            {isId ? 'Ekspor CSV' : 'Export CSV'}
          </Button>
          {canCreate ? (
            <Button
              variant="primary"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingEntry(null);
                setFormOpen(true);
              }}
            >
              {isId ? 'Buat Jurnal Baru' : 'New Journal Entry'}
            </Button>
          ) : null}
        </div>
      </header>

      {/* Top KPI Strip */}
      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label={isId ? 'Total Entri Jurnal' : 'Total Journal Entries'}
          value={String(kpiStats.totalElements)}
          badge={`${kpiStats.postedCount} Posted`}
          badgeTone="info"
          subtitle={isId ? 'Tersimpan di GL Service' : 'Persisted in GL Service'}
          accent="brand"
          icon={<BookOpen className="h-4 w-4" />}
          isLoading={journalsQuery.isLoading}
        />
        <KpiCard
          label={isId ? 'Draf Aktif (Halaman Ini)' : 'Drafts (Current Page)'}
          value={String(kpiStats.draftCount)}
          badge={
            kpiStats.draftCount > 0
              ? isId
                ? 'Perlu otorisasi'
                : 'Needs review'
              : isId
                ? 'Semua bersih'
                : 'All clear'
          }
          badgeTone={kpiStats.draftCount > 0 ? 'warning' : 'success'}
          subtitle={isId ? 'Siap ditinjau & diposting' : 'Ready for review & posting'}
          accent="amber"
          icon={<FileSpreadsheet className="h-4 w-4" />}
          isLoading={journalsQuery.isLoading}
        />
        <KpiCard
          label={isId ? 'Jurnal Diposting' : 'Posted Journals'}
          value={String(kpiStats.postedCount)}
          badge="ADR-014"
          badgeTone="brand"
          subtitle={isId ? 'Tercatat ke saldo akun' : 'Committed to GL balances'}
          accent="emerald"
          icon={<CheckCircle2 className="h-4 w-4" />}
          isLoading={journalsQuery.isLoading}
        />
        <KpiCard
          label={isId ? 'Volume Debit Diposting' : 'Posted Debit Volume'}
          value={formatCurrency(kpiStats.totalPostedDebit, baseCurrency, locale)}
          badge={isId ? 'Seimbang 100%' : '100% Balanced'}
          badgeTone="success"
          subtitle={isId ? 'Presisi desimal tetap' : 'Fixed-point decimal sum'}
          accent="sky"
          icon={<Scale className="h-4 w-4" />}
          isLoading={journalsQuery.isLoading}
        />
      </section>

      <Tabs
        value={activeTab}
        onValueChange={(val) => setActiveTab(val as 'entries' | 'trial-balance')}
      >
        <TabsList variant="underline">
          <TabsTrigger value="entries" variant="underline">
            {isId ? `Daftar Jurnal (${kpiStats.totalElements})` : `Journal Entries (${kpiStats.totalElements})`}
          </TabsTrigger>
          <TabsTrigger value="trial-balance" variant="underline">
            {isId ? 'Neraca Saldo (Trial Balance)' : 'Trial Balance'}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="entries">
          <VirtualizedTable
            data={filteredEntries}
            columns={journalColumns}
            getRowId={(row) => row.id}
            selectedRowId={drawerEntryId}
            isLoading={journalsQuery.isLoading}
            error={journalsQuery.error}
            onRetry={() => void journalsQuery.refetch()}
            onRowClick={(row) => setDrawerEntryId(row.id)}
            emptyTitle={isId ? 'Belum Ada Entri Jurnal' : 'No Journal Entries Found'}
            emptyDescription={
              isId
                ? 'Ubah filter pencarian/status atau buat draf jurnal baru.'
                : 'Adjust your search/status filters or create a new journal entry.'
            }
            onResetFilters={() => updateFilterState({ search: '', status: 'ALL', page: 1 })}
            toolbar={{
              searchValue: search,
              onSearchChange: (val) => updateFilterState({ search: val, page: 1 }),
              searchPlaceholder: isId
                ? 'Cari nomor jurnal, memo, atau sumber...'
                : 'Search journal no, memo, or source...',
              filters: (
                <div className="w-48">
                  <Select
                    aria-label={isId ? 'Filter Status Jurnal' : 'Filter Journal Status'}
                    value={statusFilter}
                    onValueChange={(val) =>
                      updateFilterState({ status: val as JournalStatusFilter, page: 1 })
                    }
                    options={statusOptions}
                  />
                </div>
              ),
            }}
            pagination={{
              page,
              size: journalsQuery.data?.page.size ?? 25,
              totalElements: journalsQuery.data?.page.totalElements ?? filteredEntries.length,
              totalPages: journalsQuery.data?.page.totalPages ?? 1,
              pageBase: 1,
              onPageChange: (nextPage) => updateFilterState({ page: nextPage }),
            }}
          />
        </TabsContent>

        <TabsContent value="trial-balance" className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-wrap items-center gap-3">
              <div className="w-56">
                <Select
                  aria-label={isId ? 'Pilih Buku Besar' : 'Select Ledger'}
                  value={selectedTbLedgerId}
                  onValueChange={setSelectedTbLedgerId}
                  options={ledgerOptions}
                  placeholder={isId ? 'Pilih Ledger...' : 'Select Ledger...'}
                />
              </div>
              <div className="w-64">
                <Select
                  aria-label={isId ? 'Pilih Periode Fiskal' : 'Select Fiscal Period'}
                  value={selectedTbPeriodId}
                  onValueChange={setSelectedTbPeriodId}
                  options={periodOptions}
                  placeholder={isId ? 'Pilih Periode...' : 'Select Period...'}
                />
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-3 text-xs">
              <span>
                {isId ? 'Mutasi Debit:' : 'Period Debit:'}{' '}
                <strong className="font-money text-slate-900 dark:text-slate-100">
                  {formatCurrency(trialBalanceTotals.periodDebit, baseCurrency, locale)}
                </strong>
              </span>
              <span>
                {isId ? 'Mutasi Kredit:' : 'Period Credit:'}{' '}
                <strong className="font-money text-slate-900 dark:text-slate-100">
                  {formatCurrency(trialBalanceTotals.periodCredit, baseCurrency, locale)}
                </strong>
              </span>
              <StatusBadge
                status={trialBalanceTotals.isBalanced ? 'POSTED' : 'REJECTED'}
                label={
                  trialBalanceTotals.isBalanced
                    ? isId
                      ? 'Seimbang'
                      : 'Balanced'
                    : isId
                      ? 'Selisih'
                      : 'Out of Balance'
                }
              />
            </div>
          </div>

          {!selectedTbLedgerId || !selectedTbPeriodId ? (
            <EmptyState
              title={isId ? 'Pilih Ledger & Periode Fiskal' : 'Select Ledger & Fiscal Period'}
              description={
                isId
                  ? 'Neraca saldo membutuhkan Ledger dan Periode Fiskal dari CoA Service.'
                  : 'Trial balance requires both an active Ledger and Fiscal Period from CoA Service.'
              }
            />
          ) : (
            <VirtualizedTable
              data={enrichedTrialBalanceRows}
              columns={trialBalanceColumns}
              getRowId={(row) => row.id}
              isLoading={trialBalanceQuery.isLoading}
              error={trialBalanceQuery.error}
              onRetry={() => void trialBalanceQuery.refetch()}
              emptyTitle={isId ? 'Belum Ada Saldo Akun' : 'No Account Balances Yet'}
              emptyDescription={
                isId
                  ? 'Belum ada jurnal yang diposting pada ledger dan periode fiskal terpilih.'
                  : 'No posted journal activity exists for the selected ledger and fiscal period.'
              }
              toolbar={{
                onExportCsv: handleExportTrialBalanceCsv,
              }}
            />
          )}
        </TabsContent>
      </Tabs>

      <JournalFormDialog
        open={formOpen}
        onOpenChange={(nextOpen) => {
          setFormOpen(nextOpen);
          if (!nextOpen) setEditingEntry(null);
        }}
        editingEntry={editingEntry}
      />

      <JournalDetailDrawer
        entryId={drawerEntryId}
        open={Boolean(drawerEntryId)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDrawerEntryId(null);
        }}
        onEdit={(entry) => {
          setEditingEntry(entry);
          setFormOpen(true);
        }}
        accountsById={accountsById}
        ledgersById={ledgersById}
        periodsById={periodsById}
      />
    </div>
  );
}
