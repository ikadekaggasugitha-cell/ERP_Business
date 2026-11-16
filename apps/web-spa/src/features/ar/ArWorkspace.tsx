import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BadgeCheck,
  Clock,
  FileSpreadsheet,
  Plus,
  Trash2,
  Wallet,
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
  arQueryKeys,
  deleteCustomerInvoice,
  fetchArAccountsLookup,
  fetchArCurrenciesLookup,
  fetchArPartiesLookup,
  fetchCustomerInvoiceDetail,
  fetchCustomerInvoices,
  markCustomerInvoicePaid,
} from './api';
import { ArInvoiceFormDialog } from './ArInvoiceFormDialog';
import type { CustomerInvoiceDto } from './types';

function arStatusTone(
  status?: string,
): 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand' {
  switch ((status ?? '').toUpperCase()) {
    case 'PAID':
      return 'success';
    case 'ISSUED':
      return 'brand';
    case 'OVERDUE':
    case 'VOIDED':
      return 'danger';
    default:
      return 'neutral';
  }
}

export function ArWorkspace() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const locale = useUiStore((s) => s.locale);
  const numberLocale = locale === 'id' ? 'id-ID' : 'en-US';
  const canWrite = useAuthStore((s) => s.hasPermission)('ar:write');

  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchTerm, setSearchTerm] = useState('');
  const [formOpen, setFormOpen] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const listQuery = useQuery({
    queryKey: arQueryKeys.list({ page, size: 25, status: statusFilter }),
    queryFn: () => fetchCustomerInvoices({ page, size: 25, status: statusFilter }),
  });

  const partiesQuery = useQuery({
    queryKey: arQueryKeys.customers,
    queryFn: fetchArPartiesLookup,
    retry: false,
  });

  const accountsQuery = useQuery({
    queryKey: arQueryKeys.accounts,
    queryFn: fetchArAccountsLookup,
    retry: false,
  });

  const currenciesQuery = useQuery({
    queryKey: arQueryKeys.currencies,
    queryFn: fetchArCurrenciesLookup,
    retry: false,
  });

  const detailQuery = useQuery({
    queryKey: arQueryKeys.detail(selectedId ?? ''),
    queryFn: () => fetchCustomerInvoiceDetail(selectedId!),
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
      const customer = partyById.get(inv.customerId);
      return (
        inv.invoiceNo.toLowerCase().includes(query) ||
        inv.customerId.toLowerCase().includes(query) ||
        (customer?.name.toLowerCase().includes(query) ?? false) ||
        (customer?.code.toLowerCase().includes(query) ?? false) ||
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
    const issuedCount = items.filter((i) => i.status.toUpperCase() === 'ISSUED').length;
    const paidCount = items.filter((i) => i.status.toUpperCase() === 'PAID').length;
    return {
      primaryCurrency,
      totalFormatted: formatCurrency(totalSum.toRawString(), primaryCurrency, numberLocale),
      outstandingFormatted: formatCurrency(
        outstandingSum.toRawString(),
        primaryCurrency,
        numberLocale,
      ),
      issuedCount,
      paidCount,
    };
  }, [items, numberLocale]);

  const markPaidMutation = useMutation({
    mutationFn: (inv: CustomerInvoiceDto) => markCustomerInvoicePaid(inv.id, inv.rowVersion),
    onSuccess: async (updated) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: arQueryKeys.all }),
        queryClient.invalidateQueries({ queryKey: arQueryKeys.detail(updated.id) }),
      ]);
      toast.success(`Customer invoice ${updated.invoiceNo} marked as Paid`);
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Failed to mark invoice as paid';
      toast.error(msg);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (inv: CustomerInvoiceDto) => deleteCustomerInvoice(inv.id, inv.rowVersion),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: arQueryKeys.all });
      setConfirmDelete(false);
      setSelectedId(null);
      toast.success('Draft customer invoice deleted');
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

  const columns: ColumnDef<CustomerInvoiceDto>[] = [
    {
      header: 'Invoice No',
      accessorKey: 'invoiceNo',
      width: '160px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900">{row.invoiceNo}</span>
      ),
    },
    {
      header: 'Customer',
      width: '230px',
      cell: (row) => {
        const party = partyById.get(row.customerId);
        return party ? (
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-slate-900">{party.name}</p>
            <p className="truncate font-mono text-[11px] text-slate-500">{party.code}</p>
          </div>
        ) : (
          <span className="font-mono text-xs text-slate-600">{row.customerId.slice(0, 13)}…</span>
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
      width: '120px',
      cell: (row) => (
        <StatusBadge
          status={row.status}
          label={row.status}
          tone={arStatusTone(row.status)}
        />
      ),
    },
  ];

  const handleExportCsv = () => {
    exportToCsv(
      filteredItems,
      [
        { header: 'Invoice No', accessor: (r) => r.invoiceNo },
        {
          header: 'Customer',
          accessor: (r) => partyById.get(r.customerId)?.name ?? r.customerId,
        },
        { header: 'Customer ID', accessor: (r) => r.customerId },
        { header: 'Invoice Date', accessor: (r) => r.invoiceDate },
        { header: 'Due Date', accessor: (r) => r.dueDate },
        { header: 'Currency', accessor: (r) => r.currencyCode },
        { header: 'Total Amount', accessor: (r) => r.totalAmount },
        { header: 'Outstanding Amount', accessor: (r) => r.outstandingAmount },
        { header: 'Status', accessor: (r) => r.status },
      ],
      'customer-invoices.csv',
    );
    toast.success('Customer invoices exported to CSV');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              {t('nav.ar')} · Customer Billing
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
              SVC-004 · AR
            </span>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            Customer invoicing, receivable collections, and payment settlement.
          </p>
        </div>

        {canWrite && (
          <Button leftIcon={<Plus className="h-4 w-4" />} onClick={() => setFormOpen(true)}>
            Issue Customer Invoice
          </Button>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Customer Invoices"
          value={pageMeta.totalElements}
          subtitle={`${kpis.paidCount} paid on page`}
          accent="brand"
          icon={<FileSpreadsheet className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
        <KpiCard
          label="Billed Volume"
          value={kpis.totalFormatted}
          subtitle={`Primary currency ${kpis.primaryCurrency}`}
          accent="sky"
          icon={<Wallet className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
        <KpiCard
          label="Open Receivables"
          value={kpis.outstandingFormatted}
          subtitle="Awaiting collection"
          accent="amber"
          icon={<Clock className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
        <KpiCard
          label="Issued & Collectible"
          value={kpis.issuedCount}
          subtitle={canWrite ? 'Eligible for Mark Paid' : 'Read-only view'}
          accent="emerald"
          icon={<BadgeCheck className="h-4 w-4" />}
          isLoading={listQuery.isLoading}
        />
      </div>

      <VirtualizedTable
        data={filteredItems}
        columns={columns}
        isLoading={listQuery.isLoading}
        error={listQuery.error}
        onRetry={() => void listQuery.refetch()}
        emptyTitle="No customer invoices found"
        emptyDescription="Issue a customer invoice to track Accounts Receivable and record incoming collections."
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
          searchPlaceholder: 'Search invoice number, customer, currency...',
          onExportCsv: filteredItems.length > 0 ? handleExportCsv : undefined,
          filters: (
            <Select
              aria-label="Filter AR status"
              className="w-40"
              value={statusFilter}
              onValueChange={(value) => {
                setStatusFilter(value);
                setPage(1);
              }}
              options={[
                { value: 'ALL', label: 'All Statuses' },
                { value: 'Issued', label: 'Issued' },
                { value: 'Paid', label: 'Paid' },
                { value: 'Draft', label: 'Draft' },
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
        title={activeDetail ? `Customer Invoice ${activeDetail.invoiceNo}` : 'Customer Invoice'}
        description={
          activeDetail
            ? `Version v${activeDetail.rowVersion} · ${activeDetail.currencyCode}`
            : undefined
        }
        footer={
          activeDetail && canWrite ? (
            <div className="flex flex-wrap items-center justify-between gap-2">
              {activeDetail.status.toUpperCase() === 'DRAFT' ? (
                <div className="flex items-center gap-2">
                  {!confirmDelete ? (
                    <Button
                      variant="danger"
                      size="sm"
                      leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                      onClick={() => setConfirmDelete(true)}
                    >
                      Delete Draft
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
                </div>
              ) : (
                <p className="text-xs text-slate-500">
                  {activeDetail.status.toUpperCase() === 'PAID'
                    ? 'Receivable fully settled.'
                    : 'Issued invoice ready for payment collection.'}
                </p>
              )}

              {activeDetail.status.toUpperCase() === 'ISSUED' && (
                <Button
                  variant="success"
                  size="sm"
                  leftIcon={<BadgeCheck className="h-4 w-4" />}
                  isLoading={markPaidMutation.isPending}
                  onClick={() => markPaidMutation.mutate(activeDetail)}
                >
                  Mark Paid
                </Button>
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
                    tone={arStatusTone(activeDetail.status)}
                  />
                </div>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Customer</p>
                <p className="mt-1 font-medium text-slate-900">
                  {partyById.get(activeDetail.customerId)?.name ?? activeDetail.customerId}
                </p>
                <p className="font-mono text-[11px] text-slate-500">{activeDetail.customerId}</p>
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
              <h3 className="mb-2 text-sm font-semibold text-slate-900">Revenue Lines</h3>
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
                  No revenue lines attached.
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      <ArInvoiceFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        customers={parties}
        accounts={accounts}
        currencies={currencies}
      />
    </div>
  );
}
