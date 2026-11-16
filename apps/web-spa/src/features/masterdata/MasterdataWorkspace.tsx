import * as React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import {
  Building2,
  Database,
  Edit3,
  Globe,
  MapPin,
  Package,
  Plus,
  RefreshCw,
  ShieldAlert,
  Tag,
  Trash2,
  Users,
} from 'lucide-react';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { ErrorState } from '../../components/ui/ErrorState';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Switch } from '../../components/ui/Switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { applyApiErrorsToForm, ApiClientError } from '../../lib/api';
import { formatNumber } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  createItem,
  createParty,
  deleteItem,
  deleteParty,
  getItem,
  getParty,
  listItems,
  listParties,
  updateItem,
  updateParty,
} from './api';
import {
  itemFormSchema,
  partyFormSchema,
  type ItemFormValues,
  type PartyFormValues,
} from './schemas';
import type { ItemRecord, PartyRecord } from './types';

function showMutationError(err: unknown, fallback: string) {
  if (err instanceof ApiClientError) {
    toast.error(err.message || fallback, {
      description: err.correlationId ? `Correlation ID: ${err.correlationId}` : undefined,
    });
    return;
  }
  toast.error(err instanceof Error ? err.message : fallback);
}

const PARTY_TYPE_BADGE: Record<string, { label: string; tone: 'brand' | 'info' | 'warning' }> = {
  customer: { label: 'Customer', tone: 'info' },
  vendor: { label: 'Vendor', tone: 'warning' },
  both: { label: 'Both', tone: 'brand' },
};

const ITEM_TYPE_BADGE: Record<string, { label: string; tone: 'success' | 'info' | 'neutral' }> = {
  stock: { label: 'Stock', tone: 'success' },
  service: { label: 'Service', tone: 'info' },
  asset: { label: 'Asset', tone: 'neutral' },
};

function MasterdataContent() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const queryClient = useQueryClient();
  const activeOrgId = activeOrg?.id ?? 'default-org';
  const canWrite = hasUserPermission(user, 'masterdata:write');

  const [activeTab, setActiveTab] = React.useState<'parties' | 'items'>('parties');

  // Parties state
  const [partyPage, setPartyPage] = React.useState(1);
  const [partyTypeFilter, setPartyTypeFilter] = React.useState<string>('ALL');
  const [partySearch, setPartySearch] = React.useState('');
  const [partyDialogOpen, setPartyDialogOpen] = React.useState(false);
  const [editingParty, setEditingParty] = React.useState<PartyRecord | null>(null);
  const [selectedPartyId, setSelectedPartyId] = React.useState<string | null>(null);
  const [deletingParty, setDeletingParty] = React.useState<PartyRecord | null>(null);

  // Items state
  const [itemPage, setItemPage] = React.useState(1);
  const [itemTypeFilter, setItemTypeFilter] = React.useState<string>('ALL');
  const [itemSearch, setItemSearch] = React.useState('');
  const [itemDialogOpen, setItemDialogOpen] = React.useState(false);
  const [editingItem, setEditingItem] = React.useState<ItemRecord | null>(null);
  const [selectedItemId, setSelectedItemId] = React.useState<string | null>(null);
  const [deletingItem, setDeletingItem] = React.useState<ItemRecord | null>(null);

  const partiesQuery = useQuery({
    queryKey: ['business-partners', activeOrgId, partyPage, partyTypeFilter],
    queryFn: () =>
      listParties({
        page: partyPage,
        size: 50,
        partyType: partyTypeFilter,
      }),
  });

  const itemsQuery = useQuery({
    queryKey: ['inventory-items', activeOrgId, itemPage, itemTypeFilter],
    queryFn: () =>
      listItems({
        page: itemPage,
        size: 50,
        itemType: itemTypeFilter,
      }),
  });

  const partyDetailQuery = useQuery({
    queryKey: ['business-partners', 'detail', activeOrgId, selectedPartyId],
    queryFn: () => getParty(selectedPartyId as string),
    enabled: Boolean(selectedPartyId),
  });

  const itemDetailQuery = useQuery({
    queryKey: ['inventory-items', 'detail', activeOrgId, selectedItemId],
    queryFn: () => getItem(selectedItemId as string),
    enabled: Boolean(selectedItemId),
  });

  const partyForm = useForm<PartyFormValues>({
    resolver: zodResolver(partyFormSchema),
    defaultValues: {
      code: '',
      name: '',
      partyType: 'vendor',
      taxId: '',
      isShared: false,
      includePrimaryAddress: false,
      addressType: 'billing',
      line1: '',
      city: '',
      postalCode: '',
      countryCode: 'ID',
    },
  });

  const itemForm = useForm<ItemFormValues>({
    resolver: zodResolver(itemFormSchema),
    defaultValues: {
      code: '',
      name: '',
      itemType: 'stock',
      baseUom: 'PCS',
    },
  });

  const openCreateParty = () => {
    setEditingParty(null);
    partyForm.reset({
      code: '',
      name: '',
      partyType: 'vendor',
      taxId: '',
      isShared: false,
      includePrimaryAddress: false,
      addressType: 'billing',
      line1: '',
      city: '',
      postalCode: '',
      countryCode: 'ID',
    });
    setPartyDialogOpen(true);
  };

  const openEditParty = (party: PartyRecord) => {
    setEditingParty(party);
    partyForm.reset({
      code: party.code,
      name: party.name,
      partyType: party.partyType,
      taxId: party.taxId ?? '',
      isShared: party.isShared,
      includePrimaryAddress: false,
      addressType: 'billing',
      line1: '',
      city: '',
      postalCode: '',
      countryCode: 'ID',
    });
    setPartyDialogOpen(true);
  };

  const openCreateItem = () => {
    setEditingItem(null);
    itemForm.reset({
      code: '',
      name: '',
      itemType: 'stock',
      baseUom: 'PCS',
    });
    setItemDialogOpen(true);
  };

  const openEditItem = (item: ItemRecord) => {
    setEditingItem(item);
    itemForm.reset({
      code: item.code,
      name: item.name,
      itemType: item.itemType,
      baseUom: item.baseUom,
    });
    setItemDialogOpen(true);
  };

  const savePartyMutation = useMutation({
    mutationFn: async (values: PartyFormValues) => {
      if (editingParty) {
        return updateParty(editingParty.id, editingParty.rowVersion, {
          code: values.code,
          name: values.name,
          partyType: values.partyType,
          taxId: values.taxId?.trim() || undefined,
          isShared: values.isShared,
        });
      }
      return createParty({
        code: values.code,
        name: values.name,
        partyType: values.partyType,
        taxId: values.taxId?.trim() || undefined,
        isShared: values.isShared,
        addresses: values.includePrimaryAddress
          ? [
              {
                addressType: values.addressType,
                line1: values.line1.trim(),
                city: values.city?.trim() || undefined,
                postalCode: values.postalCode?.trim() || undefined,
                countryCode: values.countryCode?.trim() || undefined,
                isPrimary: true,
              },
            ]
          : undefined,
      });
    },
    onSuccess: (saved) => {
      toast.success(editingParty ? `Party ${saved.code} updated` : `Party ${saved.code} created`);
      setPartyDialogOpen(false);
      setEditingParty(null);
      void queryClient.invalidateQueries({ queryKey: ['business-partners'] });
    },
    onError: (err) => {
      const mapped = applyApiErrorsToForm(err, partyForm.setError);
      if (mapped === 0) {
        showMutationError(err, 'Failed to save business partner');
      }
    },
  });

  const deletePartyMutation = useMutation({
    mutationFn: (party: PartyRecord) => deleteParty(party.id, party.rowVersion),
    onSuccess: () => {
      toast.success('Business partner deleted');
      setDeletingParty(null);
      setSelectedPartyId(null);
      void queryClient.invalidateQueries({ queryKey: ['business-partners'] });
    },
    onError: (err) => showMutationError(err, 'Failed to delete business partner'),
  });

  const saveItemMutation = useMutation({
    mutationFn: async (values: ItemFormValues) => {
      if (editingItem) {
        return updateItem(editingItem.id, editingItem.rowVersion, {
          code: values.code,
          name: values.name,
          itemType: values.itemType,
          baseUom: values.baseUom,
        });
      }
      return createItem({
        code: values.code,
        name: values.name,
        itemType: values.itemType,
        baseUom: values.baseUom,
      });
    },
    onSuccess: (saved) => {
      toast.success(editingItem ? `Item ${saved.code} updated` : `Item ${saved.code} created`);
      setItemDialogOpen(false);
      setEditingItem(null);
      void queryClient.invalidateQueries({ queryKey: ['inventory-items'] });
    },
    onError: (err) => {
      const mapped = applyApiErrorsToForm(err, itemForm.setError);
      if (mapped === 0) {
        showMutationError(err, 'Failed to save item');
      }
    },
  });

  const deleteItemMutation = useMutation({
    mutationFn: (item: ItemRecord) => deleteItem(item.id, item.rowVersion),
    onSuccess: () => {
      toast.success('Item deleted');
      setDeletingItem(null);
      setSelectedItemId(null);
      void queryClient.invalidateQueries({ queryKey: ['inventory-items'] });
    },
    onError: (err) => showMutationError(err, 'Failed to delete item'),
  });

  const rawParties = React.useMemo(() => partiesQuery.data?.items ?? [], [partiesQuery.data?.items]);
  const filteredParties = React.useMemo(() => {
    const q = partySearch.trim().toLowerCase();
    if (!q) return rawParties;
    return rawParties.filter(
      (p) =>
        p.code.toLowerCase().includes(q) ||
        p.name.toLowerCase().includes(q) ||
        (p.taxId ?? '').toLowerCase().includes(q)
    );
  }, [rawParties, partySearch]);

  const rawItems = React.useMemo(() => itemsQuery.data?.items ?? [], [itemsQuery.data?.items]);
  const filteredItems = React.useMemo(() => {
    const q = itemSearch.trim().toLowerCase();
    if (!q) return rawItems;
    return rawItems.filter(
      (i) =>
        i.code.toLowerCase().includes(q) ||
        i.name.toLowerCase().includes(q) ||
        i.baseUom.toLowerCase().includes(q)
    );
  }, [rawItems, itemSearch]);

  const vendorCount = rawParties.filter(
    (p) => p.partyType === 'vendor' || p.partyType === 'both'
  ).length;
  const stockCount = rawItems.filter((i) => i.itemType === 'stock').length;

  const partyColumns: ColumnDef<PartyRecord>[] = [
    {
      header: 'Code',
      accessorKey: 'code',
      width: '15%',
      cell: (row) => <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">{row.code}</span>,
    },
    {
      header: 'Partner Name',
      accessorKey: 'name',
      width: '28%',
      cell: (row) => <span className="font-medium text-slate-900 dark:text-slate-100">{row.name}</span>,
    },
    {
      header: 'Type',
      accessorKey: 'partyType',
      width: '14%',
      cell: (row) => {
        const cfg = PARTY_TYPE_BADGE[row.partyType] ?? { label: row.partyType, tone: 'neutral' as const };
        return <StatusBadge status={row.partyType.toUpperCase()} label={cfg.label} tone={cfg.tone} />;
      },
    },
    {
      header: 'Tax ID (NPWP)',
      accessorKey: 'taxId',
      width: '18%',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-600 dark:text-slate-300">
          {row.taxId || '—'}
        </span>
      ),
    },
    {
      header: 'Scope',
      accessorKey: 'isShared',
      width: '12%',
      cell: (row) => (
        <span className="inline-flex items-center gap-1 text-xs text-slate-600 dark:text-slate-300">
          {row.isShared ? (
            <>
              <Globe className="h-3.5 w-3.5 text-brand-600" /> Shared
            </>
          ) : (
            <>
              <Building2 className="h-3.5 w-3.5 text-slate-400" /> Org
            </>
          )}
        </span>
      ),
    },
    {
      header: 'Actions',
      width: '13%',
      align: 'right',
      cell: (row) => (
        <div
          className="inline-flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canWrite ? (
            <>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Edit party ${row.code}`}
                onClick={() => openEditParty(row)}
              >
                <Edit3 className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Delete party ${row.code}`}
                className="text-rose-600 hover:bg-rose-50 hover:text-rose-700"
                onClick={() => setDeletingParty(row)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <span className="text-[11px] text-slate-400">Read only</span>
          )}
        </div>
      ),
    },
  ];

  const itemColumns: ColumnDef<ItemRecord>[] = [
    {
      header: 'SKU Code',
      accessorKey: 'code',
      width: '18%',
      cell: (row) => <span className="font-mono font-semibold text-brand-700 dark:text-brand-400">{row.code}</span>,
    },
    {
      header: 'Item Name',
      accessorKey: 'name',
      width: '34%',
      cell: (row) => <span className="font-medium text-slate-900 dark:text-slate-100">{row.name}</span>,
    },
    {
      header: 'Type',
      accessorKey: 'itemType',
      width: '16%',
      cell: (row) => {
        const cfg = ITEM_TYPE_BADGE[row.itemType] ?? { label: row.itemType, tone: 'neutral' as const };
        return <StatusBadge status={row.itemType.toUpperCase()} label={cfg.label} tone={cfg.tone} />;
      },
    },
    {
      header: 'Base UOM',
      accessorKey: 'baseUom',
      width: '12%',
      cell: (row) => (
        <span className="inline-flex items-center rounded bg-slate-100 px-2 py-0.5 font-mono text-xs font-semibold text-slate-700 dark:bg-slate-800 dark:text-slate-200">
          {row.baseUom}
        </span>
      ),
    },
    {
      header: 'Ver',
      accessorKey: 'rowVersion',
      width: '7%',
      align: 'center',
      cell: (row) => <span className="font-mono text-xs text-slate-500">v{row.rowVersion}</span>,
    },
    {
      header: 'Actions',
      width: '13%',
      align: 'right',
      cell: (row) => (
        <div
          className="inline-flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canWrite ? (
            <>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Edit item ${row.code}`}
                onClick={() => openEditItem(row)}
              >
                <Edit3 className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                aria-label={`Delete item ${row.code}`}
                className="text-rose-600 hover:bg-rose-50 hover:text-rose-700"
                onClick={() => setDeletingItem(row)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <span className="text-[11px] text-slate-400">Read only</span>
          )}
        </div>
      ),
    },
  ];

  const includeAddress = partyForm.watch('includePrimaryAddress');

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-600/10 text-brand-700 dark:bg-brand-500/20 dark:text-brand-300">
            <Database className="h-5 w-5" />
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                Master Data Catalog
              </h1>
              {!canWrite ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-[11px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                  <ShieldAlert className="h-3 w-3" /> Read-only
                </span>
              ) : null}
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Central registry of customers, suppliers, tax IDs, SKUs, and units of measure.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={() => {
              void partiesQuery.refetch();
              void itemsQuery.refetch();
            }}
          >
            Refresh
          </Button>
          {canWrite ? (
            <Button
              variant="primary"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={activeTab === 'parties' ? openCreateParty : openCreateItem}
            >
              {activeTab === 'parties' ? 'New Partner' : 'New Item'}
            </Button>
          ) : null}
        </div>
      </header>

      <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Business Partners"
          value={String(partiesQuery.data?.page.totalElements ?? rawParties.length)}
          subtitle="Total registered parties"
          icon={<Users className="h-4 w-4" />}
          accent="brand"
          isLoading={partiesQuery.isLoading}
          onClick={() => setActiveTab('parties')}
        />
        <KpiCard
          label="Active Suppliers"
          value={String(vendorCount)}
          subtitle="Vendor & dual-role partners"
          icon={<Building2 className="h-4 w-4" />}
          accent="amber"
          isLoading={partiesQuery.isLoading}
          onClick={() => setActiveTab('parties')}
        />
        <KpiCard
          label="Catalog Items"
          value={String(itemsQuery.data?.page.totalElements ?? rawItems.length)}
          subtitle="Registered SKUs & services"
          icon={<Package className="h-4 w-4" />}
          accent="emerald"
          isLoading={itemsQuery.isLoading}
          onClick={() => setActiveTab('items')}
        />
        <KpiCard
          label="Stockable SKUs"
          value={String(stockCount)}
          subtitle="Tracked in inventory ledger"
          icon={<Tag className="h-4 w-4" />}
          accent="sky"
          isLoading={itemsQuery.isLoading}
          onClick={() => setActiveTab('items')}
        />
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'parties' | 'items')}>
        <TabsList>
          <TabsTrigger value="parties">
            Business Partners ({partiesQuery.data?.page.totalElements ?? rawParties.length})
          </TabsTrigger>
          <TabsTrigger value="items">
            Items &amp; Materials ({itemsQuery.data?.page.totalElements ?? rawItems.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="parties">
          <VirtualizedTable
            data={filteredParties}
            columns={partyColumns}
            getRowId={(r) => r.id}
            isLoading={partiesQuery.isLoading}
            error={partiesQuery.error}
            onRetry={() => void partiesQuery.refetch()}
            onRowClick={(row) => setSelectedPartyId(row.id)}
            emptyTitle="No Business Partners Found"
            emptyDescription="Create a customer or vendor record to enable procurement and sales flows."
            onResetFilters={
              partySearch || partyTypeFilter !== 'ALL'
                ? () => {
                    setPartySearch('');
                    setPartyTypeFilter('ALL');
                    setPartyPage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: partySearch,
              onSearchChange: setPartySearch,
              searchPlaceholder: 'Search by code, name, or Tax ID...',
              filters: (
                <div className="w-40">
                  <Select
                    aria-label="Filter party type"
                    value={partyTypeFilter}
                    onValueChange={(v) => {
                      setPartyTypeFilter(v);
                      setPartyPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Types' },
                      { value: 'vendor', label: 'Vendor' },
                      { value: 'customer', label: 'Customer' },
                      { value: 'both', label: 'Both' },
                    ]}
                  />
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredParties,
                  [
                    { header: 'Code', accessor: (r) => r.code },
                    { header: 'Name', accessor: (r) => r.name },
                    { header: 'Party Type', accessor: (r) => r.partyType },
                    { header: 'Tax ID', accessor: (r) => r.taxId ?? '' },
                    { header: 'Shared', accessor: (r) => (r.isShared ? 'Yes' : 'No') },
                  ],
                  'masterdata-parties.csv'
                ),
            }}
            pagination={
              partiesQuery.data
                ? {
                    page: partyPage,
                    size: partiesQuery.data.page.size || 50,
                    totalElements: partiesQuery.data.page.totalElements,
                    totalPages: Math.max(partiesQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setPartyPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="items">
          <VirtualizedTable
            data={filteredItems}
            columns={itemColumns}
            getRowId={(r) => r.id}
            isLoading={itemsQuery.isLoading}
            error={itemsQuery.error}
            onRetry={() => void itemsQuery.refetch()}
            onRowClick={(row) => setSelectedItemId(row.id)}
            emptyTitle="No Items Registered"
            emptyDescription="Add stockable materials, services, or fixed assets to the catalog."
            onResetFilters={
              itemSearch || itemTypeFilter !== 'ALL'
                ? () => {
                    setItemSearch('');
                    setItemTypeFilter('ALL');
                    setItemPage(1);
                  }
                : undefined
            }
            toolbar={{
              searchValue: itemSearch,
              onSearchChange: setItemSearch,
              searchPlaceholder: 'Search by SKU code, name, or UOM...',
              filters: (
                <div className="w-40">
                  <Select
                    aria-label="Filter item type"
                    value={itemTypeFilter}
                    onValueChange={(v) => {
                      setItemTypeFilter(v);
                      setItemPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Item Types' },
                      { value: 'stock', label: 'Stock' },
                      { value: 'service', label: 'Service' },
                      { value: 'asset', label: 'Asset' },
                    ]}
                  />
                </div>
              ),
              onExportCsv: () =>
                exportToCsv(
                  filteredItems,
                  [
                    { header: 'SKU Code', accessor: (r) => r.code },
                    { header: 'Name', accessor: (r) => r.name },
                    { header: 'Type', accessor: (r) => r.itemType },
                    { header: 'Base UOM', accessor: (r) => r.baseUom },
                  ],
                  'masterdata-items.csv'
                ),
            }}
            pagination={
              itemsQuery.data
                ? {
                    page: itemPage,
                    size: itemsQuery.data.page.size || 50,
                    totalElements: itemsQuery.data.page.totalElements,
                    totalPages: Math.max(itemsQuery.data.page.totalPages, 1),
                    pageBase: 1,
                    onPageChange: setItemPage,
                  }
                : undefined
            }
          />
        </TabsContent>
      </Tabs>

      {/* Party Create/Edit Dialog */}
      <AppDialog
        open={partyDialogOpen}
        onOpenChange={setPartyDialogOpen}
        title={editingParty ? `Edit Partner: ${editingParty.code}` : 'New Business Partner'}
        description={
          editingParty
            ? 'Update partner profile using optimistic concurrency.'
            : 'Register a vendor, customer, or dual-role partner in Master Data.'
        }
        size="lg"
        footer={
          <>
            <Button variant="outline" onClick={() => setPartyDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={savePartyMutation.isPending}
              onClick={partyForm.handleSubmit((vals) => savePartyMutation.mutate(vals))}
            >
              {editingParty ? 'Save Changes' : 'Create Partner'}
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={partyForm.handleSubmit((vals) => savePartyMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Partner Code"
              htmlFor="party-code"
              required
              error={partyForm.formState.errors.code?.message}
            >
              <Input
                id="party-code"
                placeholder="e.g. VND-001"
                disabled={Boolean(editingParty)}
                error={Boolean(partyForm.formState.errors.code)}
                {...partyForm.register('code')}
              />
            </FormField>

            <FormField
              label="Partner Role"
              htmlFor="party-type"
              required
              error={partyForm.formState.errors.partyType?.message}
            >
              <Controller
                control={partyForm.control}
                name="partyType"
                render={({ field }) => (
                  <Select
                    id="party-type"
                    value={field.value}
                    onValueChange={field.onChange}
                    options={[
                      { value: 'vendor', label: 'Vendor (Supplier)' },
                      { value: 'customer', label: 'Customer (Buyer)' },
                      { value: 'both', label: 'Both (Vendor & Customer)' },
                    ]}
                  />
                )}
              />
            </FormField>
          </div>

          <FormField
            label="Legal Name"
            htmlFor="party-name"
            required
            error={partyForm.formState.errors.name?.message}
          >
            <Input
              id="party-name"
              placeholder="e.g. PT Baja Nusantara"
              error={Boolean(partyForm.formState.errors.name)}
              {...partyForm.register('name')}
            />
          </FormField>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="Tax ID (NPWP)"
              htmlFor="party-taxId"
              error={partyForm.formState.errors.taxId?.message}
            >
              <Input
                id="party-taxId"
                placeholder="01.234.567.8-901.000"
                error={Boolean(partyForm.formState.errors.taxId)}
                {...partyForm.register('taxId')}
              />
            </FormField>

            <FormField label="Cross-Entity Visibility">
              <div className="flex h-9 items-center rounded-lg border border-slate-200 px-3 dark:border-slate-700">
                <Controller
                  control={partyForm.control}
                  name="isShared"
                  render={({ field }) => (
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      label="Share across organizations"
                    />
                  )}
                />
              </div>
            </FormField>
          </div>

          {!editingParty ? (
            <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-800 dark:bg-slate-900/60">
              <Controller
                control={partyForm.control}
                name="includePrimaryAddress"
                render={({ field }) => (
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    label="Attach primary address on creation"
                  />
                )}
              />

              {includeAddress ? (
                <div className="grid grid-cols-1 gap-3 pt-2 sm:grid-cols-2">
                  <FormField label="Address Type">
                    <Controller
                      control={partyForm.control}
                      name="addressType"
                      render={({ field }) => (
                        <Select
                          value={field.value}
                          onValueChange={field.onChange}
                          options={[
                            { value: 'billing', label: 'Billing' },
                            { value: 'shipping', label: 'Shipping' },
                            { value: 'headquarters', label: 'Headquarters' },
                          ]}
                        />
                      )}
                    />
                  </FormField>
                  <FormField label="Country Code">
                    <Input placeholder="ID" {...partyForm.register('countryCode')} />
                  </FormField>
                  <FormField
                    label="Address Line 1"
                    required
                    className="sm:col-span-2"
                    error={partyForm.formState.errors.line1?.message}
                  >
                    <Input
                      placeholder="Jl. Jend. Sudirman Kav. 52-53"
                      error={Boolean(partyForm.formState.errors.line1)}
                      {...partyForm.register('line1')}
                    />
                  </FormField>
                  <FormField label="City">
                    <Input placeholder="Jakarta Selatan" {...partyForm.register('city')} />
                  </FormField>
                  <FormField label="Postal Code">
                    <Input placeholder="12190" {...partyForm.register('postalCode')} />
                  </FormField>
                </div>
              ) : null}
            </div>
          ) : null}
        </form>
      </AppDialog>

      {/* Item Create/Edit Dialog */}
      <AppDialog
        open={itemDialogOpen}
        onOpenChange={setItemDialogOpen}
        title={editingItem ? `Edit Item: ${editingItem.code}` : 'New Item / Material'}
        description="Register a stock SKU, service, or asset with its base unit of measure."
        size="md"
        footer={
          <>
            <Button variant="outline" onClick={() => setItemDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              isLoading={saveItemMutation.isPending}
              onClick={itemForm.handleSubmit((vals) => saveItemMutation.mutate(vals))}
            >
              {editingItem ? 'Save Changes' : 'Create Item'}
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={itemForm.handleSubmit((vals) => saveItemMutation.mutate(vals))}
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField
              label="SKU Code"
              htmlFor="item-code"
              required
              error={itemForm.formState.errors.code?.message}
            >
              <Input
                id="item-code"
                placeholder="e.g. RM-STL-01"
                disabled={Boolean(editingItem)}
                error={Boolean(itemForm.formState.errors.code)}
                {...itemForm.register('code')}
              />
            </FormField>

            <FormField
              label="Item Type"
              htmlFor="item-type"
              required
              error={itemForm.formState.errors.itemType?.message}
            >
              <Controller
                control={itemForm.control}
                name="itemType"
                render={({ field }) => (
                  <Select
                    id="item-type"
                    value={field.value}
                    onValueChange={field.onChange}
                    options={[
                      { value: 'stock', label: 'Stock (Physical Inventory)' },
                      { value: 'service', label: 'Service (Non-stock)' },
                      { value: 'asset', label: 'Asset (Capital Item)' },
                    ]}
                  />
                )}
              />
            </FormField>
          </div>

          <FormField
            label="Item Name"
            htmlFor="item-name"
            required
            error={itemForm.formState.errors.name?.message}
          >
            <Input
              id="item-name"
              placeholder="e.g. Steel Coil 2mm Grade A"
              error={Boolean(itemForm.formState.errors.name)}
              {...itemForm.register('name')}
            />
          </FormField>

          <FormField
            label="Base UOM"
            htmlFor="item-uom"
            required
            hint="Unit code is normalized uppercase (e.g. PCS, KG, BOX, LTR)."
            error={itemForm.formState.errors.baseUom?.message}
          >
            <Input
              id="item-uom"
              placeholder="PCS"
              error={Boolean(itemForm.formState.errors.baseUom)}
              {...itemForm.register('baseUom')}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Party Detail Drawer */}
      <SideDrawer
        open={Boolean(selectedPartyId)}
        onOpenChange={(open) => {
          if (!open) setSelectedPartyId(null);
        }}
        title={partyDetailQuery.data ? `${partyDetailQuery.data.code} — ${partyDetailQuery.data.name}` : 'Partner Details'}
        description="Registered partner profile and physical addresses."
        footer={
          partyDetailQuery.data && canWrite ? (
            <>
              <Button
                variant="danger"
                size="sm"
                leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                onClick={() => setDeletingParty(partyDetailQuery.data)}
              >
                Delete
              </Button>
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Edit3 className="h-3.5 w-3.5" />}
                onClick={() => openEditParty(partyDetailQuery.data)}
              >
                Edit Partner
              </Button>
            </>
          ) : undefined
        }
      >
        {partyDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading partner details...</p>
        ) : partyDetailQuery.error ? (
          <ErrorState
            error={partyDetailQuery.error}
            onRetry={() => void partyDetailQuery.refetch()}
          />
        ) : partyDetailQuery.data ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Type</span>
                <div className="mt-1">
                  <StatusBadge
                    status={partyDetailQuery.data.partyType.toUpperCase()}
                    label={PARTY_TYPE_BADGE[partyDetailQuery.data.partyType]?.label}
                    tone={PARTY_TYPE_BADGE[partyDetailQuery.data.partyType]?.tone}
                  />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Tax ID</span>
                <p className="mt-1 font-mono text-xs font-medium text-slate-800 dark:text-slate-200">
                  {partyDetailQuery.data.taxId || 'Not registered'}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Visibility</span>
                <p className="mt-1 text-xs font-medium">
                  {partyDetailQuery.data.isShared ? 'Shared across orgs' : 'Organization local'}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Row Version</span>
                <p className="mt-1 font-mono text-xs">v{partyDetailQuery.data.rowVersion}</p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-slate-500">
                <MapPin className="h-3.5 w-3.5" /> Registered Addresses
              </h3>
              {partyDetailQuery.data.addresses.length === 0 ? (
                <p className="rounded-lg border border-dashed border-slate-200 p-4 text-xs text-slate-500 dark:border-slate-800">
                  No addresses recorded for this business partner.
                </p>
              ) : (
                <div className="space-y-2">
                  {partyDetailQuery.data.addresses.map((addr) => (
                    <div
                      key={addr.id}
                      className="rounded-lg border border-slate-200 bg-white p-3 text-xs dark:border-slate-800 dark:bg-slate-900"
                    >
                      <div className="mb-1 flex items-center justify-between">
                        <span className="font-semibold uppercase text-brand-700 dark:text-brand-300">
                          {addr.addressType}
                        </span>
                        {addr.isPrimary ? (
                          <StatusBadge status="ACTIVE" label="Primary" tone="success" />
                        ) : null}
                      </div>
                      <p className="text-slate-800 dark:text-slate-200">{addr.line1}</p>
                      {addr.line2 ? <p className="text-slate-600 dark:text-slate-400">{addr.line2}</p> : null}
                      <p className="text-slate-500">
                        {[addr.city, addr.stateProvince, addr.postalCode, addr.countryCode]
                          .filter(Boolean)
                          .join(', ')}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Item Detail Drawer */}
      <SideDrawer
        open={Boolean(selectedItemId)}
        onOpenChange={(open) => {
          if (!open) setSelectedItemId(null);
        }}
        title={itemDetailQuery.data ? `${itemDetailQuery.data.code} — ${itemDetailQuery.data.name}` : 'Item Details'}
        description="Item attributes and unit-of-measure conversions."
        footer={
          itemDetailQuery.data && canWrite ? (
            <>
              <Button
                variant="danger"
                size="sm"
                leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                onClick={() => setDeletingItem(itemDetailQuery.data)}
              >
                Delete
              </Button>
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Edit3 className="h-3.5 w-3.5" />}
                onClick={() => openEditItem(itemDetailQuery.data)}
              >
                Edit Item
              </Button>
            </>
          ) : undefined
        }
      >
        {itemDetailQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading item details...</p>
        ) : itemDetailQuery.error ? (
          <ErrorState error={itemDetailQuery.error} onRetry={() => void itemDetailQuery.refetch()} />
        ) : itemDetailQuery.data ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-800 dark:bg-slate-900">
              <div>
                <span className="text-xs text-slate-500">Item Type</span>
                <div className="mt-1">
                  <StatusBadge
                    status={itemDetailQuery.data.itemType.toUpperCase()}
                    label={ITEM_TYPE_BADGE[itemDetailQuery.data.itemType]?.label}
                    tone={ITEM_TYPE_BADGE[itemDetailQuery.data.itemType]?.tone}
                  />
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500">Base UOM</span>
                <p className="mt-1 font-mono text-sm font-semibold">{itemDetailQuery.data.baseUom}</p>
              </div>
              <div>
                <span className="text-xs text-slate-500">Concurrency Version</span>
                <p className="mt-1 font-mono text-xs">v{itemDetailQuery.data.rowVersion}</p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
                UOM Conversions
              </h3>
              {itemDetailQuery.data.uoms.length === 0 ? (
                <p className="rounded-lg border border-dashed border-slate-200 p-4 text-xs text-slate-500 dark:border-slate-800">
                  Uses base unit ({itemDetailQuery.data.baseUom}) only.
                </p>
              ) : (
                <div className="divide-y divide-slate-200 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
                  {itemDetailQuery.data.uoms.map((uom) => (
                    <div key={uom.id} className="flex items-center justify-between px-3 py-2 text-xs">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-semibold">{uom.uomCode}</span>
                        {uom.isBase ? <StatusBadge status="ACTIVE" label="Base" tone="brand" /> : null}
                      </div>
                      <span className="font-money text-slate-600 dark:text-slate-300">
                        1 {uom.uomCode} = {formatNumber(uom.conversionToBase, 4, locale)}{' '}
                        {itemDetailQuery.data.baseUom}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      {/* Delete Party Confirmation */}
      <AppDialog
        open={Boolean(deletingParty)}
        onOpenChange={(open) => {
          if (!open) setDeletingParty(null);
        }}
        title="Delete Business Partner"
        description={
          deletingParty
            ? `Permanently delete ${deletingParty.code} (${deletingParty.name})? Active orders referencing this partner may block deletion.`
            : undefined
        }
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeletingParty(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              isLoading={deletePartyMutation.isPending}
              onClick={() => deletingParty && deletePartyMutation.mutate(deletingParty)}
            >
              Delete Partner
            </Button>
          </>
        }
      >
        <p className="text-sm text-slate-600 dark:text-slate-300">
          This action sends an <code className="font-mono text-xs">If-Match: "{deletingParty?.rowVersion}"</code>{' '}
          precondition to prevent conflicting modifications.
        </p>
      </AppDialog>

      {/* Delete Item Confirmation */}
      <AppDialog
        open={Boolean(deletingItem)}
        onOpenChange={(open) => {
          if (!open) setDeletingItem(null);
        }}
        title="Delete Catalog Item"
        description={
          deletingItem
            ? `Permanently delete ${deletingItem.code} (${deletingItem.name})?`
            : undefined
        }
        size="sm"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeletingItem(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              isLoading={deleteItemMutation.isPending}
              onClick={() => deletingItem && deleteItemMutation.mutate(deletingItem)}
            >
              Delete Item
            </Button>
          </>
        }
      >
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Concurrency check: <code className="font-mono text-xs">v{deletingItem?.rowVersion}</code>.
        </p>
      </AppDialog>
    </div>
  );
}

export function MasterdataWorkspace() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <MasterdataContent />
    </DomainErrorBoundary>
  );
}
