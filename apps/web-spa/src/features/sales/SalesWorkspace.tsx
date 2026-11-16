import * as React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import {
  CheckCircle2,
  Edit3,
  PackageCheck,
  Plus,
  RefreshCw,
  ShieldAlert,
  ShoppingBag,
  Trash2,
  Truck,
} from 'lucide-react';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { ErrorState } from '../../components/ui/ErrorState';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { MoneyInput } from '../../components/ui/MoneyInput';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { applyApiErrorsToForm, ApiClientError } from '../../lib/api';
import {
  Decimal,
  formatCurrency,
  formatDate,
  formatNumber,
  multiplyDecimals,
} from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  confirmSalesOrder,
  createDelivery,
  createSalesOrder,
  deleteSalesOrder,
  getDelivery,
  getSalesOrder,
  listCustomerLookups,
  listDeliveries,
  listSalesItemLookups,
  listSalesOrders,
  listSalesWarehouseLookups,
  updateSalesOrder,
} from './api';
import {
  computeSalesOrderTotalDecimal,
  confirmSalesOrderFormSchema,
  createDeliveryFormSchema,
  salesOrderFormSchema,
  type ConfirmSalesOrderFormValues,
  type CreateDeliveryFormValues,
  type SalesOrderFormValues,
} from './schemas';
import type {
  CustomerLookupRecord,
  DeliveryRecord,
  SalesItemLookupRecord,
  SalesOrderDetailRecord,
  SalesOrderRecord,
  SalesWarehouseLookupRecord,
} from './types';

function showSalesError(err: unknown, fallback: string) {
  if (err instanceof ApiClientError) {
    toast.error(err.message || fallback, {
      description: err.correlationId ? `Correlation ID: ${err.correlationId}` : undefined,
    });
    return;
  }
  toast.error(err instanceof Error ? err.message : fallback);
}

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}...` : id;
}

const SO_STATUS_TONE: Record<string, 'neutral' | 'info' | 'success' | 'danger'> = {
  Draft: 'neutral',
  Confirmed: 'info',
  Delivered: 'success',
  Cancelled: 'danger',
};

function SalesContent() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const queryClient = useQueryClient();
  const activeOrgId = activeOrg?.id ?? 'default-org';
  const defaultCurrency = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, 'sales:write');

  const [activeTab, setActiveTab] = React.useState<'orders' | 'deliveries'>('orders');

  // List & filter states
  const [soPage, setSoPage] = React.useState(1);
  const [soStatusFilter, setSoStatusFilter] = React.useState('ALL');
  const [soSearch, setSoSearch] = React.useState('');

  const [delPage, setDelPage] = React.useState(1);
  const [delSearch, setDelSearch] = React.useState('');

  // Modal & Drawer state
  const [soDialogOpen, setSoDialogOpen] = React.useState(false);
  const [editingSo, setEditingSo] = React.useState<SalesOrderDetailRecord | null>(null);
  const [loadingEditSoId, setLoadingEditSoId] = React.useState<string | null>(null);
  const [selectedSoId, setSelectedSoId] = React.useState<string | null>(null);
  const [deletingSo, setDeletingSo] = React.useState<SalesOrderRecord | null>(null);
  const [confirmingSo, setConfirmingSo] = React.useState<SalesOrderRecord | null>(null);
  const [deliveringSo, setDeliveringSo] = React.useState<SalesOrderRecord | null>(null);

  const [selectedDeliveryId, setSelectedDeliveryId] = React.useState<string | null>(null);

  // Core Queries
  const salesOrdersQuery = useQuery({
    queryKey: ['sales-orders', activeOrgId, soPage, soStatusFilter],
    queryFn: () =>
      listSalesOrders({
        page: soPage,
        size: 50,
        status: soStatusFilter,
      }),
  });

  const deliveriesQuery = useQuery({
    queryKey: ['sales-deliveries', activeOrgId, delPage],
    queryFn: () => listDeliveries(delPage, 50),
  });

  const soDetailQuery = useQuery({
    queryKey: ['sales-orders', 'detail', activeOrgId, selectedSoId],
    queryFn: () => getSalesOrder(selectedSoId as string),
    enabled: Boolean(selectedSoId),
  });

  const deliveryDetailQuery = useQuery({
    queryKey: ['sales-deliveries', 'detail', activeOrgId, selectedDeliveryId],
    queryFn: () => getDelivery(selectedDeliveryId as string),
    enabled: Boolean(selectedDeliveryId),
  });

  // Lookup Queries
  const customersLookupQuery = useQuery({
    queryKey: ['business-partners', 'customers', activeOrgId],
    queryFn: listCustomerLookups,
    staleTime: 60_000,
  });

  const itemsLookupQuery = useQuery({
    queryKey: ['inventory-items', 'lookup', activeOrgId],
    queryFn: listSalesItemLookups,
    staleTime: 60_000,
  });

  const warehousesLookupQuery = useQuery({
    queryKey: ['warehouses', 'lookup', activeOrgId],
    queryFn: listSalesWarehouseLookups,
    staleTime: 60_000,
  });

  const customerMap = React.useMemo(() => {
    const map = new Map<string, CustomerLookupRecord>();
    for (const c of customersLookupQuery.data ?? []) map.set(c.id, c);
    return map;
  }, [customersLookupQuery.data]);

  const itemMap = React.useMemo(() => {
    const map = new Map<string, SalesItemLookupRecord>();
    for (const i of itemsLookupQuery.data ?? []) map.set(i.id, i);
    return map;
  }, [itemsLookupQuery.data]);

  const warehouseMap = React.useMemo(() => {
    const map = new Map<string, SalesWarehouseLookupRecord>();
    for (const w of warehousesLookupQuery.data ?? []) map.set(w.id, w);
    return map;
  }, [warehousesLookupQuery.data]);

  const formatCustomer = React.useCallback(
    (id: string) => {
      const c = customerMap.get(id);
      return c ? `${c.code} — ${c.name}` : shortId(id);
    },
    [customerMap]
  );

  const formatItem = React.useCallback(
    (id: string) => {
      const i = itemMap.get(id);
      return i ? `${i.code} — ${i.name}` : shortId(id);
    },
    [itemMap]
  );

  const formatWarehouse = React.useCallback(
    (id: string) => {
      const w = warehouseMap.get(id);
      return w ? `${w.code} — ${w.name}` : shortId(id);
    },
    [warehouseMap]
  );

  // Forms
  const soForm = useForm<SalesOrderFormValues>({
    resolver: zodResolver(salesOrderFormSchema),
    defaultValues: {
      soNo: '',
      customerId: '',
      orderDate: new Date().toISOString().slice(0, 10),
      currencyCode: defaultCurrency,
      lines: [{ itemId: '', orderQty: '1.00', unitPrice: '0.00' }],
    },
  });

  const soLinesArray = useFieldArray({
    control: soForm.control,
    name: 'lines',
  });

  const confirmForm = useForm<ConfirmSalesOrderFormValues>({
    resolver: zodResolver(confirmSalesOrderFormSchema),
    defaultValues: { warehouseId: '' },
  });

  const deliveryForm = useForm<CreateDeliveryFormValues>({
    resolver: zodResolver(createDeliveryFormSchema),
    defaultValues: {
      deliveryNo: '',
      warehouseId: '',
      deliveryDate: new Date().toISOString().slice(0, 10),
    },
  });

  const watchedSoLines = soForm.watch('lines');
  const watchedSoCurrency = soForm.watch('currencyCode') || defaultCurrency;
  const computedSoTotal = React.useMemo(
    () => computeSalesOrderTotalDecimal(watchedSoLines ?? []),
    [watchedSoLines]
  );

  const invalidateSales = () => {
    void queryClient.invalidateQueries({ queryKey: ['sales-orders'] });
    void queryClient.invalidateQueries({ queryKey: ['sales-deliveries'] });
    void queryClient.invalidateQueries({ queryKey: ['stock-balances'] });
    void queryClient.invalidateQueries({ queryKey: ['reservations'] });
  };

  // Actions
  const openCreateSo = () => {
    setEditingSo(null);
    soForm.reset({
      soNo: `SO-${new Date().getFullYear()}-${String(Math.floor(100 + Math.random() * 900))}`,
      customerId: '',
      orderDate: new Date().toISOString().slice(0, 10),
      currencyCode: defaultCurrency,
      lines: [{ itemId: '', orderQty: '1.00', unitPrice: '0.00' }],
    });
    setSoDialogOpen(true);
  };

  const openEditSo = async (so: SalesOrderRecord) => {
    setLoadingEditSoId(so.id);
    try {
      const detail = await getSalesOrder(so.id);
      setEditingSo(detail);
      soForm.reset({
        soNo: detail.soNo,
        customerId: detail.customerId,
        orderDate: detail.orderDate,
        currencyCode: detail.currencyCode,
        lines:
          detail.lines.length > 0
            ? detail.lines.map((l) => ({
                itemId: l.itemId,
                orderQty: l.orderQty,
                unitPrice: l.unitPrice,
              }))
            : [{ itemId: '', orderQty: '1.00', unitPrice: '0.00' }],
      });
      setSoDialogOpen(true);
    } catch (err) {
      showSalesError(err, 'Failed to load sales order lines');
    } finally {
      setLoadingEditSoId(null);
    }
  };

  const openConfirmSo = (so: SalesOrderRecord) => {
    confirmForm.reset({
      warehouseId: (warehousesLookupQuery.data ?? [])[0]?.id ?? '',
    });
    setConfirmingSo(so);
  };

  const openCreateDelivery = (so: SalesOrderRecord) => {
    deliveryForm.reset({
      deliveryNo: `DO-${so.soNo}`,
      warehouseId: (warehousesLookupQuery.data ?? [])[0]?.id ?? '',
      deliveryDate: new Date().toISOString().slice(0, 10),
    });
    setDeliveringSo(so);
  };

  // Mutations
  const saveSoMutation = useMutation({
    mutationFn: (vals: SalesOrderFormValues) => {
      const totalAmount = computeSalesOrderTotalDecimal(vals.lines);
      const payload = {
        soNo: vals.soNo,
        customerId: vals.customerId,
        orderDate: vals.orderDate,
        currencyCode: vals.currencyCode,
        totalAmount,
        lines: vals.lines.map((l, idx) => ({
          lineNo: idx + 1,
          itemId: l.itemId,
          orderQty: l.orderQty,
          unitPrice: l.unitPrice,
        })),
      };
      if (editingSo) {
        return updateSalesOrder(editingSo.id, editingSo.rowVersion, payload);
      }
      return createSalesOrder(payload);
    },
    onSuccess: (saved) => {
      toast.success(editingSo ? `Sales order ${saved.soNo} updated` : `Sales order ${saved.soNo} created`);
      setSoDialogOpen(false);
      setEditingSo(null);
      invalidateSales();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, soForm.setError) === 0) {
        showSalesError(err, 'Failed to save sales order');
      }
    },
  });

  const deleteSoMutation = useMutation({
    mutationFn: (so: SalesOrderRecord) => deleteSalesOrder(so.id, so.rowVersion),
    onSuccess: () => {
      toast.success('Draft sales order deleted');
      setDeletingSo(null);
      setSelectedSoId(null);
      invalidateSales();
    },
    onError: (err) => showSalesError(err, 'Failed to delete sales order'),
  });

  const confirmSoMutation = useMutation({
    mutationFn: ({ so, warehouseId }: { so: SalesOrderRecord; warehouseId?: string }) =>
      confirmSalesOrder(so.id, so.rowVersion, {
        warehouseId: warehouseId?.trim() || undefined,
      }),
    onSuccess: (updated) => {
      toast.success(`Sales order ${updated.soNo} confirmed`);
      setConfirmingSo(null);
      invalidateSales();
    },
    onError: (err) => showSalesError(err, 'Failed to confirm sales order'),
  });

  const createDeliveryMutation = useMutation({
    mutationFn: ({
      so,
      vals,
    }: {
      so: SalesOrderRecord;
      vals: CreateDeliveryFormValues;
    }) =>
      createDelivery(so.id, so.rowVersion, {
        deliveryNo: vals.deliveryNo?.trim() || undefined,
        warehouseId: vals.warehouseId?.trim() || undefined,
        deliveryDate: vals.deliveryDate,
      }),
    onSuccess: (del) => {
      toast.success(`Delivery ${del.deliveryNo} created`);
      setDeliveringSo(null);
      setSelectedSoId(null);
      invalidateSales();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, deliveryForm.setError) === 0) {
        showSalesError(err, 'Failed to create delivery');
      }
    },
  });

  // Filtered rows & KPIs
  const rawOrders = React.useMemo(
    () => salesOrdersQuery.data?.items ?? [],
    [salesOrdersQuery.data?.items]
  );
  const filteredOrders = React.useMemo(() => {
    const q = soSearch.trim().toLowerCase();
    if (!q) return rawOrders;
    return rawOrders.filter(
      (so) =>
        so.soNo.toLowerCase().includes(q) ||
        formatCustomer(so.customerId).toLowerCase().includes(q) ||
        so.status.toLowerCase().includes(q)
    );
  }, [rawOrders, soSearch, formatCustomer]);

  const rawDeliveries = React.useMemo(
    () => deliveriesQuery.data?.items ?? [],
    [deliveriesQuery.data?.items]
  );
  const filteredDeliveries = React.useMemo(() => {
    const q = delSearch.trim().toLowerCase();
    if (!q) return rawDeliveries;
    return rawDeliveries.filter(
      (d) =>
        d.deliveryNo.toLowerCase().includes(q) ||
        formatWarehouse(d.warehouseId).toLowerCase().includes(q) ||
        d.salesOrderId.toLowerCase().includes(q)
    );
  }, [rawDeliveries, delSearch, formatWarehouse]);

  const totalSalesPipeline = React.useMemo(
    () =>
      Decimal.sum(
        rawOrders.map((o) => o.totalAmount || '0'),
        2
      ).toRawString(),
    [rawOrders]
  );
  const draftCount = rawOrders.filter((o) => o.status === 'Draft').length;
  const confirmedCount = rawOrders.filter((o) => o.status === 'Confirmed').length;

  const customerOptions = React.useMemo(
    () =>
      (customersLookupQuery.data ?? []).map((c) => ({
        value: c.id,
        label: `${c.code} — ${c.name}`,
        keywords: [c.code, c.name],
      })),
    [customersLookupQuery.data]
  );

  const itemOptions = React.useMemo(
    () =>
      (itemsLookupQuery.data ?? []).map((i) => ({
        value: i.id,
        label: `${i.code} — ${i.name}`,
        subtitle: i.baseUom,
        keywords: [i.code, i.name, i.baseUom],
      })),
    [itemsLookupQuery.data]
  );

  const warehouseOptions = React.useMemo(
    () =>
      (warehousesLookupQuery.data ?? []).map((w) => ({
        value: w.id,
        label: `${w.code} — ${w.name}`,
      })),
    [warehousesLookupQuery.data]
  );

  const soColumns: ColumnDef<SalesOrderRecord>[] = [
    {
      header: 'SO Number',
      accessorKey: 'soNo',
      width: '15%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.soNo}
        </span>
      ),
    },
    {
      header: 'Customer',
      accessorKey: 'customerId',
      width: '25%',
      cell: (row) => <span className="font-medium">{formatCustomer(row.customerId)}</span>,
    },
    {
      header: 'Order Date',
      accessorKey: 'orderDate',
      width: '12%',
      cell: (row) => <span className="font-mono text-xs">{formatDate(row.orderDate, locale)}</span>,
    },
    {
      header: 'Total Amount',
      accessorKey: 'totalAmount',
      width: '18%',
      align: 'right',
      cell: (row) => (
        <span className="font-money font-semibold">
          {formatCurrency(row.totalAmount, row.currencyCode, locale, 2)}
        </span>
      ),
    },
    {
      header: 'Status',
      accessorKey: 'status',
      width: '12%',
      cell: (row) => (
        <StatusBadge
          status={row.status.toUpperCase()}
          label={row.status}
          tone={SO_STATUS_TONE[row.status] ?? 'neutral'}
        />
      ),
    },
    {
      header: 'Actions',
      width: '18%',
      align: 'right',
      cell: (row) => (
        <div
          className="inline-flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canWrite && row.status === 'Draft' ? (
            <>
              <Button
                variant="outline"
                size="xs"
                aria-label={`Confirm ${row.soNo}`}
                leftIcon={<CheckCircle2 className="h-3 w-3 text-emerald-600" />}
                onClick={() => openConfirmSo(row)}
              >
                Confirm
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Edit SO ${row.soNo}`}
                isLoading={loadingEditSoId === row.id}
                onClick={() => void openEditSo(row)}
              >
                <Edit3 className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Delete SO ${row.soNo}`}
                className="text-rose-600"
                onClick={() => setDeletingSo(row)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : canWrite && row.status === 'Confirmed' ? (
            <Button
              variant="primary"
              size="xs"
              aria-label={`Deliver ${row.soNo}`}
              leftIcon={<Truck className="h-3 w-3" />}
              onClick={() => openCreateDelivery(row)}
            >
              Deliver
            </Button>
          ) : (
            <span className="text-[11px] text-slate-400">Closed</span>
          )}
        </div>
      ),
    },
  ];

  const deliveryColumns: ColumnDef<DeliveryRecord>[] = [
    {
      header: 'Delivery No',
      accessorKey: 'deliveryNo',
      width: '20%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.deliveryNo}
        </span>
      ),
    },
    {
      header: 'Sales Order',
      accessorKey: 'salesOrderId',
      width: '22%',
      cell: (row) => {
        const matched = rawOrders.find((so) => so.id === row.salesOrderId);
        return (
          <span className="font-mono text-xs">
            {matched ? matched.soNo : shortId(row.salesOrderId)}
          </span>
        );
      },
    },
    {
      header: 'Warehouse',
      accessorKey: 'warehouseId',
      width: '25%',
      cell: (row) => <span>{formatWarehouse(row.warehouseId)}</span>,
    },
    {
      header: 'Delivery Date',
      accessorKey: 'deliveryDate',
      width: '16%',
      cell: (row) => <span className="font-mono text-xs">{formatDate(row.deliveryDate, locale)}</span>,
    },
    {
      header: 'Status',
      accessorKey: 'status',
      width: '17%',
      cell: (row) => <StatusBadge status="DELIVERED" label={row.status} tone="success" />,
    },
  ];

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-600/10 text-brand-700 dark:bg-brand-500/20 dark:text-brand-300">
            <ShoppingBag className="h-5 w-5" />
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                Sales &amp; Order Fulfillment
              </h1>
              {!canWrite ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-[11px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                  <ShieldAlert className="h-3 w-3" /> Read-only
                </span>
              ) : null}
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Customer sales orders, order confirmation, inventory allocation, and outbound deliveries.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={invalidateSales}
          >
            Refresh
          </Button>
          {canWrite ? (
            <Button
              variant="primary"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={openCreateSo}
            >
              New Sales Order
            </Button>
          ) : null}
        </div>
      </header>

      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Order Book (Page)"
          value={formatCurrency(totalSalesPipeline, defaultCurrency, locale, 2)}
          subtitle={`Total orders: ${salesOrdersQuery.data?.page.totalElements ?? rawOrders.length}`}
          icon={<ShoppingBag className="h-4 w-4" />}
          accent="brand"
          isLoading={salesOrdersQuery.isLoading}
          onClick={() => setActiveTab('orders')}
        />
        <KpiCard
          label="Draft Orders"
          value={String(draftCount)}
          subtitle="Awaiting confirmation"
          icon={<Edit3 className="h-4 w-4" />}
          accent="amber"
          isLoading={salesOrdersQuery.isLoading}
          onClick={() => setActiveTab('orders')}
        />
        <KpiCard
          label="Ready to Deliver"
          value={String(confirmedCount)}
          subtitle="Confirmed customer orders"
          icon={<PackageCheck className="h-4 w-4" />}
          accent="sky"
          isLoading={salesOrdersQuery.isLoading}
          onClick={() => setActiveTab('orders')}
        />
        <KpiCard
          label="Outbound Deliveries"
          value={String(deliveriesQuery.data?.page.totalElements ?? rawDeliveries.length)}
          subtitle="Posted shipments"
          icon={<Truck className="h-4 w-4" />}
          accent="emerald"
          isLoading={deliveriesQuery.isLoading}
          onClick={() => setActiveTab('deliveries')}
        />
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'orders' | 'deliveries')}
      >
        <TabsList>
          <TabsTrigger value="orders">
            Sales Orders ({salesOrdersQuery.data?.page.totalElements ?? rawOrders.length})
          </TabsTrigger>
          <TabsTrigger value="deliveries">
            Deliveries ({deliveriesQuery.data?.page.totalElements ?? rawDeliveries.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="orders">
          <VirtualizedTable
            data={filteredOrders}
            columns={soColumns}
            getRowId={(r) => r.id}
            isLoading={salesOrdersQuery.isLoading}
            error={salesOrdersQuery.error}
            onRetry={() => void salesOrdersQuery.refetch()}
            onRowClick={(row) => setSelectedSoId(row.id)}
            emptyTitle="No Sales Orders Found"
            emptyDescription="Create a customer sales order to track fulfillment and shipping deliveries."
            onResetFilters={
              soSearch || soStatusFilter !== 'ALL'
                ? () => {
                    setSoSearch('');
                    setSoStatusFilter('ALL');
                    setSoPage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: soSearch,
              onSearchChange: setSoSearch,
              searchPlaceholder: 'Search SO number or customer...',
              filters: (
                <div className="w-40">
                  <Select
                    aria-label="Filter SO status"
                    value={soStatusFilter}
                    onValueChange={(v) => {
                      setSoStatusFilter(v);
                      setSoPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Statuses' },
                      { value: 'Draft', label: 'Draft' },
                      { value: 'Confirmed', label: 'Confirmed' },
                      { value: 'Delivered', label: 'Delivered' },
                      { value: 'Cancelled', label: 'Cancelled' },
                    ]}
                  />
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredOrders,
                  [
                    { header: 'SO Number', accessor: (r) => r.soNo },
                    { header: 'Customer', accessor: (r) => formatCustomer(r.customerId) },
                    { header: 'Order Date', accessor: (r) => r.orderDate },
                    { header: 'Currency', accessor: (r) => r.currencyCode },
                    { header: 'Total Amount', accessor: (r) => r.totalAmount },
                    { header: 'Status', accessor: (r) => r.status },
                  ],
                  'sales-orders.csv'
                ),
            }}
            pagination={
              salesOrdersQuery.data
                ? {
                    page: soPage,
                    size: salesOrdersQuery.data.page.size || 50,
                    totalElements: salesOrdersQuery.data.page.totalElements,
                    totalPages: Math.max(salesOrdersQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setSoPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="deliveries">
          <VirtualizedTable
            data={filteredDeliveries}
            columns={deliveryColumns}
            getRowId={(r) => r.id}
            isLoading={deliveriesQuery.isLoading}
            error={deliveriesQuery.error}
            onRetry={() => void deliveriesQuery.refetch()}
            onRowClick={(row) => setSelectedDeliveryId(row.id)}
            emptyTitle="No Deliveries Posted"
            emptyDescription="Confirm a Sales Order and create a delivery to ship ordered goods."
            toolbar={{
              searchValue: delSearch,
              onSearchChange: setDelSearch,
              searchPlaceholder: 'Search delivery no or warehouse...',
              onExportCsv: () =>
                exportToCsv(
                  filteredDeliveries,
                  [
                    { header: 'Delivery No', accessor: (r) => r.deliveryNo },
                    { header: 'Sales Order ID', accessor: (r) => r.salesOrderId },
                    { header: 'Warehouse', accessor: (r) => formatWarehouse(r.warehouseId) },
                    { header: 'Delivery Date', accessor: (r) => r.deliveryDate },
                    { header: 'Status', accessor: (r) => r.status },
                  ],
                  'sales-deliveries.csv'
                ),
            }}
            pagination={
              deliveriesQuery.data
                ? {
                    page: delPage,
                    size: deliveriesQuery.data.page.size || 50,
                    totalElements: deliveriesQuery.data.page.totalElements,
                    totalPages: Math.max(deliveriesQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setDelPage,
                  }
                : undefined
            }
          />
        </TabsContent>
      </Tabs>

      {/* Create / Edit Sales Order Dialog */}
      <AppDialog
        open={soDialogOpen}
        onOpenChange={setSoDialogOpen}
        title={editingSo ? `Edit Draft SO: ${editingSo.soNo}` : 'New Sales Order'}
        description="Totals are calculated automatically using fixed-point arithmetic."
        size="xl"
        footer={
          <>
            <Button variant="outline" onClick={() => setSoDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={saveSoMutation.isPending}
              onClick={soForm.handleSubmit((vals) => saveSoMutation.mutate(vals))}
            >
              {editingSo ? 'Update Sales Order' : 'Create Sales Order'}
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={soForm.handleSubmit((vals) => saveSoMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <FormField label="SO Number" required error={soForm.formState.errors.soNo?.message}>
              <Input
                disabled={Boolean(editingSo)}
                error={Boolean(soForm.formState.errors.soNo)}
                {...soForm.register('soNo')}
              />
            </FormField>
            <FormField label="Order Date" required error={soForm.formState.errors.orderDate?.message}>
              <Input
                type="date"
                error={Boolean(soForm.formState.errors.orderDate)}
                {...soForm.register('orderDate')}
              />
            </FormField>
            <FormField
              label="Currency (ISO 3)"
              required
              error={soForm.formState.errors.currencyCode?.message}
            >
              <Input
                maxLength={3}
                error={Boolean(soForm.formState.errors.currencyCode)}
                {...soForm.register('currencyCode')}
              />
            </FormField>
          </div>

          <FormField
            label="Customer"
            required
            error={soForm.formState.errors.customerId?.message}
          >
            {customerOptions.length > 0 ? (
              <Controller
                control={soForm.control}
                name="customerId"
                render={({ field }) => (
                  <Combobox
                    value={field.value}
                    onValueChange={field.onChange}
                    options={customerOptions}
                    placeholder="Select customer..."
                    error={Boolean(soForm.formState.errors.customerId)}
                  />
                )}
              />
            ) : (
              <Input
                placeholder="Paste Customer UUID..."
                error={Boolean(soForm.formState.errors.customerId)}
                {...soForm.register('customerId')}
              />
            )}
          </FormField>

          <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-800 dark:bg-slate-900/60">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                Order Line Items
              </h3>
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() => soLinesArray.append({ itemId: '', orderQty: '1.00', unitPrice: '0.00' })}
              >
                Add Line
              </Button>
            </div>

            <div className="space-y-2.5">
              {soLinesArray.fields.map((fieldItem, idx) => {
                const lineQty = watchedSoLines?.[idx]?.orderQty ?? '0';
                const linePrice = watchedSoLines?.[idx]?.unitPrice ?? '0';
                const lineSubtotal = multiplyDecimals(lineQty || '0', linePrice || '0', 2);
                const lineErr = soForm.formState.errors.lines?.[idx];
                return (
                  <div
                    key={fieldItem.id}
                    className="grid grid-cols-1 items-end gap-2 rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-800 dark:bg-slate-900 sm:grid-cols-12"
                  >
                    <FormField
                      label={`Item #${idx + 1}`}
                      required
                      className="sm:col-span-5"
                      error={lineErr?.itemId?.message}
                    >
                      {itemOptions.length > 0 ? (
                        <Controller
                          control={soForm.control}
                          name={`lines.${idx}.itemId`}
                          render={({ field }) => (
                            <Combobox
                              value={field.value}
                              onValueChange={field.onChange}
                              options={itemOptions}
                              placeholder="Select item..."
                              error={Boolean(lineErr?.itemId)}
                            />
                          )}
                        />
                      ) : (
                        <Input
                          placeholder="Item UUID..."
                          error={Boolean(lineErr?.itemId)}
                          {...soForm.register(`lines.${idx}.itemId`)}
                        />
                      )}
                    </FormField>

                    <FormField
                      label="Qty"
                      required
                      className="sm:col-span-2"
                      error={lineErr?.orderQty?.message}
                    >
                      <Controller
                        control={soForm.control}
                        name={`lines.${idx}.orderQty`}
                        render={({ field }) => (
                          <MoneyInput
                            currency=""
                            precision={2}
                            value={field.value}
                            onValueChange={field.onChange}
                            error={Boolean(lineErr?.orderQty)}
                          />
                        )}
                      />
                    </FormField>

                    <FormField
                      label="Unit Price"
                      required
                      className="sm:col-span-3"
                      error={lineErr?.unitPrice?.message}
                    >
                      <Controller
                        control={soForm.control}
                        name={`lines.${idx}.unitPrice`}
                        render={({ field }) => (
                          <MoneyInput
                            currency={watchedSoCurrency}
                            precision={2}
                            value={field.value}
                            onValueChange={field.onChange}
                            error={Boolean(lineErr?.unitPrice)}
                          />
                        )}
                      />
                    </FormField>

                    <div className="flex items-center justify-between gap-2 sm:col-span-2 sm:justify-end">
                      <span className="font-money text-xs font-semibold text-slate-600 dark:text-slate-300">
                        {formatCurrency(lineSubtotal, watchedSoCurrency, locale, 2)}
                      </span>
                      {soLinesArray.fields.length > 1 ? (
                        <Button
                          type="button"
                          variant="ghost"
                          size="xs"
                          className="text-rose-600"
                          onClick={() => soLinesArray.remove(idx)}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      ) : null}
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="flex items-center justify-between border-t border-slate-200 pt-3 dark:border-slate-800">
              <span className="text-xs font-medium text-slate-500">Exact SO Total</span>
              <span className="font-money text-base font-bold text-slate-900 dark:text-slate-100">
                {formatCurrency(computedSoTotal, watchedSoCurrency, locale, 2)}
              </span>
            </div>
          </div>
        </form>
      </AppDialog>

      {/* Confirm Sales Order Dialog */}
      <AppDialog
        open={Boolean(confirmingSo)}
        onOpenChange={(open) => {
          if (!open) setConfirmingSo(null);
        }}
        title={confirmingSo ? `Confirm Order: ${confirmingSo.soNo}` : 'Confirm Sales Order'}
        description="Lock customer order and optionally specify fulfillment warehouse."
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setConfirmingSo(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={confirmSoMutation.isPending}
              onClick={confirmForm.handleSubmit((vals) => {
                if (confirmingSo) {
                  confirmSoMutation.mutate({ so: confirmingSo, warehouseId: vals.warehouseId });
                }
              })}
            >
              Confirm Sales Order
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={confirmForm.handleSubmit((vals) => {
            if (confirmingSo) {
              confirmSoMutation.mutate({ so: confirmingSo, warehouseId: vals.warehouseId });
            }
          })}
        >
          <FormField label="Fulfillment Warehouse (Optional)">
            {warehouseOptions.length > 0 ? (
              <Controller
                control={confirmForm.control}
                name="warehouseId"
                render={({ field }) => (
                  <Select
                    value={field.value || 'NONE'}
                    onValueChange={(v) => field.onChange(v === 'NONE' ? '' : v)}
                    options={[{ value: 'NONE', label: 'Default / Specify at Delivery' }, ...warehouseOptions]}
                  />
                )}
              />
            ) : (
              <Input placeholder="Optional Warehouse UUID..." {...confirmForm.register('warehouseId')} />
            )}
          </FormField>
        </form>
      </AppDialog>

      {/* Create Delivery Dialog */}
      <AppDialog
        open={Boolean(deliveringSo)}
        onOpenChange={(open) => {
          if (!open) setDeliveringSo(null);
        }}
        title={deliveringSo ? `Create Delivery: ${deliveringSo.soNo}` : 'Create Delivery'}
        description="Dispatch goods from warehouse and transition sales order to Delivered."
        size="md"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeliveringSo(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={createDeliveryMutation.isPending}
              onClick={deliveryForm.handleSubmit((vals) => {
                if (deliveringSo) {
                  createDeliveryMutation.mutate({ so: deliveringSo, vals });
                }
              })}
            >
              Post Delivery
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={deliveryForm.handleSubmit((vals) => {
            if (deliveringSo) {
              createDeliveryMutation.mutate({ so: deliveringSo, vals });
            }
          })}
        >
          <FormField label="Dispatch Warehouse" error={deliveryForm.formState.errors.warehouseId?.message}>
            {warehouseOptions.length > 0 ? (
              <Controller
                control={deliveryForm.control}
                name="warehouseId"
                render={({ field }) => (
                  <Select
                    value={field.value || 'NONE'}
                    onValueChange={(v) => field.onChange(v === 'NONE' ? '' : v)}
                    options={[{ value: 'NONE', label: 'Use Default Warehouse' }, ...warehouseOptions]}
                  />
                )}
              />
            ) : (
              <Input placeholder="Optional Warehouse UUID..." {...deliveryForm.register('warehouseId')} />
            )}
          </FormField>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Delivery Number" error={deliveryForm.formState.errors.deliveryNo?.message}>
              <Input placeholder="DO-2026-001" {...deliveryForm.register('deliveryNo')} />
            </FormField>
            <FormField
              label="Delivery Date"
              required
              error={deliveryForm.formState.errors.deliveryDate?.message}
            >
              <Input
                type="date"
                error={Boolean(deliveryForm.formState.errors.deliveryDate)}
                {...deliveryForm.register('deliveryDate')}
              />
            </FormField>
          </div>
        </form>
      </AppDialog>

      {/* Sales Order Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedSoId)}
        onOpenChange={(open) => {
          if (!open) setSelectedSoId(null);
        }}
        title={soDetailQuery.data ? `Sales Order ${soDetailQuery.data.soNo}` : 'Sales Order Details'}
        description="Customer order summary and fulfillment lines."
        size="lg"
        footer={
          soDetailQuery.data && canWrite ? (
            <>
              {soDetailQuery.data.status === 'Draft' ? (
                <>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => setDeletingSo(soDetailQuery.data)}
                  >
                    Delete
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => void openEditSo(soDetailQuery.data)}
                  >
                    Edit Draft
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => openConfirmSo(soDetailQuery.data)}
                  >
                    Confirm Order
                  </Button>
                </>
              ) : null}
              {soDetailQuery.data.status === 'Confirmed' ? (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => openCreateDelivery(soDetailQuery.data)}
                >
                  Create Delivery
                </Button>
              ) : null}
            </>
          ) : undefined
        }
      >
        {soDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading sales order details...</p>
        ) : soDetailQuery.error ? (
          <ErrorState error={soDetailQuery.error} onRetry={() => void soDetailQuery.refetch()} />
        ) : soDetailQuery.data ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge
                    status={soDetailQuery.data.status.toUpperCase()}
                    label={soDetailQuery.data.status}
                    tone={SO_STATUS_TONE[soDetailQuery.data.status] ?? 'neutral'}
                  />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Total Amount</span>
                <p className="mt-1 font-money text-base font-bold">
                  {formatCurrency(
                    soDetailQuery.data.totalAmount,
                    soDetailQuery.data.currencyCode,
                    locale,
                    2
                  )}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Customer</span>
                <p className="mt-1 text-xs font-medium">
                  {formatCustomer(soDetailQuery.data.customerId)}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Order Date</span>
                <p className="mt-1 font-mono text-xs">{formatDate(soDetailQuery.data.orderDate, locale)}</p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
                Sales Order Lines
              </h3>
              <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
                {soDetailQuery.data.lines.map((line) => (
                  <div key={line.id} className="flex items-center justify-between px-3 py-2.5 text-xs">
                    <div>
                      <p className="font-medium text-slate-900 dark:text-slate-100">
                        #{line.lineNo} · {formatItem(line.itemId)}
                      </p>
                      <p className="text-slate-500">
                        Ordered: <span className="font-money">{formatNumber(line.orderQty, 2, locale)}</span>{' '}
                        · Delivered:{' '}
                        <span className="font-money">{formatNumber(line.deliveredQty, 2, locale)}</span>
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="font-money font-semibold">
                        {formatCurrency(
                          multiplyDecimals(line.orderQty, line.unitPrice, 2),
                          soDetailQuery.data.currencyCode,
                          locale,
                          2
                        )}
                      </p>
                      <p className="font-money text-slate-500">
                        @ {formatCurrency(line.unitPrice, soDetailQuery.data.currencyCode, locale, 2)}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Delivery Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedDeliveryId)}
        onOpenChange={(open) => {
          if (!open) setSelectedDeliveryId(null);
        }}
        title={
          deliveryDetailQuery.data ? `Delivery ${deliveryDetailQuery.data.deliveryNo}` : 'Delivery Details'
        }
        description="Outbound shipment lines."
      >
        {deliveryDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading delivery lines...</p>
        ) : deliveryDetailQuery.error ? (
          <ErrorState
            error={deliveryDetailQuery.error}
            onRetry={() => void deliveryDetailQuery.refetch()}
          />
        ) : deliveryDetailQuery.data ? (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge status="DELIVERED" label={deliveryDetailQuery.data.status} tone="success" />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Delivery Date</span>
                <p className="mt-1 font-mono text-xs">
                  {formatDate(deliveryDetailQuery.data.deliveryDate, locale)}
                </p>
              </div>
              <div className="col-span-2">
                <span className="text-xs text-slate-500">Warehouse</span>
                <p className="mt-1 text-xs font-medium">
                  {formatWarehouse(deliveryDetailQuery.data.warehouseId)}
                </p>
              </div>
            </div>

            <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
              {deliveryDetailQuery.data.lines.map((line) => (
                <div key={line.id} className="flex items-center justify-between px-3 py-2 text-xs">
                  <span>
                    #{line.lineNo} · {formatItem(line.itemId)}
                  </span>
                  <span className="font-money font-semibold">
                    {formatNumber(line.deliveredQty, 4, locale)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Delete Draft SO Confirmation */}
      <AppDialog
        open={Boolean(deletingSo)}
        onOpenChange={(open) => {
          if (!open) setDeletingSo(null);
        }}
        title="Delete Draft Sales Order"
        description={
          deletingSo ? `Delete draft sales order ${deletingSo.soNo}?` : undefined
        }
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeletingSo(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              isLoading={deleteSoMutation.isPending}
              onClick={() => deletingSo && deleteSoMutation.mutate(deletingSo)}
            >
              Delete SO
            </Button>
          </>
        }
      >
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Optimistic concurrency check: <code className="font-mono text-xs">v{deletingSo?.rowVersion}</code>.
        </p>
      </AppDialog>
    </div>
  );
}

export function SalesWorkspace() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <SalesContent />
    </DomainErrorBoundary>
  );
}
