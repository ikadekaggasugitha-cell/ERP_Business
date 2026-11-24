import * as React from 'react';
import { Building2, Calculator, CheckCircle2, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { addDecimals, formatCurrency, formatDate } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useAssetCategoriesQuery,
  useAssetDepreciationsQuery,
  useAssetTransactionsQuery,
  useCapitalizeAssetMutation,
  useCreateCategoryMutation,
  useCreateFixedAssetMutation,
  useDepreciateAssetMutation,
  useDisposeAssetMutation,
  useFixedAssetsQuery,
  type AssetCategoryDto,
  type FixedAssetDto,
} from './fixedasset-api';

const ZERO_UUID = '00000000-0000-0000-0000-000000000001';

export function FixedAssetsWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const orgId = activeOrg?.id;
  const currency = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, ['fixedasset:write', 'fixedasset.create']);

  const [tab, setTab] = React.useState('assets');
  const [statusFilter, setStatusFilter] = React.useState('ALL');
  const [page, setPage] = React.useState(1);
  const [selectedAsset, setSelectedAsset] = React.useState<FixedAssetDto | null>(null);
  const [createAssetOpen, setCreateAssetOpen] = React.useState(false);
  const [createCatOpen, setCreateCatOpen] = React.useState(false);
  const [depreciateOpen, setDepreciateOpen] = React.useState(false);
  const [disposeOpen, setDisposeOpen] = React.useState(false);

  const [catCode, setCatCode] = React.useState('');
  const [catName, setCatName] = React.useState('');
  const [catMethod, setCatMethod] = React.useState('StraightLine');
  const [catLife, setCatLife] = React.useState('60');

  const [assetNo, setAssetNo] = React.useState('');
  const [assetName, setAssetName] = React.useState('');
  const [assetCategoryId, setAssetCategoryId] = React.useState('');
  const [acqDate, setAcqDate] = React.useState(new Date().toISOString().slice(0, 10));
  const [acqCost, setAcqCost] = React.useState('100000000');
  const [salvageVal, setSalvageVal] = React.useState('0');
  const [usefulLife, setUsefulLife] = React.useState('60');
  const [depMethod, setDepMethod] = React.useState('StraightLine');

  const [fiscalPeriodRef, setFiscalPeriodRef] = React.useState(ZERO_UUID);
  const [runDate, setRunDate] = React.useState(new Date().toISOString().slice(0, 10));
  const [proceedsAmount, setProceedsAmount] = React.useState('0');

  const assetsQ = useFixedAssetsQuery(orgId, statusFilter, page);
  const categoriesQ = useAssetCategoriesQuery(orgId);
  const depQ = useAssetDepreciationsQuery(orgId, selectedAsset?.id);
  const txnQ = useAssetTransactionsQuery(orgId, selectedAsset?.id);

  const createCat = useCreateCategoryMutation();
  const createAsset = useCreateFixedAssetMutation();
  const capitalize = useCapitalizeAssetMutation();
  const depreciate = useDepreciateAssetMutation();
  const dispose = useDisposeAssetMutation();

  const assets = assetsQ.data?.items ?? [];
  const categories = categoriesQ.data?.items ?? [];

  const totals = React.useMemo(() => {
    let totalCost = '0';
    let totalNbv = '0';
    let activeCount = 0;
    for (const a of assets) {
      totalCost = addDecimals(totalCost, a.acquisitionCost, 4);
      totalNbv = addDecimals(totalNbv, a.netBookValue, 4);
      if (a.status === 'Active') activeCount++;
    }
    return { totalCost, totalNbv, activeCount };
  }, [assets]);

  const assetColumns: ColumnDef<FixedAssetDto>[] = [
    { header: 'Asset No', accessorKey: 'assetNo', width: '130px', className: 'font-mono font-semibold' },
    { header: 'Name', accessorKey: 'name' },
    { header: 'Method', accessorKey: 'depreciationMethod', width: '140px' },
    {
      header: 'Acquisition Cost',
      align: 'right',
      width: '160px',
      cell: (r) => <span className="font-mono tabular-nums">{formatCurrency(r.acquisitionCost, currency, locale)}</span>,
    },
    {
      header: 'Net Book Value',
      align: 'right',
      width: '160px',
      cell: (r) => <span className="font-mono tabular-nums font-semibold">{formatCurrency(r.netBookValue, currency, locale)}</span>,
    },
    {
      header: 'Status',
      width: '120px',
      cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} />,
    },
  ];

  const categoryColumns: ColumnDef<AssetCategoryDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '130px', className: 'font-mono font-semibold' },
    { header: 'Category Name', accessorKey: 'name' },
    { header: 'Default Method', accessorKey: 'defaultDepreciationMethod', width: '160px' },
    {
      header: 'Useful Life (Mo)',
      align: 'right',
      width: '130px',
      cell: (r) => <span className="font-mono tabular-nums">{r.defaultUsefulLifeMonths ?? '—'}</span>,
    },
    {
      header: 'Scope',
      width: '110px',
      cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} />,
    },
  ];

  const handleCreateCategory = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createCat.mutateAsync({
        code: catCode.trim(),
        name: catName.trim(),
        defaultDepreciationMethod: catMethod,
        defaultUsefulLifeMonths: Number(catLife) || 60,
      });
      toast.success('Asset category created');
      setCreateCatOpen(false);
      setCatCode('');
      setCatName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create category');
    }
  };

  const handleCreateAsset = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!orgId) return;
    try {
      await createAsset.mutateAsync({
        assetNo: assetNo.trim(),
        name: assetName.trim(),
        assetCategoryId: assetCategoryId || categories[0]?.id || ZERO_UUID,
        financialOrganizationRef: orgId,
        acquisitionDate: acqDate,
        acquisitionCost: acqCost,
        salvageValue: salvageVal,
        usefulLifeMonths: Number(usefulLife) || 60,
        depreciationMethod: depMethod,
        assetAccountId: ZERO_UUID,
        accumDepreciationAccountId: ZERO_UUID,
        depreciationExpenseAccountId: ZERO_UUID,
      });
      toast.success('Fixed asset draft created');
      setCreateAssetOpen(false);
      setAssetNo('');
      setAssetName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create fixed asset');
    }
  };

  const handleCapitalize = async (asset: FixedAssetDto) => {
    try {
      await capitalize.mutateAsync({
        id: asset.id,
        rowVersion: asset.rowVersion,
        transactionDate: new Date().toISOString().slice(0, 10),
      });
      toast.success(`Asset ${asset.assetNo} capitalized`);
      setSelectedAsset(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Capitalize failed');
    }
  };

  const handleDepreciate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedAsset) return;
    try {
      const res = await depreciate.mutateAsync({
        id: selectedAsset.id,
        rowVersion: selectedAsset.rowVersion,
        fiscalPeriodRef,
        runDate,
      });
      toast.success(`Depreciation posted: ${formatCurrency(res.depreciationAmount, currency, locale)}`);
      setDepreciateOpen(false);
      setSelectedAsset(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Depreciation failed');
    }
  };

  const handleDispose = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedAsset) return;
    try {
      await dispose.mutateAsync({
        id: selectedAsset.id,
        rowVersion: selectedAsset.rowVersion,
        transactionDate: runDate,
        proceedsAmount,
      });
      toast.success(`Asset ${selectedAsset.assetNo} disposed`);
      setDisposeOpen(false);
      setSelectedAsset(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Disposal failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Finance' }, { label: 'Fixed Assets' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Fixed Assets & Depreciation Register
          </h1>
        </div>
        {canWrite ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateCatOpen(true)}>
              New Category
            </Button>
            <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateAssetOpen(true)}>
              New Fixed Asset
            </Button>
          </div>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard
          label="Total Acquisition Cost"
          value={<span className="font-mono tabular-nums">{formatCurrency(totals.totalCost, currency, locale)}</span>}
          icon={<Building2 className="h-4 w-4" />}
          accent="brand"
          isLoading={assetsQ.isLoading}
        />
        <KpiCard
          label="Total Net Book Value"
          value={<span className="font-mono tabular-nums">{formatCurrency(totals.totalNbv, currency, locale)}</span>}
          icon={<Calculator className="h-4 w-4" />}
          accent="emerald"
          isLoading={assetsQ.isLoading}
        />
        <KpiCard
          label="Active Assets / Categories"
          value={`${totals.activeCount} / ${categories.length}`}
          icon={<CheckCircle2 className="h-4 w-4" />}
          accent="sky"
          isLoading={assetsQ.isLoading}
        />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="assets">Fixed Assets ({assets.length})</TabsTrigger>
          <TabsTrigger value="categories">Categories ({categories.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="assets" className="pt-4">
          <VirtualizedTable
            data={assets}
            columns={assetColumns}
            isLoading={assetsQ.isLoading}
            error={assetsQ.error}
            onRetry={() => void assetsQ.refetch()}
            onRowClick={(r) => setSelectedAsset(r)}
            emptyTitle="No fixed assets found"
            emptyDescription="Register a fixed asset draft and capitalize it to begin depreciation tracking."
            toolbar={{
              filters: (
                <Select
                  value={statusFilter}
                  onValueChange={(v) => {
                    setStatusFilter(v);
                    setPage(1);
                  }}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Draft', label: 'Draft' },
                    { value: 'Active', label: 'Active' },
                    { value: 'Disposed', label: 'Disposed' },
                  ]}
                />
              ),
            }}
            pagination={
              assetsQ.data?.page
                ? {
                    page,
                    size: assetsQ.data.page.size,
                    totalElements: assetsQ.data.page.totalElements,
                    totalPages: assetsQ.data.page.totalPages,
                    onPageChange: setPage,
                  }
                : undefined
            }
          />
        </TabsContent>

        <TabsContent value="categories" className="pt-4">
          <VirtualizedTable
            data={categories}
            columns={categoryColumns}
            isLoading={categoriesQ.isLoading}
            error={categoriesQ.error}
            onRetry={() => void categoriesQ.refetch()}
            emptyTitle="No asset categories"
          />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedAsset)}
        onOpenChange={(open) => !open && setSelectedAsset(null)}
        title={selectedAsset ? `${selectedAsset.assetNo} · ${selectedAsset.name}` : 'Asset Detail'}
        titleBadge={selectedAsset ? <StatusBadge status={selectedAsset.status.toUpperCase()} label={selectedAsset.status} /> : null}
        size="xl"
        footer={
          selectedAsset && canWrite ? (
            <div className="flex flex-wrap gap-2 justify-end">
              {selectedAsset.status === 'Draft' ? (
                <Button variant="primary" size="sm" isLoading={capitalize.isPending} onClick={() => void handleCapitalize(selectedAsset)}>
                  Capitalize Asset
                </Button>
              ) : null}
              {selectedAsset.status === 'Active' ? (
                <>
                  <Button variant="outline" size="sm" onClick={() => setDepreciateOpen(true)}>
                    Post Depreciation
                  </Button>
                  <Button variant="danger" size="sm" leftIcon={<Trash2 className="h-3.5 w-3.5" />} onClick={() => setDisposeOpen(true)}>
                    Dispose Asset
                  </Button>
                </>
              ) : null}
            </div>
          ) : null
        }
      >
        {selectedAsset ? (
          <div className="space-y-5 text-xs">
            <div className="grid grid-cols-2 gap-3 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
              <div>
                <p className="text-slate-500">Acquisition Date</p>
                <p className="font-mono font-semibold">{formatDate(selectedAsset.acquisitionDate, locale)}</p>
              </div>
              <div>
                <p className="text-slate-500">Depreciation Method</p>
                <p className="font-semibold">{selectedAsset.depreciationMethod} ({selectedAsset.usefulLifeMonths} mo)</p>
              </div>
              <div>
                <p className="text-slate-500">Acquisition Cost</p>
                <p className="font-mono tabular-nums font-semibold">{formatCurrency(selectedAsset.acquisitionCost, currency, locale)}</p>
              </div>
              <div>
                <p className="text-slate-500">Net Book Value</p>
                <p className="font-mono tabular-nums font-bold text-emerald-700 dark:text-emerald-400">
                  {formatCurrency(selectedAsset.netBookValue, currency, locale)}
                </p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">Depreciation Schedule</h3>
              {(depQ.data ?? []).length === 0 ? (
                <p className="text-slate-500">No depreciation entries posted yet.</p>
              ) : (
                <div className="space-y-1.5">
                  {(depQ.data ?? []).map((d) => (
                    <div key={d.id} className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 px-3 py-2">
                      <span className="font-mono">{formatDate(d.runDate, locale)}</span>
                      <span className="font-mono tabular-nums">{formatCurrency(d.depreciationAmount, currency, locale)}</span>
                      <span className="font-mono tabular-nums text-slate-500">NBV: {formatCurrency(d.nbvAfter, currency, locale)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div>
              <h3 className="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">Transactions</h3>
              {(txnQ.data ?? []).length === 0 ? (
                <p className="text-slate-500">No asset transactions recorded.</p>
              ) : (
                <div className="space-y-1.5">
                  {(txnQ.data ?? []).map((t) => (
                    <div key={t.id} className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 px-3 py-2">
                      <span className="font-semibold">{t.transactionType}</span>
                      <span className="font-mono">{formatDate(t.transactionDate, locale)}</span>
                      <span className="font-mono tabular-nums">{formatCurrency(t.amount, currency, locale)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      <AppDialog open={createCatOpen} onOpenChange={setCreateCatOpen} title="Create Asset Category">
        <form onSubmit={handleCreateCategory} className="space-y-4">
          <FormField label="Code" htmlFor="faCatCode" required>
            <Input id="faCatCode" value={catCode} onChange={(e) => setCatCode(e.target.value)} placeholder="BLDG" required />
          </FormField>
          <FormField label="Name" htmlFor="faCatName" required>
            <Input id="faCatName" value={catName} onChange={(e) => setCatName(e.target.value)} placeholder="Buildings & Structures" required />
          </FormField>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Default Method" htmlFor="faCatMethod">
              <Select
                id="faCatMethod"
                value={catMethod}
                onValueChange={setCatMethod}
                options={[
                  { value: 'StraightLine', label: 'Straight Line' },
                  { value: 'DecliningBalance', label: 'Declining Balance' },
                  { value: 'UnitsOfProduction', label: 'Units of Production' },
                ]}
              />
            </FormField>
            <FormField label="Useful Life (Months)" htmlFor="faCatLife">
              <Input id="faCatLife" type="number" min={1} value={catLife} onChange={(e) => setCatLife(e.target.value)} />
            </FormField>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateCatOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createCat.isPending}>Create</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={createAssetOpen} onOpenChange={setCreateAssetOpen} title="Register Fixed Asset">
        <form onSubmit={handleCreateAsset} className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Asset No" htmlFor="faNo" required>
              <Input id="faNo" value={assetNo} onChange={(e) => setAssetNo(e.target.value)} placeholder="FA-2026-001" required />
            </FormField>
            <FormField label="Category" htmlFor="faCat" required>
              <Select
                id="faCat"
                value={assetCategoryId || categories[0]?.id || ''}
                onValueChange={setAssetCategoryId}
                options={categories.map((c) => ({ value: c.id, label: `${c.code} · ${c.name}` }))}
              />
            </FormField>
          </div>
          <FormField label="Asset Name" htmlFor="faName" required>
            <Input id="faName" value={assetName} onChange={(e) => setAssetName(e.target.value)} placeholder="Tower A HVAC Chiller" required />
          </FormField>
          <div className="grid grid-cols-3 gap-3">
            <FormField label="Acquisition Date" htmlFor="faAcqDate" required>
              <Input id="faAcqDate" type="date" value={acqDate} onChange={(e) => setAcqDate(e.target.value)} required />
            </FormField>
            <FormField label="Acquisition Cost" htmlFor="faCost" required>
              <Input id="faCost" value={acqCost} onChange={(e) => setAcqCost(e.target.value)} required />
            </FormField>
            <FormField label="Salvage Value" htmlFor="faSalvage">
              <Input id="faSalvage" value={salvageVal} onChange={(e) => setSalvageVal(e.target.value)} />
            </FormField>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Method" htmlFor="faMethod">
              <Select
                id="faMethod"
                value={depMethod}
                onValueChange={setDepMethod}
                options={[
                  { value: 'StraightLine', label: 'Straight Line' },
                  { value: 'DecliningBalance', label: 'Declining Balance' },
                  { value: 'UnitsOfProduction', label: 'Units of Production' },
                ]}
              />
            </FormField>
            <FormField label="Useful Life (Months)" htmlFor="faLife">
              <Input id="faLife" type="number" min={1} value={usefulLife} onChange={(e) => setUsefulLife(e.target.value)} />
            </FormField>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={() => setCreateAssetOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createAsset.isPending}>Create Draft</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={depreciateOpen} onOpenChange={setDepreciateOpen} title="Post Monthly Depreciation" size="sm">
        <form onSubmit={handleDepreciate} className="space-y-4">
          <FormField label="Fiscal Period UUID" htmlFor="depPeriod" required>
            <Input id="depPeriod" value={fiscalPeriodRef} onChange={(e) => setFiscalPeriodRef(e.target.value)} required />
          </FormField>
          <FormField label="Run Date" htmlFor="depDate" required>
            <Input id="depDate" type="date" value={runDate} onChange={(e) => setRunDate(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setDepreciateOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={depreciate.isPending}>Post Depreciation</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={disposeOpen} onOpenChange={setDisposeOpen} title="Dispose Fixed Asset" size="sm">
        <form onSubmit={handleDispose} className="space-y-4">
          <FormField label="Disposal Date" htmlFor="dispDate" required>
            <Input id="dispDate" type="date" value={runDate} onChange={(e) => setRunDate(e.target.value)} required />
          </FormField>
          <FormField label="Proceeds Amount" htmlFor="dispProceeds" required>
            <Input id="dispProceeds" value={proceedsAmount} onChange={(e) => setProceedsAmount(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setDisposeOpen(false)}>Cancel</Button>
            <Button type="submit" variant="danger" isLoading={dispose.isPending}>Confirm Disposal</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
