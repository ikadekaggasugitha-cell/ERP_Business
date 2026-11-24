import * as React from 'react';
import { CheckCircle2, Layers, Plus, ShieldAlert } from 'lucide-react';
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
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import {
  useConsolidationGroupsQuery,
  useConsolidationRunsQuery,
  useCreateConsolidationGroupMutation,
  useCreateConsolidationRunMutation,
  useEliminationRulesQuery,
  useFinalizeConsolidationRunMutation,
  useGroupMembersQuery,
  type ConsolidationGroupDto,
  type ConsolidationRunDto,
  type EliminationRuleDto,
} from './consolidation-api';

const ZERO_UUID = '00000000-0000-0000-0000-000000000001';

export function ConsolidationWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const orgId = activeOrg?.id;

  const isHeadOfficeTier = React.useMemo(() => {
    if (!user) return false;
    if (typeof user.adminTier === 'number' && user.adminTier <= 1) return true;
    const roles = new Set((user.roles ?? []).map((r) => r.trim().toUpperCase()));
    return roles.has('SYSTEM_ADMIN') || roles.has('HOLDING_ADMIN');
  }, [user]);

  const canWrite = isHeadOfficeTier && hasUserPermission(user, ['consolidation:write', 'consolidation.create']);

  const [tab, setTab] = React.useState('groups');
  const [selectedGroup, setSelectedGroup] = React.useState<ConsolidationGroupDto | null>(null);
  const [createGroupOpen, setCreateGroupOpen] = React.useState(false);
  const [createRunOpen, setCreateRunOpen] = React.useState(false);

  const [code, setCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [currencyCode, setCurrencyCode] = React.useState('IDR');
  const [runGroupId, setRunGroupId] = React.useState('');
  const [fiscalPeriodRef, setFiscalPeriodRef] = React.useState(ZERO_UUID);

  const groupsQ = useConsolidationGroupsQuery(orgId);
  const membersQ = useGroupMembersQuery(orgId, selectedGroup?.id);
  const rulesQ = useEliminationRulesQuery(orgId);
  const runsQ = useConsolidationRunsQuery(orgId);

  const createGroup = useCreateConsolidationGroupMutation();
  const createRun = useCreateConsolidationRunMutation();
  const finalizeRun = useFinalizeConsolidationRunMutation();

  const groups = groupsQ.data?.items ?? [];
  const rules = rulesQ.data?.items ?? [];
  const runs = runsQ.data?.items ?? [];

  const groupCols: ColumnDef<ConsolidationGroupDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Group Name', accessorKey: 'name' },
    { header: 'Presentation CCY', accessorKey: 'presentationCurrencyCode', width: '140px', className: 'font-mono' },
    { header: 'Scope', width: '110px', cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} /> },
  ];

  const ruleCols: ColumnDef<EliminationRuleDto>[] = [
    { header: 'Rule Type', accessorKey: 'ruleType', width: '180px', className: 'font-semibold' },
    { header: 'Group ID', accessorKey: 'consolidationGroupId', className: 'font-mono text-xs' },
    { header: 'Status', width: '110px', cell: (r) => <StatusBadge status={r.isActive ? 'ACTIVE' : 'INACTIVE'} label={r.isActive ? 'Active' : 'Inactive'} /> },
  ];

  const runCols: ColumnDef<ConsolidationRunDto>[] = [
    { header: 'Group ID', accessorKey: 'consolidationGroupId', className: 'font-mono text-xs' },
    { header: 'Fiscal Period', accessorKey: 'fiscalPeriodRef', className: 'font-mono text-xs' },
    { header: 'Version', width: '90px', align: 'right', cell: (r) => <span className="font-mono">v{r.version}</span> },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '120px',
      cell: (r) =>
        canWrite && r.status !== 'Finalized' ? (
          <Button
            size="sm"
            variant="primary"
            onClick={(e) => {
              e.stopPropagation();
              void finalizeRun.mutateAsync({ id: r.id, rowVersion: r.rowVersion }).then(
                () => toast.success('Consolidation run finalized'),
                (err) => toast.error(err instanceof Error ? err.message : 'Finalize failed')
              );
            }}
          >
            Finalize
          </Button>
        ) : null,
    },
  ];

  const handleCreateGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createGroup.mutateAsync({ code: code.trim(), name: name.trim(), presentationCurrencyCode: currencyCode.toUpperCase() });
      toast.success('Consolidation group created');
      setCreateGroupOpen(false);
      setCode('');
      setName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Create failed');
    }
  };

  const handleCreateRun = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createRun.mutateAsync({
        consolidationGroupId: runGroupId || groups[0]?.id || ZERO_UUID,
        fiscalPeriodRef,
        version: 1,
      });
      toast.success('Consolidation run created');
      setCreateRunOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Create run failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Finance' }, { label: 'Group Consolidation' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Multi-Entity Group Consolidation
          </h1>
        </div>
        {canWrite ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateGroupOpen(true)}>
              New Group
            </Button>
            <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateRunOpen(true)}>
              New Run
            </Button>
          </div>
        ) : null}
      </div>

      {!isHeadOfficeTier ? (
        <div className="flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50/70 dark:border-amber-900/50 dark:bg-amber-950/20 px-4 py-3 text-xs text-amber-900 dark:text-amber-200">
          <ShieldAlert className="h-4 w-4 shrink-0" />
          <span>Head Office Tier Guard: Consolidation mutations require HOLDING_ADMIN or SYSTEM_ADMIN role. Read-only view active.</span>
        </div>
      ) : null}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Consolidation Groups" value={groups.length} icon={<Layers className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Elimination Rules" value={rules.length} icon={<Layers className="h-4 w-4" />} accent="amber" />
        <KpiCard label="Consolidation Runs" value={runs.length} icon={<CheckCircle2 className="h-4 w-4" />} accent="emerald" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="groups">Groups ({groups.length})</TabsTrigger>
          <TabsTrigger value="rules">Elimination Rules ({rules.length})</TabsTrigger>
          <TabsTrigger value="runs">Runs ({runs.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="groups" className="pt-4">
          <VirtualizedTable data={groups} columns={groupCols} isLoading={groupsQ.isLoading} error={groupsQ.error} onRetry={() => void groupsQ.refetch()} onRowClick={setSelectedGroup} emptyTitle="No consolidation groups" />
        </TabsContent>
        <TabsContent value="rules" className="pt-4">
          <VirtualizedTable data={rules} columns={ruleCols} isLoading={rulesQ.isLoading} error={rulesQ.error} onRetry={() => void rulesQ.refetch()} emptyTitle="No elimination rules" />
        </TabsContent>
        <TabsContent value="runs" className="pt-4">
          <VirtualizedTable data={runs} columns={runCols} isLoading={runsQ.isLoading} error={runsQ.error} onRetry={() => void runsQ.refetch()} emptyTitle="No consolidation runs" />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedGroup)}
        onOpenChange={(open) => !open && setSelectedGroup(null)}
        title={selectedGroup ? `${selectedGroup.code} · ${selectedGroup.name}` : 'Group Members'}
      >
        <div className="space-y-3 text-xs">
          <h3 className="font-bold uppercase tracking-wider text-slate-500">Group Members & Ownership</h3>
          {(membersQ.data ?? []).length === 0 ? (
            <p className="text-slate-500">No subsidiary members assigned yet.</p>
          ) : (
            (membersQ.data ?? []).map((m) => (
              <div key={m.id} className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 px-3 py-2">
                <span className="font-mono">{m.companyRef}</span>
                <span className="font-semibold">{m.consolidationMethod}</span>
                <span className="font-mono tabular-nums">{m.ownershipPercentage}%</span>
              </div>
            ))
          )}
        </div>
      </SideDrawer>

      <AppDialog open={createGroupOpen} onOpenChange={setCreateGroupOpen} title="Create Consolidation Group" size="md">
        <form onSubmit={handleCreateGroup} className="space-y-4">
          <FormField label="Group Code" htmlFor="cgCode" required>
            <Input id="cgCode" value={code} onChange={(e) => setCode(e.target.value)} placeholder="GRP-SINERGI" required />
          </FormField>
          <FormField label="Group Name" htmlFor="cgName" required>
            <Input id="cgName" value={name} onChange={(e) => setName(e.target.value)} placeholder="Sinergi Consolidated Group" required />
          </FormField>
          <FormField label="Presentation Currency" htmlFor="cgCcy" required>
            <Input id="cgCcy" maxLength={3} value={currencyCode} onChange={(e) => setCurrencyCode(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateGroupOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createGroup.isPending}>Create</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={createRunOpen} onOpenChange={setCreateRunOpen} title="Start Consolidation Run" size="sm">
        <form onSubmit={handleCreateRun} className="space-y-4">
          <FormField label="Consolidation Group" htmlFor="crGrp" required>
            <Select
              id="crGrp"
              value={runGroupId || groups[0]?.id || ''}
              onValueChange={setRunGroupId}
              options={groups.map((g) => ({ value: g.id, label: `${g.code} · ${g.name}` }))}
            />
          </FormField>
          <FormField label="Fiscal Period UUID" htmlFor="crPeriod" required>
            <Input id="crPeriod" value={fiscalPeriodRef} onChange={(e) => setFiscalPeriodRef(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateRunOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createRun.isPending}>Start Run</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
