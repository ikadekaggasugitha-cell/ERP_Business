import * as React from 'react';
import { CheckCircle2, PieChart, Plus, ShieldCheck } from 'lucide-react';
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
import { addDecimals, formatCurrency } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useAddBudgetLineMutation,
  useBudgetCommitmentsQuery,
  useBudgetDetailQuery,
  useBudgetPoliciesQuery,
  useBudgetsQuery,
  useCreateBudgetMutation,
  useCreatePolicyMutation,
  useTransitionBudgetMutation,
  type BudgetCommitmentDto,
  type BudgetDto,
  type BudgetPolicyDto,
} from './budget-api';

const ZERO_UUID = '00000000-0000-0000-0000-000000000001';

export function BudgetsWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const orgId = activeOrg?.id;
  const currency = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, ['budget:write', 'budget.create']);
  const canApprove = hasUserPermission(user, ['budget:approve', 'budget.approve']);

  const [tab, setTab] = React.useState('budgets');
  const [statusFilter, setStatusFilter] = React.useState('ALL');
  const [selectedBudget, setSelectedBudget] = React.useState<BudgetDto | null>(null);
  const [createBudgetOpen, setCreateBudgetOpen] = React.useState(false);
  const [createPolicyOpen, setCreatePolicyOpen] = React.useState(false);
  const [addLineOpen, setAddLineOpen] = React.useState(false);

  const [code, setCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [fiscalYearRef, setFiscalYearRef] = React.useState(ZERO_UUID);
  const [txType, setTxType] = React.useState('PO');
  const [controlType, setControlType] = React.useState('HardStop');
  const [linePeriod, setLinePeriod] = React.useState('1');
  const [lineAmount, setLineAmount] = React.useState('50000000');

  const budgetsQ = useBudgetsQuery(orgId, statusFilter);
  const detailQ = useBudgetDetailQuery(orgId, selectedBudget?.id);
  const policiesQ = useBudgetPoliciesQuery(orgId);
  const commitmentsQ = useBudgetCommitmentsQuery(orgId);

  const createBudget = useCreateBudgetMutation();
  const transitionBudget = useTransitionBudgetMutation();
  const createPolicy = useCreatePolicyMutation();
  const addLine = useAddBudgetLineMutation();

  const budgets = budgetsQ.data?.items ?? [];
  const policies = policiesQ.data?.items ?? [];
  const commitments = commitmentsQ.data?.items ?? [];

  const totalCommitted = React.useMemo(
    () => commitments.reduce((acc, c) => addDecimals(acc, c.committedAmount, 4), '0'),
    [commitments]
  );

  const budgetCols: ColumnDef<BudgetDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Budget Name', accessorKey: 'name' },
    { header: 'Version', width: '90px', align: 'right', cell: (r) => <span className="font-mono">v{r.version}</span> },
    { header: 'Status', width: '130px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const policyCols: ColumnDef<BudgetPolicyDto>[] = [
    { header: 'Transaction Type', width: '160px', cell: (r) => r.transactionType || 'All Transactions' },
    { header: 'Control Type', width: '140px', cell: (r) => <StatusBadge status={r.controlType === 'HardStop' ? 'LOCKED' : 'PENDING'} label={r.controlType} /> },
    { header: 'Scope', width: '120px', cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} /> },
  ];

  const commitCols: ColumnDef<BudgetCommitmentDto>[] = [
    { header: 'Source Document', accessorKey: 'sourceDocumentRef', className: 'font-mono font-semibold' },
    {
      header: 'Committed Amount',
      align: 'right',
      width: '180px',
      cell: (r) => <span className="font-mono tabular-nums">{formatCurrency(r.committedAmount, currency, locale)}</span>,
    },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const handleCreateBudget = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!orgId) return;
    try {
      await createBudget.mutateAsync({
        code: code.trim(),
        name: name.trim(),
        financialOrganizationRef: orgId,
        fiscalYearRef,
        version: 1,
      });
      toast.success('Budget created');
      setCreateBudgetOpen(false);
      setCode('');
      setName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create budget');
    }
  };

  const handleTransition = async (b: BudgetDto, action: 'submit' | 'approve' | 'close') => {
    try {
      await transitionBudget.mutateAsync({ id: b.id, action, rowVersion: b.rowVersion });
      toast.success(`Budget ${b.code} ${action}d`);
      setSelectedBudget(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Failed to ${action} budget`);
    }
  };

  const handleCreatePolicy = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createPolicy.mutateAsync({ transactionType: txType, controlType });
      toast.success('Budget control policy saved');
      setCreatePolicyOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create policy');
    }
  };

  const handleAddLine = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedBudget) return;
    try {
      await addLine.mutateAsync({
        budgetId: selectedBudget.id,
        accountRef: ZERO_UUID,
        periodNo: Number(linePeriod) || 1,
        budgetAmount: lineAmount,
      });
      toast.success('Budget line added');
      setAddLineOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to add line');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Finance' }, { label: 'Budgets & Controls' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Budgeting, Policies & Commitments
          </h1>
        </div>
        {canWrite ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreatePolicyOpen(true)}>
              New Policy
            </Button>
            <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateBudgetOpen(true)}>
              New Budget
            </Button>
          </div>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Budgets" value={budgets.length} icon={<PieChart className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Control Policies" value={policies.length} icon={<ShieldCheck className="h-4 w-4" />} accent="amber" />
        <KpiCard
          label="Total Commitments"
          value={<span className="font-mono tabular-nums">{formatCurrency(totalCommitted, currency, locale)}</span>}
          icon={<CheckCircle2 className="h-4 w-4" />}
          accent="emerald"
        />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="budgets">Budgets ({budgets.length})</TabsTrigger>
          <TabsTrigger value="policies">Policies ({policies.length})</TabsTrigger>
          <TabsTrigger value="commitments">Commitments ({commitments.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="budgets" className="pt-4">
          <VirtualizedTable
            data={budgets}
            columns={budgetCols}
            isLoading={budgetsQ.isLoading}
            error={budgetsQ.error}
            onRetry={() => void budgetsQ.refetch()}
            onRowClick={setSelectedBudget}
            emptyTitle="No budgets configured"
            toolbar={{
              filters: (
                <Select
                  value={statusFilter}
                  onValueChange={setStatusFilter}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Draft', label: 'Draft' },
                    { value: 'Submitted', label: 'Submitted' },
                    { value: 'Approved', label: 'Approved' },
                    { value: 'Closed', label: 'Closed' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
        <TabsContent value="policies" className="pt-4">
          <VirtualizedTable data={policies} columns={policyCols} isLoading={policiesQ.isLoading} error={policiesQ.error} onRetry={() => void policiesQ.refetch()} emptyTitle="No policies" />
        </TabsContent>
        <TabsContent value="commitments" className="pt-4">
          <VirtualizedTable data={commitments} columns={commitCols} isLoading={commitmentsQ.isLoading} error={commitmentsQ.error} onRetry={() => void commitmentsQ.refetch()} emptyTitle="No budget commitments" />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedBudget)}
        onOpenChange={(open) => !open && setSelectedBudget(null)}
        title={selectedBudget ? `${selectedBudget.code} · ${selectedBudget.name}` : 'Budget Detail'}
        titleBadge={selectedBudget ? <StatusBadge status={selectedBudget.status.toUpperCase()} label={selectedBudget.status} /> : null}
        footer={
          selectedBudget ? (
            <div className="flex flex-wrap gap-2 justify-end">
              {selectedBudget.status === 'Draft' && canWrite ? (
                <>
                  <Button variant="outline" size="sm" onClick={() => setAddLineOpen(true)}>Add Line</Button>
                  <Button variant="primary" size="sm" onClick={() => void handleTransition(selectedBudget, 'submit')}>Submit</Button>
                </>
              ) : null}
              {selectedBudget.status === 'Submitted' && canApprove ? (
                <Button variant="primary" size="sm" onClick={() => void handleTransition(selectedBudget, 'approve')}>Approve</Button>
              ) : null}
              {selectedBudget.status === 'Approved' && canApprove ? (
                <Button variant="danger" size="sm" onClick={() => void handleTransition(selectedBudget, 'close')}>Close Budget</Button>
              ) : null}
            </div>
          ) : null
        }
      >
        <div className="space-y-4 text-xs">
          <h3 className="font-bold uppercase tracking-wider text-slate-500">Budget Lines</h3>
          {(detailQ.data?.lines ?? []).length === 0 ? (
            <p className="text-slate-500">No budget lines added yet.</p>
          ) : (
            <div className="space-y-1.5">
              {(detailQ.data?.lines ?? []).map((l) => (
                <div key={l.id} className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 px-3 py-2">
                  <span className="font-mono">Period {l.periodNo}</span>
                  <span className="font-mono tabular-nums font-semibold">{formatCurrency(l.budgetAmount, currency, locale)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </SideDrawer>

      <AppDialog open={createBudgetOpen} onOpenChange={setCreateBudgetOpen} title="Create Budget" size="md">
        <form onSubmit={handleCreateBudget} className="space-y-4">
          <FormField label="Code" htmlFor="bdgCode" required>
            <Input id="bdgCode" value={code} onChange={(e) => setCode(e.target.value)} placeholder="BDG-2026-OPEX" required />
          </FormField>
          <FormField label="Name" htmlFor="bdgName" required>
            <Input id="bdgName" value={name} onChange={(e) => setName(e.target.value)} placeholder="FY2026 Operating Budget" required />
          </FormField>
          <FormField label="Fiscal Year UUID" htmlFor="bdgFy" required>
            <Input id="bdgFy" value={fiscalYearRef} onChange={(e) => setFiscalYearRef(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateBudgetOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createBudget.isPending}>Create</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={createPolicyOpen} onOpenChange={setCreatePolicyOpen} title="Create Budget Policy" size="sm">
        <form onSubmit={handleCreatePolicy} className="space-y-4">
          <FormField label="Transaction Type" htmlFor="polTx">
            <Select
              id="polTx"
              value={txType}
              onValueChange={setTxType}
              options={[
                { value: 'PO', label: 'Purchase Order (PO)' },
                { value: 'PR', label: 'Purchase Requisition (PR)' },
                { value: 'AP', label: 'Vendor Invoice (AP)' },
              ]}
            />
          </FormField>
          <FormField label="Control Type" htmlFor="polCtrl">
            <Select
              id="polCtrl"
              value={controlType}
              onValueChange={setControlType}
              options={[
                { value: 'HardStop', label: 'Hard Stop (Reject)' },
                { value: 'Warn', label: 'Warn Only' },
                { value: 'TrackOnly', label: 'Track Only' },
              ]}
            />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreatePolicyOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createPolicy.isPending}>Save</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={addLineOpen} onOpenChange={setAddLineOpen} title="Add Budget Line" size="sm">
        <form onSubmit={handleAddLine} className="space-y-4">
          <FormField label="Period No (1-12)" htmlFor="linePeriod" required>
            <Input id="linePeriod" type="number" min={1} max={12} value={linePeriod} onChange={(e) => setLinePeriod(e.target.value)} required />
          </FormField>
          <FormField label="Budget Amount" htmlFor="lineAmt" required>
            <Input id="lineAmt" value={lineAmount} onChange={(e) => setLineAmount(e.target.value)} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setAddLineOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={addLine.isPending}>Add Line</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
