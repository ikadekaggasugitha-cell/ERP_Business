import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircle2,
  CreditCard,
  FileText,
  Pencil,
  Plus,
  Receipt,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Button } from '../../components/ui/Button';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { ApiClientError } from '../../lib/api';
import { Decimal } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { useTranslation } from 'react-i18next';
import { formatCurrency, formatDate } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  apQueryKeys,
  deleteVendorInvoice,
  fetchCoaAccountsLookup,
  fetchCurrenciesLookup,
  fetchPartiesLookup,
  fetchVendorInvoiceDetail,
  fetchVendorInvoices,
  recordVendorInvoice,
} from './api';
import { ApInvoiceFormDialog } from './ApInvoiceFormDialog';
import type { VendorInvoiceDto } from './types';

function apStatusTone(status?: string): 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand' {
  switch ((status ?? '').toUpperCase()) {
    case 'RECORDED':
    case 'PAID':
    case 'POSTED':
      return 'success';
    case 'PARTIAL':
    case 'PENDING':
      return 'warning';
    case 'VOIDED':
    case 'CANCELLED':
      return 'danger';
    default:
      return 'neutral';
  }
}

export function ApWorkspace() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const locale = useUiStore((s) => s.locale);
  const numberLocale = locale === 'id' ? 'id-ID' : 'en-US';
  const canWrite = useAuthStore((s) => s.hasPermission)('ap:write');

  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchTerm, setSearchTerm] = useState('');
  const [formOpen, setFormOpen] = useState(false);
  const [editingInvoice, setEditingInvoice] = useState<VendorInvoiceDto | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const listQuery = useQuery({
    queryKey: apQueryKeys.list({ page, size: 25, status: statusFilter }),
    queryFn: () => fetchVendorInvoices({ page, size: 25, status: statusFilter }),
  });

  const partiesQuery = useQuery({
    queryKey: apQueryKeys.vendors,
    queryFn: fetchPartiesLookup,
    retry: false,
  });

  const accountsQuery = useQuery({
    queryKey: apQueryKeys.accounts,
    queryFn: fetchCoaAccountsLookup,
    retry: false,
  });

  const currenciesQuery = useQuery({
    queryKey: apQueryKeys.currencies,
    queryFn: fetchCurrenciesLookup,
    retry: false,
  });

  const detailQuery = useQuery({
    queryKey: apQueryKeys.detail(selectedId ?? ''),
    queryFn: () => fetchVendorInvoiceDetail(selectedId!),
    enabled: Boolean(selectedId),
  });

  const parties = useMemo(() => partiesQuery.data ?? [], [partiesQuery.data]);
  const accounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);
  const currencies = useMemo(() => currenciesQuery.data ?? [], [currenciesQuery.data]);

  const partyById = useMemo(
    () => new Map(parties.map((party) => [party.id, party])),
    [parties],
  );
  const accountById = useMemo(
    () => new Map(accounts.map((account) => [account.id, account])),
    [accounts],
  );

  const items = useMemo(() => listQuery.data?.items ?? [], [listQuery.data]);
  const pageMeta = listQuery.data?.page ?? {
    number: page,
    size: 25,
    totalElements: items.length,
    totalPages: 1,
  };

  const filteredItems = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) return items;
    return items.filter((inv) => {
      const vendor = partyById.get(inv.vendorId);
      return (
        inv.invoiceNo.toLowerCase().includes(query) ||
        inv.vendorId.toLowerCase().includes(query) ||
        (vendor?.name.toLowerCase().includes(query) ?? false) ||
        (vendor?.code.toLowerCase().includes(query) ?? false) ||
        inv.currencyCode.toLowerCase().includes(query)
      );
    });
  }, [items, searchTerm, partyById]);

  const kpis = useMemo(() => {
    const primaryCurrency = items[0]?.currencyCode || 'IDR';
    const totalSum = Decimal.sum(
      items.map((item) => item.totalAmount || '0'),
      2,
    );
    const outstandingSum = Decimal.sum(
      items.map((item) => item.outstandingAmount || '0'),
      2,
    );
    const draftCount = items.filter((i) => i.status.toUpperCase() === 'DRAFT').length;
    const recordedCount = items.filter((i) => i.status.toUpperCase() === 'RECORDED').length;
    return {
      primaryCurrency,
      totalFormatted: formatCurrency(totalSum.toRawString(), primaryCurrency, numberLocale),
      outstandingFormatted: formatCurrency(
        outstandingSum.toRawString(),
        primaryCurrency,
        numberLocale,
      ),
      draftCount,
      recordedCount,
    };
  }, [items, numberLocale]);

  const recordMutation = useMutation({
    mutationFn: (inv: VendorInvoiceDto) => recordVendorInvoice(inv.id, inv.rowVersion),
    onSuccess: async (updated) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: apQueryKeys.all }),
        queryClient.invalidateQueries({ queryKey: apQueryKeys.detail(updated.id) }),
      ]);
      toast.success(`Vendor invoice ${updated.invoiceNo} recorded to General Ledger`);
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Failed to record vendor invoice';
      toast.error(msg);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (inv: VendorInvoiceDto) => deleteVendorInvoice(inv.id, inv.rowVersion),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: apQueryKeys.all });
      setConfirmDelete(false);
      setSelectedId(null);
      toast.success('Draft vendor invoice deleted');
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Failed to delete draft invoice';
      toast.error(msg);
    },
  });

  const activeDetail = useMemo(() => {
    const fallback = items.find((item) => item.id === selectedId) ?? null;
    if (detailQuery.data && detailQuery.data.id) {
      return detailQuery.data;
    }
    return fallback;
  }, [detailQuery.data, items, selectedId]);

  const columns: ColumnDef<VendorInvoiceDto>[] = [
    {
      header: 'Invoice No',
      accessorKey: 'invoiceNo',
      width: '160px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900">{row.invoiceNo}</span>
      ),
    },
    {
      header: 'Vendor',
      width: '230px',
      cell: (row) => {
        const party = partyById.get(row.vendorId);
        return party ? (
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-slate-900">{party.name}</p>
            <p className="truncate font-mono text-[11px] text-slate-500">{party.code}</p>
          </div>
        ) : (
          <span className="font-mono text-xs text-slate-600">{row.vendorId.slice(0, 13)}…</span>
        );
      },
    },
    {
      header: 'Invoice Date',
      width: '125px',
      cell: (row) => (
        <span className="text-xs text-slate-600">{formatDate(row.invoiceDate, numberLocale)}</span>
      ),
    },
    {
      header: 'Due Date',
      width: '125px',
      cell: (row) => (
        <span className="text-xs text-slate-600">{formatDate(row.dueDate, numberLocale)}</span>
      ),
    },
    {
      header: 'Total',
      align: 'right',
      width: '160px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900 tabular-nums">
          {formatCurrency(row.totalAmount, row.currencyCode, numberLocale)}
        </span>
      ),
    },
    {
      header: 'Outstanding',
      align: 'right',
      width: '160px',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-700 tabular-nums">
          {formatCurrency(row.outstandingAmount, row.currencyCode, numberLocale)}
        </span>
      ),
    },
    {
      header: 'Status',
      width: '125px',
      cell: (row) => (
        <StatusBadge
          status={row.status}
          label={row.status}
          tone={apStatusTone(row.status)}
        />
      ),
    },
  ];

  const handleOpenCreate = () => {
    setEditingInvoice(null);
    setFormOpen(true);
  };

  const handleOpenEdit = (invoice: VendorInvoiceDto) => {
    setSelectedId(null);
    setEditingInvoice(invoice);
    setFormOpen(true);
  };

  const handleExportCsv = () => {
    exportToCsv(
      filteredItems,
      [
        { header: 'Invoice No', accessor: (r) => r.invoiceNo },
        {
          header: 'Vendor',
          accessor: (r) => partyById.get(r.vendorId)?.name ?? r.vendorId,
        },
        { header: 'Vendor ID', accessor: (r) => r.vendorId },
        { header: 'Invoice Date', accessor: (r) => r.invoiceDate },
        { header: 'Due Date', accessor: (r) => r.dueDate },
        { header: 'Currency', accessor: (r) => r.currencyCode },
        { header: 'Total Amount', accessor: (r) => r.totalAmount },
        { header: 'Outstanding Amount', accessor: (r) => r.outstandingAmount },
        { header: 'Status', accessor: (r) => r.status },
      ],
      'vendor-invoices.csv',
    );
    toast.success('Vendor invoices exported to CSV');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              {t('nav.ap')} · Vendor Bills
            </h1>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-700">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  listQuery.isError
                    ? 'bg-rose-500'
                    : listQuery.isFetching
                      ? 'bg-amber-500'
                      : 'bg-emerald-500'
                }`}
              />
              SVC-003 · AP
            </span>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            Supplier invoice intake, multi-line GL cost distribution, and General Ledger posting.
          </p>
        </div>

        {canWrite && (
          <Button leftIcon={<Plus className="h-4 w-4" />} onClick={handleOpenCreate}>
            New Vendor Invoice
          </Button>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Vendor Invoices"
          value={pageMeta.totalElements}
          subtitle={`${kpis.recordedCount} recorded on page`}
          accent="brand"
          icon={<FileText className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
        <KpiCard
          label="Page Payable Volume"
          value={kpis.totalFormatted}
          subtitle={`Primary currency ${kpis.primaryCurrency}`}
          accent="sky"
          icon={<Receipt className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
        <KpiCard
          label="Outstanding Liability"
          value={kpis.outstandingFormatted}
          subtitle="Open payable balance"
          accent="amber"
          icon={<CreditCard className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
        <KpiCard
          label="Drafts Awaiting Record"
          value={kpis.draftCount}
          subtitle={canWrite ? 'Ready to post' : 'Read-only view'}
          accent="emerald"
          icon={<CheckCircle2 className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
      </div>

      <VirtualizedTable
        data={filteredItems}
        columns={columns}
        isLoading={listQuery.isLoading}
        error={listQuery.error}
        onRetry={() => void listQuery.refetch()}
        emptyTitle="No vendor invoices found"
        emptyDescription="Create a supplier bill to track accounts payable obligations and post expense lines."
        onResetFilters={
          searchTerm || statusFilter !== 'ALL'
            ? () => {
                setSearchTerm('');
                setStatusFilter('ALL');
                setPage(1);
              }
            : undefined
        }
        onRowClick={(row) => {
          setConfirmDelete(false);
          setSelectedId(row.id);
        }}
        pagination={{
          page: pageMeta.number || page,
          size: pageMeta.size || 25,
          totalElements: pageMeta.totalElements,
          totalPages: pageMeta.totalPages || 1,
          pageBase: 1,
          onPageChange: setPage,
        }}
        toolbar={{
          searchValue: searchTerm,
          onSearchChange: setSearchTerm,
          searchPlaceholder: 'Search invoice number, vendor, currency...',
          onExportCsv: filteredItems.length > 0 ? handleExportCsv : undefined,
          filters: (
            <Select
              aria-label="Filter AP status"
              className="w-40"
              value={statusFilter}
              onValueChange={(value) => {
                setStatusFilter(value);
                setPage(1);
              }}
              options={[
                { value: 'ALL', label: 'All Statuses' },
                { value: 'Draft', label: 'Draft' },
                { value: 'Recorded', label: 'Recorded' },
                { value: 'Paid', label: 'Paid' },
              ]}
            />
          ),
        }}
      />

      <SideDrawer
        open={Boolean(selectedId)}
        onOpenChange={(open) => {
          if (!open) {
            setConfirmDelete(false);
            setSelectedId(null);
          }
        }}
        size="lg"
        title={activeDetail ? `Vendor Invoice ${activeDetail.invoiceNo}` : 'Vendor Invoice'}
        description={
          activeDetail
            ? `Version v${activeDetail.rowVersion} · ${activeDetail.currencyCode}`
            : undefined
        }
        footer={
          activeDetail && canWrite ? (
            <div className="flex flex-wrap items-center justify-between gap-2">
              {(activeDetail.status ?? '').toUpperCase() === 'DRAFT' ? (
                <>
                  <div className="flex items-center gap-2">
                    {!confirmDelete ? (
                      <Button
                        variant="danger"
                        size="sm"
                        leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                        onClick={() => setConfirmDelete(true)}
                      >
                        Delete
                      </Button>
                    ) : (
                      <>
                        <Button
                          variant="danger"
                          size="sm"
                          isLoading={deleteMutation.isPending}
                          onClick={() => deleteMutation.mutate(activeDetail)}
                        >
                          Confirm Delete
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => setConfirmDelete(false)}
                        >
                          Cancel
                        </Button>
                      </>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      leftIcon={<Pencil className="h-3.5 w-3.5" />}
                      onClick={() => handleOpenEdit(activeDetail)}
                    >
                      Edit Draft
                    </Button>
                  </div>
                  <Button
                    variant="success"
                    size="sm"
                    leftIcon={<CheckCircle2 className="h-4 w-4" />}
                    isLoading={recordMutation.isPending}
                    onClick={() => recordMutation.mutate(activeDetail)}
                  >
                    Record / Post
                  </Button>
                </>
              ) : (
                <p className="text-xs text-slate-500">
                  Recorded invoices are locked for audit integrity.
                </p>
              )}
            </div>
          ) : undefined
        }
      >
        {activeDetail ? (
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm">
              <div>
                <p className="text-xs font-medium text-slate-500">Status</p>
                <div className="mt-1">
                  <StatusBadge
                    status={activeDetail.status}
                    label={activeDetail.status}
                    tone={apStatusTone(activeDetail.status)}
                  />
                </div>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Vendor</p>
                <p className="mt-1 font-medium text-slate-900">
                  {partyById.get(activeDetail.vendorId)?.name ?? activeDetail.vendorId}
                </p>
                <p className="font-mono text-[11px] text-slate-500">{activeDetail.vendorId}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Invoice Date</p>
                <p className="mt-1 text-slate-800">
                  {formatDate(activeDetail.invoiceDate, numberLocale)}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Due Date</p>
                <p className="mt-1 text-slate-800">
                  {formatDate(activeDetail.dueDate, numberLocale)}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Total Amount</p>
                <p className="mt-1 font-mono font-semibold text-slate-900 tabular-nums">
                  {formatCurrency(
                    activeDetail.totalAmount,
                    activeDetail.currencyCode,
                    numberLocale,
                  )}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Outstanding</p>
                <p className="mt-1 font-mono font-semibold text-amber-700 tabular-nums">
                  {formatCurrency(
                    activeDetail.outstandingAmount,
                    activeDetail.currencyCode,
                    numberLocale,
                  )}
                </p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-sm font-semibold text-slate-900">Distribution Lines</h3>
              {detailQuery.isLoading ? (
                <div className="rounded-lg border border-slate-200 p-4 text-xs text-slate-500">
                  Loading invoice lines...
                </div>
              ) : activeDetail.lines && activeDetail.lines.length > 0 ? (
                <div className="overflow-hidden rounded-xl border border-slate-200">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-slate-50 text-slate-600">
                      <tr>
                        <th className="px-3 py-2 font-semibold">#</th>
                        <th className="px-3 py-2 font-semibold">GL Account</th>
                        <th className="px-3 py-2 font-semibold">Description</th>
                        <th className="px-3 py-2 text-right font-semibold">Amount</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {activeDetail.lines.map((line, index) => {
                        const acc = accountById.get(line.accountId);
                        return (
                          <tr key={line.id ?? index}>
                            <td className="px-3 py-2 font-mono text-slate-500">{line.lineNo}</td>
                            <td className="px-3 py-2">
                              {acc ? (
                                <span>
                                  <span className="font-mono font-semibold text-slate-900">
                                    {acc.code}
                                  </span>{' '}
                                  {acc.name}
                                </span>
                              ) : (
                                <span className="font-mono text-slate-600">{line.accountId}</span>
                              )}
                            </td>
                            <td className="px-3 py-2 text-slate-700">{line.description}</td>
                            <td className="px-3 py-2 text-right font-mono font-medium text-slate-900 tabular-nums">
                              {formatCurrency(
                                line.amount,
                                activeDetail.currencyCode,
                                numberLocale,
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="rounded-lg border border-slate-200 p-4 text-xs text-slate-500">
                  No distribution lines attached.
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      <ApInvoiceFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        editingInvoice={editingInvoice}
        vendors={parties}
        accounts={accounts}
        currencies={currencies}
      />
    </div>
  );
}
