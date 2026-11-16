import { addDecimals, Decimal, subtractDecimals } from '../../lib/decimal';

export interface GlBalanceSnapshotDto {
  id: string;
  fiscalPeriodId: string;
  accountCode: string;
  closingBalance: string;
  refreshedAt: string;
}

export interface SalesSummaryDto {
  id: string;
  organizationId: string;
  periodYear: number;
  periodMonth: number;
  customerId: string;
  totalSalesAmount: string;
  orderCount: number;
  refreshedAt: string;
}

export interface InventoryPositionDto {
  id: string;
  organizationId: string;
  warehouseCode: string;
  itemCode: string;
  onHandQty: string;
  availableQty: string;
  refreshedAt: string;
}

export interface WorkflowTaskDto {
  id: string;
  organizationId: string;
  instanceId: string;
  assigneeId: string;
  decision?: string | null;
  documentRef: string;
  instanceStatus: string;
  createdBy: string;
  rowVersion: number;
}

export interface DashboardJournalSummaryDto {
  id: string;
  journalNo: string;
  postingDate: string;
  currencyCode: string;
  totalDebit: string;
  status: string;
}

export interface DashboardCurrencyDto {
  id: string;
  code: string;
  name: string;
  precision: number;
  isActive: boolean;
}

export type AccountCategoryKey = 'Asset' | 'Liability' | 'Equity' | 'Revenue' | 'Expense' | 'Other';

export interface CategoryBreakdownDatum {
  category: AccountCategoryKey;
  labelId: string;
  labelEn: string;
  amount: string;
  chartValue: number;
  accountCount: number;
  color: string;
}

export interface GlDashboardSummary {
  snapshotCount: number;
  assetsTotal: string;
  liabilitiesTotal: string;
  equityTotal: string;
  revenueTotal: string;
  expenseTotal: string;
  netOperatingMargin: string;
  accountingEquationDiff: string;
  isAccountingEquationBalanced: boolean;
  categorySeries: CategoryBreakdownDatum[];
  lastRefreshedAt: string | null;
}

export interface SalesPeriodDatum {
  periodKey: string;
  totalSalesAmount: string;
  chartSales: number;
  orderCount: number;
}

export interface WarehouseStockDatum {
  warehouseCode: string;
  onHandQty: string;
  availableQty: string;
  reservedQty: string;
  chartOnHand: number;
  chartAvailable: number;
  skuCount: number;
}

export interface SalesInventorySummary {
  totalSalesAmount: string;
  totalOrders: number;
  totalOnHandQty: string;
  totalAvailableQty: string;
  totalReservedQty: string;
  salesSeries: SalesPeriodDatum[];
  warehouseSeries: WarehouseStockDatum[];
  lastRefreshedAt: string | null;
}

const CATEGORY_METADATA: Record<
  AccountCategoryKey,
  { labelId: string; labelEn: string; color: string }
> = {
  Asset: { labelId: 'Aset', labelEn: 'Asset', color: '#0284c7' },
  Liability: { labelId: 'Kewajiban', labelEn: 'Liability', color: '#d97706' },
  Equity: { labelId: 'Ekuitas', labelEn: 'Equity', color: '#4338ca' },
  Revenue: { labelId: 'Pendapatan', labelEn: 'Revenue', color: '#0f766e' },
  Expense: { labelId: 'Beban', labelEn: 'Expense', color: '#be123c' },
  Other: { labelId: 'Lainnya', labelEn: 'Other', color: '#64748b' },
};

/**
 * Converts a validated fixed-point decimal string purely at the chart rendering boundary.
 * Financial totals and arithmetic MUST always stay in `Decimal`.
 */
export function toChartCoordinate(decimalValue: string): number {
  const normalized = new Decimal(decimalValue || '0', 2).abs().toRawString();
  const numeric = Number(normalized);
  return Number.isFinite(numeric) ? numeric : 0;
}

export function classifyAccountCategory(accountCode: string): AccountCategoryKey {
  const firstDigitMatch = accountCode.trim().match(/\d/);
  const digit = firstDigitMatch ? firstDigitMatch[0] : '';
  switch (digit) {
    case '1':
      return 'Asset';
    case '2':
      return 'Liability';
    case '3':
      return 'Equity';
    case '4':
      return 'Revenue';
    case '5':
    case '6':
    case '7':
    case '8':
    case '9':
      return 'Expense';
    default:
      return 'Other';
  }
}

export function pickLatestTimestamp(timestamps: Array<string | null | undefined>): string | null {
  let latestIso: string | null = null;
  let latestMs = -1;
  for (const raw of timestamps) {
    if (!raw) continue;
    const ms = Date.parse(raw);
    if (!Number.isNaN(ms) && ms >= latestMs) {
      latestMs = ms;
      latestIso = raw;
    }
  }
  return latestIso;
}

export function summarizeGlSnapshots(rows: GlBalanceSnapshotDto[]): GlDashboardSummary {
  const totals: Record<AccountCategoryKey, string> = {
    Asset: '0.00',
    Liability: '0.00',
    Equity: '0.00',
    Revenue: '0.00',
    Expense: '0.00',
    Other: '0.00',
  };
  const counts: Record<AccountCategoryKey, number> = {
    Asset: 0,
    Liability: 0,
    Equity: 0,
    Revenue: 0,
    Expense: 0,
    Other: 0,
  };

  for (const row of rows) {
    const category = classifyAccountCategory(row.accountCode);
    totals[category] = addDecimals(totals[category], row.closingBalance || '0', 2);
    counts[category] += 1;
  }

  const orderedCategories: AccountCategoryKey[] = [
    'Asset',
    'Liability',
    'Equity',
    'Revenue',
    'Expense',
    'Other',
  ];

  const categorySeries: CategoryBreakdownDatum[] = orderedCategories
    .filter((cat) => counts[cat] > 0 || !new Decimal(totals[cat], 2).isZero())
    .map((cat) => ({
      category: cat,
      labelId: CATEGORY_METADATA[cat].labelId,
      labelEn: CATEGORY_METADATA[cat].labelEn,
      amount: totals[cat],
      chartValue: toChartCoordinate(totals[cat]),
      accountCount: counts[cat],
      color: CATEGORY_METADATA[cat].color,
    }));

  const liabilitiesPlusEquity = addDecimals(totals.Liability, totals.Equity, 2);
  const accountingEquationDiff = subtractDecimals(totals.Asset, liabilitiesPlusEquity, 2);
  const isAccountingEquationBalanced =
    rows.length > 0 && new Decimal(accountingEquationDiff, 2).isZero();

  return {
    snapshotCount: rows.length,
    assetsTotal: totals.Asset,
    liabilitiesTotal: totals.Liability,
    equityTotal: totals.Equity,
    revenueTotal: totals.Revenue,
    expenseTotal: totals.Expense,
    netOperatingMargin: subtractDecimals(totals.Revenue, totals.Expense, 2),
    accountingEquationDiff,
    isAccountingEquationBalanced,
    categorySeries,
    lastRefreshedAt: pickLatestTimestamp(rows.map((r) => r.refreshedAt)),
  };
}

export function summarizeSalesAndInventory(
  salesRows: SalesSummaryDto[],
  inventoryRows: InventoryPositionDto[]
): SalesInventorySummary {
  let totalSalesAmount = '0.00';
  let totalOrders = 0;

  const salesByPeriod = new Map<string, { total: string; orders: number }>();

  for (const row of salesRows) {
    totalSalesAmount = addDecimals(totalSalesAmount, row.totalSalesAmount || '0', 2);
    totalOrders += row.orderCount || 0;
    const periodKey = `${row.periodYear}-${String(row.periodMonth).padStart(2, '0')}`;
    const prev = salesByPeriod.get(periodKey) ?? { total: '0.00', orders: 0 };
    salesByPeriod.set(periodKey, {
      total: addDecimals(prev.total, row.totalSalesAmount || '0', 2),
      orders: prev.orders + (row.orderCount || 0),
    });
  }

  const salesSeries: SalesPeriodDatum[] = Array.from(salesByPeriod.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([periodKey, val]) => ({
      periodKey,
      totalSalesAmount: val.total,
      chartSales: toChartCoordinate(val.total),
      orderCount: val.orders,
    }));

  let totalOnHandQty = '0.00';
  let totalAvailableQty = '0.00';
  const stockByWarehouse = new Map<
    string,
    { onHand: string; available: string; skuCount: number }
  >();

  for (const item of inventoryRows) {
    totalOnHandQty = addDecimals(totalOnHandQty, item.onHandQty || '0', 2);
    totalAvailableQty = addDecimals(totalAvailableQty, item.availableQty || '0', 2);
    const wh = item.warehouseCode || 'UNASSIGNED';
    const prev = stockByWarehouse.get(wh) ?? { onHand: '0.00', available: '0.00', skuCount: 0 };
    stockByWarehouse.set(wh, {
      onHand: addDecimals(prev.onHand, item.onHandQty || '0', 2),
      available: addDecimals(prev.available, item.availableQty || '0', 2),
      skuCount: prev.skuCount + 1,
    });
  }

  const warehouseSeries: WarehouseStockDatum[] = Array.from(stockByWarehouse.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([warehouseCode, val]) => ({
      warehouseCode,
      onHandQty: val.onHand,
      availableQty: val.available,
      reservedQty: subtractDecimals(val.onHand, val.available, 2),
      chartOnHand: toChartCoordinate(val.onHand),
      chartAvailable: toChartCoordinate(val.available),
      skuCount: val.skuCount,
    }));

  const lastRefreshedAt = pickLatestTimestamp([
    ...salesRows.map((r) => r.refreshedAt),
    ...inventoryRows.map((r) => r.refreshedAt),
  ]);

  return {
    totalSalesAmount,
    totalOrders,
    totalOnHandQty,
    totalAvailableQty,
    totalReservedQty: subtractDecimals(totalOnHandQty, totalAvailableQty, 2),
    salesSeries,
    warehouseSeries,
    lastRefreshedAt,
  };
}
