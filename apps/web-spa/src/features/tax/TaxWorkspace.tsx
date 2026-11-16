import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  CheckCircle2,
  FileCheck2,
  Percent,
  Plus,
  Receipt,
  Send,
} from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Button } from '../../components/ui/Button';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { Decimal } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { useTranslation } from 'react-i18next';
import { formatCurrency } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  fetchTaxAccountsLookup,
  fetchTaxCodes,
  fetchTaxInvoiceDetail,
  fetchTaxInvoices,
  taxQueryKeys,
} from './api';
import {
  CreateTaxCodeDialog,
  CreateTaxInvoiceDialog,
  ReportEfakturDialog,
} from './TaxForms';
import type { TaxCodeDto, TaxInvoiceDto } from './types';

function efakturTone(
  status?: string,
): 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand' {
  switch ((status ?? '').toUpperCase()) {
    case 'REPORTED':
      return 'brand';
    case 'APPROVED':
      return 'success';
    case 'REJECTED':
      return 'danger';
    default:
      return 'neutral';
  }
}

export function TaxWorkspace() {
  const { t } = useTranslation();
  const locale = useUiStore((s) => s.locale);
  const numberLocale = locale === 'id' ? 'id-ID' : 'en-US';
  const canWrite = useAuthStore((s) => s.hasPermission)('tax:write');

  const [activeTab, setActiveTab] = useState('invoices');
  const [invoicePage, setInvoicePage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [invoiceSearch, setInvoiceSearch] = useState('');

  const [codePage, setCodePage] = useState(1);
  const [codeSearch, setCodeSearch] = useState('');

  const [createInvoiceOpen, setCreateInvoiceOpen] = useState(false);
  const [createCodeOpen, setCreateCodeOpen] = useState(false);
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string | null>(null);
  const [reportingInvoice, setReportingInvoice] = useState<TaxInvoiceDto | null>(null);

  const invoicesQuery = useQuery({
    queryKey: taxQueryKeys.invoicesList({
      page: invoicePage,
      size: 25,
      efakturStatus: statusFilter,
    }),
    queryFn: () =>
      fetchTaxInvoices({
        page: invoicePage,
        size: 25,
        efakturStatus: statusFilter,
      }),
  });

  const codesQuery = useQuery({
    queryKey: taxQueryKeys.codesList({ page: codePage, size: 50 }),
    queryFn: () => fetchTaxCodes({ page: codePage, size: 50 }),
  });

  const accountsQuery = useQuery({
    queryKey: taxQueryKeys.accounts,
    queryFn: fetchTaxAccountsLookup,
    retry: false,
  });

  const detailQuery = useQuery({
    queryKey: taxQueryKeys.invoiceDetail(selectedInvoiceId ?? ''),
    queryFn: () => fetchTaxInvoiceDetail(selectedInvoiceId!),
    enabled: Boolean(selectedInvoiceId),
  });

  const invoices = useMemo(() => invoicesQuery.data?.items ?? [], [invoicesQuery.data]);
  const invoicePageMeta = invoicesQuery.data?.page ?? {
    number: invoicePage,
    size: 25,
    totalElements: invoices.length,
    totalPages: 1,
  };

  const taxCodes = useMemo(() => codesQuery.data?.items ?? [], [codesQuery.data]);
  const codePageMeta = codesQuery.data?.page ?? {
    number: codePage,
    size: 50,
    totalElements: taxCodes.length,
    totalPages: 1,
  };

  const accounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);
  const taxCodeById = useMemo(
    () => new Map(taxCodes.map((code) => [code.id, code])),
    [taxCodes],
  );
  const accountById = useMemo(
    () => new Map(accounts.map((acc) => [acc.id, acc])),
    [accounts],
  );

  const filteredInvoices = useMemo(() => {
    const q = invoiceSearch.trim().toLowerCase();
    if (!q) return invoices;
    return invoices.filter((inv) => {
      const code = taxCodeById.get(inv.taxCodeId);
      return (
        inv.sourceDocumentRef.toLowerCase().includes(q) ||
        (inv.efakturNo ?? '').toLowerCase().includes(q) ||
        (code?.code.toLowerCase().includes(q) ?? false) ||
        inv.efakturStatus.toLowerCase().includes(q)
      );
    });
  }, [invoices, invoiceSearch, taxCodeById]);

  const filteredCodes = useMemo(() => {
    const q = codeSearch.trim().toLowerCase();
    if (!q) return taxCodes;
    return taxCodes.filter(
      (c) =>
        c.code.toLowerCase().includes(q) ||
        c.taxType.toLowerCase().includes(q) ||
        c.rate.toLowerCase().includes(q),
    );
  }, [taxCodes, codeSearch]);

  const kpis = useMemo(() => {
    const totalBase = Decimal.sum(
      invoices.map((i) => i.baseAmount || '0'),
      2,
    );
    const totalTax = Decimal.sum(
      invoices.map((i) => i.taxAmount || '0'),
      2,
    );
    const reportedCount = invoices.filter(
      (i) => i.efakturStatus.toUpperCase() === 'REPORTED',
    ).length;
    return {
      totalBaseFormatted: formatCurrency(totalBase.toRawString(), 'IDR', numberLocale),
      totalTaxFormatted: formatCurrency(totalTax.toRawString(), 'IDR', numberLocale),
      reportedCount,
      draftCount: invoices.length - reportedCount,
    };
  }, [invoices, numberLocale]);

  const activeDetail = useMemo(() => {
    const fallback = invoices.find((item) => item.id === selectedInvoiceId) ?? null;
    if (detailQuery.data && detailQuery.data.id) {
      return detailQuery.data;
    }
    return fallback;
  }, [detailQuery.data, invoices, selectedInvoiceId]);

  const invoiceColumns: ColumnDef<TaxInvoiceDto>[] = [
    {
      header: 'Source Document',
      accessorKey: 'sourceDocumentRef',
      width: '185px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900">
          {row.sourceDocumentRef}
        </span>
      ),
    },
    {
      header: 'e-Faktur No (NSFP)',
      width: '185px',
      cell: (row) =>
        row.efakturNo ? (
          <span className="font-mono text-xs text-slate-800">{row.efakturNo}</span>
        ) : (
          <span className="text-xs italic text-slate-400">Unassigned</span>
        ),
    },
    {
      header: 'Tax Code',
      width: '165px',
      cell: (row) => {
        const code = taxCodeById.get(row.taxCodeId);
        return code ? (
          <div>
            <p className="text-xs font-semibold text-slate-900">{code.code}</p>
            <p className="text-[11px] text-slate-500">
              {code.taxType} · {code.rate}%
            </p>
          </div>
        ) : (
          <span className="font-mono text-xs text-slate-500">{row.taxCodeId.slice(0, 13)}…</span>
        );
      },
    },
    {
      header: 'DPP / Base',
      align: 'right',
      width: '160px',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-700 tabular-nums">
          {formatCurrency(row.baseAmount, 'IDR', numberLocale)}
        </span>
      ),
    },
    {
      header: 'Tax Amount',
      align: 'right',
      width: '155px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900 tabular-nums">
          {formatCurrency(row.taxAmount, 'IDR', numberLocale)}
        </span>
      ),
    },
    {
      header: 'DJP Status',
      width: '130px',
      cell: (row) => (
        <StatusBadge
          status={row.efakturStatus}
          label={row.efakturStatus}
          tone={efakturTone(row.efakturStatus)}
        />
      ),
    },
  ];

  const codeColumns: ColumnDef<TaxCodeDto>[] = [
    {
      header: 'Code',
      accessorKey: 'code',
      width: '160px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900">{row.code}</span>
      ),
    },
    {
      header: 'Tax Type',
      accessorKey: 'taxType',
      width: '180px',
      cell: (row) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700">
          {row.taxType}
        </span>
      ),
    },
    {
      header: 'Rate (%)',
      align: 'right',
      width: '130px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900 tabular-nums">
          {row.rate}%
        </span>
      ),
    },
    {
      header: 'GL Tax Account',
      width: '260px',
      cell: (row) => {
        if (!row.glAccountId) {
          return <span className="text-xs text-slate-400">—</span>;
        }
        const acc = accountById.get(row.glAccountId);
        return acc ? (
          <span className="text-xs text-slate-700">
            <span className="font-mono font-semibold">{acc.code}</span> {acc.name}
          </span>
        ) : (
          <span className="font-mono text-xs text-slate-600">{row.glAccountId}</span>
        );
      },
    },
    {
      header: 'Version',
      align: 'right',
      width: '100px',
      cell: (row) => <span className="font-mono text-xs text-slate-500">v{row.rowVersion}</span>,
    },
  ];

  const handleExportInvoices = () => {
    exportToCsv(
      filteredInvoices,
      [
        { header: 'Source Document', accessor: (r) => r.sourceDocumentRef },
        { header: 'e-Faktur No', accessor: (r) => r.efakturNo ?? '' },
        { header: 'Tax Code', accessor: (r) => taxCodeById.get(r.taxCodeId)?.code ?? r.taxCodeId },
        { header: 'Base Amount', accessor: (r) => r.baseAmount },
        { header: 'Tax Amount', accessor: (r) => r.taxAmount },
        { header: 'Status', accessor: (r) => r.efakturStatus },
      ],
      'tax-efaktur-invoices.csv',
    );
    toast.success('e-Faktur register exported to CSV');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              {t('nav.tax')} & e-Faktur
            </h1>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-700">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  invoicesQuery.isError || codesQuery.isError
                    ? 'bg-rose-500'
                    : invoicesQuery.isFetching
                      ? 'bg-amber-500'
                      : 'bg-emerald-500'
                }`}
              />
              SVC-005 · TAX
            </span>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            PPN & PPh tax codes, DPP calculation, and DJP e-Faktur compliance reporting.
          </p>
        </div>

        {canWrite && (
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              leftIcon={<Percent className="h-4 w-4" />}
              onClick={() => setCreateCodeOpen(true)}
            >
              New Tax Code
            </Button>
            <Button
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => setCreateInvoiceOpen(true)}
            >
              New Tax Invoice
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Tax Invoices"
          value={invoicePageMeta.totalElements}
          subtitle={`${kpis.draftCount} draft · ${kpis.reportedCount} reported`}
          accent="brand"
          icon={<FileCheck2 className="h-4 w-4" />}
          isLoading={invoicesQuery.isLoading}
        />
        <KpiCard
          label="Taxable Base (DPP)"
          value={kpis.totalBaseFormatted}
          subtitle="Current page sum"
          accent="sky"
          icon={<Receipt className="h-4 w-4" />}
          isLoading={invoicesQuery.isLoading}
        />
        <KpiCard
          label="Computed Tax (PPN/PPh)"
          value={kpis.totalTaxFormatted}
          subtitle="Exact fixed-point total"
          accent="amber"
          icon={<Percent className="h-4 w-4" />}
          isLoading={invoicesQuery.isLoading}
        />
        <KpiCard
          label="Active Tax Codes"
          value={codePageMeta.totalElements}
          subtitle="PPN, PPh 21/23/4(2) rules"
          accent="emerald"
          icon={<CheckCircle2 className="h-4 w-4" />}
          isLoading={codesQuery.isLoading}
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList variant="underline">
          <TabsTrigger value="invoices" variant="underline">
            Tax Invoices / e-Faktur ({invoicePageMeta.totalElements})
          </TabsTrigger>
          <TabsTrigger value="codes" variant="underline">
            Tax Codes ({codePageMeta.totalElements})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="invoices">
          <VirtualizedTable
            data={filteredInvoices}
            columns={invoiceColumns}
            isLoading={invoicesQuery.isLoading}
            error={invoicesQuery.error}
            onRetry={() => void invoicesQuery.refetch()}
            emptyTitle="No tax invoices recorded"
            emptyDescription="Create an e-Faktur entry linked to an AP or AR document to compute DPP & tax."
            onResetFilters={
              invoiceSearch || statusFilter !== 'ALL'
                ? () => {
                    setInvoiceSearch('');
                    setStatusFilter('ALL');
                    setInvoicePage(1);
                  }
                : undefined
            }
            onRowClick={(row) => setSelectedInvoiceId(row.id)}
            pagination={{
              page: invoicePageMeta.number || invoicePage,
              size: invoicePageMeta.size || 25,
              totalElements: invoicePageMeta.totalElements,
              totalPages: invoicePageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setInvoicePage,
            }}
            toolbar={{
              searchValue: invoiceSearch,
              onSearchChange: setInvoiceSearch,
              searchPlaceholder: 'Search source ref, e-Faktur NSFP, tax code...',
              onExportCsv: filteredInvoices.length > 0 ? handleExportInvoices : undefined,
              filters: (
                <Select
                  aria-label="Filter e-Faktur status"
                  className="w-40"
                  value={statusFilter}
                  onValueChange={(value) => {
                    setStatusFilter(value);
                    setInvoicePage(1);
                  }}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Draft', label: 'Draft' },
                    { value: 'Reported', label: 'Reported' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>

        <TabsContent value="codes">
          <VirtualizedTable
            data={filteredCodes}
            columns={codeColumns}
            isLoading={codesQuery.isLoading}
            error={codesQuery.error}
            onRetry={() => void codesQuery.refetch()}
            emptyTitle="No tax codes configured"
            emptyDescription="Add PPN (11%) or PPh withholding tax codes to apply across invoices."
            onResetFilters={codeSearch ? () => setCodeSearch('') : undefined}
            pagination={{
              page: codePageMeta.number || codePage,
              size: codePageMeta.size || 50,
              totalElements: codePageMeta.totalElements,
              totalPages: codePageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setCodePage,
            }}
            toolbar={{
              searchValue: codeSearch,
              onSearchChange: setCodeSearch,
              searchPlaceholder: 'Search tax code, type, rate...',
            }}
          />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedInvoiceId)}
        onOpenChange={(open) => {
          if (!open) setSelectedInvoiceId(null);
        }}
        size="md"
        title={
          activeDetail
            ? `Tax Invoice · ${activeDetail.sourceDocumentRef}`
            : 'Tax Invoice Detail'
        }
        description={
          activeDetail
            ? `Version v${activeDetail.rowVersion} · Status ${activeDetail.efakturStatus}`
            : undefined
        }
        footer={
          activeDetail &&
          canWrite &&
          activeDetail.efakturStatus.toUpperCase() !== 'REPORTED' ? (
            <div className="flex justify-end">
              <Button
                variant="primary"
                leftIcon={<Send className="h-4 w-4" />}
                onClick={() => setReportingInvoice(activeDetail)}
              >
                Report to DJP
              </Button>
            </div>
          ) : undefined
        }
      >
        {activeDetail ? (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm">
              <div>
                <p className="text-xs font-medium text-slate-500">e-Faktur Status</p>
                <div className="mt-1">
                  <StatusBadge
                    status={activeDetail.efakturStatus}
                    label={activeDetail.efakturStatus}
                    tone={efakturTone(activeDetail.efakturStatus)}
                  />
                </div>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">e-Faktur NSFP</p>
                <p className="mt-1 font-mono text-xs font-semibold text-slate-900">
                  {activeDetail.efakturNo || 'Not assigned yet'}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Tax Code</p>
                <p className="mt-1 font-medium text-slate-900">
                  {taxCodeById.get(activeDetail.taxCodeId)?.code ?? activeDetail.taxCodeId}
                </p>
                <p className="text-xs text-slate-500">
                  {taxCodeById.get(activeDetail.taxCodeId)
                    ? `${taxCodeById.get(activeDetail.taxCodeId)?.taxType} · ${
                        taxCodeById.get(activeDetail.taxCodeId)?.rate
                      }%`
                    : ''}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Source Reference</p>
                <p className="mt-1 font-mono text-xs font-semibold text-slate-900">
                  {activeDetail.sourceDocumentRef}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Base Amount (DPP)</p>
                <p className="mt-1 font-mono font-semibold text-slate-900 tabular-nums">
                  {formatCurrency(activeDetail.baseAmount, 'IDR', numberLocale)}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Tax Amount</p>
                <p className="mt-1 font-mono font-semibold text-brand-700 tabular-nums">
                  {formatCurrency(activeDetail.taxAmount, 'IDR', numberLocale)}
                </p>
              </div>
            </div>
          </div>
        ) : null}
      </SideDrawer>

      <CreateTaxInvoiceDialog
        open={createInvoiceOpen}
        onOpenChange={setCreateInvoiceOpen}
        taxCodes={taxCodes}
      />

      <CreateTaxCodeDialog
        open={createCodeOpen}
        onOpenChange={setCreateCodeOpen}
        accounts={accounts}
      />

      <ReportEfakturDialog
        invoice={reportingInvoice}
        onOpenChange={(open) => {
          if (!open) setReportingInvoice(null);
        }}
      />
    </div>
  );
}
