import * as React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import {
  ArrowLeftRight,
  Boxes,
  CheckCircle2,
  ClipboardList,
  PackageCheck,
  Plus,
  RefreshCw,
  ShieldAlert,
  Undo2,
  Warehouse,
} from 'lucide-react';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
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
  createReservation,
  createStockMovement,
  createWarehouse,
  issueReservation,
  listInventoryItemLookups,
  listReservations,
  listStockBalances,
  listStockMovements,
  listWarehouses,
  releaseReservation,
} from './api';
import {
  issueReservationFormSchema,
  reservationFormSchema,
  stockMovementFormSchema,
  warehouseFormSchema,
  type IssueReservationFormValues,
  type ReservationFormValues,
  type StockMovementFormValues,
  type WarehouseFormValues,
} from './schemas';
import type {
  ItemLookupRecord,
  ReservationRecord,
  StockBalanceRecord,
  StockMovementRecord,
  WarehouseRecord,
} from './types';

function showInventoryError(err: unknown, fallback: string) {
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

const MOVEMENT_BADGE_TONE: Record<string, 'success' | 'danger' | 'warning'> = {
  Receipt: 'success',
  Issue: 'danger',
  Adjustment: 'warning',
};

const RESERVATION_BADGE_TONE: Record<string, 'warning' | 'success' | 'neutral'> = {
  Reserved: 'warning',
  Consumed: 'success',
  Released: 'neutral',
};

function InventoryContent() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const queryClient = useQueryClient();
  const activeOrgId = activeOrg?.id ?? 'default-org';
  const currencyCode = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, 'inventory:write');

  const [activeTab, setActiveTab] = React.useState<
    'balances' | 'warehouses' | 'movements' | 'reservations'
  >('balances');

  // Filters & Pagination
  const [balancePage, setBalancePage] = React.useState(1);
  const [balanceWarehouseFilter, setBalanceWarehouseFilter] = React.useState('ALL');
  const [balanceItemFilter, setBalanceItemFilter] = React.useState('');
  const [balanceSearch, setBalanceSearch] = React.useState('');

  const [warehousePage, setWarehousePage] = React.useState(1);
  const [warehouseSearch, setWarehouseSearch] = React.useState('');

  const [movementPage, setMovementPage] = React.useState(1);
  const [movementWarehouseFilter, setMovementWarehouseFilter] = React.useState('ALL');
  const [movementSearch, setMovementSearch] = React.useState('');

  const [reservationPage, setReservationPage] = React.useState(1);
  const [reservationSearch, setReservationSearch] = React.useState('');

  // Dialog & Drawer state
  const [warehouseDialogOpen, setWarehouseDialogOpen] = React.useState(false);
  const [movementDialogOpen, setMovementDialogOpen] = React.useState(false);
  const [reservationDialogOpen, setReservationDialogOpen] = React.useState(false);
  const [selectedReservation, setSelectedReservation] = React.useState<ReservationRecord | null>(
    null
  );
  const [issuingReservation, setIssuingReservation] = React.useState<ReservationRecord | null>(
    null
  );
  const [releasingReservation, setReleasingReservation] =
    React.useState<ReservationRecord | null>(null);

  // Core inventory-service queries (fail loudly with ErrorState, zero silent mocks)
  const balancesQuery = useQuery({
    queryKey: [
      'stock-balances',
      activeOrgId,
      balancePage,
      balanceWarehouseFilter,
      balanceItemFilter,
    ],
    queryFn: () =>
      listStockBalances({
        page: balancePage,
        size: 50,
        warehouseId: balanceWarehouseFilter,
        itemId: balanceItemFilter || undefined,
      }),
  });

  const warehousesQuery = useQuery({
    queryKey: ['warehouses', activeOrgId, warehousePage],
    queryFn: () => listWarehouses(warehousePage, 100),
  });

  const movementsQuery = useQuery({
    queryKey: ['stock-movements', activeOrgId, movementPage, movementWarehouseFilter],
    queryFn: () =>
      listStockMovements({
        page: movementPage,
        size: 50,
        warehouseId: movementWarehouseFilter,
      }),
  });

  const reservationsQuery = useQuery({
    queryKey: ['reservations', activeOrgId, reservationPage],
    queryFn: () => listReservations(reservationPage, 50),
  });

  // Optional item catalog enrichment for readable SKU names
  const itemLookupQuery = useQuery({
    queryKey: ['inventory-items', 'lookup', activeOrgId],
    queryFn: listInventoryItemLookups,
    staleTime: 60_000,
  });

  const warehouseMap = React.useMemo(() => {
    const map = new Map<string, WarehouseRecord>();
    for (const wh of warehousesQuery.data?.items ?? []) {
      map.set(wh.id, wh);
    }
    return map;
  }, [warehousesQuery.data?.items]);

  const itemMap = React.useMemo(() => {
    const map = new Map<string, ItemLookupRecord>();
    for (const item of itemLookupQuery.data ?? []) {
      map.set(item.id, item);
    }
    return map;
  }, [itemLookupQuery.data]);

  const formatWarehouseLabel = React.useCallback(
    (whId: string) => {
      const wh = warehouseMap.get(whId);
      return wh ? `${wh.code} — ${wh.name}` : shortId(whId);
    },
    [warehouseMap]
  );

  const formatItemLabel = React.useCallback(
    (itemId: string) => {
      const item = itemMap.get(itemId);
      return item ? `${item.code} — ${item.name}` : shortId(itemId);
    },
    [itemMap]
  );

  // Forms
  const warehouseForm = useForm<WarehouseFormValues>({
    resolver: zodResolver(warehouseFormSchema),
    defaultValues: { code: '', name: '' },
  });

  const movementForm = useForm<StockMovementFormValues>({
    resolver: zodResolver(stockMovementFormSchema),
    defaultValues: {
      mode: 'Receipt',
      movementDate: new Date().toISOString().slice(0, 10),
      warehouseId: '',
      destinationWarehouseId: '',
      itemId: '',
      quantity: '1.0000',
      unitCost: '',
      sourceDocumentRef: '',
    },
  });

  const reservationForm = useForm<ReservationFormValues>({
    resolver: zodResolver(reservationFormSchema),
    defaultValues: {
      warehouseId: '',
      itemId: '',
      reservedQty: '1.0000',
      salesOrderRef: '',
    },
  });

  const issueForm = useForm<IssueReservationFormValues>({
    resolver: zodResolver(issueReservationFormSchema),
    defaultValues: { amount: '' },
  });

  const invalidateInventory = () => {
    void queryClient.invalidateQueries({ queryKey: ['stock-balances'] });
    void queryClient.invalidateQueries({ queryKey: ['warehouses'] });
    void queryClient.invalidateQueries({ queryKey: ['stock-movements'] });
    void queryClient.invalidateQueries({ queryKey: ['reservations'] });
  };

  // Mutations
  const createWarehouseMutation = useMutation({
    mutationFn: (vals: WarehouseFormValues) =>
      createWarehouse({
        code: vals.code,
        name: vals.name,
        organizationRefId: activeOrg?.id,
      }),
    onSuccess: (created) => {
      toast.success(`Warehouse ${created.code} created`);
      setWarehouseDialogOpen(false);
      warehouseForm.reset();
      invalidateInventory();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, warehouseForm.setError) === 0) {
        showInventoryError(err, 'Failed to create warehouse');
      }
    },
  });

  const createMovementMutation = useMutation({
    mutationFn: async (vals: StockMovementFormValues) => {
      const movementDate = vals.movementDate?.trim() || undefined;
      const sourceDocumentRef = vals.sourceDocumentRef?.trim() || undefined;
      const unitCost = vals.unitCost?.trim() || undefined;

      if (vals.mode === 'Transfer') {
        const transferRef = sourceDocumentRef || `TRF-${Date.now()}`;
        await createStockMovement({
          movementDate,
          warehouseId: vals.warehouseId,
          itemId: vals.itemId,
          movementType: 'Issue',
          quantity: vals.quantity,
          unitCost,
          sourceDocumentRef: `${transferRef} (OUT)`,
        });
        try {
          return await createStockMovement({
            movementDate,
            warehouseId: vals.destinationWarehouseId,
            itemId: vals.itemId,
            movementType: 'Receipt',
            quantity: vals.quantity,
            unitCost,
            sourceDocumentRef: `${transferRef} (IN)`,
          });
        } catch (receiptErr) {
          throw new Error(
            `Source Issue succeeded for ${transferRef}, but destination Receipt failed: ${
              receiptErr instanceof Error ? receiptErr.message : 'Unknown error'
            }`
          );
        }
      }

      return createStockMovement({
        movementDate,
        warehouseId: vals.warehouseId,
        itemId: vals.itemId,
        movementType: vals.mode,
        quantity: vals.quantity,
        unitCost,
        sourceDocumentRef,
      });
    },
    onSuccess: () => {
      toast.success('Stock movement recorded');
      setMovementDialogOpen(false);
      movementForm.reset({
        mode: 'Receipt',
        movementDate: new Date().toISOString().slice(0, 10),
        warehouseId: warehousesQuery.data?.items[0]?.id ?? '',
        destinationWarehouseId: '',
        itemId: '',
        quantity: '1.0000',
        unitCost: '',
        sourceDocumentRef: '',
      });
      invalidateInventory();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, movementForm.setError) === 0) {
        showInventoryError(err, 'Failed to record stock movement');
      }
    },
  });

  const createReservationMutation = useMutation({
    mutationFn: (vals: ReservationFormValues) =>
      createReservation({
        warehouseId: vals.warehouseId,
        itemId: vals.itemId,
        reservedQty: vals.reservedQty,
        salesOrderRef: vals.salesOrderRef?.trim() || undefined,
      }),
    onSuccess: () => {
      toast.success('Stock reservation created');
      setReservationDialogOpen(false);
      reservationForm.reset();
      invalidateInventory();
    },
    onError: (err) => {
      if (applyApiErrorsToForm(err, reservationForm.setError) === 0) {
        showInventoryError(err, 'Failed to create reservation');
      }
    },
  });

  const issueReservationMutation = useMutation({
    mutationFn: ({ resv, amount }: { resv: ReservationRecord; amount?: string }) =>
      issueReservation(resv.id, resv.rowVersion, { amount }),
    onSuccess: (updated) => {
      toast.success(`Reservation updated to ${updated.status}`);
      setIssuingReservation(null);
      setSelectedReservation(null);
      issueForm.reset();
      invalidateInventory();
    },
    onError: (err) => showInventoryError(err, 'Failed to issue reserved stock'),
  });

  const releaseReservationMutation = useMutation({
    mutationFn: (resv: ReservationRecord) => releaseReservation(resv.id, resv.rowVersion),
    onSuccess: (updated) => {
      toast.success(`Reservation ${updated.status}`);
      setReleasingReservation(null);
      setSelectedReservation(null);
      invalidateInventory();
    },
    onError: (err) => showInventoryError(err, 'Failed to release reservation'),
  });

  // Filtered datasets
  const rawBalances = React.useMemo(
    () => balancesQuery.data?.items ?? [],
    [balancesQuery.data?.items]
  );
  const filteredBalances = React.useMemo(() => {
    const q = balanceSearch.trim().toLowerCase();
    if (!q) return rawBalances;
    return rawBalances.filter((row) => {
      const itemText = formatItemLabel(row.itemId).toLowerCase();
      const whText = formatWarehouseLabel(row.warehouseId).toLowerCase();
      return (
        itemText.includes(q) ||
        whText.includes(q) ||
        row.itemId.toLowerCase().includes(q) ||
        row.warehouseId.toLowerCase().includes(q)
      );
    });
  }, [rawBalances, balanceSearch, formatItemLabel, formatWarehouseLabel]);

  const rawWarehouses = React.useMemo(
    () => warehousesQuery.data?.items ?? [],
    [warehousesQuery.data?.items]
  );
  const filteredWarehouses = React.useMemo(() => {
    const q = warehouseSearch.trim().toLowerCase();
    if (!q) return rawWarehouses;
    return rawWarehouses.filter(
      (w) => w.code.toLowerCase().includes(q) || w.name.toLowerCase().includes(q)
    );
  }, [rawWarehouses, warehouseSearch]);

  const rawMovements = React.useMemo(
    () => movementsQuery.data?.items ?? [],
    [movementsQuery.data?.items]
  );
  const filteredMovements = React.useMemo(() => {
    const q = movementSearch.trim().toLowerCase();
    if (!q) return rawMovements;
    return rawMovements.filter((m) => {
      const itemText = formatItemLabel(m.itemId).toLowerCase();
      const whText = formatWarehouseLabel(m.warehouseId).toLowerCase();
      return (
        itemText.includes(q) ||
        whText.includes(q) ||
        m.movementType.toLowerCase().includes(q) ||
        (m.sourceDocumentRef ?? '').toLowerCase().includes(q)
      );
    });
  }, [rawMovements, movementSearch, formatItemLabel, formatWarehouseLabel]);

  const rawReservations = React.useMemo(
    () => reservationsQuery.data?.items ?? [],
    [reservationsQuery.data?.items]
  );
  const filteredReservations = React.useMemo(() => {
    const q = reservationSearch.trim().toLowerCase();
    if (!q) return rawReservations;
    return rawReservations.filter((r) => {
      const itemText = formatItemLabel(r.itemId).toLowerCase();
      const whText = formatWarehouseLabel(r.warehouseId).toLowerCase();
      return (
        itemText.includes(q) ||
        whText.includes(q) ||
        r.salesOrderRef.toLowerCase().includes(q) ||
        r.status.toLowerCase().includes(q)
      );
    });
  }, [rawReservations, reservationSearch, formatItemLabel, formatWarehouseLabel]);

  // Exact fixed-point Decimal KPIs
  const totalValuation = React.useMemo(() => {
    const lineValues = rawBalances.map((row) =>
      multiplyDecimals(row.onHandQty || '0', row.avgCost || '0', 4)
    );
    return Decimal.sum(lineValues, 2).toRawString();
  }, [rawBalances]);

  const totalOnHandUnits = React.useMemo(() => {
    return Decimal.sum(
      rawBalances.map((r) => r.onHandQty || '0'),
      2
    ).toRawString();
  }, [rawBalances]);

  const activeReservationsCount = rawReservations.filter((r) => r.status === 'Reserved').length;

  const warehouseSelectOptions = React.useMemo(
    () =>
      rawWarehouses.map((w) => ({
        value: w.id,
        label: `${w.code} — ${w.name}`,
      })),
    [rawWarehouses]
  );

  const itemComboboxOptions = React.useMemo(
    () =>
      (itemLookupQuery.data ?? []).map((i) => ({
        value: i.id,
        label: `${i.code} — ${i.name}`,
        subtitle: `${i.itemType.toUpperCase()} · ${i.baseUom}`,
        keywords: [i.code, i.name, i.baseUom],
      })),
    [itemLookupQuery.data]
  );

  // Table Columns
  const balanceColumns: ColumnDef<StockBalanceRecord>[] = [
    {
      header: 'Item',
      accessorKey: 'itemId',
      width: '24%',
      cell: (row) => {
        const item = itemMap.get(row.itemId);
        return (
          <div className="truncate">
            <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
              {item ? item.code : shortId(row.itemId)}
            </span>
            {item ? (
              <span className="ml-1.5 text-slate-700 dark:text-slate-300">{item.name}</span>
            ) : null}
          </div>
        );
      },
    },
    {
      header: 'Warehouse',
      accessorKey: 'warehouseId',
      width: '18%',
      cell: (row) => (
        <span className="text-slate-700 dark:text-slate-300">
          {formatWarehouseLabel(row.warehouseId)}
        </span>
      ),
    },
    {
      header: 'On Hand',
      accessorKey: 'onHandQty',
      width: '11%',
      align: 'right',
      cell: (row) => (
        <span className="font-money font-medium">{formatNumber(row.onHandQty, 2, locale)}</span>
      ),
    },
    {
      header: 'Reserved',
      accessorKey: 'reservedQty',
      width: '11%',
      align: 'right',
      cell: (row) => (
        <span className="font-money text-amber-700 dark:text-amber-300">
          {formatNumber(row.reservedQty, 2, locale)}
        </span>
      ),
    },
    {
      header: 'Available',
      accessorKey: 'availableQty',
      width: '11%',
      align: 'right',
      cell: (row) => {
        const avail = new Decimal(row.availableQty || '0');
        return (
          <span
            className={
              avail.lte(0)
                ? 'font-money font-semibold text-rose-600'
                : 'font-money font-semibold text-emerald-700 dark:text-emerald-300'
            }
          >
            {formatNumber(row.availableQty, 2, locale)}
          </span>
        );
      },
    },
    {
      header: 'Avg Cost',
      accessorKey: 'avgCost',
      width: '12%',
      align: 'right',
      cell: (row) => (
        <span className="font-money text-slate-600 dark:text-slate-300">
          {formatCurrency(row.avgCost, currencyCode, locale, 2)}
        </span>
      ),
    },
    {
      header: 'Status',
      width: '13%',
      align: 'center',
      cell: (row) => {
        const avail = new Decimal(row.availableQty || '0');
        const reserved = new Decimal(row.reservedQty || '0');
        if (avail.lte(0)) {
          return <StatusBadge status="OUT_OF_STOCK" label="Empty" tone="danger" />;
        }
        if (reserved.isPositive()) {
          return <StatusBadge status="RESERVED" label="Partial Resv" tone="warning" />;
        }
        return <StatusBadge status="ACTIVE" label="Available" tone="success" />;
      },
    },
  ];

  const warehouseColumns: ColumnDef<WarehouseRecord>[] = [
    {
      header: 'Code',
      accessorKey: 'code',
      width: '22%',
      cell: (row) => (
        <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">
          {row.code}
        </span>
      ),
    },
    {
      header: 'Warehouse Name',
      accessorKey: 'name',
      width: '44%',
      cell: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      header: 'Organization Ref',
      accessorKey: 'organizationRefId',
      width: '24%',
      cell: (row) => <span className="font-mono text-xs text-slate-500">{row.organizationRefId}</span>,
    },
    {
      header: 'Version',
      accessorKey: 'rowVersion',
      width: '10%',
      align: 'center',
      cell: (row) => <span className="font-mono text-xs text-slate-500">v{row.rowVersion}</span>,
    },
  ];

  const movementColumns: ColumnDef<StockMovementRecord>[] = [
    {
      header: 'Date',
      accessorKey: 'movementDate',
      width: '13%',
      cell: (row) => <span className="font-mono text-xs">{formatDate(row.movementDate, locale)}</span>,
    },
    {
      header: 'Type',
      accessorKey: 'movementType',
      width: '13%',
      cell: (row) => (
        <StatusBadge
          status={row.movementType.toUpperCase()}
          label={row.movementType}
          tone={MOVEMENT_BADGE_TONE[row.movementType] ?? 'neutral'}
        />
      ),
    },
    {
      header: 'Item',
      accessorKey: 'itemId',
      width: '25%',
      cell: (row) => <span>{formatItemLabel(row.itemId)}</span>,
    },
    {
      header: 'Warehouse',
      accessorKey: 'warehouseId',
      width: '20%',
      cell: (row) => <span>{formatWarehouseLabel(row.warehouseId)}</span>,
    },
    {
      header: 'Quantity',
      accessorKey: 'quantity',
      width: '13%',
      align: 'right',
      cell: (row) => (
        <span className="font-money font-semibold">{formatNumber(row.quantity, 4, locale)}</span>
      ),
    },
    {
      header: 'Source Ref',
      accessorKey: 'sourceDocumentRef',
      width: '16%',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-600 dark:text-slate-300">
          {row.sourceDocumentRef || '—'}
        </span>
      ),
    },
  ];

  const reservationColumns: ColumnDef<ReservationRecord>[] = [
    {
      header: 'Item',
      accessorKey: 'itemId',
      width: '24%',
      cell: (row) => <span>{formatItemLabel(row.itemId)}</span>,
    },
    {
      header: 'Warehouse',
      accessorKey: 'warehouseId',
      width: '19%',
      cell: (row) => <span>{formatWarehouseLabel(row.warehouseId)}</span>,
    },
    {
      header: 'Reserved Qty',
      accessorKey: 'reservedQty',
      width: '13%',
      align: 'right',
      cell: (row) => (
        <span className="font-money font-semibold">{formatNumber(row.reservedQty, 4, locale)}</span>
      ),
    },
    {
      header: 'Sales Order Ref',
      accessorKey: 'salesOrderRef',
      width: '18%',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-600 dark:text-slate-300">
          {row.salesOrderRef || '—'}
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
          tone={RESERVATION_BADGE_TONE[row.status] ?? 'neutral'}
        />
      ),
    },
    {
      header: 'Actions',
      width: '15%',
      align: 'right',
      cell: (row) => (
        <div
          className="inline-flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canWrite && row.status === 'Reserved' ? (
            <>
              <Button
                variant="outline"
                size="xs"
                aria-label={`Issue reservation ${row.salesOrderRef || row.id}`}
                leftIcon={<CheckCircle2 className="h-3 w-3 text-emerald-600" />}
                onClick={() => {
                  issueForm.reset({ amount: row.reservedQty });
                  setIssuingReservation(row);
                }}
              >
                Issue
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Release reservation ${row.salesOrderRef || row.id}`}
                leftIcon={<Undo2 className="h-3 w-3 text-amber-600" />}
                onClick={() => setReleasingReservation(row)}
              >
                Release
              </Button>
            </>
          ) : (
            <span className="text-[11px] text-slate-400">Closed</span>
          )}
        </div>
      ),
    },
  ];

  const movementMode = movementForm.watch('mode');

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-600/10 text-brand-700 dark:bg-brand-500/20 dark:text-brand-300">
            <Boxes className="h-5 w-5" />
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                Inventory &amp; Warehousing
              </h1>
              {!canWrite ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-[11px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                  <ShieldAlert className="h-3 w-3" /> Read-only
                </span>
              ) : null}
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Real-time stock balances, multi-warehouse locations, stock movements, and allocations.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={invalidateInventory}
          >
            Refresh
          </Button>
          {canWrite ? (
            <>
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Warehouse className="h-3.5 w-3.5" />}
                onClick={() => {
                  warehouseForm.reset();
                  setWarehouseDialogOpen(true);
                }}
              >
                New Warehouse
              </Button>
              <Button
                variant="outline"
                size="sm"
                leftIcon={<ClipboardList className="h-3.5 w-3.5" />}
                onClick={() => {
                  reservationForm.reset({
                    warehouseId: rawWarehouses[0]?.id ?? '',
                    itemId: '',
                    reservedQty: '1.0000',
                    salesOrderRef: '',
                  });
                  setReservationDialogOpen(true);
                }}
              >
                New Reservation
              </Button>
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => {
                  movementForm.reset({
                    mode: 'Receipt',
                    movementDate: new Date().toISOString().slice(0, 10),
                    warehouseId: rawWarehouses[0]?.id ?? '',
                    destinationWarehouseId: '',
                    itemId: '',
                    quantity: '1.0000',
                    unitCost: '',
                    sourceDocumentRef: '',
                  });
                  setMovementDialogOpen(true);
                }}
              >
                Record Movement
              </Button>
            </>
          ) : null}
        </div>
      </header>

      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Inventory Valuation"
          value={formatCurrency(totalValuation, currencyCode, locale, 2)}
          subtitle={`On-hand qty: ${formatNumber(totalOnHandUnits, 2, locale)} units`}
          icon={<Boxes className="h-4 w-4" />}
          accent="brand"
          isLoading={balancesQuery.isLoading}
          onClick={() => setActiveTab('balances')}
        />
        <KpiCard
          label="Active Warehouses"
          value={String(warehousesQuery.data?.page.totalElements ?? rawWarehouses.length)}
          subtitle="Storage locations"
          icon={<Warehouse className="h-4 w-4" />}
          accent="sky"
          isLoading={warehousesQuery.isLoading}
          onClick={() => setActiveTab('warehouses')}
        />
        <KpiCard
          label="Stock Movements"
          value={String(movementsQuery.data?.page.totalElements ?? rawMovements.length)}
          subtitle="Receipts, issues & adjustments"
          icon={<ArrowLeftRight className="h-4 w-4" />}
          accent="emerald"
          isLoading={movementsQuery.isLoading}
          onClick={() => setActiveTab('movements')}
        />
        <KpiCard
          label="Open Reservations"
          value={String(activeReservationsCount)}
          subtitle={`Total records: ${reservationsQuery.data?.page.totalElements ?? rawReservations.length}`}
          icon={<PackageCheck className="h-4 w-4" />}
          accent="amber"
          isLoading={reservationsQuery.isLoading}
          onClick={() => setActiveTab('reservations')}
        />
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v) =>
          setActiveTab(v as 'balances' | 'warehouses' | 'movements' | 'reservations')
        }
      >
        <TabsList>
          <TabsTrigger value="balances">
            Stock Balances ({balancesQuery.data?.page.totalElements ?? rawBalances.length})
          </TabsTrigger>
          <TabsTrigger value="warehouses">
            Warehouses ({warehousesQuery.data?.page.totalElements ?? rawWarehouses.length})
          </TabsTrigger>
          <TabsTrigger value="movements">
            Stock Movements ({movementsQuery.data?.page.totalElements ?? rawMovements.length})
          </TabsTrigger>
          <TabsTrigger value="reservations">
            Reservations ({reservationsQuery.data?.page.totalElements ?? rawReservations.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="balances">
          <VirtualizedTable
            data={filteredBalances}
            columns={balanceColumns}
            getRowId={(r) => r.id}
            isLoading={balancesQuery.isLoading}
            error={balancesQuery.error}
            onRetry={() => void balancesQuery.refetch()}
            emptyTitle="No Stock Balances Recorded"
            emptyDescription="Record a stock receipt movement or goods receipt to initialize warehouse balances."
            onResetFilters={
              balanceSearch || balanceWarehouseFilter !== 'ALL' || balanceItemFilter
                ? () => {
                    setBalanceSearch('');
                    setBalanceWarehouseFilter('ALL');
                    setBalanceItemFilter('');
                    setBalancePage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: balanceSearch,
              onSearchChange: setBalanceSearch,
              searchPlaceholder: 'Filter by SKU or warehouse...',
              filters: (
                <div className="flex flex-wrap items-center gap-2">
                  <div className="w-48">
                    <Select
                      aria-label="Filter stock balances by warehouse"
                      value={balanceWarehouseFilter}
                      onValueChange={(v) => {
                        setBalanceWarehouseFilter(v);
                        setBalancePage(1);
                      }}
                      options={[{ value: 'ALL', label: 'All Warehouses' }, ...warehouseSelectOptions]}
                    />
                  </div>
                  {itemComboboxOptions.length > 0 ? (
                    <div className="w-56">
                      <Combobox
                        value={balanceItemFilter}
                        onValueChange={(v) => {
                          setBalanceItemFilter(v);
                          setBalancePage(1);
                        }}
                        options={itemComboboxOptions}
                        placeholder="Filter by item..."
                        allowClear
                      />
                    </div>
                  ) : null}
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredBalances,
                  [
                    { header: 'Item', accessor: (r) => formatItemLabel(r.itemId) },
                    { header: 'Warehouse', accessor: (r) => formatWarehouseLabel(r.warehouseId) },
                    { header: 'On Hand Qty', accessor: (r) => r.onHandQty },
                    { header: 'Reserved Qty', accessor: (r) => r.reservedQty },
                    { header: 'Available Qty', accessor: (r) => r.availableQty },
                    { header: 'Avg Cost', accessor: (r) => r.avgCost },
                  ],
                  'inventory-stock-balances.csv'
                ),
            }}
            pagination={
              balancesQuery.data
                ? {
                    page: balancePage,
                    size: balancesQuery.data.page.size || 50,
                    totalElements: balancesQuery.data.page.totalElements,
                    totalPages: Math.max(balancesQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setBalancePage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="warehouses">
          <VirtualizedTable
            data={filteredWarehouses}
            columns={warehouseColumns}
            getRowId={(r) => r.id}
            isLoading={warehousesQuery.isLoading}
            error={warehousesQuery.error}
            onRetry={() => void warehousesQuery.refetch()}
            emptyTitle="No Warehouses Configured"
            emptyDescription="Create a warehouse location before receiving or issuing inventory."
            toolbar={{
              searchValue: warehouseSearch,
              onSearchChange: setWarehouseSearch,
              searchPlaceholder: 'Search warehouse code or name...',
              actions: canWrite ? (
                <Button
                  variant="primary"
                  size="sm"
                  leftIcon={<Plus className="h-3.5 w-3.5" />}
                  onClick={() => setWarehouseDialogOpen(true)}
                >
                  Create Warehouse
                </Button>
              ) : undefined,
            }}
            pagination={
              warehousesQuery.data
                ? {
                    page: warehousePage,
                    size: warehousesQuery.data.page.size || 100,
                    totalElements: warehousesQuery.data.page.totalElements,
                    totalPages: Math.max(warehousesQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setWarehousePage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="movements">
          <VirtualizedTable
            data={filteredMovements}
            columns={movementColumns}
            getRowId={(r) => r.id}
            isLoading={movementsQuery.isLoading}
            error={movementsQuery.error}
            onRetry={() => void movementsQuery.refetch()}
            emptyTitle="No Stock Movements Recorded"
            emptyDescription="Use Record Movement to post receipts, issues, adjustments, or transfers."
            toolbar={{
              searchValue: movementSearch,
              onSearchChange: setMovementSearch,
              searchPlaceholder: 'Search item, warehouse, or source ref...',
              filters: (
                <div className="w-48">
                  <Select
                    aria-label="Filter stock movements by warehouse"
                    value={movementWarehouseFilter}
                    onValueChange={(v) => {
                      setMovementWarehouseFilter(v);
                      setMovementPage(1);
                    }}
                    options={[{ value: 'ALL', label: 'All Warehouses' }, ...warehouseSelectOptions]}
                  />
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredMovements,
                  [
                    { header: 'Date', accessor: (r) => r.movementDate },
                    { header: 'Type', accessor: (r) => r.movementType },
                    { header: 'Item', accessor: (r) => formatItemLabel(r.itemId) },
                    { header: 'Warehouse', accessor: (r) => formatWarehouseLabel(r.warehouseId) },
                    { header: 'Quantity', accessor: (r) => r.quantity },
                    { header: 'Source Ref', accessor: (r) => r.sourceDocumentRef ?? '' },
                  ],
                  'inventory-stock-movements.csv'
                ),
            }}
            pagination={
              movementsQuery.data
                ? {
                    page: movementPage,
                    size: movementsQuery.data.page.size || 50,
                    totalElements: movementsQuery.data.page.totalElements,
                    totalPages: Math.max(movementsQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setMovementPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="reservations">
          <VirtualizedTable
            data={filteredReservations}
            columns={reservationColumns}
            getRowId={(r) => r.id}
            isLoading={reservationsQuery.isLoading}
            error={reservationsQuery.error}
            onRetry={() => void reservationsQuery.refetch()}
            onRowClick={(row) => setSelectedReservation(row)}
            emptyTitle="No Stock Reservations"
            emptyDescription="Reserve available warehouse stock for upcoming sales or production orders."
            toolbar={{
              searchValue: reservationSearch,
              onSearchChange: setReservationSearch,
              searchPlaceholder: 'Search by item, status, or sales order ref...',
            }}
            pagination={
              reservationsQuery.data
                ? {
                    page: reservationPage,
                    size: reservationsQuery.data.page.size || 50,
                    totalElements: reservationsQuery.data.page.totalElements,
                    totalPages: Math.max(reservationsQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setReservationPage,
                  }
                : undefined
            }
          />
        </TabsContent>
      </Tabs>

      {/* Create Warehouse Dialog */}
      <AppDialog
        open={warehouseDialogOpen}
        onOpenChange={setWarehouseDialogOpen}
        title="New Warehouse"
        description="Register a storage location in Inventory Service."
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setWarehouseDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={createWarehouseMutation.isPending}
              onClick={warehouseForm.handleSubmit((vals) => createWarehouseMutation.mutate(vals))}
            >
              Create Warehouse
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={warehouseForm.handleSubmit((vals) => createWarehouseMutation.mutate(vals))}
        >
          <FormField
            label="Warehouse Code"
            htmlFor="wh-code"
            required
            error={warehouseForm.formState.errors.code?.message}
          >
            <Input
              id="wh-code"
              placeholder="e.g. WH-JKT-01"
              error={Boolean(warehouseForm.formState.errors.code)}
              {...warehouseForm.register('code')}
            />
          </FormField>
          <FormField
            label="Warehouse Name"
            htmlFor="wh-name"
            required
            error={warehouseForm.formState.errors.name?.message}
          >
            <Input
              id="wh-name"
              placeholder="e.g. Jakarta Main Distribution Center"
              error={Boolean(warehouseForm.formState.errors.name)}
              {...warehouseForm.register('name')}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Record Stock Movement Dialog */}
      <AppDialog
        open={movementDialogOpen}
        onOpenChange={setMovementDialogOpen}
        title="Record Stock Movement"
        description="Post inventory Receipt, Issue, Adjustment, or paired Warehouse Transfer."
        size="md"
        footer={
          <>
            <Button variant="outline" onClick={() => setMovementDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={createMovementMutation.isPending}
              onClick={movementForm.handleSubmit((vals) => createMovementMutation.mutate(vals))}
            >
              Post Movement
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={movementForm.handleSubmit((vals) => createMovementMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Movement Type"
              required
              error={movementForm.formState.errors.mode?.message}
            >
              <Controller
                control={movementForm.control}
                name="mode"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    options={[
                      { value: 'Receipt', label: 'Receipt (+ In)' },
                      { value: 'Issue', label: 'Issue (- Out)' },
                      { value: 'Adjustment', label: 'Adjustment (+/-)' },
                      { value: 'Transfer', label: 'Transfer (Issue + Receipt)' },
                    ]}
                  />
                )}
              />
            </FormField>

            <FormField label="Movement Date" error={movementForm.formState.errors.movementDate?.message}>
              <Input type="date" {...movementForm.register('movementDate')} />
            </FormField>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label={movementMode === 'Transfer' ? 'Source Warehouse' : 'Warehouse'}
              required
              error={movementForm.formState.errors.warehouseId?.message}
            >
              <Controller
                control={movementForm.control}
                name="warehouseId"
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    placeholder="Select warehouse..."
                    options={warehouseSelectOptions}
                    error={Boolean(movementForm.formState.errors.warehouseId)}
                  />
                )}
              />
            </FormField>

            {movementMode === 'Transfer' ? (
              <FormField
                label="Destination Warehouse"
                required
                error={movementForm.formState.errors.destinationWarehouseId?.message}
              >
                <Controller
                  control={movementForm.control}
                  name="destinationWarehouseId"
                  render={({ field }) => (
                    <Select
                      value={field.value}
                      onValueChange={field.onChange}
                      placeholder="Select destination..."
                      options={warehouseSelectOptions}
                      error={Boolean(movementForm.formState.errors.destinationWarehouseId)}
                    />
                  )}
                />
              </FormField>
            ) : (
              <FormField
                label="Source Document Ref"
                error={movementForm.formState.errors.sourceDocumentRef?.message}
              >
                <Input placeholder="e.g. GR-2026-001" {...movementForm.register('sourceDocumentRef')} />
              </FormField>
            )}
          </div>

          <FormField
            label="Item / SKU"
            required
            error={movementForm.formState.errors.itemId?.message}
          >
            {itemComboboxOptions.length > 0 ? (
              <Controller
                control={movementForm.control}
                name="itemId"
                render={({ field }) => (
                  <Combobox
                    value={field.value}
                    onValueChange={field.onChange}
                    options={itemComboboxOptions}
                    placeholder="Search item SKU..."
                    error={Boolean(movementForm.formState.errors.itemId)}
                  />
                )}
              />
            ) : (
              <Input
                placeholder="Paste Item UUID..."
                error={Boolean(movementForm.formState.errors.itemId)}
                {...movementForm.register('itemId')}
              />
            )}
          </FormField>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Quantity"
              required
              hint={
                movementMode === 'Adjustment'
                  ? 'Negative values deduct inventory.'
                  : 'Must be positive.'
              }
              error={movementForm.formState.errors.quantity?.message}
            >
              <Controller
                control={movementForm.control}
                name="quantity"
                render={({ field }) => (
                  <MoneyInput
                    currency=""
                    precision={4}
                    allowNegative={movementMode === 'Adjustment'}
                    value={field.value}
                    onValueChange={field.onChange}
                    error={Boolean(movementForm.formState.errors.quantity)}
                  />
                )}
              />
            </FormField>

            <FormField
              label="Unit Cost (Optional)"
              error={movementForm.formState.errors.unitCost?.message}
            >
              <Controller
                control={movementForm.control}
                name="unitCost"
                render={({ field }) => (
                  <MoneyInput
                    currency={currencyCode}
                    precision={2}
                    value={field.value ?? ''}
                    onValueChange={field.onChange}
                    error={Boolean(movementForm.formState.errors.unitCost)}
                  />
                )}
              />
            </FormField>
          </div>
        </form>
      </AppDialog>

      {/* Create Reservation Dialog */}
      <AppDialog
        open={reservationDialogOpen}
        onOpenChange={setReservationDialogOpen}
        title="New Stock Reservation"
        description="Lock available warehouse stock for a sales or manufacturing demand."
        size="md"
        footer={
          <>
            <Button variant="outline" onClick={() => setReservationDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={createReservationMutation.isPending}
              onClick={reservationForm.handleSubmit((vals) =>
                createReservationMutation.mutate(vals)
              )}
            >
              Reserve Stock
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={reservationForm.handleSubmit((vals) => createReservationMutation.mutate(vals))}
        >
          <FormField
            label="Warehouse"
            required
            error={reservationForm.formState.errors.warehouseId?.message}
          >
            <Controller
              control={reservationForm.control}
              name="warehouseId"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  placeholder="Select warehouse..."
                  options={warehouseSelectOptions}
                  error={Boolean(reservationForm.formState.errors.warehouseId)}
                />
              )}
            />
          </FormField>

          <FormField
            label="Item / SKU"
            required
            error={reservationForm.formState.errors.itemId?.message}
          >
            {itemComboboxOptions.length > 0 ? (
              <Controller
                control={reservationForm.control}
                name="itemId"
                render={({ field }) => (
                  <Combobox
                    value={field.value}
                    onValueChange={field.onChange}
                    options={itemComboboxOptions}
                    placeholder="Select item..."
                    error={Boolean(reservationForm.formState.errors.itemId)}
                  />
                )}
              />
            ) : (
              <Input
                placeholder="Paste Item UUID..."
                error={Boolean(reservationForm.formState.errors.itemId)}
                {...reservationForm.register('itemId')}
              />
            )}
          </FormField>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Reserved Quantity"
              required
              error={reservationForm.formState.errors.reservedQty?.message}
            >
              <Controller
                control={reservationForm.control}
                name="reservedQty"
                render={({ field }) => (
                  <MoneyInput
                    currency=""
                    precision={4}
                    value={field.value}
                    onValueChange={field.onChange}
                    error={Boolean(reservationForm.formState.errors.reservedQty)}
                  />
                )}
              />
            </FormField>

            <FormField
              label="Sales Order Ref"
              error={reservationForm.formState.errors.salesOrderRef?.message}
            >
              <Input placeholder="e.g. SO-2026-0042" {...reservationForm.register('salesOrderRef')} />
            </FormField>
          </div>
        </form>
      </AppDialog>

      {/* Issue Reservation Dialog */}
      <AppDialog
        open={Boolean(issuingReservation)}
        onOpenChange={(open) => {
          if (!open) setIssuingReservation(null);
        }}
        title="Issue Reserved Stock"
        description={
          issuingReservation
            ? `Consume reserved stock for ${formatItemLabel(issuingReservation.itemId)} at ${formatWarehouseLabel(issuingReservation.warehouseId)}.`
            : undefined
        }
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setIssuingReservation(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={issueReservationMutation.isPending}
              onClick={issueForm.handleSubmit((vals) => {
                if (issuingReservation) {
                  issueReservationMutation.mutate({
                    resv: issuingReservation,
                    amount: vals.amount,
                  });
                }
              })}
            >
              Confirm Issue
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={issueForm.handleSubmit((vals) => {
            if (issuingReservation) {
              issueReservationMutation.mutate({
                resv: issuingReservation,
                amount: vals.amount,
              });
            }
          })}
        >
          <FormField
            label="Issue Quantity"
            hint={`Reserved quantity: ${issuingReservation?.reservedQty ?? '0'}`}
            error={issueForm.formState.errors.amount?.message}
          >
            <Controller
              control={issueForm.control}
              name="amount"
              render={({ field }) => (
                <MoneyInput
                  currency=""
                  precision={4}
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                  error={Boolean(issueForm.formState.errors.amount)}
                />
              )}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Release Reservation Confirmation */}
      <AppDialog
        open={Boolean(releasingReservation)}
        onOpenChange={(open) => {
          if (!open) setReleasingReservation(null);
        }}
        title="Release Stock Reservation"
        description="Return reserved quantity back to available stock?"
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setReleasingReservation(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              isLoading={releaseReservationMutation.isPending}
              onClick={() =>
                releasingReservation && releaseReservationMutation.mutate(releasingReservation)
              }
            >
              Release Reservation
            </Button>
          </>
        }
      >
        {releasingReservation ? (
          <p className="text-sm text-slate-600 dark:text-slate-300">
            Item: <strong>{formatItemLabel(releasingReservation.itemId)}</strong> · Qty:{' '}
            <strong className="font-money">{releasingReservation.reservedQty}</strong>
          </p>
        ) : null}
      </AppDialog>

      {/* Reservation Detail Drawer */}
      <SideDrawer
        open={Boolean(selectedReservation)}
        onOpenChange={(open) => {
          if (!open) setSelectedReservation(null);
        }}
        title="Reservation Details"
        description={selectedReservation ? `ID ${shortId(selectedReservation.id)}` : undefined}
        footer={
          selectedReservation && canWrite && selectedReservation.status === 'Reserved' ? (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setReleasingReservation(selectedReservation)}
              >
                Release
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={() => {
                  issueForm.reset({ amount: selectedReservation.reservedQty });
                  setIssuingReservation(selectedReservation);
                }}
              >
                Issue Stock
              </Button>
            </>
          ) : undefined
        }
      >
        {selectedReservation ? (
          <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
            <div className="flex justify-between">
              <span className="text-xs text-slate-500">Status</span>
              <StatusBadge
                status={selectedReservation.status.toUpperCase()}
                label={selectedReservation.status}
                tone={RESERVATION_BADGE_TONE[selectedReservation.status] ?? 'neutral'}
              />
            </div>
            <div>
              <span className="text-xs text-slate-500">Item</span>
              <p className="font-medium">{formatItemLabel(selectedReservation.itemId)}</p>
            </div>
            <div>
              <span className="text-xs text-slate-500">Warehouse</span>
              <p className="font-medium">{formatWarehouseLabel(selectedReservation.warehouseId)}</p>
            </div>
            <div className="flex justify-between">
              <span className="text-xs text-slate-500">Reserved Quantity</span>
              <span className="font-money font-semibold">
                {formatNumber(selectedReservation.reservedQty, 4, locale)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-xs text-slate-500">Sales Order Ref</span>
              <span className="font-mono text-xs">{selectedReservation.salesOrderRef || '—'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-xs text-slate-500">Concurrency Version</span>
              <span className="font-mono text-xs">v{selectedReservation.rowVersion}</span>
            </div>
          </div>
        ) : null}
      </SideDrawer>
    </div>
  );
}

export function InventoryWorkspace() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <InventoryContent />
    </DomainErrorBoundary>
  );
}
