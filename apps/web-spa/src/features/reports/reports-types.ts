import { addDecimals, Decimal, divideDecimals, subtractDecimals } from '../../lib/decimal';
import {
  classifyAccountCategory,
  toChartCoordinate,
  type GlBalanceSnapshotDto,
  type InventoryPositionDto,
  type SalesSummaryDto,
} from '../dashboard/dashboard-utils';

export type { GlBalanceSnapshotDto, InventoryPositionDto, SalesSummaryDto };

export interface ReportsTrialBalanceDto {
  id: string;
  ledgerId: string;
  accountId: string;
  fiscalPeriodId: string;
  openingBalance: string;
  periodDebit: string;
  periodCredit: string;
  closingBalance: string;
  rowVersion: number;
}

export interface ReportLedgerLookupDto {
  id: string;
  code: string;
  ledgerType: string;
  currencyCode: string;
}

export interface ReportFiscalPeriodLookupDto {
  id: string;
  periodNo: number;
  startDate: string;
  endDate: string;
  status: string;
}

export interface ReportAccountLookupDto {
  id: string;
  code: string;
  name: string;
  accountType: string;
}

export interface EnrichedTrialBalanceItem extends ReportsTrialBalanceDto {
  accountCode: string;
  accountName: string;
  accountType: string;
}

export interface GlCategoryReportDatum {
  category: string;
  closingTotal: string;
  chartValue: number;
  count: number;
  color: string;
}

export interface SalesReportMetrics {
  totalRevenue: string;
  totalOrders: number;
  averageOrderValue: string;
  monthlySeries: Array<{
    period: string;
    revenue: string;
    chartRevenue: number;
    orders: number;
  }>;
}

export interface InventoryReportMetrics {
  totalOnHand: string;
  totalAvailable: string;
  totalReserved: string;
  skuCount: number;
  warehouseSeries: Array<{
    warehouseCode: string;
    onHand: string;
    available: string;
    chartOnHand: number;
    chartAvailable: number;
    skuCount: number;
  }>;
}

const CATEGORY_COLORS: Record<string, string> = {
  Asset: '#0284c7',
  Liability: '#d97706',
  Equity: '#4338ca',
  Revenue: '#0f766e',
  Expense: '#be123c',
  Other: '#64748b',
};

export function summarizeGlReport(rows: GlBalanceSnapshotDto[]): {
  netClosing: string;
  series: GlCategoryReportDatum[];
} {
  const byCat = new Map<string, { total: string; count: number }>();
  let netClosing = '0.00';

  for (const row of rows) {
    netClosing = addDecimals(netClosing, row.closingBalance || '0', 2);
    const cat = classifyAccountCategory(row.accountCode);
    const prev = byCat.get(cat) ?? { total: '0.00', count: 0 };
    byCat.set(cat, {
      total: addDecimals(prev.total, row.closingBalance || '0', 2),
      count: prev.count + 1,
    });
  }

  const series: GlCategoryReportDatum[] = Array.from(byCat.entries()).map(([category, val]) => ({
    category,
    closingTotal: val.total,
    chartValue: toChartCoordinate(val.total),
    count: val.count,
    color: CATEGORY_COLORS[category] ?? '#64748b',
  }));

  return { netClosing, series };
}

export function summarizeSalesReport(rows: SalesSummaryDto[]): SalesReportMetrics {
  let totalRevenue = '0.00';
  let totalOrders = 0;
  const byPeriod = new Map<string, { revenue: string; orders: number }>();

  for (const row of rows) {
    totalRevenue = addDecimals(totalRevenue, row.totalSalesAmount || '0', 2);
    totalOrders += row.orderCount || 0;
    const key = `${row.periodYear}-${String(row.periodMonth).padStart(2, '0')}`;
    const prev = byPeriod.get(key) ?? { revenue: '0.00', orders: 0 };
    byPeriod.set(key, {
      revenue: addDecimals(prev.revenue, row.totalSalesAmount || '0', 2),
      orders: prev.orders + (row.orderCount || 0),
    });
  }

  const averageOrderValue =
    totalOrders > 0 ? divideDecimals(totalRevenue, String(totalOrders), 2) : '0.00';

  const monthlySeries = Array.from(byPeriod.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([period, val]) => ({
      period,
      revenue: val.revenue,
      chartRevenue: toChartCoordinate(val.revenue),
      orders: val.orders,
    }));

  return {
    totalRevenue,
    totalOrders,
    averageOrderValue,
    monthlySeries,
  };
}

export function summarizeInventoryReport(rows: InventoryPositionDto[]): InventoryReportMetrics {
  let totalOnHand = '0.00';
  let totalAvailable = '0.00';
  const byWarehouse = new Map<string, { onHand: string; available: string; skuCount: number }>();

  for (const row of rows) {
    totalOnHand = addDecimals(totalOnHand, row.onHandQty || '0', 2);
    totalAvailable = addDecimals(totalAvailable, row.availableQty || '0', 2);
    const wh = row.warehouseCode || 'UNASSIGNED';
    const prev = byWarehouse.get(wh) ?? { onHand: '0.00', available: '0.00', skuCount: 0 };
    byWarehouse.set(wh, {
      onHand: addDecimals(prev.onHand, row.onHandQty || '0', 2),
      available: addDecimals(prev.available, row.availableQty || '0', 2),
      skuCount: prev.skuCount + 1,
    });
  }

  const warehouseSeries = Array.from(byWarehouse.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([warehouseCode, val]) => ({
      warehouseCode,
      onHand: val.onHand,
      available: val.available,
      chartOnHand: toChartCoordinate(val.onHand),
      chartAvailable: toChartCoordinate(val.available),
      skuCount: val.skuCount,
    }));

  return {
    totalOnHand,
    totalAvailable,
    totalReserved: subtractDecimals(totalOnHand, totalAvailable, 2),
    skuCount: rows.length,
    warehouseSeries,
  };
}

export function summarizeTrialBalanceReport(rows: EnrichedTrialBalanceItem[]): {
  opening: string;
  periodDebit: string;
  periodCredit: string;
  closing: string;
  isBalanced: boolean;
} {
  let opening = '0.00';
  let periodDebit = '0.00';
  let periodCredit = '0.00';
  let closing = '0.00';

  for (const r of rows) {
    opening = addDecimals(opening, r.openingBalance || '0', 2);
    periodDebit = addDecimals(periodDebit, r.periodDebit || '0', 2);
    periodCredit = addDecimals(periodCredit, r.periodCredit || '0', 2);
    closing = addDecimals(closing, r.closingBalance || '0', 2);
  }

  return {
    opening,
    periodDebit,
    periodCredit,
    closing,
    isBalanced: new Decimal(periodDebit, 2).equals(new Decimal(periodCredit, 2)),
  };
}
