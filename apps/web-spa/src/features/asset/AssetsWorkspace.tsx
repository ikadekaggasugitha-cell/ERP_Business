import * as React from 'react';
import { ArrowRightLeft, Boxes, CheckCircle2, Plus } from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { formatDate } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useAssetMovementsQuery,
  useAssetVerificationsQuery,
  useCreateOperationalAssetMutation,
  useOperationalAssetsQuery,
  useTransitionMovementMutation,
  type AssetMovementDto,
  type AssetVerificationDto,
  type OperationalAssetDto,
} from './asset-api';

export function AssetsWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, ['asset:write', 'asset.create']);

  const [tab, setTab] = React.useState('assets');
  const [statusFilter, setStatusFilter] = React.useState('ALL');
  const [createOpen, setCreateOpen] = React.useState(false);

  const [assetCode, setAssetCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [category, setCategory] = React.useState('HVAC');
  const [barcode, setBarcode] = React.useState('');
  const [status, setStatus] = React.useState('Active');

  const assetsQ = useOperationalAssetsQuery(orgId, statusFilter);
  const movementsQ = useAssetMovementsQuery(orgId);
  const verificationsQ = useAssetVerificationsQuery(orgId);
  const createAsset = useCreateOperationalAssetMutation();
  const transitionMovement = useTransitionMovementMutation();

  const assets = assetsQ.data?.items ?? [];
  const movements = movementsQ.data?.items ?? [];
  const verifications = verificationsQ.data?.items ?? [];

  const assetCols: ColumnDef<OperationalAssetDto>[] = [
    { header: 'Asset Code', accessorKey: 'assetCode', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Name', accessorKey: 'name' },
    { header: 'Category', accessorKey: 'category', width: '130px' },
    { header: 'Barcode', width: '140px', cell: (r) => <span className="font-mono text-xs">{r.barcode ?? '—'}</span> },
    { header: 'Status', width: '140px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const movementCols: ColumnDef<AssetMovementDto>[] = [
    { header: 'Asset ID', accessorKey: 'assetId', className: 'font-mono text-xs' },
    { header: 'Target Org', accessorKey: 'toOrganizationRef', className: 'font-mono text-xs' },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '130px',
      cell: (r) => {
        if (!canWrite) return null;
        if (r.status === 'Draft') {
          return (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void transitionMovement.mutateAsync({ id: r.id, action: 'approve', rowVersion: r.rowVersion }).then(
                  () => toast.success('Movement approved'),
                  (err) => toast.error(err instanceof Error ? err.message : 'Approve failed')
                )
              }
            >
              Approve
            </Button>
          );
        }
        if (r.status === 'Approved') {
          return (
            <Button
              size="sm"
              variant="primary"
              onClick={() =>
                void transitionMovement.mutateAsync({ id: r.id, action: 'complete', rowVersion: r.rowVersion }).then(
                  () => toast.success('Movement completed'),
                  (err) => toast.error(err instanceof Error ? err.message : 'Complete failed')
                )
              }
            >
              Complete
            </Button>
          );
        }
        return null;
      },
    },
  ];

  const verCols: ColumnDef<AssetVerificationDto>[] = [
    { header: 'Date', width: '130px', cell: (r) => formatDate(r.verificationDate, locale) },
    { header: 'Asset ID', accessorKey: 'assetId', className: 'font-mono text-xs' },
    { header: 'System', width: '120px', cell: (r) => r.systemStatus ?? '—' },
    { header: 'Physical', width: '120px', cell: (r) => r.physicalStatus ?? '—' },
    { header: 'Note', cell: (r) => r.discrepancyNote ?? '—' },
  ];

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createAsset.mutateAsync({
        assetCode: assetCode.trim(),
        name: name.trim(),
        category: category.trim(),
        barcode: barcode.trim() || undefined,
        status,
      });
      toast.success('Operational asset registered');
      setCreateOpen(false);
      setAssetCode('');
      setName('');
      setBarcode('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Registration failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Operations' }, { label: 'Operational Assets' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Operational Asset Registry, Movements & Verification
          </h1>
        </div>
        {canWrite ? (
          <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateOpen(true)}>
            Register Asset
          </Button>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Operational Assets" value={assets.length} icon={<Boxes className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Asset Movements" value={movements.length} icon={<ArrowRightLeft className="h-4 w-4" />} accent="amber" />
        <KpiCard label="Physical Verifications" value={verifications.length} icon={<CheckCircle2 className="h-4 w-4" />} accent="emerald" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="assets">Assets ({assets.length})</TabsTrigger>
          <TabsTrigger value="movements">Movements ({movements.length})</TabsTrigger>
          <TabsTrigger value="verifications">Verifications ({verifications.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="assets" className="pt-4">
          <VirtualizedTable
            data={assets}
            columns={assetCols}
            isLoading={assetsQ.isLoading}
            error={assetsQ.error}
            onRetry={() => void assetsQ.refetch()}
            emptyTitle="No operational assets"
            toolbar={{
              filters: (
                <Select
                  value={statusFilter}
                  onValueChange={setStatusFilter}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Active', label: 'Active' },
                    { value: 'UnderMaintenance', label: 'Under Maintenance' },
                    { value: 'Idle', label: 'Idle' },
                    { value: 'Damaged', label: 'Damaged' },
                    { value: 'Disposed', label: 'Disposed' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
        <TabsContent value="movements" className="pt-4">
          <VirtualizedTable data={movements} columns={movementCols} isLoading={movementsQ.isLoading} error={movementsQ.error} onRetry={() => void movementsQ.refetch()} emptyTitle="No movements" />
        </TabsContent>
        <TabsContent value="verifications" className="pt-4">
          <VirtualizedTable data={verifications} columns={verCols} isLoading={verificationsQ.isLoading} error={verificationsQ.error} onRetry={() => void verificationsQ.refetch()} emptyTitle="No verifications" />
        </TabsContent>
      </Tabs>

      <AppDialog open={createOpen} onOpenChange={setCreateOpen} title="Register Operational Asset" size="md">
        <form onSubmit={handleCreate} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Asset Code" htmlFor="opAstCode" required>
              <Input id="opAstCode" value={assetCode} onChange={(e) => setAssetCode(e.target.value)} placeholder="AST-HVAC-01" required />
            </FormField>
            <FormField label="Category" htmlFor="opAstCat" required>
              <Input id="opAstCat" value={category} onChange={(e) => setCategory(e.target.value)} required />
            </FormField>
          </div>
          <FormField label="Asset Name" htmlFor="opAstName" required>
            <Input id="opAstName" value={name} onChange={(e) => setName(e.target.value)} placeholder="Chiller Unit Lobby A" required />
          </FormField>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Barcode / Tag" htmlFor="opAstBar">
              <Input id="opAstBar" value={barcode} onChange={(e) => setBarcode(e.target.value)} placeholder="BC-000123" />
            </FormField>
            <FormField label="Status" htmlFor="opAstStatus">
              <Select
                id="opAstStatus"
                value={status}
                onValueChange={setStatus}
                options={[
                  { value: 'Active', label: 'Active' },
                  { value: 'Idle', label: 'Idle' },
                  { value: 'UnderMaintenance', label: 'Under Maintenance' },
                ]}
              />
            </FormField>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createAsset.isPending}>Register</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
