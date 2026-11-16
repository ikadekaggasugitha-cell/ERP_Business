import * as React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import {
  CheckCircle2,
  ClipboardList,
  Edit3,
  Factory,
  Layers,
  Play,
  Plus,
  RefreshCw,
  ShieldAlert,
  Trash2,
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
import { Decimal, formatNumber, multiplyDecimals } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  completeWorkOrder,
  createBom,
  createWorkOrder,
  deleteWorkOrder,
  getBom,
  listBoms,
  listManufacturingItemLookups,
  listWorkOrders,
  releaseWorkOrder,
  updateWorkOrder,
} from './api';
import {
  bomFormSchema,
  completeWorkOrderFormSchema,
  workOrderFormSchema,
  type BomFormValues,
  type CompleteWorkOrderFormValues,
  type WorkOrderFormValues,
} from './schemas';
import type { BomRecord, ManufacturingItemLookupRecord, WorkOrderRecord } from './types';

function showManufacturingError(err: unknown, fallback: string) {
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

const WO_STATUS_TONE: Record<string, 'neutral' | 'info' | 'success'> = {
  Planned: 'neutral',
  Released: 'info',
  Completed: 'success',
};

function ManufacturingContent() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const queryClient = useQueryClient();
  const activeOrgId = activeOrg?.id ?? 'default-org';
  const canWrite = hasUserPermission(user, 'manufacturing:write');

  const [activeTab, setActiveTab] = React.useState<'work-orders' | 'boms'>('work-orders');

  // Work Orders state
  const [woPage, setWoPage] = React.useState(1);
  const [woStatusFilter, setWoStatusFilter] = React.useState('ALL');
  const [woSearch, setWoSearch] = React.useState('');
  const [woDialogOpen, setWoDialogOpen] = React.useState(false);
  const [editingWo, setEditingWo] = React.useState<WorkOrderRecord | null>(null);
  const [selectedWo, setSelectedWo] = React.useState<WorkOrderRecord | null>(null);
  const [completingWo, setCompletingWo] = React.useState<WorkOrderRecord | null>(null);
  const [deletingWo, setDeletingWo] = React.useState<WorkOrderRecord | null>(null);

  // BOMs state
  const [bomPage, setBomPage] = React.useState(1);
  const [bomSearch, setBomSearch] = React.useState('');
  const [bomDialogOpen, setBomDialogOpen] = React.useState(false);
  const [selectedBomId, setSelectedBomId] = React.useState<string | null>(null);

  // Queries
  const workOrdersQuery = useQuery({
    queryKey: ['work-orders', activeOrgId, woPage, woStatusFilter],
    queryFn: () =>
      listWorkOrders({
        page: woPage,
        size: 50,
        status: woStatusFilter,
      }),
  });

  const bomsQuery = useQuery({
    queryKey: ['boms', activeOrgId, bomPage],
    queryFn: () => listBoms(bomPage, 50),
  });

  const activeBomDetailId = selectedBomId ?? selectedWo?.bomId ?? null;
  const bomDetailQuery = useQuery({
    queryKey: ['boms', 'detail', activeOrgId, activeBomDetailId],
    queryFn: () => getBom(activeBomDetailId as string),
    enabled: Boolean(activeBomDetailId),
  });

  const itemsLookupQuery = useQuery({
    queryKey: ['inventory-items', 'lookup', activeOrgId],
    queryFn: listManufacturingItemLookups,
    staleTime: 60_000,
  });

  const itemMap = React.useMemo(() => {
    const map = new Map<string, ManufacturingItemLookupRecord>();
    for (const i of itemsLookupQuery.data ?? []) map.set(i.id, i);
    return map;
  }, [itemsLookupQuery.data]);

  const rawBoms = React.useMemo(() => bomsQuery.data?.items ?? [], [bomsQuery.data?.items]);
  const bomMap = React.useMemo(() => {
    const map = new Map<string, BomRecord>();
    for (const b of rawBoms) map.set(b.id, b);
    return map;
  }, [rawBoms]);

  const formatItem = React.useCallback(
    (id: string) => {
      const i = itemMap.get(id);
      return i ? `${i.code} — ${i.name}` : shortId(id);
    },
    [itemMap]
  );

  // Forms
  const woForm = useForm<WorkOrderFormValues>({
    resolver: zodResolver(workOrderFormSchema),
    defaultValues: {
      woNo: '',
      productItemId: '',
      bomId: '',
      plannedQty: '1.00',
      salesOrderRef: '',
    },
  });

  const completeForm = useForm<CompleteWorkOrderFormValues>({
    resolver: zodResolver(completeWorkOrderFormSchema),
    defaultValues: { producedQty: '1.00' },
  });

  const bomForm = useForm<BomFormValues>({
    resolver: zodResolver(bomFormSchema),
    defaultValues: {
      code: '',
      productItemId: '',
      version: 1,
      lines: [{ componentItemId: '', quantity: '1.00', uom: 'PCS' }],
    },
  });

  const bomLinesArray = useFieldArray({
    control: bomForm.control,
    name: 'lines',
  });

  const invalidateManufacturing = () => {
    void queryClient.invalidateQueries({ queryKey: ['work-orders'] });
    void queryClient.invalidateQueries({ queryKey: ['boms'] });
    void queryClient.invalidateQueries({ queryKey: ['stock-balances'] });
    void queryClient.invalidateQueries({ queryKey: ['stock-movements'] });
  };

  // Mutations
  const saveWoMutation = useMutation({
    mutationFn: (vals: WorkOrderFormValues) => {
      const payload = {
        woNo: vals.woNo,
        productItemId: vals.productItemId,
        bomId: vals.bomId,
        plannedQty: vals.plannedQty,
        salesOrderRef: vals.salesOrderRef?.trim() || undefined,
      };
      if (editingWo) {
        return updateWorkOrder(editingWo.id, editingWo.rowVersion, payload);
      }
      return createWorkOrder(payload);
    },
    onSuccess: (saved) => {
      toast.success(editingWo ? `Work order ${saved.woNo} updated` : `Work order ${saved.woNo} created`);
      setWoDialogOpen(false);
      setEditingWo(null);
      invalidateManufacturing();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, woForm.setError) === 0) {
        showManufacturingError(err, 'Failed to save work order');
      }
    },
  });

  const deleteWoMutation = useMutation({
    mutationFn: (wo: WorkOrderRecord) => deleteWorkOrder(wo.id, wo.rowVersion),
    onSuccess: () => {
      toast.success('Planned work order deleted');
      setDeletingWo(null);
      setSelectedWo(null);
      invalidateManufacturing();
    },
    onError: (err) => showManufacturingError(err, 'Failed to delete work order'),
  });

  const releaseWoMutation = useMutation({
    mutationFn: (wo: WorkOrderRecord) => releaseWorkOrder(wo.id, wo.rowVersion),
    onSuccess: (updated) => {
      toast.success(`Work order ${updated.woNo} released to shop floor`);
      setSelectedWo(null);
      invalidateManufacturing();
    },
    onError: (err) => showManufacturingError(err, 'Failed to release work order'),
  });

  const completeWoMutation = useMutation({
    mutationFn: ({ wo, producedQty }: { wo: WorkOrderRecord; producedQty: string }) =>
      completeWorkOrder(wo.id, wo.rowVersion, { producedQty }),
    onSuccess: (updated) => {
      toast.success(`Work order ${updated.woNo} completed`);
      setCompletingWo(null);
      setSelectedWo(null);
      completeForm.reset();
      invalidateManufacturing();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, completeForm.setError) === 0) {
        showManufacturingError(err, 'Failed to complete work order');
      }
    },
  });

  const createBomMutation = useMutation({
    mutationFn: (vals: BomFormValues) =>
      createBom({
        code: vals.code,
        productItemId: vals.productItemId,
        version: vals.version,
        lines: vals.lines.map((l, idx) => ({
          lineNo: idx + 1,
          componentItemId: l.componentItemId,
          quantity: l.quantity,
          uom: l.uom,
        })),
      }),
    onSuccess: (created) => {
      toast.success(`BOM ${created.code} v${created.version} created`);
      setBomDialogOpen(false);
      bomForm.reset();
      invalidateManufacturing();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, bomForm.setError) === 0) {
        showManufacturingError(err, 'Failed to create bill of materials');
      }
    },
  });

  const openCreateWo = () => {
    setEditingWo(null);
    woForm.reset({
      woNo: `WO-${new Date().getFullYear()}-${String(Math.floor(100 + Math.random() * 900))}`,
      productItemId: rawBoms[0]?.productItemId ?? '',
      bomId: rawBoms[0]?.id ?? '',
      plannedQty: '1.00',
      salesOrderRef: '',
    });
    setWoDialogOpen(true);
  };

  const openEditWo = (wo: WorkOrderRecord) => {
    setEditingWo(wo);
    woForm.reset({
      woNo: wo.woNo,
      productItemId: wo.productItemId,
      bomId: wo.bomId,
      plannedQty: wo.plannedQty,
      salesOrderRef: wo.salesOrderRef ?? '',
    });
    setWoDialogOpen(true);
  };

  const openCompleteWo = (wo: WorkOrderRecord) => {
    completeForm.reset({ producedQty: wo.plannedQty || '1.00' });
    setCompletingWo(wo);
  };

  // Filtered rows
  const rawWorkOrders = React.useMemo(
    () => workOrdersQuery.data?.items ?? [],
    [workOrdersQuery.data?.items]
  );
  const filteredWorkOrders = React.useMemo(() => {
    const q = woSearch.trim().toLowerCase();
    if (!q) return rawWorkOrders;
    return rawWorkOrders.filter(
      (wo) =>
        wo.woNo.toLowerCase().includes(q) ||
        formatItem(wo.productItemId).toLowerCase().includes(q) ||
        (wo.salesOrderRef ?? '').toLowerCase().includes(q) ||
        wo.status.toLowerCase().includes(q)
    );
  }, [rawWorkOrders, woSearch, formatItem]);

  const filteredBoms = React.useMemo(() => {
    const q = bomSearch.trim().toLowerCase();
    if (!q) return rawBoms;
    return rawBoms.filter(
      (b) =>
        b.code.toLowerCase().includes(q) ||
        formatItem(b.productItemId).toLowerCase().includes(q)
    );
  }, [rawBoms, bomSearch, formatItem]);

  // Exact Decimal KPIs
  const plannedCount = rawWorkOrders.filter((w) => w.status === 'Planned').length;
  const releasedCount = rawWorkOrders.filter((w) => w.status === 'Released').length;
  const totalProducedQty = React.useMemo(
    () =>
      Decimal.sum(
        rawWorkOrders.map((w) => w.producedQty || '0'),
        2
      ).toRawString(),
    [rawWorkOrders]
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

  const watchedWoProductId = woForm.watch('productItemId');
  const bomSelectOptions = React.useMemo(() => {
    const matching = watchedWoProductId
      ? rawBoms.filter((b) => b.productItemId === watchedWoProductId)
      : rawBoms;
    const pool = matching.length > 0 ? matching : rawBoms;
    return pool.map((b) => ({
      value: b.id,
      label: `${b.code} (v${b.version}) — ${formatItem(b.productItemId)}`,
    }));
  }, [rawBoms, watchedWoProductId, formatItem]);

  const woColumns: ColumnDef<WorkOrderRecord>[] = [
    {
      header: 'WO Number',
      accessorKey: 'woNo',
      width: '14%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.woNo}
        </span>
      ),
    },
    {
      header: 'Finished Product',
      accessorKey: 'productItemId',
      width: '25%',
      cell: (row) => <span className="font-medium">{formatItem(row.productItemId)}</span>,
    },
    {
      header: 'BOM',
      accessorKey: 'bomId',
      width: '13%',
      cell: (row) => {
        const bom = bomMap.get(row.bomId);
        return (
          <span className="font-mono text-xs text-slate-600 dark:text-slate-300">
            {bom ? `${bom.code} v${bom.version}` : shortId(row.bomId)}
          </span>
        );
      },
    },
    {
      header: 'Planned',
      accessorKey: 'plannedQty',
      width: '10%',
      align: 'right',
      cell: (row) => <span className="font-money">{formatNumber(row.plannedQty, 2, locale)}</span>,
    },
    {
      header: 'Produced',
      accessorKey: 'producedQty',
      width: '10%',
      align: 'right',
      cell: (row) => (
        <span className="font-money font-semibold text-emerald-700 dark:text-emerald-300">
          {formatNumber(row.producedQty, 2, locale)}
        </span>
      ),
    },
    {
      header: 'Status',
      accessorKey: 'status',
      width: '11%',
      cell: (row) => (
        <StatusBadge
          status={row.status.toUpperCase()}
          label={row.status}
          tone={WO_STATUS_TONE[row.status] ?? 'neutral'}
        />
      ),
    },
    {
      header: 'Actions',
      width: '17%',
      align: 'right',
      cell: (row) => (
        <div
          className="inline-flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canWrite && row.status === 'Planned' ? (
            <>
              <Button
                variant="outline"
                size="xs"
                aria-label={`Release ${row.woNo}`}
                leftIcon={<Play className="h-3 w-3 text-brand-600" />}
                isLoading={releaseWoMutation.isPending}
                onClick={() => releaseWoMutation.mutate(row)}
              >
                Release
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Edit WO ${row.woNo}`}
                onClick={() => openEditWo(row)}
              >
                <Edit3 className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Delete WO ${row.woNo}`}
                className="text-rose-600"
                onClick={() => setDeletingWo(row)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : canWrite && row.status === 'Released' ? (
            <Button
              variant="primary"
              size="xs"
              aria-label={`Complete ${row.woNo}`}
              leftIcon={<CheckCircle2 className="h-3 w-3" />}
              onClick={() => openCompleteWo(row)}
            >
              Complete
            </Button>
          ) : (
            <span className="text-[11px] text-slate-400">Closed</span>
          )}
        </div>
      ),
    },
  ];

  const bomColumns: ColumnDef<BomRecord>[] = [
    {
      header: 'BOM Code',
      accessorKey: 'code',
      width: '20%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.code}
        </span>
      ),
    },
    {
      header: 'Product Item',
      accessorKey: 'productItemId',
      width: '40%',
      cell: (row) => <span className="font-medium">{formatItem(row.productItemId)}</span>,
    },
    {
      header: 'Version',
      accessorKey: 'version',
      width: '13%',
      align: 'center',
      cell: (row) => <span className="font-mono text-xs font-semibold">v{row.version}</span>,
    },
    {
      header: 'Status',
      accessorKey: 'isActive',
      width: '14%',
      cell: (row) => (
        <StatusBadge
          status={row.isActive ? 'ACTIVE' : 'INACTIVE'}
          label={row.isActive ? 'Active' : 'Inactive'}
          tone={row.isActive ? 'success' : 'neutral'}
        />
      ),
    },
    {
      header: 'Row Ver',
      accessorKey: 'rowVersion',
      width: '13%',
      align: 'center',
      cell: (row) => <span className="font-mono text-xs text-slate-500">v{row.rowVersion}</span>,
    },
  ];

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-600/10 text-brand-700 dark:bg-brand-500/20 dark:text-brand-300">
            <Factory className="h-5 w-5" />
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                Manufacturing &amp; Production
              </h1>
              {!canWrite ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-[11px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                  <ShieldAlert className="h-3 w-3" /> Read-only
                </span>
              ) : null}
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Multi-level bills of materials, production work order release, and output completion.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={invalidateManufacturing}
          >
            Refresh
          </Button>
          {canWrite ? (
            <>
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Layers className="h-3.5 w-3.5" />}
                onClick={() => {
                  bomForm.reset({
                    code: `BOM-${String(Math.floor(100 + Math.random() * 900))}`,
                    productItemId: '',
                    version: 1,
                    lines: [{ componentItemId: '', quantity: '1.00', uom: 'PCS' }],
                  });
                  setBomDialogOpen(true);
                }}
              >
                New BOM
              </Button>
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={openCreateWo}
              >
                New Work Order
              </Button>
            </>
          ) : null}
        </div>
      </header>

      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Work Orders"
          value={String(workOrdersQuery.data?.page.totalElements ?? rawWorkOrders.length)}
          subtitle={`Planned: ${plannedCount} · Released: ${releasedCount}`}
          icon={<ClipboardList className="h-4 w-4" />}
          accent="brand"
          isLoading={workOrdersQuery.isLoading}
          onClick={() => setActiveTab('work-orders')}
        />
        <KpiCard
          label="In Production"
          value={String(releasedCount)}
          subtitle="Released to shop floor"
          icon={<Play className="h-4 w-4" />}
          accent="sky"
          isLoading={workOrdersQuery.isLoading}
          onClick={() => setActiveTab('work-orders')}
        />
        <KpiCard
          label="Total Output (Page)"
          value={formatNumber(totalProducedQty, 2, locale)}
          subtitle="Finished units completed"
          icon={<CheckCircle2 className="h-4 w-4" />}
          accent="emerald"
          isLoading={workOrdersQuery.isLoading}
          onClick={() => setActiveTab('work-orders')}
        />
        <KpiCard
          label="Bills of Materials"
          value={String(bomsQuery.data?.page.totalElements ?? rawBoms.length)}
          subtitle="Production recipes"
          icon={<Layers className="h-4 w-4" />}
          accent="amber"
          isLoading={bomsQuery.isLoading}
          onClick={() => setActiveTab('boms')}
        />
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'work-orders' | 'boms')}
      >
        <TabsList>
          <TabsTrigger value="work-orders">
            Work Orders ({workOrdersQuery.data?.page.totalElements ?? rawWorkOrders.length})
          </TabsTrigger>
          <TabsTrigger value="boms">
            Bill of Materials ({bomsQuery.data?.page.totalElements ?? rawBoms.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="work-orders">
          <VirtualizedTable
            data={filteredWorkOrders}
            columns={woColumns}
            getRowId={(r) => r.id}
            isLoading={workOrdersQuery.isLoading}
            error={workOrdersQuery.error}
            onRetry={() => void workOrdersQuery.refetch()}
            onRowClick={(row) => {
              setSelectedBomId(null);
              setSelectedWo(row);
            }}
            emptyTitle="No Work Orders Found"
            emptyDescription="Create a work order linked to an active Bill of Materials to schedule production."
            onResetFilters={
              woSearch || woStatusFilter !== 'ALL'
                ? () => {
                    setWoSearch('');
                    setWoStatusFilter('ALL');
                    setWoPage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: woSearch,
              onSearchChange: setWoSearch,
              searchPlaceholder: 'Search WO number, product, or sales ref...',
              filters: (
                <div className="w-40">
                  <Select
                    aria-label="Filter work order status"
                    value={woStatusFilter}
                    onValueChange={(v) => {
                      setWoStatusFilter(v);
                      setWoPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Statuses' },
                      { value: 'Planned', label: 'Planned' },
                      { value: 'Released', label: 'Released' },
                      { value: 'Completed', label: 'Completed' },
                    ]}
                  />
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredWorkOrders,
                  [
                    { header: 'WO Number', accessor: (r) => r.woNo },
                    { header: 'Product', accessor: (r) => formatItem(r.productItemId) },
                    { header: 'BOM ID', accessor: (r) => r.bomId },
                    { header: 'Planned Qty', accessor: (r) => r.plannedQty },
                    { header: 'Produced Qty', accessor: (r) => r.producedQty },
                    { header: 'Status', accessor: (r) => r.status },
                  ],
                  'manufacturing-work-orders.csv'
                ),
            }}
            pagination={
              workOrdersQuery.data
                ? {
                    page: woPage,
                    size: workOrdersQuery.data.page.size || 50,
                    totalElements: workOrdersQuery.data.page.totalElements,
                    totalPages: Math.max(workOrdersQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setWoPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="boms">
          <VirtualizedTable
            data={filteredBoms}
            columns={bomColumns}
            getRowId={(r) => r.id}
            isLoading={bomsQuery.isLoading}
            error={bomsQuery.error}
            onRetry={() => void bomsQuery.refetch()}
            onRowClick={(row) => {
              setSelectedWo(null);
              setSelectedBomId(row.id);
            }}
            emptyTitle="No Bills of Materials"
            emptyDescription="Define product recipes with component quantities and units of measure."
            toolbar={{
              searchValue: bomSearch,
              onSearchChange: setBomSearch,
              searchPlaceholder: 'Search BOM code or product...',
              onExportCsv: () =>
                exportToCsv(
                  filteredBoms,
                  [
                    { header: 'BOM Code', accessor: (r) => r.code },
                    { header: 'Product', accessor: (r) => formatItem(r.productItemId) },
                    { header: 'Version', accessor: (r) => r.version },
                    { header: 'Active', accessor: (r) => (r.isActive ? 'Yes' : 'No') },
                  ],
                  'manufacturing-boms.csv'
                ),
            }}
            pagination={
              bomsQuery.data
                ? {
                    page: bomPage,
                    size: bomsQuery.data.page.size || 50,
                    totalElements: bomsQuery.data.page.totalElements,
                    totalPages: Math.max(bomsQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setBomPage,
                  }
                : undefined
            }
          />
        </TabsContent>
      </Tabs>

      {/* Create / Edit Work Order Dialog */}
      <AppDialog
        open={woDialogOpen}
        onOpenChange={setWoDialogOpen}
        title={editingWo ? `Edit Work Order: ${editingWo.woNo}` : 'New Work Order'}
        description="Schedule manufacturing output against a registered Bill of Materials."
        size="md"
        footer={
          <>
            <Button variant="outline" onClick={() => setWoDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={saveWoMutation.isPending}
              onClick={woForm.handleSubmit((vals) => saveWoMutation.mutate(vals))}
            >
              {editingWo ? 'Update Work Order' : 'Create Work Order'}
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={woForm.handleSubmit((vals) => saveWoMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="WO Number" required error={woForm.formState.errors.woNo?.message}>
              <Input
                disabled={Boolean(editingWo)}
                error={Boolean(woForm.formState.errors.woNo)}
                {...woForm.register('woNo')}
              />
            </FormField>
            <FormField label="Sales Order Ref" error={woForm.formState.errors.salesOrderRef?.message}>
              <Input placeholder="Optional SO ref..." {...woForm.register('salesOrderRef')} />
            </FormField>
          </div>

          <FormField
            label="Finished Product Item"
            required
            error={woForm.formState.errors.productItemId?.message}
          >
            {itemOptions.length > 0 ? (
              <Controller
                control={woForm.control}
                name="productItemId"
                render={({ field }) => (
                  <Combobox
                    value={field.value}
                    onValueChange={(nextProdId) => {
                      field.onChange(nextProdId);
                      const matchedBom = rawBoms.find((b) => b.productItemId === nextProdId);
                      if (matchedBom) {
                        woForm.setValue('bomId', matchedBom.id);
                      }
                    }}
                    options={itemOptions}
                    placeholder="Select product item..."
                    error={Boolean(woForm.formState.errors.productItemId)}
                  />
                )}
              />
            ) : (
              <Input
                placeholder="Paste Product Item UUID..."
                error={Boolean(woForm.formState.errors.productItemId)}
                {...woForm.register('productItemId')}
              />
            )}
          </FormField>

          <FormField
            label="Bill of Materials (BOM)"
            required
            error={woForm.formState.errors.bomId?.message}
          >
            {bomSelectOptions.length > 0 ? (
              <Controller
                control={woForm.control}
                name="bomId"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    placeholder="Select BOM..."
                    options={bomSelectOptions}
                    error={Boolean(woForm.formState.errors.bomId)}
                  />
                )}
              />
            ) : (
              <Input
                placeholder="Paste BOM UUID..."
                error={Boolean(woForm.formState.errors.bomId)}
                {...woForm.register('bomId')}
              />
            )}
          </FormField>

          <FormField
            label="Planned Quantity"
            required
            error={woForm.formState.errors.plannedQty?.message}
          >
            <Controller
              control={woForm.control}
              name="plannedQty"
              render={({ field }) => (
                <MoneyInput
                  currency=""
                  precision={2}
                  value={field.value}
                  onValueChange={field.onChange}
                  error={Boolean(woForm.formState.errors.plannedQty)}
                />
              )}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Complete Work Order Dialog */}
      <AppDialog
        open={Boolean(completingWo)}
        onOpenChange={(open) => {
          if (!open) setCompletingWo(null);
        }}
        title={completingWo ? `Complete Work Order: ${completingWo.woNo}` : 'Complete Work Order'}
        description="Record finished output quantity to complete manufacturing."
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setCompletingWo(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={completeWoMutation.isPending}
              onClick={completeForm.handleSubmit((vals) => {
                if (completingWo) {
                  completeWoMutation.mutate({ wo: completingWo, producedQty: vals.producedQty });
                }
              })}
            >
              Complete Production
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={completeForm.handleSubmit((vals) => {
            if (completingWo) {
              completeWoMutation.mutate({ wo: completingWo, producedQty: vals.producedQty });
            }
          })}
        >
          <FormField
            label="Produced Quantity"
            required
            hint={`Planned target: ${completingWo?.plannedQty ?? '0'}`}
            error={completeForm.formState.errors.producedQty?.message}
          >
            <Controller
              control={completeForm.control}
              name="producedQty"
              render={({ field }) => (
                <MoneyInput
                  currency=""
                  precision={2}
                  value={field.value}
                  onValueChange={field.onChange}
                  error={Boolean(completeForm.formState.errors.producedQty)}
                />
              )}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Create BOM Dialog */}
      <AppDialog
        open={bomDialogOpen}
        onOpenChange={setBomDialogOpen}
        title="New Bill of Materials (BOM)"
        description="Define finished good composition and component requirements per 1 unit."
        size="xl"
        footer={
          <>
            <Button variant="outline" onClick={() => setBomDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={createBomMutation.isPending}
              onClick={bomForm.handleSubmit((vals) => createBomMutation.mutate(vals))}
            >
              Create BOM
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={bomForm.handleSubmit((vals) => createBomMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <FormField label="BOM Code" required error={bomForm.formState.errors.code?.message}>
              <Input
                error={Boolean(bomForm.formState.errors.code)}
                {...bomForm.register('code')}
              />
            </FormField>
            <FormField
              label="Finished Product"
              required
              className="sm:col-span-2"
              error={bomForm.formState.errors.productItemId?.message}
            >
              {itemOptions.length > 0 ? (
                <Controller
                  control={bomForm.control}
                  name="productItemId"
                  render={({ field }) => (
                    <Combobox
                      value={field.value}
                      onValueChange={field.onChange}
                      options={itemOptions}
                      placeholder="Select finished product..."
                      error={Boolean(bomForm.formState.errors.productItemId)}
                    />
                  )}
                />
              ) : (
                <Input
                  placeholder="Finished Product UUID..."
                  error={Boolean(bomForm.formState.errors.productItemId)}
                  {...bomForm.register('productItemId')}
                />
              )}
            </FormField>
          </div>

          <FormField label="BOM Version" required error={bomForm.formState.errors.version?.message}>
            <Input
              type="number"
              min={1}
              className="w-32"
              error={Boolean(bomForm.formState.errors.version)}
              {...bomForm.register('version', { valueAsNumber: true })}
            />
          </FormField>

          <div className="space-y-2.5 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-800 dark:bg-slate-900/60">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                BOM Component Lines
              </h3>
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() =>
                  bomLinesArray.append({ componentItemId: '', quantity: '1.00', uom: 'PCS' })
                }
              >
                Add Component
              </Button>
            </div>

            {bomLinesArray.fields.map((fieldItem, idx) => {
              const lineErr = bomForm.formState.errors.lines?.[idx];
              return (
                <div
                  key={fieldItem.id}
                  className="grid grid-cols-1 items-end gap-2 rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-800 dark:bg-slate-900 sm:grid-cols-12"
                >
                  <FormField
                    label={`Component #${idx + 1}`}
                    required
                    className="sm:col-span-6"
                    error={lineErr?.componentItemId?.message}
                  >
                    {itemOptions.length > 0 ? (
                      <Controller
                        control={bomForm.control}
                        name={`lines.${idx}.componentItemId`}
                        render={({ field }) => (
                          <Combobox
                            value={field.value}
                            onValueChange={(nextId) => {
                              field.onChange(nextId);
                              const matchedItem = itemMap.get(nextId);
                              if (matchedItem?.baseUom) {
                                bomForm.setValue(`lines.${idx}.uom`, matchedItem.baseUom);
                              }
                            }}
                            options={itemOptions}
                            placeholder="Select component item..."
                            error={Boolean(lineErr?.componentItemId)}
                          />
                        )}
                      />
                    ) : (
                      <Input
                        placeholder="Component Item UUID..."
                        error={Boolean(lineErr?.componentItemId)}
                        {...bomForm.register(`lines.${idx}.componentItemId`)}
                      />
                    )}
                  </FormField>

                  <FormField
                    label="Qty / Unit"
                    required
                    className="sm:col-span-3"
                    error={lineErr?.quantity?.message}
                  >
                    <Controller
                      control={bomForm.control}
                      name={`lines.${idx}.quantity`}
                      render={({ field }) => (
                        <MoneyInput
                          currency=""
                          precision={4}
                          value={field.value}
                          onValueChange={field.onChange}
                          error={Boolean(lineErr?.quantity)}
                        />
                      )}
                    />
                  </FormField>

                  <FormField
                    label="UOM"
                    required
                    className="sm:col-span-2"
                    error={lineErr?.uom?.message}
                  >
                    <Input
                      placeholder="PCS"
                      error={Boolean(lineErr?.uom)}
                      {...bomForm.register(`lines.${idx}.uom`)}
                    />
                  </FormField>

                  <div className="flex justify-end sm:col-span-1">
                    {bomLinesArray.fields.length > 1 ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        className="text-rose-600"
                        onClick={() => bomLinesArray.remove(idx)}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        </form>
      </AppDialog>

      {/* Work Order Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedWo)}
        onOpenChange={(open) => {
          if (!open) setSelectedWo(null);
        }}
        title={selectedWo ? `Work Order ${selectedWo.woNo}` : 'Work Order'}
        description="Production schedule and BOM component requirements."
        size="lg"
        footer={
          selectedWo && canWrite ? (
            <>
              {selectedWo.status === 'Planned' ? (
                <>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => setDeletingWo(selectedWo)}
                  >
                    Delete
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openEditWo(selectedWo)}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    isLoading={releaseWoMutation.isPending}
                    onClick={() => releaseWoMutation.mutate(selectedWo)}
                  >
                    Release to Floor
                  </Button>
                </>
              ) : null}
              {selectedWo.status === 'Released' ? (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => openCompleteWo(selectedWo)}
                >
                  Complete Production
                </Button>
              ) : null}
            </>
          ) : undefined
        }
      >
        {selectedWo ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge
                    status={selectedWo.status.toUpperCase()}
                    label={selectedWo.status}
                    tone={WO_STATUS_TONE[selectedWo.status] ?? 'neutral'}
                  />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Finished Product</span>
                <p className="mt-1 text-xs font-semibold">{formatItem(selectedWo.productItemId)}</p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Planned Qty</span>
                <p className="mt-1 font-money font-semibold">
                  {formatNumber(selectedWo.plannedQty, 2, locale)}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Produced Qty</span>
                <p className="mt-1 font-money font-semibold text-emerald-700 dark:text-emerald-300">
                  {formatNumber(selectedWo.producedQty, 2, locale)}
                </p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
                Required Material Breakdown (BOM)
              </h3>
              {bomDetailQuery.isLoading ? (
                <p className="text-xs text-slate-500">Loading BOM components...</p>
              ) : bomDetailQuery.data ? (
                <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
                  {bomDetailQuery.data.lines.map((line) => {
                    const totalReq = multiplyDecimals(selectedWo.plannedQty, line.quantity, 4);
                    return (
                      <div key={line.id} className="flex items-center justify-between px-3 py-2 text-xs">
                        <span>
                          #{line.lineNo} · {formatItem(line.componentItemId)}
                        </span>
                        <span className="font-money font-semibold">
                          {formatNumber(totalReq, 2, locale)} {line.uom}
                        </span>
                      </div>
                    );
                  })}
                </div>
              ) : null}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* BOM Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedBomId)}
        onOpenChange={(open) => {
          if (!open) setSelectedBomId(null);
        }}
        title={bomDetailQuery.data ? `BOM ${bomDetailQuery.data.code} (v${bomDetailQuery.data.version})` : 'Bill of Materials'}
        description="Component recipe per 1 finished unit."
        size="lg"
      >
        {bomDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading BOM components...</p>
        ) : bomDetailQuery.error ? (
          <ErrorState error={bomDetailQuery.error} onRetry={() => void bomDetailQuery.refetch()} />
        ) : bomDetailQuery.data ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Product</span>
                <p className="mt-1 text-xs font-semibold">
                  {formatItem(bomDetailQuery.data.productItemId)}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge
                    status={bomDetailQuery.data.isActive ? 'ACTIVE' : 'INACTIVE'}
                    label={bomDetailQuery.data.isActive ? 'Active' : 'Inactive'}
                    tone={bomDetailQuery.data.isActive ? 'success' : 'neutral'}
                  />
                </div>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
                Components ({bomDetailQuery.data.lines.length})
              </h3>
              <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
                {bomDetailQuery.data.lines.map((line) => (
                  <div key={line.id} className="flex items-center justify-between px-3 py-2.5 text-xs">
                    <div>
                      <p className="font-medium text-slate-900 dark:text-slate-100">
                        #{line.lineNo} · {formatItem(line.componentItemId)}
                      </p>
                    </div>
                    <span className="font-money font-semibold">
                      {formatNumber(line.quantity, 4, locale)} {line.uom}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Delete Work Order Confirmation */}
      <AppDialog
        open={Boolean(deletingWo)}
        onOpenChange={(open) => {
          if (!open) setDeletingWo(null);
        }}
        title="Delete Planned Work Order"
        description={deletingWo ? `Delete planned WO ${deletingWo.woNo}?` : undefined}
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeletingWo(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              isLoading={deleteWoMutation.isPending}
              onClick={() => deletingWo && deleteWoMutation.mutate(deletingWo)}
            >
              Delete WO
            </Button>
          </>
        }
      >
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Optimistic concurrency check: <code className="font-mono text-xs">v{deletingWo?.rowVersion}</code>.
        </p>
      </AppDialog>
    </div>
  );
}

export function ManufacturingWorkspace() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <ManufacturingContent />
    </DomainErrorBoundary>
  );
}
