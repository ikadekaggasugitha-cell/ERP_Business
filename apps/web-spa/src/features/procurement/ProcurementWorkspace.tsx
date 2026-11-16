import * as React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import {
  CheckCircle,
  ClipboardCheck,
  Edit3,
  FilePlus2,
  PackageCheck,
  Plus,
  RefreshCw,
  Send,
  ShieldAlert,
  ShoppingCart,
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
  approvePurchaseOrder,
  approvePurchaseRequisition,
  createPurchaseOrder,
  createPurchaseRequisition,
  deletePurchaseOrder,
  getGoodsReceipt,
  getPurchaseOrder,
  getPurchaseRequisition,
  listGoodsReceipts,
  listProcurementItemLookups,
  listPurchaseOrders,
  listPurchaseRequisitions,
  listVendorLookups,
  listWarehouseLookups,
  receivePurchaseOrder,
  submitPurchaseRequisition,
  updatePurchaseOrder,
} from './api';
import {
  computeOrderTotalDecimal,
  purchaseOrderFormSchema,
  purchaseRequisitionFormSchema,
  receivePurchaseOrderSchema,
  type PurchaseOrderFormValues,
  type PurchaseRequisitionFormValues,
  type ReceivePurchaseOrderFormValues,
} from './schemas';
import type {
  GoodsReceiptRecord,
  ProcurementItemLookupRecord,
  PurchaseOrderDetailRecord,
  PurchaseOrderRecord,
  PurchaseRequisitionRecord,
  VendorLookupRecord,
  WarehouseLookupRecord,
} from './types';

function showProcurementError(err: unknown, fallback: string) {
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

const PO_STATUS_TONE: Record<string, 'neutral' | 'info' | 'success'> = {
  Draft: 'neutral',
  Approved: 'info',
  Received: 'success',
};

const PR_STATUS_TONE: Record<string, 'neutral' | 'warning' | 'success'> = {
  Draft: 'neutral',
  Submitted: 'warning',
  Approved: 'success',
};

function ProcurementContent() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const queryClient = useQueryClient();
  const activeOrgId = activeOrg?.id ?? 'default-org';
  const defaultCurrency = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, 'procurement:write');

  const [activeTab, setActiveTab] = React.useState<'orders' | 'requisitions' | 'receipts'>(
    'orders'
  );

  // List states
  const [poPage, setPoPage] = React.useState(1);
  const [poStatusFilter, setPoStatusFilter] = React.useState('ALL');
  const [poSearch, setPoSearch] = React.useState('');

  const [prPage, setPrPage] = React.useState(1);
  const [prStatusFilter, setPrStatusFilter] = React.useState('ALL');
  const [prSearch, setPrSearch] = React.useState('');

  const [grPage, setGrPage] = React.useState(1);
  const [grSearch, setGrSearch] = React.useState('');

  // Modal & Drawer state
  const [poDialogOpen, setPoDialogOpen] = React.useState(false);
  const [editingPo, setEditingPo] = React.useState<PurchaseOrderDetailRecord | null>(null);
  const [loadingEditPoId, setLoadingEditPoId] = React.useState<string | null>(null);
  const [selectedPoId, setSelectedPoId] = React.useState<string | null>(null);
  const [deletingPo, setDeletingPo] = React.useState<PurchaseOrderRecord | null>(null);
  const [receivingPo, setReceivingPo] = React.useState<PurchaseOrderRecord | null>(null);

  const [prDialogOpen, setPrDialogOpen] = React.useState(false);
  const [selectedPrId, setSelectedPrId] = React.useState<string | null>(null);
  const [selectedGrId, setSelectedGrId] = React.useState<string | null>(null);

  // Core procurement-service queries
  const ordersQuery = useQuery({
    queryKey: ['purchase-orders', activeOrgId, poPage, poStatusFilter],
    queryFn: () =>
      listPurchaseOrders({
        page: poPage,
        size: 50,
        status: poStatusFilter,
      }),
  });

  const requisitionsQuery = useQuery({
    queryKey: ['purchase-requisitions', activeOrgId, prPage, prStatusFilter],
    queryFn: () =>
      listPurchaseRequisitions({
        page: prPage,
        size: 50,
        status: prStatusFilter,
      }),
  });

  const receiptsQuery = useQuery({
    queryKey: ['goods-receipts', activeOrgId, grPage],
    queryFn: () => listGoodsReceipts(grPage, 50),
  });

  // Detail Drawer queries
  const poDetailQuery = useQuery({
    queryKey: ['purchase-orders', 'detail', activeOrgId, selectedPoId ?? receivingPo?.id],
    queryFn: () => getPurchaseOrder((selectedPoId ?? receivingPo?.id) as string),
    enabled: Boolean(selectedPoId || receivingPo?.id),
  });

  const prDetailQuery = useQuery({
    queryKey: ['purchase-requisitions', 'detail', activeOrgId, selectedPrId],
    queryFn: () => getPurchaseRequisition(selectedPrId as string),
    enabled: Boolean(selectedPrId),
  });

  const grDetailQuery = useQuery({
    queryKey: ['goods-receipts', 'detail', activeOrgId, selectedGrId],
    queryFn: () => getGoodsReceipt(selectedGrId as string),
    enabled: Boolean(selectedGrId),
  });

  // Lookup queries
  const vendorsLookupQuery = useQuery({
    queryKey: ['business-partners', 'vendors', activeOrgId],
    queryFn: listVendorLookups,
    staleTime: 60_000,
  });

  const itemsLookupQuery = useQuery({
    queryKey: ['inventory-items', 'lookup', activeOrgId],
    queryFn: listProcurementItemLookups,
    staleTime: 60_000,
  });

  const warehousesLookupQuery = useQuery({
    queryKey: ['warehouses', 'lookup', activeOrgId],
    queryFn: listWarehouseLookups,
    staleTime: 60_000,
  });

  const vendorMap = React.useMemo(() => {
    const map = new Map<string, VendorLookupRecord>();
    for (const v of vendorsLookupQuery.data ?? []) map.set(v.id, v);
    return map;
  }, [vendorsLookupQuery.data]);

  const itemMap = React.useMemo(() => {
    const map = new Map<string, ProcurementItemLookupRecord>();
    for (const i of itemsLookupQuery.data ?? []) map.set(i.id, i);
    return map;
  }, [itemsLookupQuery.data]);

  const warehouseMap = React.useMemo(() => {
    const map = new Map<string, WarehouseLookupRecord>();
    for (const w of warehousesLookupQuery.data ?? []) map.set(w.id, w);
    return map;
  }, [warehousesLookupQuery.data]);

  const formatVendor = React.useCallback(
    (id: string) => {
      const v = vendorMap.get(id);
      return v ? `${v.code} — ${v.name}` : shortId(id);
    },
    [vendorMap]
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
  const poForm = useForm<PurchaseOrderFormValues>({
    resolver: zodResolver(purchaseOrderFormSchema),
    defaultValues: {
      poNo: '',
      vendorId: '',
      purchaseRequisitionId: '',
      orderDate: new Date().toISOString().slice(0, 10),
      currencyCode: defaultCurrency,
      lines: [{ itemId: '', orderQty: '1.00', unitPrice: '0.00' }],
    },
  });

  const poLinesArray = useFieldArray({
    control: poForm.control,
    name: 'lines',
  });

  const receiveForm = useForm<ReceivePurchaseOrderFormValues>({
    resolver: zodResolver(receivePurchaseOrderSchema),
    defaultValues: {
      grNo: '',
      warehouseId: '',
      receiptDate: new Date().toISOString().slice(0, 10),
    },
  });

  const prForm = useForm<PurchaseRequisitionFormValues>({
    resolver: zodResolver(purchaseRequisitionFormSchema),
    defaultValues: {
      prNo: '',
      neededDate: '',
      workflowRef: '',
      lines: [{ itemId: '', requestQty: '1.00', neededDate: '' }],
    },
  });

  const prLinesArray = useFieldArray({
    control: prForm.control,
    name: 'lines',
  });

  const watchedPoLines = poForm.watch('lines');
  const watchedPoCurrency = poForm.watch('currencyCode') || defaultCurrency;
  const computedPoTotal = React.useMemo(
    () => computeOrderTotalDecimal(watchedPoLines ?? []),
    [watchedPoLines]
  );

  const invalidateProcurement = () => {
    void queryClient.invalidateQueries({ queryKey: ['purchase-orders'] });
    void queryClient.invalidateQueries({ queryKey: ['purchase-requisitions'] });
    void queryClient.invalidateQueries({ queryKey: ['goods-receipts'] });
    void queryClient.invalidateQueries({ queryKey: ['stock-balances'] });
    void queryClient.invalidateQueries({ queryKey: ['stock-movements'] });
  };

  // Actions & Mutations
  const openCreatePo = () => {
    setEditingPo(null);
    poForm.reset({
      poNo: `PO-${new Date().getFullYear()}-${String(Math.floor(100 + Math.random() * 900))}`,
      vendorId: '',
      purchaseRequisitionId: '',
      orderDate: new Date().toISOString().slice(0, 10),
      currencyCode: defaultCurrency,
      lines: [{ itemId: '', orderQty: '1.00', unitPrice: '0.00' }],
    });
    setPoDialogOpen(true);
  };

  const openEditPo = async (po: PurchaseOrderRecord) => {
    setLoadingEditPoId(po.id);
    try {
      const detail = await getPurchaseOrder(po.id);
      setEditingPo(detail);
      poForm.reset({
        poNo: detail.poNo,
        vendorId: detail.vendorId,
        purchaseRequisitionId: detail.purchaseRequisitionId ?? '',
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
      setPoDialogOpen(true);
    } catch (err) {
      showProcurementError(err, 'Failed to load purchase order lines for editing');
    } finally {
      setLoadingEditPoId(null);
    }
  };

  const openReceivePo = (po: PurchaseOrderRecord) => {
    receiveForm.reset({
      grNo: `GR-${po.poNo}`,
      warehouseId: (warehousesLookupQuery.data ?? [])[0]?.id ?? '',
      receiptDate: new Date().toISOString().slice(0, 10),
    });
    setReceivingPo(po);
  };

  const savePoMutation = useMutation({
    mutationFn: async (vals: PurchaseOrderFormValues) => {
      const totalAmount = computeOrderTotalDecimal(vals.lines);
      const payload = {
        poNo: vals.poNo,
        vendorId: vals.vendorId,
        purchaseRequisitionId: vals.purchaseRequisitionId?.trim() || undefined,
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
      if (editingPo) {
        return updatePurchaseOrder(editingPo.id, editingPo.rowVersion, payload);
      }
      return createPurchaseOrder(payload);
    },
    onSuccess: (saved) => {
      toast.success(
        editingPo ? `Purchase order ${saved.poNo} updated` : `Purchase order ${saved.poNo} created`
      );
      setPoDialogOpen(false);
      setEditingPo(null);
      invalidateProcurement();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, poForm.setError) === 0) {
        showProcurementError(err, 'Failed to save purchase order');
      }
    },
  });

  const deletePoMutation = useMutation({
    mutationFn: (po: PurchaseOrderRecord) => deletePurchaseOrder(po.id, po.rowVersion),
    onSuccess: () => {
      toast.success('Draft purchase order deleted');
      setDeletingPo(null);
      setSelectedPoId(null);
      invalidateProcurement();
    },
    onError: (err) => showProcurementError(err, 'Failed to delete purchase order'),
  });

  const approvePoMutation = useMutation({
    mutationFn: (po: PurchaseOrderRecord) => approvePurchaseOrder(po.id, po.rowVersion),
    onSuccess: (updated) => {
      toast.success(`Purchase order ${updated.poNo} approved`);
      invalidateProcurement();
    },
    onError: (err) => showProcurementError(err, 'Failed to approve purchase order'),
  });

  const receivePoMutation = useMutation({
    mutationFn: ({
      po,
      vals,
    }: {
      po: PurchaseOrderRecord;
      vals: ReceivePurchaseOrderFormValues;
    }) =>
      receivePurchaseOrder(po.id, po.rowVersion, {
        grNo: vals.grNo?.trim() || undefined,
        warehouseId: vals.warehouseId,
        receiptDate: vals.receiptDate,
      }),
    onSuccess: (gr) => {
      toast.success(`Goods receipt ${gr.grNo} recorded`);
      setReceivingPo(null);
      setSelectedPoId(null);
      invalidateProcurement();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, receiveForm.setError) === 0) {
        showProcurementError(err, 'Failed to receive purchase order');
      }
    },
  });

  const createPrMutation = useMutation({
    mutationFn: (vals: PurchaseRequisitionFormValues) =>
      createPurchaseRequisition({
        prNo: vals.prNo,
        neededDate: vals.neededDate?.trim() || undefined,
        workflowRef: vals.workflowRef?.trim() || undefined,
        lines: vals.lines.map((l, idx) => ({
          lineNo: idx + 1,
          itemId: l.itemId,
          requestQty: l.requestQty,
          neededDate: l.neededDate?.trim() || undefined,
        })),
      }),
    onSuccess: (created) => {
      toast.success(`Requisition ${created.prNo} created`);
      setPrDialogOpen(false);
      prForm.reset();
      invalidateProcurement();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, prForm.setError) === 0) {
        showProcurementError(err, 'Failed to create purchase requisition');
      }
    },
  });

  const submitPrMutation = useMutation({
    mutationFn: (pr: PurchaseRequisitionRecord) => submitPurchaseRequisition(pr.id, pr.rowVersion),
    onSuccess: (updated) => {
      toast.success(`Requisition ${updated.prNo} submitted`);
      invalidateProcurement();
    },
    onError: (err) => showProcurementError(err, 'Failed to submit requisition'),
  });

  const approvePrMutation = useMutation({
    mutationFn: (pr: PurchaseRequisitionRecord) =>
      approvePurchaseRequisition(pr.id, pr.rowVersion),
    onSuccess: (updated) => {
      toast.success(`Requisition ${updated.prNo} approved`);
      invalidateProcurement();
    },
    onError: (err) => showProcurementError(err, 'Failed to approve requisition'),
  });

  // Memoized lists & lookups
  const rawOrders = React.useMemo(() => ordersQuery.data?.items ?? [], [ordersQuery.data?.items]);
  const filteredOrders = React.useMemo(() => {
    const q = poSearch.trim().toLowerCase();
    if (!q) return rawOrders;
    return rawOrders.filter(
      (po) =>
        po.poNo.toLowerCase().includes(q) ||
        formatVendor(po.vendorId).toLowerCase().includes(q) ||
        po.status.toLowerCase().includes(q)
    );
  }, [rawOrders, poSearch, formatVendor]);

  const rawRequisitions = React.useMemo(
    () => requisitionsQuery.data?.items ?? [],
    [requisitionsQuery.data?.items]
  );
  const filteredRequisitions = React.useMemo(() => {
    const q = prSearch.trim().toLowerCase();
    if (!q) return rawRequisitions;
    return rawRequisitions.filter(
      (pr) =>
        pr.prNo.toLowerCase().includes(q) ||
        (pr.workflowRef ?? '').toLowerCase().includes(q) ||
        pr.status.toLowerCase().includes(q)
    );
  }, [rawRequisitions, prSearch]);

  const rawReceipts = React.useMemo(
    () => receiptsQuery.data?.items ?? [],
    [receiptsQuery.data?.items]
  );
  const filteredReceipts = React.useMemo(() => {
    const q = grSearch.trim().toLowerCase();
    if (!q) return rawReceipts;
    return rawReceipts.filter(
      (gr) =>
        gr.grNo.toLowerCase().includes(q) ||
        formatWarehouse(gr.warehouseId).toLowerCase().includes(q) ||
        gr.purchaseOrderId.toLowerCase().includes(q)
    );
  }, [rawReceipts, grSearch, formatWarehouse]);

  // Exact Decimal KPIs
  const totalPoExposure = React.useMemo(
    () =>
      Decimal.sum(
        rawOrders.map((po) => po.totalAmount || '0'),
        2
      ).toRawString(),
    [rawOrders]
  );
  const draftPoCount = rawOrders.filter((po) => po.status === 'Draft').length;
  const approvedPoCount = rawOrders.filter((po) => po.status === 'Approved').length;

  const vendorOptions = React.useMemo(
    () =>
      (vendorsLookupQuery.data ?? []).map((v) => ({
        value: v.id,
        label: `${v.code} — ${v.name}`,
        keywords: [v.code, v.name],
      })),
    [vendorsLookupQuery.data]
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

  const approvedPrOptions = React.useMemo(
    () =>
      rawRequisitions
        .filter((pr) => pr.status === 'Approved')
        .map((pr) => ({ value: pr.id, label: `${pr.prNo} (${pr.status})` })),
    [rawRequisitions]
  );

  // Columns
  const poColumns: ColumnDef<PurchaseOrderRecord>[] = [
    {
      header: 'PO Number',
      accessorKey: 'poNo',
      width: '15%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.poNo}
        </span>
      ),
    },
    {
      header: 'Vendor',
      accessorKey: 'vendorId',
      width: '24%',
      cell: (row) => <span className="font-medium">{formatVendor(row.vendorId)}</span>,
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
          tone={PO_STATUS_TONE[row.status] ?? 'neutral'}
        />
      ),
    },
    {
      header: 'Actions',
      width: '19%',
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
                aria-label={`Approve PO ${row.poNo}`}
                isLoading={approvePoMutation.isPending}
                onClick={() => approvePoMutation.mutate(row)}
              >
                Approve
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Edit PO ${row.poNo}`}
                isLoading={loadingEditPoId === row.id}
                onClick={() => void openEditPo(row)}
              >
                <Edit3 className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Delete PO ${row.poNo}`}
                className="text-rose-600 hover:bg-rose-50 hover:text-rose-700"
                onClick={() => setDeletingPo(row)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : canWrite && row.status === 'Approved' ? (
            <Button
              variant="primary"
              size="xs"
              aria-label={`Receive ${row.poNo}`}
              leftIcon={<PackageCheck className="h-3 w-3" />}
              onClick={() => openReceivePo(row)}
            >
              Receive
            </Button>
          ) : (
            <span className="text-[11px] text-slate-400">View only</span>
          )}
        </div>
      ),
    },
  ];

  const prColumns: ColumnDef<PurchaseRequisitionRecord>[] = [
    {
      header: 'PR Number',
      accessorKey: 'prNo',
      width: '18%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.prNo}
        </span>
      ),
    },
    {
      header: 'Needed Date',
      accessorKey: 'neededDate',
      width: '15%',
      cell: (row) => (
        <span className="font-mono text-xs">
          {row.neededDate ? formatDate(row.neededDate, locale) : '—'}
        </span>
      ),
    },
    {
      header: 'Workflow Ref',
      accessorKey: 'workflowRef',
      width: '25%',
      cell: (row) => <span className="text-xs text-slate-600">{row.workflowRef || '—'}</span>,
    },
    {
      header: 'Status',
      accessorKey: 'status',
      width: '15%',
      cell: (row) => (
        <StatusBadge
          status={row.status.toUpperCase()}
          label={row.status}
          tone={PR_STATUS_TONE[row.status] ?? 'neutral'}
        />
      ),
    },
    {
      header: 'Actions',
      width: '27%',
      align: 'right',
      cell: (row) => (
        <div
          className="inline-flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canWrite && row.status === 'Draft' ? (
            <Button
              variant="outline"
              size="xs"
              aria-label={`Submit PR ${row.prNo}`}
              leftIcon={<Send className="h-3 w-3" />}
              isLoading={submitPrMutation.isPending}
              onClick={() => submitPrMutation.mutate(row)}
            >
              Submit
            </Button>
          ) : canWrite && row.status === 'Submitted' ? (
            <Button
              variant="primary"
              size="xs"
              aria-label={`Approve PR ${row.prNo}`}
              leftIcon={<CheckCircle className="h-3 w-3" />}
              isLoading={approvePrMutation.isPending}
              onClick={() => approvePrMutation.mutate(row)}
            >
              Approve
            </Button>
          ) : (
            <span className="text-[11px] text-slate-400">{row.status}</span>
          )}
        </div>
      ),
    },
  ];

  const grColumns: ColumnDef<GoodsReceiptRecord>[] = [
    {
      header: 'GR Number',
      accessorKey: 'grNo',
      width: '20%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.grNo}
        </span>
      ),
    },
    {
      header: 'Purchase Order',
      accessorKey: 'purchaseOrderId',
      width: '22%',
      cell: (row) => {
        const matched = rawOrders.find((po) => po.id === row.purchaseOrderId);
        return (
          <span className="font-mono text-xs">
            {matched ? matched.poNo : shortId(row.purchaseOrderId)}
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
      header: 'Receipt Date',
      accessorKey: 'receiptDate',
      width: '16%',
      cell: (row) => <span className="font-mono text-xs">{formatDate(row.receiptDate, locale)}</span>,
    },
    {
      header: 'Status',
      accessorKey: 'status',
      width: '17%',
      cell: (row) => <StatusBadge status="RECEIVED" label={row.status} tone="success" />,
    },
  ];

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-600/10 text-brand-700 dark:bg-brand-500/20 dark:text-brand-300">
            <ShoppingCart className="h-5 w-5" />
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                Procurement &amp; Purchasing
              </h1>
              {!canWrite ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-[11px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                  <ShieldAlert className="h-3 w-3" /> Read-only
                </span>
              ) : null}
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Purchase requisitions, supplier purchase orders, approvals, and warehouse receiving.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={invalidateProcurement}
          >
            Refresh
          </Button>
          {canWrite ? (
            <>
              <Button
                variant="outline"
                size="sm"
                leftIcon={<FilePlus2 className="h-3.5 w-3.5" />}
                onClick={() => {
                  prForm.reset({
                    prNo: `PR-${new Date().getFullYear()}-${String(Math.floor(100 + Math.random() * 900))}`,
                    neededDate: '',
                    workflowRef: '',
                    lines: [{ itemId: '', requestQty: '1.00', neededDate: '' }],
                  });
                  setPrDialogOpen(true);
                }}
              >
                New Requisition
              </Button>
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={openCreatePo}
              >
                New Purchase Order
              </Button>
            </>
          ) : null}
        </div>
      </header>

      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="PO Value (Page)"
          value={formatCurrency(totalPoExposure, defaultCurrency, locale, 2)}
          subtitle={`Total POs: ${ordersQuery.data?.page.totalElements ?? rawOrders.length}`}
          icon={<ShoppingCart className="h-4 w-4" />}
          accent="brand"
          isLoading={ordersQuery.isLoading}
          onClick={() => setActiveTab('orders')}
        />
        <KpiCard
          label="Draft POs"
          value={String(draftPoCount)}
          subtitle="Pending approval"
          icon={<Edit3 className="h-4 w-4" />}
          accent="amber"
          isLoading={ordersQuery.isLoading}
          onClick={() => setActiveTab('orders')}
        />
        <KpiCard
          label="Ready to Receive"
          value={String(approvedPoCount)}
          subtitle="Approved purchase orders"
          icon={<PackageCheck className="h-4 w-4" />}
          accent="sky"
          isLoading={ordersQuery.isLoading}
          onClick={() => setActiveTab('orders')}
        />
        <KpiCard
          label="Goods Receipts"
          value={String(receiptsQuery.data?.page.totalElements ?? rawReceipts.length)}
          subtitle={`Requisitions: ${requisitionsQuery.data?.page.totalElements ?? rawRequisitions.length}`}
          icon={<ClipboardCheck className="h-4 w-4" />}
          accent="emerald"
          isLoading={receiptsQuery.isLoading}
          onClick={() => setActiveTab('receipts')}
        />
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'orders' | 'requisitions' | 'receipts')}
      >
        <TabsList>
          <TabsTrigger value="orders">
            Purchase Orders ({ordersQuery.data?.page.totalElements ?? rawOrders.length})
          </TabsTrigger>
          <TabsTrigger value="requisitions">
            Purchase Requisitions ({requisitionsQuery.data?.page.totalElements ?? rawRequisitions.length})
          </TabsTrigger>
          <TabsTrigger value="receipts">
            Goods Receipts ({receiptsQuery.data?.page.totalElements ?? rawReceipts.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="orders">
          <VirtualizedTable
            data={filteredOrders}
            columns={poColumns}
            getRowId={(r) => r.id}
            isLoading={ordersQuery.isLoading}
            error={ordersQuery.error}
            onRetry={() => void ordersQuery.refetch()}
            onRowClick={(row) => setSelectedPoId(row.id)}
            emptyTitle="No Purchase Orders Found"
            emptyDescription="Create a supplier purchase order to track purchasing commitments and receipts."
            onResetFilters={
              poSearch || poStatusFilter !== 'ALL'
                ? () => {
                    setPoSearch('');
                    setPoStatusFilter('ALL');
                    setPoPage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: poSearch,
              onSearchChange: setPoSearch,
              searchPlaceholder: 'Search PO number or vendor...',
              filters: (
                <div className="w-40">
                  <Select
                    aria-label="Filter PO status"
                    value={poStatusFilter}
                    onValueChange={(v) => {
                      setPoStatusFilter(v);
                      setPoPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Statuses' },
                      { value: 'Draft', label: 'Draft' },
                      { value: 'Approved', label: 'Approved' },
                      { value: 'Received', label: 'Received' },
                    ]}
                  />
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredOrders,
                  [
                    { header: 'PO Number', accessor: (r) => r.poNo },
                    { header: 'Vendor', accessor: (r) => formatVendor(r.vendorId) },
                    { header: 'Order Date', accessor: (r) => r.orderDate },
                    { header: 'Currency', accessor: (r) => r.currencyCode },
                    { header: 'Total Amount', accessor: (r) => r.totalAmount },
                    { header: 'Status', accessor: (r) => r.status },
                  ],
                  'procurement-purchase-orders.csv'
                ),
            }}
            pagination={
              ordersQuery.data
                ? {
                    page: poPage,
                    size: ordersQuery.data.page.size || 50,
                    totalElements: ordersQuery.data.page.totalElements,
                    totalPages: Math.max(ordersQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setPoPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="requisitions">
          <VirtualizedTable
            data={filteredRequisitions}
            columns={prColumns}
            getRowId={(r) => r.id}
            isLoading={requisitionsQuery.isLoading}
            error={requisitionsQuery.error}
            onRetry={() => void requisitionsQuery.refetch()}
            onRowClick={(row) => setSelectedPrId(row.id)}
            emptyTitle="No Purchase Requisitions"
            emptyDescription="Submit internal material requests for review and approval before issuing POs."
            onResetFilters={
              prSearch || prStatusFilter !== 'ALL'
                ? () => {
                    setPrSearch('');
                    setPrStatusFilter('ALL');
                    setPrPage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: prSearch,
              onSearchChange: setPrSearch,
              searchPlaceholder: 'Search PR number or workflow ref...',
              filters: (
                <div className="w-40">
                  <Select
                    aria-label="Filter PR status"
                    value={prStatusFilter}
                    onValueChange={(v) => {
                      setPrStatusFilter(v);
                      setPrPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Statuses' },
                      { value: 'Draft', label: 'Draft' },
                      { value: 'Submitted', label: 'Submitted' },
                      { value: 'Approved', label: 'Approved' },
                    ]}
                  />
                </div>
              ),
            }}
            pagination={
              requisitionsQuery.data
                ? {
                    page: prPage,
                    size: requisitionsQuery.data.page.size || 50,
                    totalElements: requisitionsQuery.data.page.totalElements,
                    totalPages: Math.max(requisitionsQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setPrPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="receipts">
          <VirtualizedTable
            data={filteredReceipts}
            columns={grColumns}
            getRowId={(r) => r.id}
            isLoading={receiptsQuery.isLoading}
            error={receiptsQuery.error}
            onRetry={() => void receiptsQuery.refetch()}
            onRowClick={(row) => setSelectedGrId(row.id)}
            emptyTitle="No Goods Receipts Posted"
            emptyDescription="Approve a Purchase Order and receive goods into a warehouse to generate receipts."
            toolbar={{
              searchValue: grSearch,
              onSearchChange: setGrSearch,
              searchPlaceholder: 'Search GR number or warehouse...',
              onExportCsv: () =>
                exportToCsv(
                  filteredReceipts,
                  [
                    { header: 'GR Number', accessor: (r) => r.grNo },
                    { header: 'Purchase Order ID', accessor: (r) => r.purchaseOrderId },
                    { header: 'Warehouse', accessor: (r) => formatWarehouse(r.warehouseId) },
                    { header: 'Receipt Date', accessor: (r) => r.receiptDate },
                    { header: 'Status', accessor: (r) => r.status },
                  ],
                  'procurement-goods-receipts.csv'
                ),
            }}
            pagination={
              receiptsQuery.data
                ? {
                    page: grPage,
                    size: receiptsQuery.data.page.size || 50,
                    totalElements: receiptsQuery.data.page.totalElements,
                    totalPages: Math.max(receiptsQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setGrPage,
                  }
                : undefined
            }
          />
        </TabsContent>
      </Tabs>

      {/* Create / Edit Purchase Order Dialog */}
      <AppDialog
        open={poDialogOpen}
        onOpenChange={setPoDialogOpen}
        title={editingPo ? `Edit Draft PO: ${editingPo.poNo}` : 'New Purchase Order'}
        description="Totals are calculated with exact fixed-point decimal precision."
        size="xl"
        footer={
          <>
            <Button variant="outline" onClick={() => setPoDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={savePoMutation.isPending}
              onClick={poForm.handleSubmit((vals) => savePoMutation.mutate(vals))}
            >
              {editingPo ? 'Update Purchase Order' : 'Create Purchase Order'}
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={poForm.handleSubmit((vals) => savePoMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <FormField
              label="PO Number"
              htmlFor="po-number"
              required
              error={poForm.formState.errors.poNo?.message}
            >
              <Input
                id="po-number"
                error={Boolean(poForm.formState.errors.poNo)}
                {...poForm.register('poNo')}
              />
            </FormField>
            <FormField
              label="Order Date"
              htmlFor="po-date"
              required
              error={poForm.formState.errors.orderDate?.message}
            >
              <Input
                id="po-date"
                type="date"
                error={Boolean(poForm.formState.errors.orderDate)}
                {...poForm.register('orderDate')}
              />
            </FormField>
            <FormField
              label="Currency (ISO 3)"
              htmlFor="po-currency"
              required
              error={poForm.formState.errors.currencyCode?.message}
            >
              <Input
                id="po-currency"
                maxLength={3}
                error={Boolean(poForm.formState.errors.currencyCode)}
                {...poForm.register('currencyCode')}
              />
            </FormField>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Vendor"
              required
              error={poForm.formState.errors.vendorId?.message}
            >
              {vendorOptions.length > 0 ? (
                <Controller
                  control={poForm.control}
                  name="vendorId"
                  render={({ field }) => (
                    <Combobox
                      value={field.value}
                      onValueChange={field.onChange}
                      options={vendorOptions}
                      placeholder="Select supplier..."
                      error={Boolean(poForm.formState.errors.vendorId)}
                    />
                  )}
                />
              ) : (
                <Input
                  placeholder="Paste Vendor UUID..."
                  error={Boolean(poForm.formState.errors.vendorId)}
                  {...poForm.register('vendorId')}
                />
              )}
            </FormField>

            <FormField label="Linked Purchase Requisition (Optional)">
              {approvedPrOptions.length > 0 ? (
                <Controller
                  control={poForm.control}
                  name="purchaseRequisitionId"
                  render={({ field }) => (
                    <Select
                      value={field.value || 'NONE'}
                      onValueChange={(v) => field.onChange(v === 'NONE' ? '' : v)}
                      options={[
                        { value: 'NONE', label: 'None (Direct PO)' },
                        ...approvedPrOptions,
                      ]}
                    />
                  )}
                />
              ) : (
                <Input placeholder="Optional PR UUID..." {...poForm.register('purchaseRequisitionId')} />
              )}
            </FormField>
          </div>

          <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-800 dark:bg-slate-900/60">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                Line Items
              </h3>
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() => poLinesArray.append({ itemId: '', orderQty: '1.00', unitPrice: '0.00' })}
              >
                Add Line
              </Button>
            </div>

            <div className="space-y-2.5">
              {poLinesArray.fields.map((fieldItem, idx) => {
                const lineQty = watchedPoLines?.[idx]?.orderQty ?? '0';
                const linePrice = watchedPoLines?.[idx]?.unitPrice ?? '0';
                const lineSubtotal = multiplyDecimals(lineQty || '0', linePrice || '0', 2);
                const lineErr = poForm.formState.errors.lines?.[idx];
                return (
                  <div
                    key={fieldItem.id}
                    className="grid grid-cols-1 items-end gap-2 rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-800 dark:bg-slate-900 sm:grid-cols-12"
                  >
                    <FormField
                      label={`Line #${idx + 1} Item`}
                      required
                      className="sm:col-span-5"
                      error={lineErr?.itemId?.message}
                    >
                      {itemOptions.length > 0 ? (
                        <Controller
                          control={poForm.control}
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
                          {...poForm.register(`lines.${idx}.itemId`)}
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
                        control={poForm.control}
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
                        control={poForm.control}
                        name={`lines.${idx}.unitPrice`}
                        render={({ field }) => (
                          <MoneyInput
                            currency={watchedPoCurrency}
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
                        {formatCurrency(lineSubtotal, watchedPoCurrency, locale, 2)}
                      </span>
                      {poLinesArray.fields.length > 1 ? (
                        <Button
                          type="button"
                          variant="ghost"
                          size="xs"
                          aria-label={`Remove line ${idx + 1}`}
                          className="text-rose-600"
                          onClick={() => poLinesArray.remove(idx)}
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
              <span className="text-xs font-medium text-slate-500">Exact PO Total</span>
              <span className="font-money text-base font-bold text-slate-900 dark:text-slate-100">
                {formatCurrency(computedPoTotal, watchedPoCurrency, locale, 2)}
              </span>
            </div>
          </div>
        </form>
      </AppDialog>

      {/* Receive Goods Dialog */}
      <AppDialog
        open={Boolean(receivingPo)}
        onOpenChange={(open) => {
          if (!open) setReceivingPo(null);
        }}
        title={receivingPo ? `Receive Goods: ${receivingPo.poNo}` : 'Receive Goods'}
        description="Post warehouse goods receipt for all ordered PO lines."
        size="md"
        footer={
          <>
            <Button variant="outline" onClick={() => setReceivingPo(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={receivePoMutation.isPending}
              onClick={receiveForm.handleSubmit((vals) => {
                if (receivingPo) {
                  receivePoMutation.mutate({ po: receivingPo, vals });
                }
              })}
            >
              Confirm Goods Receipt
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={receiveForm.handleSubmit((vals) => {
            if (receivingPo) {
              receivePoMutation.mutate({ po: receivingPo, vals });
            }
          })}
        >
          <FormField
            label="Receiving Warehouse"
            required
            error={receiveForm.formState.errors.warehouseId?.message}
          >
            {warehouseOptions.length > 0 ? (
              <Controller
                control={receiveForm.control}
                name="warehouseId"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    placeholder="Select warehouse..."
                    options={warehouseOptions}
                    error={Boolean(receiveForm.formState.errors.warehouseId)}
                  />
                )}
              />
            ) : (
              <Input
                placeholder="Paste Warehouse UUID..."
                error={Boolean(receiveForm.formState.errors.warehouseId)}
                {...receiveForm.register('warehouseId')}
              />
            )}
          </FormField>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Goods Receipt No" error={receiveForm.formState.errors.grNo?.message}>
              <Input placeholder="GR-2026-001" {...receiveForm.register('grNo')} />
            </FormField>
            <FormField
              label="Receipt Date"
              required
              error={receiveForm.formState.errors.receiptDate?.message}
            >
              <Input
                type="date"
                error={Boolean(receiveForm.formState.errors.receiptDate)}
                {...receiveForm.register('receiptDate')}
              />
            </FormField>
          </div>

          {poDetailQuery.data && poDetailQuery.data.id === receivingPo?.id ? (
            <div className="space-y-1.5 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-slate-800 dark:bg-slate-900">
              <p className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                PO Ordered Quantities
              </p>
              {poDetailQuery.data.lines.map((line) => (
                <div key={line.id} className="flex justify-between text-xs">
                  <span className="truncate pr-2">
                    #{line.lineNo} · {formatItem(line.itemId)}
                  </span>
                  <span className="font-money font-semibold">
                    {formatNumber(line.orderQty, 2, locale)}
                  </span>
                </div>
              ))}
            </div>
          ) : null}
        </form>
      </AppDialog>

      {/* Create Purchase Requisition Dialog */}
      <AppDialog
        open={prDialogOpen}
        onOpenChange={setPrDialogOpen}
        title="New Purchase Requisition"
        description="Submit internal material demand lines for procurement approval."
        size="lg"
        footer={
          <>
            <Button variant="outline" onClick={() => setPrDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={createPrMutation.isPending}
              onClick={prForm.handleSubmit((vals) => createPrMutation.mutate(vals))}
            >
              Create Requisition
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={prForm.handleSubmit((vals) => createPrMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <FormField label="PR Number" required error={prForm.formState.errors.prNo?.message}>
              <Input
                error={Boolean(prForm.formState.errors.prNo)}
                {...prForm.register('prNo')}
              />
            </FormField>
            <FormField label="Needed Date">
              <Input type="date" {...prForm.register('neededDate')} />
            </FormField>
            <FormField label="Workflow Ref">
              <Input placeholder="e.g. CAPEX-Q2" {...prForm.register('workflowRef')} />
            </FormField>
          </div>

          <div className="space-y-2 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-800 dark:bg-slate-900/60">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                Requested Items
              </h3>
              <Button
                type="button"
                variant="outline"
                size="xs"
                leftIcon={<Plus className="h-3.5 w-3.5" />}
                onClick={() => prLinesArray.append({ itemId: '', requestQty: '1.00', neededDate: '' })}
              >
                Add Line
              </Button>
            </div>
            {prLinesArray.fields.map((fieldItem, idx) => {
              const lineErr = prForm.formState.errors.lines?.[idx];
              return (
                <div
                  key={fieldItem.id}
                  className="grid grid-cols-1 items-end gap-2 rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-800 dark:bg-slate-900 sm:grid-cols-12"
                >
                  <FormField
                    label={`Item #${idx + 1}`}
                    required
                    className="sm:col-span-6"
                    error={lineErr?.itemId?.message}
                  >
                    {itemOptions.length > 0 ? (
                      <Controller
                        control={prForm.control}
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
                        {...prForm.register(`lines.${idx}.itemId`)}
                      />
                    )}
                  </FormField>
                  <FormField
                    label="Qty"
                    required
                    className="sm:col-span-3"
                    error={lineErr?.requestQty?.message}
                  >
                    <Controller
                      control={prForm.control}
                      name={`lines.${idx}.requestQty`}
                      render={({ field }) => (
                        <MoneyInput
                          currency=""
                          precision={2}
                          value={field.value}
                          onValueChange={field.onChange}
                          error={Boolean(lineErr?.requestQty)}
                        />
                      )}
                    />
                  </FormField>
                  <FormField label="Line Date" className="sm:col-span-2">
                    <Input type="date" {...prForm.register(`lines.${idx}.neededDate`)} />
                  </FormField>
                  <div className="flex justify-end sm:col-span-1">
                    {prLinesArray.fields.length > 1 ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        className="text-rose-600"
                        onClick={() => prLinesArray.remove(idx)}
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

      {/* PO Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedPoId)}
        onOpenChange={(open) => {
          if (!open) setSelectedPoId(null);
        }}
        title={poDetailQuery.data ? `PO ${poDetailQuery.data.poNo}` : 'Purchase Order Details'}
        description="Supplier commitment header and line items."
        size="lg"
        footer={
          poDetailQuery.data && canWrite ? (
            <>
              {poDetailQuery.data.status === 'Draft' ? (
                <>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => setDeletingPo(poDetailQuery.data)}
                  >
                    Delete
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => void openEditPo(poDetailQuery.data)}
                  >
                    Edit Draft
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    isLoading={approvePoMutation.isPending}
                    onClick={() => approvePoMutation.mutate(poDetailQuery.data)}
                  >
                    Approve PO
                  </Button>
                </>
              ) : null}
              {poDetailQuery.data.status === 'Approved' ? (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => openReceivePo(poDetailQuery.data)}
                >
                  Receive Goods
                </Button>
              ) : null}
            </>
          ) : undefined
        }
      >
        {poDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading purchase order details...</p>
        ) : poDetailQuery.error ? (
          <ErrorState error={poDetailQuery.error} onRetry={() => void poDetailQuery.refetch()} />
        ) : poDetailQuery.data ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge
                    status={poDetailQuery.data.status.toUpperCase()}
                    label={poDetailQuery.data.status}
                    tone={PO_STATUS_TONE[poDetailQuery.data.status] ?? 'neutral'}
                  />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Total Amount</span>
                <p className="mt-1 font-money text-base font-bold">
                  {formatCurrency(
                    poDetailQuery.data.totalAmount,
                    poDetailQuery.data.currencyCode,
                    locale,
                    2
                  )}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Vendor</span>
                <p className="mt-1 text-xs font-medium">{formatVendor(poDetailQuery.data.vendorId)}</p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Order Date</span>
                <p className="mt-1 font-mono text-xs">{formatDate(poDetailQuery.data.orderDate, locale)}</p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
                Purchase Order Lines
              </h3>
              <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
                {poDetailQuery.data.lines.map((line) => (
                  <div key={line.id} className="flex items-center justify-between px-3 py-2.5 text-xs">
                    <div>
                      <p className="font-medium text-slate-900 dark:text-slate-100">
                        #{line.lineNo} · {formatItem(line.itemId)}
                      </p>
                      <p className="text-slate-500">
                        Ordered: <span className="font-money">{formatNumber(line.orderQty, 2, locale)}</span>{' '}
                        · Received:{' '}
                        <span className="font-money">{formatNumber(line.receivedQty, 2, locale)}</span>
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="font-money font-semibold">
                        {formatCurrency(
                          multiplyDecimals(line.orderQty, line.unitPrice, 2),
                          poDetailQuery.data.currencyCode,
                          locale,
                          2
                        )}
                      </p>
                      <p className="font-money text-slate-500">
                        @ {formatCurrency(line.unitPrice, poDetailQuery.data.currencyCode, locale, 2)}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* PR Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedPrId)}
        onOpenChange={(open) => {
          if (!open) setSelectedPrId(null);
        }}
        title={
          prDetailQuery.data ? `Requisition ${prDetailQuery.data.prNo}` : 'Requisition Details'
        }
        description="Requested items and approval status."
        footer={
          prDetailQuery.data && canWrite ? (
            <>
              {prDetailQuery.data.status === 'Draft' ? (
                <Button
                  variant="primary"
                  size="sm"
                  isLoading={submitPrMutation.isPending}
                  onClick={() => submitPrMutation.mutate(prDetailQuery.data)}
                >
                  Submit Requisition
                </Button>
              ) : null}
              {prDetailQuery.data.status === 'Submitted' ? (
                <Button
                  variant="primary"
                  size="sm"
                  isLoading={approvePrMutation.isPending}
                  onClick={() => approvePrMutation.mutate(prDetailQuery.data)}
                >
                  Approve Requisition
                </Button>
              ) : null}
            </>
          ) : undefined
        }
      >
        {prDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading requisition...</p>
        ) : prDetailQuery.error ? (
          <ErrorState error={prDetailQuery.error} onRetry={() => void prDetailQuery.refetch()} />
        ) : prDetailQuery.data ? (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge
                    status={prDetailQuery.data.status.toUpperCase()}
                    label={prDetailQuery.data.status}
                    tone={PR_STATUS_TONE[prDetailQuery.data.status] ?? 'neutral'}
                  />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Needed Date</span>
                <p className="mt-1 font-mono text-xs">
                  {prDetailQuery.data.neededDate
                    ? formatDate(prDetailQuery.data.neededDate, locale)
                    : '—'}
                </p>
              </div>
            </div>
            <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
              {prDetailQuery.data.lines.map((l) => (
                <div key={l.id} className="flex justify-between px-3 py-2 text-xs">
                  <span>
                    #{l.lineNo} · {formatItem(l.itemId)}
                  </span>
                  <span className="font-money font-semibold">
                    {formatNumber(l.requestQty, 2, locale)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Goods Receipt Detail SideDrawer */}
      <SideDrawer
        open={Boolean(selectedGrId)}
        onOpenChange={(open) => {
          if (!open) setSelectedGrId(null);
        }}
        title={grDetailQuery.data ? `Goods Receipt ${grDetailQuery.data.grNo}` : 'Goods Receipt'}
        description="Warehouse receiving confirmation lines."
      >
        {grDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading goods receipt...</p>
        ) : grDetailQuery.error ? (
          <ErrorState error={grDetailQuery.error} onRetry={() => void grDetailQuery.refetch()} />
        ) : grDetailQuery.data ? (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Status</span>
                <div className="mt-1">
                  <StatusBadge status="RECEIVED" label={grDetailQuery.data.status} tone="success" />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Receipt Date</span>
                <p className="mt-1 font-mono text-xs">
                  {formatDate(grDetailQuery.data.receiptDate, locale)}
                </p>
              </div>
              <div className="col-span-2">
                <span className="text-xs text-slate-500">Warehouse</span>
                <p className="mt-1 text-xs font-medium">
                  {formatWarehouse(grDetailQuery.data.warehouseId)}
                </p>
              </div>
            </div>
            <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
              {grDetailQuery.data.lines.map((l) => (
                <div key={l.id} className="flex justify-between px-3 py-2 text-xs">
                  <span>
                    Line #{l.lineNo} · PO Line {shortId(l.poLineId)}
                  </span>
                  <span className="font-money font-semibold">
                    {formatNumber(l.receivedQty, 4, locale)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Delete PO Confirmation */}
      <AppDialog
        open={Boolean(deletingPo)}
        onOpenChange={(open) => {
          if (!open) setDeletingPo(null);
        }}
        title="Delete Draft Purchase Order"
        description={
          deletingPo ? `Delete draft PO ${deletingPo.poNo}? This cannot be undone.` : undefined
        }
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeletingPo(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              isLoading={deletePoMutation.isPending}
              onClick={() => deletingPo && deletePoMutation.mutate(deletingPo)}
            >
              Delete PO
            </Button>
          </>
        }
      >
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Optimistic concurrency check: <code className="font-mono text-xs">v{deletingPo?.rowVersion}</code>.
        </p>
      </AppDialog>
    </div>
  );
}

export function ProcurementWorkspace() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <ProcurementContent />
    </DomainErrorBoundary>
  );
}
