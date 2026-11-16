import { useMemo, useState } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import {
  Activity,
  BarChart3,
  BookOpen,
  Boxes,
  CheckSquare,
  ChevronRight,
  Coins,
  FileSpreadsheet,
  FolderTree,
  Landmark,
  Plus,
  Receipt,
  RefreshCw,
  Table2,
  TrendingUp,
} from 'lucide-react';
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { Button } from '../../components/ui/Button';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { KpiCard } from '../../components/ui/KpiCard';
import { CardSkeleton, KpiSkeleton } from '../../components/ui/Skeleton';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { apiClient, unwrapListResponse } from '../../lib/api';
import { formatCurrency, formatDateTime, formatNumber } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { ListEnvelope } from '../../types';
import { JournalFormDialog } from '../journals/JournalFormDialog';
import {
  pickLatestTimestamp,
  summarizeGlSnapshots,
  summarizeSalesAndInventory,
  type CategoryBreakdownDatum,
  type DashboardCurrencyDto,
  type DashboardJournalSummaryDto,
  type GlBalanceSnapshotDto,
  type InventoryPositionDto,
  type SalesPeriodDatum,
  type SalesSummaryDto,
  type WarehouseStockDatum,
  type WorkflowTaskDto,
} from './dashboard-utils';

interface TooltipPayloadEntry<T> {
  payload?: T;
}

interface CustomTooltipProps<T> {
  active?: boolean;
  payload?: Array<TooltipPayloadEntry<T>>;
  currency: string;
  locale: 'id' | 'en';
}

const BAR_GRADIENTS: Record<string, { top: string; bottom: string }> = {
  Asset: { top: '#0ea5e9', bottom: '#0284c7' },
  Liability: { top: '#f59e0b', bottom: '#d97706' },
  Equity: { top: '#6366f1', bottom: '#4338ca' },
  Revenue: { top: '#14b8a6', bottom: '#0f766e' },
  Expense: { top: '#f43f5e', bottom: '#be123c' },
  Other: { top: '#94a3b8', bottom: '#64748b' },
};

function GlCategoryTooltip({
  active,
  payload,
  currency,
  locale,
}: CustomTooltipProps<CategoryBreakdownDatum>) {
  if (!active || !payload?.length || !payload[0]?.payload) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900 px-3 py-2.5 text-xs text-slate-50 shadow-[var(--shadow-popover)]">
      <p className="font-semibold text-slate-300">
        {locale === 'id' ? item.labelId : item.labelEn}
      </p>
      <p className="mt-1 font-money text-sm font-bold text-white">
        {formatCurrency(item.amount, currency, locale)}
      </p>
      <p className="mt-0.5 text-[11px] text-slate-400">
        {item.accountCount} {locale === 'id' ? 'akun snapshot' : 'snapshot accounts'}
      </p>
    </div>
  );
}

function SalesTooltip({
  active,
  payload,
  currency,
  locale,
}: CustomTooltipProps<SalesPeriodDatum>) {
  if (!active || !payload?.length || !payload[0]?.payload) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900 px-3 py-2.5 text-xs text-slate-50 shadow-[var(--shadow-popover)]">
      <div className="flex items-center justify-between gap-3">
        <span className="font-mono font-semibold text-slate-300">{item.periodKey}</span>
        <span className="font-semibold text-emerald-400">
          {item.orderCount} {locale === 'id' ? 'pesanan' : 'orders'}
        </span>
      </div>
      <p className="mt-1 font-money text-sm font-bold text-white">
        {formatCurrency(item.totalSalesAmount, currency, locale)}
      </p>
    </div>
  );
}

function InventoryTooltip({
  active,
  payload,
  locale,
}: Omit<CustomTooltipProps<WarehouseStockDatum>, 'currency'>) {
  if (!active || !payload?.length || !payload[0]?.payload) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900 px-3 py-2.5 text-xs text-slate-50 shadow-[var(--shadow-popover)]">
      <p className="font-mono font-semibold text-white">{item.warehouseCode}</p>
      <p className="mt-1 text-slate-300">
        {locale === 'id' ? 'Fisik' : 'On Hand'}:{' '}
        <span className="font-money font-semibold text-white">
          {formatNumber(item.onHandQty, 2, locale)}
        </span>
      </p>
      <p className="text-emerald-400">
        {locale === 'id' ? 'Tersedia' : 'Available'}:{' '}
        <span className="font-money font-semibold">
          {formatNumber(item.availableQty, 2, locale)}
        </span>
      </p>
    </div>
  );
}

export function DashboardWorkspace() {
  const navigate = useNavigate();
  const locale = useUiStore((s) => s.locale);
  const isRealtimeConnected = useUiStore((s) => s.isRealtimeConnected);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const user = useAuthStore((s) => s.user);
  const orgId = activeOrg?.id ?? 'default';
  const baseCurrency = activeOrg?.functionalCurrency || 'IDR';
  const isId = locale === 'id';

  const [showGlTable, setShowGlTable] = useState(false);
  const [showOpsTable, setShowOpsTable] = useState(false);
  const [opsChartMode, setOpsChartMode] = useState<'sales' | 'inventory'>('sales');
  const [journalFormOpen, setJournalFormOpen] = useState(false);

  const canReadReporting = hasUserPermission(user, ['reporting.read', 'reporting:read']);
  const canReadGl = hasUserPermission(user, ['gl:read', 'gl.read', 'gl.*']);
  const canCreateGl = hasUserPermission(user, ['gl:create', 'gl.create', 'gl.*']);
  const canReadWorkflow = hasUserPermission(user, ['workflow:read', 'workflow.read', 'workflow.*']);
  const canReadCurrencies = hasUserPermission(user, ['currency:read', 'currency.read', 'currency.*']);

  const glSnapshotQuery = useQuery({
    queryKey: ['gl-balances', 'dashboard', orgId],
    enabled: canReadReporting,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<GlBalanceSnapshotDto> | GlBalanceSnapshotDto[]>(
        '/gl-balance-snapshots',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res);
    },
  });

  const salesSummaryQuery = useQuery({
    queryKey: ['balances', 'sales-summaries', 'dashboard', orgId],
    enabled: canReadReporting,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<SalesSummaryDto> | SalesSummaryDto[]>(
        '/sales-summaries',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res);
    },
  });

  const inventoryPositionQuery = useQuery({
    queryKey: ['balances', 'inventory-positions', 'dashboard', orgId],
    enabled: canReadReporting,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<InventoryPositionDto> | InventoryPositionDto[]>(
        '/inventory-positions',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res);
    },
  });

  const journalsQuery = useQuery({
    queryKey: ['journals', 'dashboard', orgId],
    enabled: canReadGl,
    queryFn: async () => {
      const res = await apiClient<
        ListEnvelope<DashboardJournalSummaryDto> | DashboardJournalSummaryDto[]
      >('/journal-entries', { query: { page: 1, size: 25 } });
      return unwrapListResponse(res);
    },
  });

  const workflowTasksQuery = useQuery({
    queryKey: ['approvals', 'dashboard', orgId],
    enabled: canReadWorkflow,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<WorkflowTaskDto> | WorkflowTaskDto[]>(
        '/workflow-tasks',
        { query: new URLSearchParams({ status: 'PENDING', page: '1', size: '10' }) }
      );
      return unwrapListResponse(res);
    },
  });

  const currenciesQuery = useQuery({
    queryKey: ['currencies', 'dashboard', orgId],
    enabled: canReadCurrencies,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<DashboardCurrencyDto> | DashboardCurrencyDto[]>(
        '/currencies',
        { query: new URLSearchParams({ isActive: 'true', page: '1', size: '20' }) }
      );
      return unwrapListResponse(res);
    },
  });

  const glSummary = useMemo(
    () => summarizeGlSnapshots(glSnapshotQuery.data?.items ?? []),
    [glSnapshotQuery.data?.items]
  );

  const salesInvSummary = useMemo(
    () =>
      summarizeSalesAndInventory(
        salesSummaryQuery.data?.items ?? [],
        inventoryPositionQuery.data?.items ?? []
      ),
    [salesSummaryQuery.data?.items, inventoryPositionQuery.data?.items]
  );

  const activeCurrencyCodes = useMemo(() => {
    const rows = currenciesQuery.data?.items ?? [];
    if (rows.length === 0) return [baseCurrency];
    return rows.slice(0, 4).map((c) => c.code);
  }, [currenciesQuery.data?.items, baseCurrency]);

  const journalStats = useMemo(() => {
    const items = journalsQuery.data?.items ?? [];
    const total = journalsQuery.data?.page.totalElements ?? items.length;
    const draftCount = items.filter((j) => j.status.trim().toUpperCase() === 'DRAFT').length;
    return { total, draftCount };
  }, [journalsQuery.data]);

  const pendingTasks = workflowTasksQuery.data?.items ?? [];
  const lastRefreshedIso = pickLatestTimestamp([
    glSummary.lastRefreshedAt,
    salesInvSummary.lastRefreshedAt,
  ]);

  const isReportingLoading =
    canReadReporting &&
    (glSnapshotQuery.isLoading ||
      salesSummaryQuery.isLoading ||
      inventoryPositionQuery.isLoading);

  const reportingError =
    glSnapshotQuery.error ?? salesSummaryQuery.error ?? inventoryPositionQuery.error;

  const isRefreshing =
    glSnapshotQuery.isFetching ||
    salesSummaryQuery.isFetching ||
    inventoryPositionQuery.isFetching ||
    journalsQuery.isFetching ||
    workflowTasksQuery.isFetching;

  const handleRefreshAll = () => {
    if (canReadReporting) {
      void glSnapshotQuery.refetch();
      void salesSummaryQuery.refetch();
      void inventoryPositionQuery.refetch();
    }
    if (canReadGl) void journalsQuery.refetch();
    if (canReadWorkflow) void workflowTasksQuery.refetch();
    if (canReadCurrencies) void currenciesQuery.refetch();
  };

  return (
    <div className="space-y-5">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <Breadcrumbs items={[{ label: isId ? 'Dashboard Eksekutif' : 'Executive Dashboard' }]} />
          <div className="mt-1.5 flex flex-wrap items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
              {isId ? 'Dashboard Eksekutif' : 'Executive Dashboard'}
            </h1>
            <span
              className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold ${
                isRealtimeConnected
                  ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/70 dark:bg-emerald-950/50 dark:text-emerald-300'
                  : 'border-border bg-surface-muted text-slate-600 dark:text-slate-300'
              }`}
            >
              <Activity className="h-3 w-3" aria-hidden="true" />
              {isRealtimeConnected
                ? isId
                  ? 'SSE Real-time Aktif'
                  : 'SSE Live'
                : isId
                  ? 'Mode Polling API'
                  : 'API Polling'}
            </span>
          </div>
          <p className="mt-1 text-xs sm:text-sm text-slate-500 dark:text-slate-400">
            {isId
              ? 'Ikhtisar keuangan konsolidasi & persetujuan operasional'
              : 'Consolidated financial overview & pending approvals'}{' '}
            ·{' '}
            <span className="font-medium text-slate-700 dark:text-slate-300">
              {activeOrg ? `${activeOrg.code} — ${activeOrg.name}` : 'Enterprise ERP'}
            </span>{' '}
            ·{' '}
            <span className="font-mono font-semibold text-slate-700 dark:text-slate-200">
              {baseCurrency}
            </span>
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-border bg-surface px-2.5 text-xs shadow-2xs">
            <Coins className="h-3.5 w-3.5 text-brand-600 dark:text-brand-400 shrink-0" aria-hidden="true" />
            <span className="text-slate-500 dark:text-slate-400">
              {isId ? 'Valas:' : 'CCY:'}
            </span>
            <span className="font-mono font-semibold text-slate-800 dark:text-slate-200">
              {activeCurrencyCodes.join(' · ')}
            </span>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleRefreshAll}
            isLoading={isRefreshing}
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
          >
            {isId ? 'Segarkan Data' : 'Refresh'}
          </Button>

          {canCreateGl ? (
            <Button
              variant="primary"
              size="sm"
              leftIcon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => setJournalFormOpen(true)}
            >
              {isId ? 'Buat Jurnal' : 'New Journal'}
            </Button>
          ) : null}
        </div>
      </header>

      {/* Top Tri-Currency KPI Strip */}
      {isReportingLoading ? (
        <section
          aria-label={isId ? 'Memuat indikator utama' : 'Loading key metrics'}
          className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4"
        >
          <KpiSkeleton />
          <KpiSkeleton />
          <KpiSkeleton />
          <KpiSkeleton />
        </section>
      ) : reportingError ? (
        <ErrorState
          error={reportingError}
          title={
            isId
              ? 'Gagal Memuat Snapshot BI Dashboard'
              : 'Failed to Load Dashboard BI Snapshots'
          }
          onRetry={handleRefreshAll}
        />
      ) : (
        <section
          aria-label={isId ? 'Indikator kinerja keuangan' : 'Financial key metrics'}
          className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4"
        >
          <KpiCard
            label={isId ? 'Total Aset Buku Besar (GL)' : 'GL Total Assets'}
            value={
              canReadReporting
                ? formatCurrency(glSummary.assetsTotal, baseCurrency, locale)
                : isId
                  ? 'Akses Dibatasi'
                  : 'Restricted'
            }
            badge={
              canReadGl
                ? `${journalStats.total} ${isId ? 'Jurnal' : 'Journals'}`
                : `${glSummary.snapshotCount} ${isId ? 'Akun' : 'Accounts'}`
            }
            badgeTone="info"
            subtitle={
              canReadGl
                ? `${journalStats.draftCount} ${isId ? 'draf aktif' : 'active drafts'}`
                : undefined
            }
            accent="brand"
            icon={<BookOpen className="h-4 w-4" />}
            onClick={() => navigate({ to: '/journals' })}
          />

          <KpiCard
            label={isId ? 'Kewajiban & Hutang (AP)' : 'Liabilities & Payables (AP)'}
            value={
              canReadReporting
                ? formatCurrency(glSummary.liabilitiesTotal, baseCurrency, locale)
                : isId
                  ? 'Akses Dibatasi'
                  : 'Restricted'
            }
            badge={
              canReadWorkflow
                ? `${pendingTasks.length} ${isId ? 'Tertunda' : 'Pending'}`
                : isId
                  ? 'Modul AP'
                  : 'AP Module'
            }
            badgeTone={pendingTasks.length > 0 ? 'warning' : 'neutral'}
            subtitle={
              canReadReporting
                ? `${isId ? 'Ekuitas' : 'Equity'} ${formatCurrency(glSummary.equityTotal, baseCurrency, locale)}`
                : isId
                  ? 'Membutuhkan izin reporting.read'
                  : 'Requires reporting.read'
            }
            accent="amber"
            icon={<Receipt className="h-4 w-4" />}
            onClick={() => navigate({ to: '/ap' as never })}
          />

          <KpiCard
            label={isId ? 'Pendapatan & Piutang (AR)' : 'Revenue & Receivables (AR)'}
            value={
              canReadReporting
                ? formatCurrency(salesInvSummary.totalSalesAmount, baseCurrency, locale)
                : isId
                  ? 'Akses Dibatasi'
                  : 'Restricted'
            }
            badge={`${salesInvSummary.totalOrders} ${isId ? 'Pesanan' : 'Orders'}`}
            badgeTone="success"
            subtitle={`${isId ? 'Laba GL' : 'GL Margin'} ${formatCurrency(glSummary.netOperatingMargin, baseCurrency, locale)}`}
            accent="emerald"
            icon={<TrendingUp className="h-4 w-4" />}
            onClick={() => navigate({ to: '/ar' as never })}
          />

          <KpiCard
            label={isId ? 'Kas, Valas & Persediaan' : 'Treasury, FX & Inventory'}
            value={
              canReadReporting
                ? `${formatNumber(salesInvSummary.totalAvailableQty, 2, locale)} Unit`
                : activeCurrencyCodes.join(' / ')
            }
            badge={`${activeCurrencyCodes.length} ${isId ? 'Valas Aktif' : 'Currencies'}`}
            badgeTone="brand"
            subtitle={`${isId ? 'Fisik' : 'On-Hand'} ${formatNumber(salesInvSummary.totalOnHandQty, 2, locale)} · ${activeCurrencyCodes.slice(0, 3).join('/')}`}
            accent="sky"
            icon={<Landmark className="h-4 w-4" />}
            onClick={() => navigate({ to: '/treasury' as never })}
          />
        </section>
      )}

      {/* Middle Analytics Charts + Accessible Summary Tables */}
      <section className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        {/* Chart 1: GL Balance by Account Category */}
        <article className="rounded-xl border border-border bg-surface p-5 shadow-[var(--shadow-card)]">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="flex items-center gap-2 text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100">
                <BarChart3 className="h-4 w-4 text-brand-600 dark:text-brand-400" aria-hidden="true" />
                {isId ? 'Saldo Buku Besar per Kategori Akun' : 'GL Balance by Account Category'}
              </h2>
              <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                {isId
                  ? 'Neraca & Laba Rugi konsolidasi · A = L + E'
                  : 'Consolidated Balance Sheet & P&L · A = L + E'}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {canReadReporting && glSummary.snapshotCount > 0 ? (
                <StatusBadge
                  status={glSummary.isAccountingEquationBalanced ? 'ISSUED' : 'OPEN'}
                  tone={glSummary.isAccountingEquationBalanced ? 'brand' : 'neutral'}
                  label={
                    glSummary.isAccountingEquationBalanced
                      ? 'A = L + E · Valid'
                      : `${glSummary.snapshotCount} ${isId ? 'Akun' : 'Accts'}`
                  }
                />
              ) : null}
              <Button
                variant="outline"
                size="xs"
                aria-pressed={showGlTable}
                onClick={() => setShowGlTable((prev) => !prev)}
                leftIcon={<Table2 className="h-3.5 w-3.5" />}
              >
                {showGlTable
                  ? isId
                    ? 'Lihat Grafik'
                    : 'View Chart'
                  : isId
                    ? 'Ringkasan Tabel'
                    : 'Summary Table'}
              </Button>
            </div>
          </div>

          <div className="mt-4">
            {isReportingLoading ? (
              <CardSkeleton />
            ) : !canReadReporting ? (
              <EmptyState
                title={isId ? 'Izin Pelaporan Diperlukan' : 'Reporting Permission Required'}
                description={
                  isId
                    ? 'Akun Anda tidak memiliki izin reporting.read untuk menampilkan grafik BI.'
                    : 'Your role does not have reporting.read permission to view BI charts.'
                }
              />
            ) : glSummary.categorySeries.length === 0 ? (
              <EmptyState
                title={isId ? 'Belum Ada Snapshot GL' : 'No GL Snapshots Yet'}
                description={
                  isId
                    ? 'Posting jurnal di modul Buku Besar atau sinkronkan laporan snapshot BI.'
                    : 'Post journal entries in General Ledger or refresh BI balance snapshots.'
                }
                action={
                  <Button size="sm" variant="outline" onClick={() => navigate({ to: '/journals' })}>
                    {isId ? 'Buka Jurnal GL' : 'Open GL Journals'}
                  </Button>
                }
              />
            ) : showGlTable ? (
              <div className="overflow-x-auto">
                <table
                  className="w-full border-collapse text-left text-xs"
                  aria-label={
                    isId ? 'Tabel Ringkasan Saldo GL per Kategori' : 'GL Balance Category Summary Table'
                  }
                >
                  <thead>
                    <tr className="border-b border-border text-slate-500 dark:text-slate-400">
                      <th scope="col" className="py-2 pr-3 font-semibold">
                        {isId ? 'Kategori' : 'Category'}
                      </th>
                      <th scope="col" className="py-2 px-3 text-right font-semibold">
                        {isId ? 'Akun' : 'Accounts'}
                      </th>
                      <th scope="col" className="py-2 pl-3 text-right font-semibold">
                        {isId ? 'Total Saldo' : 'Closing Balance'}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {glSummary.categorySeries.map((row) => (
                      <tr
                        key={row.category}
                        className="border-b border-border-subtle last:border-b-0 dark:border-slate-800/70"
                      >
                        <td className="py-2.5 pr-3 font-medium text-slate-800 dark:text-slate-200">
                          <span className="inline-flex items-center gap-2">
                            <span
                              className="h-2.5 w-2.5 rounded-full"
                              style={{ backgroundColor: row.color }}
                              aria-hidden="true"
                            />
                            {isId ? row.labelId : row.labelEn}
                          </span>
                        </td>
                        <td className="py-2.5 px-3 text-right font-money text-slate-600 dark:text-slate-400">
                          {row.accountCount}
                        </td>
                        <td className="py-2.5 pl-3 text-right font-money font-semibold text-slate-900 dark:text-slate-100">
                          {formatCurrency(row.amount, baseCurrency, locale)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="h-[252px] w-full" aria-label={isId ? 'Grafik Saldo GL' : 'GL Balance Chart'}>
                <ResponsiveContainer width="100%" height={252} minWidth={220}>
                  <BarChart
                    data={glSummary.categorySeries}
                    margin={{ top: 10, right: 12, left: 4, bottom: 4 }}
                  >
                    <defs>
                      {Object.entries(BAR_GRADIENTS).map(([cat, grad]) => (
                        <linearGradient key={cat} id={`glBarGrad-${cat}`} x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor={grad.top} stopOpacity={0.96} />
                          <stop offset="100%" stopColor={grad.bottom} stopOpacity={1} />
                        </linearGradient>
                      ))}
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" opacity={0.35} vertical={false} />
                    <XAxis
                      dataKey={isId ? 'labelId' : 'labelEn'}
                      tick={{ fontSize: 11, fill: '#64748b', fontWeight: 500 }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fontSize: 11, fill: '#64748b' }}
                      axisLine={false}
                      tickLine={false}
                      width={56}
                    />
                    <Tooltip
                      cursor={{ fill: 'rgba(148, 163, 184, 0.1)' }}
                      content={<GlCategoryTooltip currency={baseCurrency} locale={locale} />}
                    />
                    <Bar dataKey="chartValue" radius={[7, 7, 0, 0]} maxBarSize={52}>
                      {glSummary.categorySeries.map((entry) => (
                        <Cell
                          key={entry.category}
                          fill={`url(#glBarGrad-${entry.category in BAR_GRADIENTS ? entry.category : 'Other'})`}
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
          </div>
        </article>

        {/* Chart 2: Sales Trend & Inventory Overview */}
        <article className="rounded-xl border border-border bg-surface p-5 shadow-[var(--shadow-card)]">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="flex items-center gap-2 text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100">
                <Boxes className="h-4 w-4 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
                {opsChartMode === 'sales'
                  ? isId
                    ? 'Tren Penjualan per Periode'
                    : 'Sales Revenue by Period'
                  : isId
                    ? 'Posisi Stok per Gudang'
                    : 'Inventory Position by Warehouse'}
              </h2>
              <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                {opsChartMode === 'sales'
                  ? isId
                    ? 'Pendapatan penjualan live dari /sales-summaries'
                    : 'Live sales summary trend from /sales-summaries'
                  : isId
                    ? 'Perbandingan stok fisik vs tersedia dari /inventory-positions'
                    : 'On-hand vs available stock from /inventory-positions'}
              </p>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="inline-flex items-center rounded-lg border border-border bg-surface-muted p-0.5">
                <button
                  type="button"
                  onClick={() => setOpsChartMode('sales')}
                  className={`rounded-md px-2.5 py-1 text-[11px] font-semibold transition-colors cursor-pointer ${
                    opsChartMode === 'sales'
                      ? 'bg-surface text-slate-900 shadow-2xs dark:text-slate-100'
                      : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'
                  }`}
                >
                  {isId ? 'Penjualan' : 'Sales'}
                </button>
                <button
                  type="button"
                  onClick={() => setOpsChartMode('inventory')}
                  className={`rounded-md px-2.5 py-1 text-[11px] font-semibold transition-colors cursor-pointer ${
                    opsChartMode === 'inventory'
                      ? 'bg-surface text-slate-900 shadow-2xs dark:text-slate-100'
                      : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'
                  }`}
                >
                  {isId ? 'Persediaan' : 'Inventory'}
                </button>
              </div>
              <Button
                variant="outline"
                size="xs"
                aria-pressed={showOpsTable}
                onClick={() => setShowOpsTable((prev) => !prev)}
                leftIcon={<Table2 className="h-3.5 w-3.5" />}
              >
                {showOpsTable
                  ? isId
                    ? 'Grafik'
                    : 'Chart'
                  : isId
                    ? 'Tabel'
                    : 'Table'}
              </Button>
            </div>
          </div>

          <div className="mt-4">
            {isReportingLoading ? (
              <CardSkeleton />
            ) : !canReadReporting ? (
              <EmptyState
                title={isId ? 'Izin Pelaporan Diperlukan' : 'Reporting Permission Required'}
                description={
                  isId
                    ? 'Akun Anda tidak memiliki izin reporting.read untuk melihat metrik penjualan & stok.'
                    : 'Your role does not have reporting.read permission for sales & inventory analytics.'
                }
              />
            ) : opsChartMode === 'sales' ? (
              salesInvSummary.salesSeries.length === 0 ? (
                <EmptyState
                  title={isId ? 'Belum Ada Data Ringkasan Penjualan' : 'No Sales Summaries Yet'}
                  description={
                    isId
                      ? 'Ringkasan penjualan akan muncul setelah pesanan penjualan disinkronkan ke modul BI.'
                      : 'Sales summaries appear once sales orders synchronize to BI reporting.'
                  }
                />
              ) : showOpsTable ? (
                <div className="overflow-x-auto">
                  <table
                    className="w-full border-collapse text-left text-xs"
                    aria-label={isId ? 'Tabel Ringkasan Penjualan' : 'Sales Summary Table'}
                  >
                    <thead>
                      <tr className="border-b border-border text-slate-500 dark:text-slate-400">
                        <th scope="col" className="py-2 pr-3 font-semibold">
                          {isId ? 'Periode' : 'Period'}
                        </th>
                        <th scope="col" className="py-2 px-3 text-right font-semibold">
                          {isId ? 'Pesanan' : 'Orders'}
                        </th>
                        <th scope="col" className="py-2 pl-3 text-right font-semibold">
                          {isId ? 'Total Penjualan' : 'Total Sales'}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {salesInvSummary.salesSeries.map((row) => (
                        <tr
                          key={row.periodKey}
                          className="border-b border-border-subtle last:border-b-0 dark:border-slate-800/70"
                        >
                          <td className="py-2.5 pr-3 font-mono font-medium text-slate-800 dark:text-slate-200">
                            {row.periodKey}
                          </td>
                          <td className="py-2.5 px-3 text-right font-money text-slate-600 dark:text-slate-400">
                            {row.orderCount}
                          </td>
                          <td className="py-2.5 pl-3 text-right font-money font-semibold text-slate-900 dark:text-slate-100">
                            {formatCurrency(row.totalSalesAmount, baseCurrency, locale)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="h-[252px] w-full">
                  <ResponsiveContainer width="100%" height={252} minWidth={220}>
                    <AreaChart
                      data={salesInvSummary.salesSeries}
                      margin={{ top: 10, right: 12, left: 4, bottom: 4 }}
                    >
                      <defs>
                        <linearGradient id="erpSalesAreaGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="#14b8a6" stopOpacity={0.28} />
                          <stop offset="100%" stopColor="#14b8a6" stopOpacity={0.01} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" opacity={0.35} vertical={false} />
                      <XAxis
                        dataKey="periodKey"
                        tick={{ fontSize: 11, fill: '#64748b' }}
                        axisLine={false}
                        tickLine={false}
                      />
                      <YAxis
                        tick={{ fontSize: 11, fill: '#64748b' }}
                        axisLine={false}
                        tickLine={false}
                        width={56}
                      />
                      <Tooltip
                        content={<SalesTooltip currency={baseCurrency} locale={locale} />}
                      />
                      <Area
                        type="monotone"
                        dataKey="chartSales"
                        stroke="#0f766e"
                        strokeWidth={2.5}
                        activeDot={{ r: 5, fill: '#0f766e', stroke: '#ffffff', strokeWidth: 2 }}
                        fill="url(#erpSalesAreaGrad)"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              )
            ) : salesInvSummary.warehouseSeries.length === 0 ? (
              <EmptyState
                title={isId ? 'Belum Ada Posisi Persediaan' : 'No Inventory Positions Yet'}
                description={
                  isId
                    ? 'Data stok per gudang akan tampil setelah snapshot persediaan tersedia.'
                    : 'Warehouse stock balances appear once inventory snapshots are refreshed.'
                }
              />
            ) : showOpsTable ? (
              <div className="overflow-x-auto">
                <table
                  className="w-full border-collapse text-left text-xs"
                  aria-label={isId ? 'Tabel Stok per Gudang' : 'Warehouse Stock Summary Table'}
                >
                  <thead>
                    <tr className="border-b border-border text-slate-500 dark:text-slate-400">
                      <th scope="col" className="py-2 pr-3 font-semibold">
                        {isId ? 'Gudang' : 'Warehouse'}
                      </th>
                      <th scope="col" className="py-2 px-3 text-right font-semibold">
                        {isId ? 'Stok Fisik' : 'On Hand'}
                      </th>
                      <th scope="col" className="py-2 pl-3 text-right font-semibold">
                        {isId ? 'Tersedia' : 'Available'}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {salesInvSummary.warehouseSeries.map((row) => (
                      <tr
                        key={row.warehouseCode}
                        className="border-b border-border-subtle last:border-b-0 dark:border-slate-800/70"
                      >
                        <td className="py-2.5 pr-3 font-mono font-medium text-slate-800 dark:text-slate-200">
                          {row.warehouseCode}
                        </td>
                        <td className="py-2.5 px-3 text-right font-money text-slate-700 dark:text-slate-300">
                          {formatNumber(row.onHandQty, 2, locale)}
                        </td>
                        <td className="py-2.5 pl-3 text-right font-money font-semibold text-emerald-700 dark:text-emerald-400">
                          {formatNumber(row.availableQty, 2, locale)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="h-[252px] w-full">
                <ResponsiveContainer width="100%" height={252} minWidth={220}>
                  <BarChart
                    data={salesInvSummary.warehouseSeries}
                    margin={{ top: 10, right: 12, left: 4, bottom: 4 }}
                  >
                    <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" opacity={0.35} vertical={false} />
                    <XAxis
                      dataKey="warehouseCode"
                      tick={{ fontSize: 11, fill: '#64748b' }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fontSize: 11, fill: '#64748b' }}
                      axisLine={false}
                      tickLine={false}
                      width={52}
                    />
                    <Tooltip content={<InventoryTooltip locale={locale} />} />
                    <Bar
                      dataKey="chartOnHand"
                      name={isId ? 'Stok Fisik' : 'On Hand'}
                      fill="#0284c7"
                      radius={[5, 5, 0, 0]}
                    />
                    <Bar
                      dataKey="chartAvailable"
                      name={isId ? 'Tersedia' : 'Available'}
                      fill="#0f766e"
                      radius={[5, 5, 0, 0]}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
          </div>
        </article>
      </section>

      {/* Bottom Row: Pending Workflow Approvals + Quick Actions & Sync Status */}
      <section className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* Pending Workflow Approvals Widget */}
        <article className="lg:col-span-2 rounded-xl border border-border bg-surface p-5 shadow-[var(--shadow-card)]">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-2.5">
              <h2 className="flex items-center gap-2 text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100">
                <CheckSquare className="h-4 w-4 text-amber-600 dark:text-amber-400" aria-hidden="true" />
                {isId ? 'Antrean Persetujuan Workflow Tertunda' : 'Pending Workflow Approvals'}
              </h2>
              {canReadWorkflow ? (
                <StatusBadge
                  status="PENDING"
                  label={`${pendingTasks.length} ${isId ? 'Menunggu' : 'Pending'}`}
                />
              ) : null}
            </div>
            <Link
              to={'/workflow' as never}
              className="inline-flex items-center gap-1 text-xs font-semibold text-brand-700 hover:text-brand-800 dark:text-brand-400 dark:hover:text-brand-300"
            >
              {isId ? 'Buka Antrean Otorisasi' : 'Open Workflow Queue'}
              <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
            </Link>
          </div>

          <div className="mt-4">
            {!canReadWorkflow ? (
              <EmptyState
                title={isId ? 'Izin Workflow Diperlukan' : 'Workflow Permission Required'}
                description={
                  isId
                    ? 'Anda memerlukan izin workflow:read untuk melihat tugas persetujuan.'
                    : 'Requires workflow:read permission to list pending tasks.'
                }
              />
            ) : workflowTasksQuery.isLoading ? (
              <CardSkeleton />
            ) : workflowTasksQuery.error ? (
              <ErrorState
                error={workflowTasksQuery.error}
                title={isId ? 'Gagal Memuat Antrean Persetujuan' : 'Failed to Load Approval Tasks'}
                onRetry={() => void workflowTasksQuery.refetch()}
              />
            ) : pendingTasks.length === 0 ? (
              <EmptyState
                title={isId ? 'Tidak Ada Persetujuan Tertunda' : 'No Pending Approvals'}
                description={
                  isId
                    ? 'Semua dokumen persetujuan untuk entitas ini telah diproses.'
                    : 'All approval workflow tasks for this entity have been processed.'
                }
              />
            ) : (
              <ul className="space-y-2.5">
                {pendingTasks.slice(0, 6).map((task) => (
                  <li
                    key={task.id}
                    className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border-subtle bg-surface-alt/70 dark:border-slate-800 dark:bg-slate-800/40 px-4 py-3 transition-colors hover:border-border"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-brand-200 bg-brand-50 text-brand-700 dark:border-brand-900 dark:bg-brand-950/60 dark:text-brand-400">
                        <CheckSquare className="h-4 w-4" aria-hidden="true" />
                      </span>
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono text-xs sm:text-sm font-bold text-slate-900 dark:text-slate-100">
                            {task.documentRef}
                          </span>
                          <StatusBadge status={task.instanceStatus || 'PENDING'} />
                        </div>
                        <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400 truncate">
                          {isId ? 'Pengaju:' : 'Requested by:'}{' '}
                          <span className="font-medium text-slate-700 dark:text-slate-300">
                            {task.createdBy}
                          </span>{' '}
                          · {isId ? 'Otorisator:' : 'Assignee:'}{' '}
                          <span className="font-mono">{task.assigneeId}</span>
                        </p>
                      </div>
                    </div>

                    <Link
                      to={'/workflow' as never}
                      className="inline-flex h-8 items-center gap-1 rounded-lg border border-border bg-surface px-3 text-xs font-semibold text-slate-700 shadow-2xs hover:bg-surface-muted dark:text-slate-200"
                    >
                      {isId ? 'Tinjau' : 'Review'}
                      <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </article>

        {/* Right Column: Quick Actions + Synchronization Card */}
        <aside className="space-y-4">
          <div className="rounded-xl border border-border bg-surface p-4.5 shadow-[var(--shadow-card)]">
            <h2 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              {isId ? 'Aksi Cepat & Pintasan Modul' : 'Quick Actions & Module Shortcuts'}
            </h2>
            <div className="mt-3 space-y-2">
              <button
                type="button"
                onClick={() => (canCreateGl ? setJournalFormOpen(true) : navigate({ to: '/journals' }))}
                className="flex w-full items-center justify-between gap-3 rounded-xl border border-border-subtle bg-surface-alt/80 dark:border-slate-800 dark:bg-slate-800/40 px-3 py-2.5 text-left transition-colors hover:border-brand-300 hover:bg-brand-50/40 dark:hover:bg-slate-800 cursor-pointer"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-brand-200 bg-brand-50 text-brand-700 dark:border-brand-900 dark:bg-brand-950/60 dark:text-brand-400">
                    <BookOpen className="h-3.5 w-3.5" />
                  </span>
                  <div className="min-w-0">
                    <p className="text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">
                      {isId ? 'Entri Jurnal GL' : 'GL Journal Entry'}
                    </p>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {isId ? 'Posting multi-valas ADR-014' : 'Multi-currency ADR-014 posting'}
                    </p>
                  </div>
                </div>
                <ChevronRight className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              </button>

              <button
                type="button"
                onClick={() => navigate({ to: '/coa' as never })}
                className="flex w-full items-center justify-between gap-3 rounded-xl border border-border-subtle bg-surface-alt/80 dark:border-slate-800 dark:bg-slate-800/40 px-3 py-2.5 text-left transition-colors hover:border-brand-300 hover:bg-brand-50/40 dark:hover:bg-slate-800 cursor-pointer"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-900 dark:bg-sky-950/60 dark:text-sky-400">
                    <FolderTree className="h-3.5 w-3.5" />
                  </span>
                  <div className="min-w-0">
                    <p className="text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">
                      {isId ? 'Bagan Akun & Periode' : 'Chart of Accounts & Periods'}
                    </p>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {isId ? 'Struktur akun & tutup buku' : 'Account tree & fiscal periods'}
                    </p>
                  </div>
                </div>
                <ChevronRight className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              </button>

              <button
                type="button"
                onClick={() => navigate({ to: '/currency' as never })}
                className="flex w-full items-center justify-between gap-3 rounded-xl border border-border-subtle bg-surface-alt/80 dark:border-slate-800 dark:bg-slate-800/40 px-3 py-2.5 text-left transition-colors hover:border-brand-300 hover:bg-brand-50/40 dark:hover:bg-slate-800 cursor-pointer"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/60 dark:text-amber-400">
                    <Coins className="h-3.5 w-3.5" />
                  </span>
                  <div className="min-w-0">
                    <p className="text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">
                      {isId ? 'Mata Uang & Kurs FX' : 'Currencies & FX Rates'}
                    </p>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {activeCurrencyCodes.join(' · ')}
                    </p>
                  </div>
                </div>
                <ChevronRight className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              </button>

              <button
                type="button"
                onClick={() => navigate({ to: '/reports' as never })}
                className="flex w-full items-center justify-between gap-3 rounded-xl border border-border-subtle bg-surface-alt/80 dark:border-slate-800 dark:bg-slate-800/40 px-3 py-2.5 text-left transition-colors hover:border-brand-300 hover:bg-brand-50/40 dark:hover:bg-slate-800 cursor-pointer"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-900 dark:bg-emerald-950/60 dark:text-emerald-400">
                    <FileSpreadsheet className="h-3.5 w-3.5" />
                  </span>
                  <div className="min-w-0">
                    <p className="text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">
                      {isId ? 'Laporan BI & Ekspor' : 'BI Reporting & Exports'}
                    </p>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {isId ? 'Neraca, Laba Rugi, Neraca Saldo' : 'Balance Sheet, P&L, Trial Balance'}
                    </p>
                  </div>
                </div>
                <ChevronRight className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              </button>
            </div>
          </div>

          <div className="rounded-xl border border-border bg-surface p-4.5 shadow-[var(--shadow-card)] space-y-2.5">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-xs font-bold text-slate-900 dark:text-slate-100">
                {isId ? 'Status Sinkronisasi & BI' : 'Synchronization & BI Status'}
              </h3>
              <StatusBadge
                status={isRealtimeConnected ? 'POSTED' : 'OPEN'}
                tone={isRealtimeConnected ? 'success' : 'neutral'}
                label={isRealtimeConnected ? (isId ? 'Tersinkron' : 'Synced') : 'Polling'}
              />
            </div>

            <dl className="space-y-1.5 text-xs">
              <div className="flex items-center justify-between gap-2">
                <dt className="text-slate-500 dark:text-slate-400">
                  {isId ? 'Snapshot BI Terakhir' : 'Latest BI Snapshot'}
                </dt>
                <dd className="font-money font-semibold text-slate-800 dark:text-slate-200">
                  {lastRefreshedIso
                    ? formatDateTime(lastRefreshedIso, locale)
                    : isId
                      ? 'Menunggu'
                      : 'Pending'}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-2">
                <dt className="text-slate-500 dark:text-slate-400">
                  {isId ? 'Entri Jurnal Draf' : 'Draft Journal Queue'}
                </dt>
                <dd className="font-money font-semibold text-slate-800 dark:text-slate-200">
                  {journalStats.draftCount} {isId ? 'draf' : 'drafts'}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-2">
                <dt className="text-slate-500 dark:text-slate-400">
                  {isId ? 'Mata Uang Entitas' : 'Entity Currency'}
                </dt>
                <dd className="font-mono font-semibold text-slate-800 dark:text-slate-200">
                  {baseCurrency} ({activeCurrencyCodes.length} {isId ? 'aktif' : 'active'})
                </dd>
              </div>
            </dl>
          </div>
        </aside>
      </section>

      <JournalFormDialog
        open={journalFormOpen}
        onOpenChange={setJournalFormOpen}
      />
    </div>
  );
}
