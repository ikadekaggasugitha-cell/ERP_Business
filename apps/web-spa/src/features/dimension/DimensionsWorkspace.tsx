import * as React from 'react';
import { Play, Plus, SlidersHorizontal } from 'lucide-react';
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
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import {
  useAllocationRulesQuery,
  useCostCentersQuery,
  useCreateAllocationRuleMutation,
  useCreateCostCenterMutation,
  useCreateProfitCenterMutation,
  useExecuteAllocationRuleMutation,
  useProfitCentersQuery,
  type AllocationRuleDto,
  type CostCenterDto,
  type ProfitCenterDto,
} from './dimension-api';

export function DimensionsWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, ['dimension:write']);

  const [tab, setTab] = React.useState('cost-centers');
  const [createType, setCreateType] = React.useState<'cc' | 'pc' | 'rule' | null>(null);
  const [execRule, setExecRule] = React.useState<AllocationRuleDto | null>(null);

  const [code, setCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [basis, setBasis] = React.useState('Percentage');
  const [targetCcId, setTargetCcId] = React.useState('');
  const [execAmount, setExecAmount] = React.useState('10000000');
  const [execDate, setExecDate] = React.useState(new Date().toISOString().slice(0, 10));

  const ccQ = useCostCentersQuery(orgId);
  const pcQ = useProfitCentersQuery(orgId);
  const rulesQ = useAllocationRulesQuery(orgId);

  const createCC = useCreateCostCenterMutation();
  const createPC = useCreateProfitCenterMutation();
  const createRule = useCreateAllocationRuleMutation();
  const executeRule = useExecuteAllocationRuleMutation();

  const costCenters = ccQ.data?.items ?? [];
  const profitCenters = pcQ.data?.items ?? [];
  const rules = rulesQ.data?.items ?? [];

  const ccColumns: ColumnDef<CostCenterDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Cost Center Name', accessorKey: 'name' },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const pcColumns: ColumnDef<ProfitCenterDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Profit Center Name', accessorKey: 'name' },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const ruleColumns: ColumnDef<AllocationRuleDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Rule Name', accessorKey: 'name' },
    { header: 'Basis', accessorKey: 'basis', width: '140px' },
    {
      header: 'Status',
      width: '110px',
      cell: (r) => <StatusBadge status={r.isActive ? 'ACTIVE' : 'INACTIVE'} label={r.isActive ? 'Active' : 'Inactive'} />,
    },
    {
      header: 'Action',
      width: '120px',
      cell: (r) =>
        canWrite ? (
          <Button
            variant="outline"
            size="sm"
            leftIcon={<Play className="h-3.5 w-3.5" />}
            onClick={(e) => {
              e.stopPropagation();
              setExecRule(r);
            }}
          >
            Execute
          </Button>
        ) : null,
    },
  ];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (createType === 'cc') {
        await createCC.mutateAsync({ code: code.trim(), name: name.trim() });
        toast.success('Cost center created');
      } else if (createType === 'pc') {
        await createPC.mutateAsync({ code: code.trim(), name: name.trim() });
        toast.success('Profit center created');
      } else if (createType === 'rule') {
        const ccId = targetCcId || costCenters[0]?.id;
        if (!ccId) {
          toast.error('Create at least one Cost Center first');
          return;
        }
        await createRule.mutateAsync({
          code: code.trim(),
          name: name.trim(),
          basis,
          targets: [{ costCenterId: ccId, weightPct: '100' }],
        });
        toast.success('Allocation rule created');
      }
      setCreateType(null);
      setCode('');
      setName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Save failed');
    }
  };

  const handleExecute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!execRule) return;
    try {
      await executeRule.mutateAsync({ id: execRule.id, amount: execAmount, periodDate: execDate });
      toast.success(`Allocation rule ${execRule.code} executed`);
      setExecRule(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Execution failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Finance' }, { label: 'Dimensions & Allocations' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Analytical Dimensions & Cost Allocation
          </h1>
        </div>
        {canWrite ? (
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateType('cc')}>
              Cost Center
            </Button>
            <Button variant="outline" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateType('pc')}>
              Profit Center
            </Button>
            <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateType('rule')}>
              Allocation Rule
            </Button>
          </div>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Cost Centers" value={costCenters.length} icon={<SlidersHorizontal className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Profit Centers" value={profitCenters.length} icon={<SlidersHorizontal className="h-4 w-4" />} accent="emerald" />
        <KpiCard label="Allocation Rules" value={rules.length} icon={<SlidersHorizontal className="h-4 w-4" />} accent="sky" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="cost-centers">Cost Centers ({costCenters.length})</TabsTrigger>
          <TabsTrigger value="profit-centers">Profit Centers ({profitCenters.length})</TabsTrigger>
          <TabsTrigger value="rules">Allocation Rules ({rules.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="cost-centers" className="pt-4">
          <VirtualizedTable data={costCenters} columns={ccColumns} isLoading={ccQ.isLoading} error={ccQ.error} onRetry={() => void ccQ.refetch()} emptyTitle="No cost centers" />
        </TabsContent>
        <TabsContent value="profit-centers" className="pt-4">
          <VirtualizedTable data={profitCenters} columns={pcColumns} isLoading={pcQ.isLoading} error={pcQ.error} onRetry={() => void pcQ.refetch()} emptyTitle="No profit centers" />
        </TabsContent>
        <TabsContent value="rules" className="pt-4">
          <VirtualizedTable data={rules} columns={ruleColumns} isLoading={rulesQ.isLoading} error={rulesQ.error} onRetry={() => void rulesQ.refetch()} emptyTitle="No allocation rules" />
        </TabsContent>
      </Tabs>

      <AppDialog
        open={createType !== null}
        onOpenChange={(open) => !open && setCreateType(null)}
        title={createType === 'cc' ? 'New Cost Center' : createType === 'pc' ? 'New Profit Center' : 'New Allocation Rule'}
        size="md"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Code" htmlFor="dimCode" required>
            <Input id="dimCode" value={code} onChange={(e) => setCode(e.target.value)} placeholder="CC-OPS-01" required />
          </FormField>
          <FormField label="Name" htmlFor="dimName" required>
            <Input id="dimName" value={name} onChange={(e) => setName(e.target.value)} placeholder="Operations Division" required />
          </FormField>
          {createType === 'rule' ? (
            <>
              <FormField label="Basis" htmlFor="dimBasis">
                <Select
                  id="dimBasis"
                  value={basis}
                  onValueChange={setBasis}
                  options={[
                    { value: 'Percentage', label: 'Percentage' },
                    { value: 'Headcount', label: 'Headcount' },
                    { value: 'FloorArea', label: 'Floor Area' },
                  ]}
                />
              </FormField>
              <FormField label="Target Cost Center (100%)" htmlFor="dimTarget">
                <Select
                  id="dimTarget"
                  value={targetCcId || costCenters[0]?.id || ''}
                  onValueChange={setTargetCcId}
                  options={costCenters.map((c) => ({ value: c.id, label: `${c.code} · ${c.name}` }))}
                />
              </FormField>
            </>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateType(null)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createCC.isPending || createPC.isPending || createRule.isPending}>
              Save
            </Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={Boolean(execRule)} onOpenChange={(open) => !open && setExecRule(null)} title={`Execute Rule ${execRule?.code ?? ''}`} size="sm">
        <form onSubmit={handleExecute} className="space-y-4">
          <FormField label="Amount to Allocate" htmlFor="execAmt" required>
            <Input id="execAmt" value={execAmount} onChange={(e) => setExecAmount(e.target.value)} required />
          </FormField>
          <FormField label="Period Date" htmlFor="execDate" required>
            <Input id="execDate" type="date" value={execDate} onChange={(e) => setExecDate(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setExecRule(null)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={executeRule.isPending}>Execute</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
