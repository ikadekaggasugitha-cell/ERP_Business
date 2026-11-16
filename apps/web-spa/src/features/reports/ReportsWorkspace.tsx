import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  BarChart3,
  Boxes,
  Download,
  FileJson,
  FileSpreadsheet,
  Printer,
  Scale,
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
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { Button } from '../../components/ui/Button';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select, type SelectOption } from '../../components/ui/Select';
import { CardSkeleton } from '../../components/ui/Skeleton';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { apiClient, unwrapListResponse } from '../../lib/api';
import { formatCurrency, formatDateTime, formatNumber } from '../../lib/decimal';
import { exportToCsv, exportToJson, triggerPrint } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { ListEnvelope } from '../../types';
import {
  summarizeGlReport,
  summarizeInventoryReport,
  summarizeSalesReport,
  summarizeTrialBalanceReport,
  type EnrichedTrialBalanceItem,
  type GlBalanceSnapshotDto,
  type GlCategoryReportDatum,
  type InventoryPositionDto,
  type ReportAccountLookupDto,
  type ReportFiscalPeriodLookupDto,
  type ReportLedgerLookupDto,
  type ReportsTrialBalanceDto,
  type SalesSummaryDto,
} from './reports-types';

type ReportTab = 'gl-snapshots' | 'sales-summaries' | 'inventory-positions' | 'trial-balance';

interface ChartTooltipEntry<T> {
  payload?: T;
}

function GlReportTooltip({
  active,
  payload,
  currency,
  locale,
}: {
  active?: boolean;
  payload?: Array<ChartTooltipEntry<GlCategoryReportDatum>>;
  currency: string;
  locale: 'id' | 'en';
}) {
  if (!active || !payload?.length || !payload[0]?.payload) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-slate-700 dark:bg-slate-900">
      <p className="font-semibold text-slate-900 dark:text-slate-100">{item.category}</p>
      <p className="mt-1 font-money font-bold text-slate-800 dark:text-slate-200">
        {formatCurrency(item.closingTotal, currency, locale)}
      </p>
      <p className="text-[11px] text-slate-500">{item.count} accounts</p>
    </div>
  );
}

function SalesReportTooltip({
  active,
  payload,
  currency,
  locale,
}: {
  active?: boolean;
  payload?: Array<ChartTooltipEntry<{ period: string; revenue: string; orders: number }>>;
  currency: string;
  locale: 'id' | 'en';
}) {
  if (!active || !payload?.length || !payload[0]?.payload) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-slate-700 dark:bg-slate-900">
      <p className="font-mono font-semibold text-slate-900 dark:text-slate-100">{item.period}</p>
      <p className="mt-1 font-money font-bold text-emerald-700 dark:text-emerald-400">
        {formatCurrency(item.revenue, currency, locale)}
      </p>
      <p className="text-[11px] text-slate-500">
        {item.orders} {locale === 'id' ? 'pesanan' : 'orders'}
      </p>
    </div>
  );
}

function InventoryReportTooltip({
  active,
  payload,
  locale,
}: {
  active?: boolean;
  payload?: Array<
    ChartTooltipEntry<{ warehouseCode: string; onHand: string; available: string }>
  >;
  locale: 'id' | 'en';
}) {
  if (!active || !payload?.length || !payload[0]?.payload) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-slate-700 dark:bg-slate-900">
      <p className="font-mono font-semibold text-slate-900 dark:text-slate-100">
        {item.warehouseCode}
      </p>
      <p className="mt-1 text-slate-700 dark:text-slate-300">
        On Hand: <span className="font-money font-semibold">{formatNumber(item.onHand, 2, locale)}</span>
      </p>
      <p className="text-emerald-700 dark:text-emerald-400">
        Available:{' '}
        <span className="font-money font-semibold">{formatNumber(item.available, 2, locale)}</span>
      </p>
    </div>
  );
}

export function ReportsWorkspace() {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const user = useAuthStore((s) => s.user);
  const orgId = activeOrg?.id ?? 'default';
  const baseCurrency = activeOrg?.functionalCurrency || 'IDR';

  const [activeTab, setActiveTab] = useState<ReportTab>('gl-snapshots');
  const [glSearch, setGlSearch] = useState('');
  const [salesSearch, setSalesSearch] = useState('');
  const [salesYearFilter, setSalesYearFilter] = useState('ALL');
  const [invSearch, setInvSearch] = useState('');
  const [invWarehouseFilter, setInvWarehouseFilter] = useState('ALL');
  const [selectedTbLedgerId, setSelectedTbLedgerId] = useState('');
  const [selectedTbPeriodId, setSelectedTbPeriodId] = useState('');
  const [showChartTable, setShowChartTable] = useState(false);

  const canReadReporting = hasUserPermission(user, ['reporting.read', 'reporting:read']);
  const canReadGl = hasUserPermission(user, ['gl:read', 'gl.read', 'gl.*']);

  const glQuery = useQuery({
    queryKey: ['gl-balances', 'reports', orgId],
    enabled: canReadReporting,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<GlBalanceSnapshotDto> | GlBalanceSnapshotDto[]>(
        '/gl-balance-snapshots',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const salesQuery = useQuery({
    queryKey: ['balances', 'sales-summaries', 'reports', orgId],
    enabled: canReadReporting,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<SalesSummaryDto> | SalesSummaryDto[]>(
        '/sales-summaries',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const inventoryQuery = useQuery({
    queryKey: ['balances', 'inventory-positions', 'reports', orgId],
    enabled: canReadReporting,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<InventoryPositionDto> | InventoryPositionDto[]>(
        '/inventory-positions',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const ledgersQuery = useQuery({
    queryKey: ['ledgers', 'reports-lookup', orgId],
    enabled: canReadGl,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<ReportLedgerLookupDto> | ReportLedgerLookupDto[]>(
        '/ledgers',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const periodsQuery = useQuery({
    queryKey: ['fiscal-periods', 'reports-lookup', orgId],
    enabled: canReadGl,
    queryFn: async () => {
      const res = await apiClient<
        ListEnvelope<ReportFiscalPeriodLookupDto> | ReportFiscalPeriodLookupDto[]
      >('/fiscal-periods', { query: { page: 1, size: 100 } });
      return unwrapListResponse(res).items;
    },
  });

  const accountsQuery = useQuery({
    queryKey: ['coa-accounts', 'reports-lookup', orgId],
    enabled: canReadGl,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<ReportAccountLookupDto> | ReportAccountLookupDto[]>(
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
    if (!selectedTbLedgerId && ledgers.length > 0) setSelectedTbLedgerId(ledgers[0].id);
  }, [selectedTbLedgerId, ledgers]);

  useEffect(() => {
    if (!selectedTbPeriodId && periods.length > 0) {
      const openPeriod =
        periods.find((p) => p.status.trim().toUpperCase() === 'OPEN') ?? periods[0];
      setSelectedTbPeriodId(openPeriod.id);
    }
  }, [selectedTbPeriodId, periods]);

  const trialBalanceQuery = useQuery({
    queryKey: ['gl-balances', 'reports-tb', orgId, selectedTbLedgerId, selectedTbPeriodId],
    enabled: canReadGl && Boolean(selectedTbLedgerId && selectedTbPeriodId),
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<ReportsTrialBalanceDto> | ReportsTrialBalanceDto[]>(
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
      return unwrapListResponse(res).items;
    },
  });

  const glRows = useMemo(() => {
    const raw = glQuery.data ?? [];
    const term = glSearch.trim().toLowerCase();
    if (!term) return raw;
    return raw.filter(
      (r) =>
        r.accountCode.toLowerCase().includes(term) ||
        r.fiscalPeriodId.toLowerCase().includes(term)
    );
  }, [glQuery.data, glSearch]);

  const salesRows = useMemo(() => {
    const raw = salesQuery.data ?? [];
    const term = salesSearch.trim().toLowerCase();
    return raw.filter((r) => {
      if (salesYearFilter !== 'ALL' && String(r.periodYear) !== salesYearFilter) return false;
      if (!term) return true;
      return r.customerId.toLowerCase().includes(term);
    });
  }, [salesQuery.data, salesSearch, salesYearFilter]);

  const inventoryRows = useMemo(() => {
    const raw = inventoryQuery.data ?? [];
    const term = invSearch.trim().toLowerCase();
    return raw.filter((r) => {
      if (invWarehouseFilter !== 'ALL' && r.warehouseCode !== invWarehouseFilter) return false;
      if (!term) return true;
      return (
        r.itemCode.toLowerCase().includes(term) ||
        r.warehouseCode.toLowerCase().includes(term)
      );
    });
  }, [inventoryQuery.data, invSearch, invWarehouseFilter]);

  const accountsById = useMemo(() => {
    const map = new Map<string, ReportAccountLookupDto>();
    for (const acc of accounts) map.set(acc.id, acc);
    return map;
  }, [accounts]);

  const enrichedTbRows: EnrichedTrialBalanceItem[] = useMemo(
    () =>
      (trialBalanceQuery.data ?? []).map((row) => {
        const acc = accountsById.get(row.accountId);
        return {
          ...row,
          accountCode: acc?.code ?? row.accountId.slice(0, 8),
          accountName: acc?.name ?? (isId ? 'Akun Tidak Ditemukan' : 'Unknown Account'),
          accountType: acc?.accountType ?? '—',
        };
      }),
    [trialBalanceQuery.data, accountsById, isId]
  );

  const glMetrics = useMemo(() => summarizeGlReport(glRows), [glRows]);
  const salesMetrics = useMemo(() => summarizeSalesReport(salesRows), [salesRows]);
  const inventoryMetrics = useMemo(() => summarizeInventoryReport(inventoryRows), [inventoryRows]);
  const tbMetrics = useMemo(() => summarizeTrialBalanceReport(enrichedTbRows), [enrichedTbRows]);

  const salesYearOptions: SelectOption[] = useMemo(() => {
    const years = Array.from(new Set((salesQuery.data ?? []).map((r) => r.periodYear))).sort(
      (a, b) => b - a
    );
    return [
      { value: 'ALL', label: isId ? 'Semua Tahun' : 'All Years' },
      ...years.map((y) => ({ value: String(y), label: String(y) })),
    ];
  }, [salesQuery.data, isId]);

  const warehouseOptions: SelectOption[] = useMemo(() => {
    const codes = Array.from(
      new Set((inventoryQuery.data ?? []).map((r) => r.warehouseCode))
    ).sort();
    return [
      { value: 'ALL', label: isId ? 'Semua Gudang' : 'All Warehouses' },
      ...codes.map((w) => ({ value: w, label: w })),
    ];
  }, [inventoryQuery.data, isId]);

  const handleExportActiveCsv = () => {
    if (activeTab === 'gl-snapshots') {
      exportToCsv(
        glRows,
        [
          { header: 'Account Code', accessor: (r) => r.accountCode },
          { header: 'Fiscal Period ID', accessor: (r) => r.fiscalPeriodId },
          { header: 'Closing Balance', accessor: (r) => r.closingBalance },
          { header: 'Refreshed At', accessor: (r) => r.refreshedAt },
        ],
        'gl-balance-snapshots.csv'
      );
      return;
    }
    if (activeTab === 'sales-summaries') {
      exportToCsv(
        salesRows,
        [
          { header: 'Year', accessor: (r) => r.periodYear },
          { header: 'Month', accessor: (r) => r.periodMonth },
          { header: 'Customer ID', accessor: (r) => r.customerId },
          { header: 'Order Count', accessor: (r) => r.orderCount },
          { header: 'Total Sales Amount', accessor: (r) => r.totalSalesAmount },
          { header: 'Refreshed At', accessor: (r) => r.refreshedAt },
        ],
        'sales-summaries.csv'
      );
      return;
    }
    if (activeTab === 'inventory-positions') {
      exportToCsv(
        inventoryRows,
        [
          { header: 'Warehouse Code', accessor: (r) => r.warehouseCode },
          { header: 'Item Code', accessor: (r) => r.itemCode },
          { header: 'On Hand Qty', accessor: (r) => r.onHandQty },
          { header: 'Available Qty', accessor: (r) => r.availableQty },
          { header: 'Refreshed At', accessor: (r) => r.refreshedAt },
        ],
        'inventory-positions.csv'
      );
      return;
    }
    exportToCsv(
      enrichedTbRows,
      [
        { header: 'Account Code', accessor: (r) => r.accountCode },
        { header: 'Account Name', accessor: (r) => r.accountName },
        { header: 'Opening Balance', accessor: (r) => r.openingBalance },
        { header: 'Period Debit', accessor: (r) => r.periodDebit },
        { header: 'Period Credit', accessor: (r) => r.periodCredit },
        { header: 'Closing Balance', accessor: (r) => r.closingBalance },
      ],
      'trial-balance-report.csv'
    );
  };

  const handleExportActiveJson = () => {
    if (activeTab === 'gl-snapshots') {
      exportToJson(glRows, 'gl-balance-snapshots.json');
      return;
    }
    if (activeTab === 'sales-summaries') {
      exportToJson(salesRows, 'sales-summaries.json');
      return;
    }
    if (activeTab === 'inventory-positions') {
      exportToJson(inventoryRows, 'inventory-positions.json');
      return;
    }
    exportToJson(enrichedTbRows, 'trial-balance-report.json');
  };

  const glColumns: ColumnDef<GlBalanceSnapshotDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Kode Akun' : 'Account Code',
        width: '150px',
        cell: (r) => <span className="font-mono font-semibold">{r.accountCode}</span>,
      },
      {
        header: isId ? 'ID Periode Fiskal' : 'Fiscal Period ID',
        width: '260px',
        cell: (r) => <span className="font-mono text-xs text-slate-600 dark:text-slate-400">{r.fiscalPeriodId}</span>,
      },
      {
        header: isId ? 'Saldo Akhir' : 'Closing Balance',
        align: 'right',
        width: '180px',
        cell: (r) => (
          <span className="font-money font-semibold text-slate-900 dark:text-slate-100">
            {formatCurrency(r.closingBalance, baseCurrency, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Diperbarui' : 'Refreshed At',
        width: '180px',
        cell: (r) => (
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {formatDateTime(r.refreshedAt, locale)}
          </span>
        ),
      },
    ],
    [baseCurrency, isId, locale]
  );

  const salesColumns: ColumnDef<SalesSummaryDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Periode' : 'Period',
        width: '120px',
        cell: (r) => (
          <span className="font-mono font-semibold">
            {r.periodYear}-{String(r.periodMonth).padStart(2, '0')}
          </span>
        ),
      },
      {
        header: isId ? 'ID Pelanggan' : 'Customer ID',
        width: '240px',
        cell: (r) => <span className="font-mono text-xs">{r.customerId}</span>,
      },
      {
        header: isId ? 'Jml Pesanan' : 'Orders',
        align: 'right',
        width: '120px',
        cell: (r) => <span className="font-money">{r.orderCount}</span>,
      },
      {
        header: isId ? 'Total Penjualan' : 'Total Sales',
        align: 'right',
        width: '180px',
        cell: (r) => (
          <span className="font-money font-semibold text-emerald-700 dark:text-emerald-400">
            {formatCurrency(r.totalSalesAmount, baseCurrency, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Diperbarui' : 'Refreshed At',
        width: '180px',
        cell: (r) => (
          <span className="text-xs text-slate-500">{formatDateTime(r.refreshedAt, locale)}</span>
        ),
      },
    ],
    [baseCurrency, isId, locale]
  );

  const inventoryColumns: ColumnDef<InventoryPositionDto>[] = useMemo(
    () => [
      {
        header: isId ? 'Gudang' : 'Warehouse',
        width: '150px',
        cell: (r) => <span className="font-mono font-semibold">{r.warehouseCode}</span>,
      },
      {
        header: isId ? 'Kode Item' : 'Item Code',
        width: '180px',
        cell: (r) => <span className="font-mono">{r.itemCode}</span>,
      },
      {
        header: isId ? 'Stok Fisik' : 'On Hand Qty',
        align: 'right',
        width: '150px',
        cell: (r) => <span className="font-money">{formatNumber(r.onHandQty, 2, locale)}</span>,
      },
      {
        header: isId ? 'Tersedia' : 'Available Qty',
        align: 'right',
        width: '150px',
        cell: (r) => (
          <span className="font-money font-semibold text-emerald-700 dark:text-emerald-400">
            {formatNumber(r.availableQty, 2, locale)}
          </span>
        ),
      },
      {
        header: isId ? 'Diperbarui' : 'Refreshed At',
        width: '180px',
        cell: (r) => (
          <span className="text-xs text-slate-500">{formatDateTime(r.refreshedAt, locale)}</span>
        ),
      },
    ],
    [isId, locale]
  );

  const tbColumns: ColumnDef<EnrichedTrialBalanceItem>[] = useMemo(
    () => [
      {
        header: isId ? 'Kode' : 'Code',
        width: '120px',
        cell: (r) => <span className="font-mono font-semibold">{r.accountCode}</span>,
      },
      {
        header: isId ? 'Nama Akun' : 'Account Name',
        width: '240px',
        cell: (r) => (
          <div>
            <span className="font-medium">{r.accountName}</span>
            <p className="text-[11px] text-slate-500">{r.accountType}</p>
          </div>
        ),
      },
      {
        header: isId ? 'Saldo Awal' : 'Opening',
        align: 'right',
        width: '150px',
        cell: (r) => <span className="font-money">{formatCurrency(r.openingBalance, baseCurrency, locale)}</span>,
      },
      {
        header: 'Debit',
        align: 'right',
        width: '150px',
        cell: (r) => <span className="font-money">{formatCurrency(r.periodDebit, baseCurrency, locale)}</span>,
      },
      {
        header: isId ? 'Kredit' : 'Credit',
        align: 'right',
        width: '150px',
        cell: (r) => <span className="font-money">{formatCurrency(r.periodCredit, baseCurrency, locale)}</span>,
      },
      {
        header: isId ? 'Saldo Akhir' : 'Closing',
        align: 'right',
        width: '160px',
        cell: (r) => (
          <span className="font-money font-bold">{formatCurrency(r.closingBalance, baseCurrency, locale)}</span>
        ),
      },
    ],
    [baseCurrency, isId, locale]
  );

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <Breadcrumbs
            items={[{ label: isId ? 'Laporan Keuangan & BI' : 'BI Financial Reporting' }]}
          />
          <div className="mt-2 flex flex-wrap items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
              {isId ? 'Portal Pelaporan Keuangan & BI' : 'BI Financial Reporting Portal'}
            </h1>
            <StatusBadge status="REPORTED" label="reporting.erp.example" dot={false} />
          </div>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {isId
              ? 'Snapshot Buku Besar, ringkasan penjualan, posisi persediaan gudang, serta neraca saldo siap ekspor.'
              : 'GL balance snapshots, sales summaries, warehouse inventory positions, and exportable trial balance.'}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2 print:hidden">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<Download className="h-3.5 w-3.5" />}
            onClick={handleExportActiveCsv}
          >
            {isId ? 'Ekspor CSV' : 'Export CSV'}
          </Button>
          <Button
            variant="outline"
            size="sm"
            leftIcon={<FileJson className="h-3.5 w-3.5" />}
            onClick={handleExportActiveJson}
          >
            {isId ? 'Ekspor JSON' : 'Export JSON'}
          </Button>
          <Button
            variant="outline"
            size="sm"
            leftIcon={<Printer className="h-3.5 w-3.5" />}
            onClick={() => triggerPrint()}
          >
            {isId ? 'Cetak' : 'Print'}
          </Button>
        </div>
      </header>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as ReportTab)}>
        <TabsList>
          <TabsTrigger value="gl-snapshots">
            {isId ? 'Snapshot Saldo GL' : 'GL Balance Snapshots'}
          </TabsTrigger>
          <TabsTrigger value="sales-summaries">
            {isId ? 'Ringkasan Penjualan' : 'Sales Summaries'}
          </TabsTrigger>
          <TabsTrigger value="inventory-positions">
            {isId ? 'Posisi Persediaan' : 'Inventory Positions'}
          </TabsTrigger>
          <TabsTrigger value="trial-balance">
            {isId ? 'Neraca Saldo' : 'Trial Balance'}
          </TabsTrigger>
        </TabsList>

        {/* Tab 1: GL Balance Snapshots */}
        <TabsContent value="gl-snapshots" className="space-y-4">
          {!canReadReporting ? (
            <ErrorState
              title={isId ? 'Izin Pelaporan Diperlukan' : 'Reporting Permission Required'}
              message={isId ? 'Dibutuhkan izin reporting.read.' : 'Requires reporting.read permission.'}
            />
          ) : (
            <>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <KpiCard
                  label={isId ? 'Total Baris Snapshot' : 'Snapshot Rows'}
                  value={String(glRows.length)}
                  accent="brand"
                  icon={<BarChart3 className="h-5 w-5" />}
                  isLoading={glQuery.isLoading}
                />
                <KpiCard
                  label={isId ? 'Kategori Akun Aktif' : 'Active Categories'}
                  value={String(glMetrics.series.length)}
                  accent="sky"
                  icon={<FileSpreadsheet className="h-5 w-5" />}
                  isLoading={glQuery.isLoading}
                />
                <KpiCard
                  label={isId ? 'Saldo Bersih Snapshot' : 'Net Snapshot Total'}
                  value={formatCurrency(glMetrics.netClosing, baseCurrency, locale)}
                  accent="emerald"
                  icon={<Scale className="h-5 w-5" />}
                  isLoading={glQuery.isLoading}
                />
              </div>

              <article className="rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
                <div className="mb-3 flex items-center justify-between gap-2">
                  <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                    {isId ? 'Distribusi Saldo Snapshot per Kategori' : 'Snapshot Balance by Category'}
                  </h2>
                  <Button
                    variant="outline"
                    size="xs"
                    aria-pressed={showChartTable}
                    leftIcon={<Table2 className="h-3.5 w-3.5" />}
                    onClick={() => setShowChartTable((p) => !p)}
                  >
                    {showChartTable ? (isId ? 'Grafik' : 'Chart') : isId ? 'Tabel' : 'Table'}
                  </Button>
                </div>

                {glQuery.isLoading ? (
                  <CardSkeleton />
                ) : glQuery.error ? (
                  <ErrorState error={glQuery.error} onRetry={() => void glQuery.refetch()} />
                ) : glMetrics.series.length === 0 ? (
                  <EmptyState
                    title={isId ? 'Belum Ada Snapshot GL' : 'No GL Snapshots'}
                    onResetFilters={() => setGlSearch('')}
                  />
                ) : showChartTable ? (
                  <div className="overflow-x-auto">
                    <table className="w-full border-collapse text-left text-xs">
                      <thead>
                        <tr className="border-b border-slate-200 text-slate-500 dark:border-slate-800">
                          <th scope="col" className="py-2 font-semibold">{isId ? 'Kategori' : 'Category'}</th>
                          <th scope="col" className="py-2 text-right font-semibold">{isId ? 'Akun' : 'Count'}</th>
                          <th scope="col" className="py-2 text-right font-semibold">{isId ? 'Total' : 'Total'}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {glMetrics.series.map((s) => (
                          <tr key={s.category} className="border-b border-slate-100 dark:border-slate-800/70">
                            <td className="py-2 font-medium">{s.category}</td>
                            <td className="py-2 text-right font-money">{s.count}</td>
                            <td className="py-2 text-right font-money font-semibold">
                              {formatCurrency(s.closingTotal, baseCurrency, locale)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="h-[240px] w-full">
                    <ResponsiveContainer width="100%" height={240} minWidth={220}>
                      <BarChart data={glMetrics.series}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" opacity={0.45} />
                        <XAxis dataKey="category" tick={{ fontSize: 11, fill: '#64748b' }} />
                        <YAxis tick={{ fontSize: 11, fill: '#64748b' }} width={56} />
                        <Tooltip content={<GlReportTooltip currency={baseCurrency} locale={locale} />} />
                        <Bar dataKey="chartValue" radius={[6, 6, 0, 0]} maxBarSize={48}>
                          {glMetrics.series.map((e) => (
                            <Cell key={e.category} fill={e.color} />
                          ))}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </article>

              <VirtualizedTable
                data={glRows}
                columns={glColumns}
                getRowId={(r) => r.id}
                isLoading={glQuery.isLoading}
                error={glQuery.error}
                onRetry={() => void glQuery.refetch()}
                emptyTitle={isId ? 'Snapshot GL Kosong' : 'No GL Snapshots'}
                onResetFilters={() => setGlSearch('')}
                toolbar={{
                  searchValue: glSearch,
                  onSearchChange: setGlSearch,
                  searchPlaceholder: isId ? 'Cari kode akun atau periode...' : 'Search account code or period...',
                  onExportCsv: handleExportActiveCsv,
                }}
              />
            </>
          )}
        </TabsContent>

        {/* Tab 2: Sales Summaries */}
        <TabsContent value="sales-summaries" className="space-y-4">
          {!canReadReporting ? (
            <ErrorState
              title={isId ? 'Izin Pelaporan Diperlukan' : 'Reporting Permission Required'}
              message={isId ? 'Dibutuhkan izin reporting.read.' : 'Requires reporting.read permission.'}
            />
          ) : (
            <>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <KpiCard
                  label={isId ? 'Total Penjualan BI' : 'Total BI Sales Revenue'}
                  value={formatCurrency(salesMetrics.totalRevenue, baseCurrency, locale)}
                  accent="emerald"
                  icon={<TrendingUp className="h-5 w-5" />}
                  isLoading={salesQuery.isLoading}
                />
                <KpiCard
                  label={isId ? 'Total Pesanan' : 'Total Orders'}
                  value={String(salesMetrics.totalOrders)}
                  accent="brand"
                  icon={<BarChart3 className="h-5 w-5" />}
                  isLoading={salesQuery.isLoading}
                />
                <KpiCard
                  label={isId ? 'Rata-rata Nilai Pesanan' : 'Average Order Value'}
                  value={formatCurrency(salesMetrics.averageOrderValue, baseCurrency, locale)}
                  accent="sky"
                  icon={<Scale className="h-5 w-5" />}
                  isLoading={salesQuery.isLoading}
                />
              </div>

              <article className="rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
                <div className="mb-3 flex items-center justify-between gap-2">
                  <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                    {isId ? 'Tren Penjualan Bulanan' : 'Monthly Sales Revenue Trend'}
                  </h2>
                  <Button
                    variant="outline"
                    size="xs"
                    aria-pressed={showChartTable}
                    leftIcon={<Table2 className="h-3.5 w-3.5" />}
                    onClick={() => setShowChartTable((p) => !p)}
                  >
                    {showChartTable ? (isId ? 'Grafik' : 'Chart') : isId ? 'Tabel' : 'Table'}
                  </Button>
                </div>

                {salesQuery.isLoading ? (
                  <CardSkeleton />
                ) : salesQuery.error ? (
                  <ErrorState error={salesQuery.error} onRetry={() => void salesQuery.refetch()} />
                ) : salesMetrics.monthlySeries.length === 0 ? (
                  <EmptyState
                    title={isId ? 'Belum Ada Data Penjualan' : 'No Sales Data'}
                    onResetFilters={() => {
                      setSalesSearch('');
                      setSalesYearFilter('ALL');
                    }}
                  />
                ) : showChartTable ? (
                  <div className="overflow-x-auto">
                    <table className="w-full border-collapse text-left text-xs">
                      <thead>
                        <tr className="border-b border-slate-200 text-slate-500 dark:border-slate-800">
                          <th scope="col" className="py-2 font-semibold">{isId ? 'Periode' : 'Period'}</th>
                          <th scope="col" className="py-2 text-right font-semibold">{isId ? 'Pesanan' : 'Orders'}</th>
                          <th scope="col" className="py-2 text-right font-semibold">{isId ? 'Pendapatan' : 'Revenue'}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {salesMetrics.monthlySeries.map((m) => (
                          <tr key={m.period} className="border-b border-slate-100 dark:border-slate-800/70">
                            <td className="py-2 font-mono">{m.period}</td>
                            <td className="py-2 text-right font-money">{m.orders}</td>
                            <td className="py-2 text-right font-money font-semibold">
                              {formatCurrency(m.revenue, baseCurrency, locale)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="h-[240px] w-full">
                    <ResponsiveContainer width="100%" height={240} minWidth={220}>
                      <AreaChart data={salesMetrics.monthlySeries}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" opacity={0.45} />
                        <XAxis dataKey="period" tick={{ fontSize: 11, fill: '#64748b' }} />
                        <YAxis tick={{ fontSize: 11, fill: '#64748b' }} width={56} />
                        <Tooltip content={<SalesReportTooltip currency={baseCurrency} locale={locale} />} />
                        <Area
                          type="monotone"
                          dataKey="chartRevenue"
                          stroke="#0f766e"
                          strokeWidth={2}
                          fill="#14b8a6"
                          fillOpacity={0.2}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </article>

              <VirtualizedTable
                data={salesRows}
                columns={salesColumns}
                getRowId={(r) => r.id}
                isLoading={salesQuery.isLoading}
                error={salesQuery.error}
                onRetry={() => void salesQuery.refetch()}
                emptyTitle={isId ? 'Belum Ada Ringkasan Penjualan' : 'No Sales Summaries'}
                onResetFilters={() => {
                  setSalesSearch('');
                  setSalesYearFilter('ALL');
                }}
                toolbar={{
                  searchValue: salesSearch,
                  onSearchChange: setSalesSearch,
                  searchPlaceholder: isId ? 'Cari ID pelanggan...' : 'Search customer ID...',
                  filters: (
                    <div className="w-40">
                      <Select
                        aria-label={isId ? 'Filter Tahun' : 'Filter Year'}
                        value={salesYearFilter}
                        onValueChange={setSalesYearFilter}
                        options={salesYearOptions}
                      />
                    </div>
                  ),
                  onExportCsv: handleExportActiveCsv,
                }}
              />
            </>
          )}
        </TabsContent>

        {/* Tab 3: Inventory Positions */}
        <TabsContent value="inventory-positions" className="space-y-4">
          {!canReadReporting ? (
            <ErrorState
              title={isId ? 'Izin Pelaporan Diperlukan' : 'Reporting Permission Required'}
              message={isId ? 'Dibutuhkan izin reporting.read.' : 'Requires reporting.read permission.'}
            />
          ) : (
            <>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <KpiCard
                  label={isId ? 'Total Stok Fisik' : 'Total On-Hand Qty'}
                  value={formatNumber(inventoryMetrics.totalOnHand, 2, locale)}
                  subtitle={`${inventoryMetrics.skuCount} ${isId ? 'baris SKU' : 'SKU positions'}`}
                  accent="sky"
                  icon={<Boxes className="h-5 w-5" />}
                  isLoading={inventoryQuery.isLoading}
                />
                <KpiCard
                  label={isId ? 'Total Stok Tersedia' : 'Total Available Qty'}
                  value={formatNumber(inventoryMetrics.totalAvailable, 2, locale)}
                  accent="emerald"
                  icon={<Boxes className="h-5 w-5" />}
                  isLoading={inventoryQuery.isLoading}
                />
                <KpiCard
                  label={isId ? 'Stok Direservasi' : 'Reserved Qty'}
                  value={formatNumber(inventoryMetrics.totalReserved, 2, locale)}
                  accent="amber"
                  icon={<Boxes className="h-5 w-5" />}
                  isLoading={inventoryQuery.isLoading}
                />
              </div>

              <article className="rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
                <div className="mb-3 flex items-center justify-between gap-2">
                  <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                    {isId ? 'Stok Fisik vs Tersedia per Gudang' : 'On-Hand vs Available by Warehouse'}
                  </h2>
                  <Button
                    variant="outline"
                    size="xs"
                    aria-pressed={showChartTable}
                    leftIcon={<Table2 className="h-3.5 w-3.5" />}
                    onClick={() => setShowChartTable((p) => !p)}
                  >
                    {showChartTable ? (isId ? 'Grafik' : 'Chart') : isId ? 'Tabel' : 'Table'}
                  </Button>
                </div>

                {inventoryQuery.isLoading ? (
                  <CardSkeleton />
                ) : inventoryQuery.error ? (
                  <ErrorState
                    error={inventoryQuery.error}
                    onRetry={() => void inventoryQuery.refetch()}
                  />
                ) : inventoryMetrics.warehouseSeries.length === 0 ? (
                  <EmptyState
                    title={isId ? 'Belum Ada Posisi Persediaan' : 'No Inventory Positions'}
                    onResetFilters={() => {
                      setInvSearch('');
                      setInvWarehouseFilter('ALL');
                    }}
                  />
                ) : showChartTable ? (
                  <div className="overflow-x-auto">
                    <table className="w-full border-collapse text-left text-xs">
                      <thead>
                        <tr className="border-b border-slate-200 text-slate-500 dark:border-slate-800">
                          <th scope="col" className="py-2 font-semibold">{isId ? 'Gudang' : 'Warehouse'}</th>
                          <th scope="col" className="py-2 text-right font-semibold">{isId ? 'Fisik' : 'On Hand'}</th>
                          <th scope="col" className="py-2 text-right font-semibold">{isId ? 'Tersedia' : 'Available'}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {inventoryMetrics.warehouseSeries.map((w) => (
                          <tr key={w.warehouseCode} className="border-b border-slate-100 dark:border-slate-800/70">
                            <td className="py-2 font-mono">{w.warehouseCode}</td>
                            <td className="py-2 text-right font-money">{formatNumber(w.onHand, 2, locale)}</td>
                            <td className="py-2 text-right font-money font-semibold">
                              {formatNumber(w.available, 2, locale)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="h-[240px] w-full">
                    <ResponsiveContainer width="100%" height={240} minWidth={220}>
                      <BarChart data={inventoryMetrics.warehouseSeries}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" opacity={0.45} />
                        <XAxis dataKey="warehouseCode" tick={{ fontSize: 11, fill: '#64748b' }} />
                        <YAxis tick={{ fontSize: 11, fill: '#64748b' }} width={52} />
                        <Tooltip content={<InventoryReportTooltip locale={locale} />} />
                        <Bar dataKey="chartOnHand" fill="#0284c7" radius={[4, 4, 0, 0]} />
                        <Bar dataKey="chartAvailable" fill="#0f766e" radius={[4, 4, 0, 0]} />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </article>

              <VirtualizedTable
                data={inventoryRows}
                columns={inventoryColumns}
                getRowId={(r) => r.id}
                isLoading={inventoryQuery.isLoading}
                error={inventoryQuery.error}
                onRetry={() => void inventoryQuery.refetch()}
                emptyTitle={isId ? 'Persediaan Kosong' : 'No Inventory Positions'}
                onResetFilters={() => {
                  setInvSearch('');
                  setInvWarehouseFilter('ALL');
                }}
                toolbar={{
                  searchValue: invSearch,
                  onSearchChange: setInvSearch,
                  searchPlaceholder: isId ? 'Cari kode item atau gudang...' : 'Search item or warehouse...',
                  filters: (
                    <div className="w-48">
                      <Select
                        aria-label={isId ? 'Filter Gudang' : 'Filter Warehouse'}
                        value={invWarehouseFilter}
                        onValueChange={setInvWarehouseFilter}
                        options={warehouseOptions}
                      />
                    </div>
                  ),
                  onExportCsv: handleExportActiveCsv,
                }}
              />
            </>
          )}
        </TabsContent>

        {/* Tab 4: Trial Balance Viewer */}
        <TabsContent value="trial-balance" className="space-y-4">
          {!canReadGl ? (
            <ErrorState
              title={isId ? 'Izin Buku Besar Diperlukan' : 'GL Permission Required'}
              message={isId ? 'Dibutuhkan izin gl:read.' : 'Requires gl:read permission.'}
            />
          ) : (
            <>
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-2xs dark:border-slate-800 dark:bg-slate-900">
                <div className="flex flex-wrap items-center gap-3">
                  <div className="w-56">
                    <Select
                      aria-label={isId ? 'Pilih Ledger' : 'Select Ledger'}
                      value={selectedTbLedgerId}
                      onValueChange={setSelectedTbLedgerId}
                      options={ledgers.map((l) => ({
                        value: l.id,
                        label: `${l.code} (${l.currencyCode})`,
                      }))}
                      placeholder={isId ? 'Pilih Ledger...' : 'Select Ledger...'}
                    />
                  </div>
                  <div className="w-64">
                    <Select
                      aria-label={isId ? 'Pilih Periode Fiskal' : 'Select Period'}
                      value={selectedTbPeriodId}
                      onValueChange={setSelectedTbPeriodId}
                      options={periods.map((p) => ({
                        value: p.id,
                        label: `#${p.periodNo} (${p.startDate} → ${p.endDate})`,
                      }))}
                      placeholder={isId ? 'Pilih Periode...' : 'Select Period...'}
                    />
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-3 text-xs">
                  <span>
                    Debit:{' '}
                    <strong className="font-money">
                      {formatCurrency(tbMetrics.periodDebit, baseCurrency, locale)}
                    </strong>
                  </span>
                  <span>
                    {isId ? 'Kredit:' : 'Credit:'}{' '}
                    <strong className="font-money">
                      {formatCurrency(tbMetrics.periodCredit, baseCurrency, locale)}
                    </strong>
                  </span>
                  <StatusBadge
                    status={tbMetrics.isBalanced ? 'POSTED' : 'REJECTED'}
                    label={
                      tbMetrics.isBalanced
                        ? isId
                          ? 'Seimbang'
                          : 'Balanced'
                        : isId
                          ? 'Tidak Seimbang'
                          : 'Unbalanced'
                    }
                  />
                </div>
              </div>

              {!selectedTbLedgerId || !selectedTbPeriodId ? (
                <EmptyState
                  title={isId ? 'Pilih Ledger & Periode Fiskal' : 'Select Ledger & Period'}
                  description={
                    isId
                      ? 'Silakan pilih Ledger dan Periode Fiskal untuk menampilkan Neraca Saldo.'
                      : 'Choose both an active Ledger and Fiscal Period to render Trial Balance.'
                  }
                />
              ) : (
                <VirtualizedTable
                  data={enrichedTbRows}
                  columns={tbColumns}
                  getRowId={(r) => r.id}
                  isLoading={trialBalanceQuery.isLoading}
                  error={trialBalanceQuery.error}
                  onRetry={() => void trialBalanceQuery.refetch()}
                  emptyTitle={isId ? 'Neraca Saldo Kosong' : 'No Trial Balance Rows'}
                  toolbar={{ onExportCsv: handleExportActiveCsv }}
                />
              )}
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
