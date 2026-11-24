import * as React from 'react';
import { CalendarCheck2, CheckSquare, Lock, Plus, RotateCcw } from 'lucide-react';
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
import { Textarea } from '../../components/ui/Textarea';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import {
  useApproveReopeningMutation,
  useClosingRunDetailQuery,
  useClosingRunsQuery,
  useClosingTemplatesQuery,
  useCompleteClosingTaskMutation,
  useCreateClosingRunMutation,
  usePeriodReopeningsQuery,
  usePeriodStatusesQuery,
  useRequestReopeningMutation,
  useTransitionClosingRunMutation,
  type ClosingRunDto,
  type ClosingTemplateDto,
  type PeriodReopeningDto,
  type PeriodStatusDto,
} from './financialclose-api';

const ZERO_UUID = '00000000-0000-0000-0000-000000000001';

export function FinancialCloseWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, ['financialclose:write', 'financialclose.create']);
  const canApprove = hasUserPermission(user, ['financialclose:approve', 'financialclose.approve']);

  const [tab, setTab] = React.useState('runs');
  const [selectedRun, setSelectedRun] = React.useState<ClosingRunDto | null>(null);
  const [createRunOpen, setCreateRunOpen] = React.useState(false);
  const [reopenOpen, setReopenOpen] = React.useState(false);

  const [fiscalPeriodRef, setFiscalPeriodRef] = React.useState(ZERO_UUID);
  const [runType, setRunType] = React.useState('MonthEnd');
  const [templateId, setTemplateId] = React.useState('');
  const [reopenReason, setReopenReason] = React.useState('');

  const runsQ = useClosingRunsQuery(orgId);
  const templatesQ = useClosingTemplatesQuery(orgId);
  const statusesQ = usePeriodStatusesQuery(orgId);
  const reopeningsQ = usePeriodReopeningsQuery(orgId);
  const runDetailQ = useClosingRunDetailQuery(orgId, selectedRun?.id);

  const createRun = useCreateClosingRunMutation();
  const completeTask = useCompleteClosingTaskMutation();
  const transitionRun = useTransitionClosingRunMutation();
  const requestReopen = useRequestReopeningMutation();
  const approveReopen = useApproveReopeningMutation();

  const runs = runsQ.data?.items ?? [];
  const templates = templatesQ.data?.items ?? [];
  const statuses = statusesQ.data?.items ?? [];
  const reopenings = reopeningsQ.data?.items ?? [];

  const runCols: ColumnDef<ClosingRunDto>[] = [
    { header: 'Run Type', accessorKey: 'runType', width: '140px', className: 'font-semibold' },
    { header: 'Fiscal Period', accessorKey: 'fiscalPeriodRef', className: 'font-mono text-xs' },
    { header: 'Status', width: '130px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const tplCols: ColumnDef<ClosingTemplateDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Template Name', accessorKey: 'name' },
    { header: 'Scope', width: '110px', cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} /> },
  ];

  const statusCols: ColumnDef<PeriodStatusDto>[] = [
    { header: 'Module', accessorKey: 'module', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Fiscal Period', accessorKey: 'fiscalPeriodRef', className: 'font-mono text-xs' },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const reopenCols: ColumnDef<PeriodReopeningDto>[] = [
    { header: 'Period', accessorKey: 'fiscalPeriodRef', width: '240px', className: 'font-mono text-xs' },
    { header: 'Reason', accessorKey: 'reason' },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '120px',
      cell: (r) =>
        r.status === 'Pending' && canApprove ? (
          <Button
            size="sm"
            variant="primary"
            onClick={(e) => {
              e.stopPropagation();
              void approveReopen.mutateAsync({ id: r.id, rowVersion: r.rowVersion }).then(
                () => toast.success('Period reopening approved'),
                (err) => toast.error(err instanceof Error ? err.message : 'Approve failed')
              );
            }}
          >
            Approve
          </Button>
        ) : null,
    },
  ];

  const handleCreateRun = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!orgId) return;
    try {
      await createRun.mutateAsync({
        financialOrganizationRef: orgId,
        fiscalPeriodRef,
        runType,
        templateId: templateId || undefined,
      });
      toast.success('Closing run created');
      setCreateRunOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create closing run');
    }
  };

  const handleRequestReopen = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!orgId) return;
    try {
      await requestReopen.mutateAsync({
        financialOrganizationRef: orgId,
        fiscalPeriodRef,
        reason: reopenReason.trim(),
      });
      toast.success('Period reopening requested');
      setReopenOpen(false);
      setReopenReason('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Request failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Finance' }, { label: 'Financial Close' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Financial Close & Period Governance
          </h1>
        </div>
        {canWrite ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" leftIcon={<RotateCcw className="h-4 w-4" />} onClick={() => setReopenOpen(true)}>
              Request Reopen
            </Button>
            <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateRunOpen(true)}>
              New Closing Run
            </Button>
          </div>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Closing Runs" value={runs.length} icon={<CalendarCheck2 className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Templates" value={templates.length} icon={<CheckSquare className="h-4 w-4" />} accent="sky" />
        <KpiCard label="Locked Periods" value={statuses.filter((s) => s.status === 'Locked').length} icon={<Lock className="h-4 w-4" />} accent="rose" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="runs">Closing Runs ({runs.length})</TabsTrigger>
          <TabsTrigger value="templates">Templates ({templates.length})</TabsTrigger>
          <TabsTrigger value="statuses">Period Statuses ({statuses.length})</TabsTrigger>
          <TabsTrigger value="reopenings">Reopenings ({reopenings.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="runs" className="pt-4">
          <VirtualizedTable data={runs} columns={runCols} isLoading={runsQ.isLoading} error={runsQ.error} onRetry={() => void runsQ.refetch()} onRowClick={setSelectedRun} emptyTitle="No closing runs" />
        </TabsContent>
        <TabsContent value="templates" className="pt-4">
          <VirtualizedTable data={templates} columns={tplCols} isLoading={templatesQ.isLoading} error={templatesQ.error} onRetry={() => void templatesQ.refetch()} emptyTitle="No templates" />
        </TabsContent>
        <TabsContent value="statuses" className="pt-4">
          <VirtualizedTable data={statuses} columns={statusCols} isLoading={statusesQ.isLoading} error={statusesQ.error} onRetry={() => void statusesQ.refetch()} emptyTitle="No period statuses" />
        </TabsContent>
        <TabsContent value="reopenings" className="pt-4">
          <VirtualizedTable data={reopenings} columns={reopenCols} isLoading={reopeningsQ.isLoading} error={reopeningsQ.error} onRetry={() => void reopeningsQ.refetch()} emptyTitle="No reopening requests" />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedRun)}
        onOpenChange={(open) => !open && setSelectedRun(null)}
        title={selectedRun ? `Closing Run · ${selectedRun.runType}` : 'Closing Run'}
        titleBadge={selectedRun ? <StatusBadge status={selectedRun.status.toUpperCase()} label={selectedRun.status} /> : null}
        footer={
          selectedRun && canApprove ? (
            <div className="flex gap-2 justify-end">
              {selectedRun.status === 'InProgress' || selectedRun.status === 'Open' ? (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() =>
                    void transitionRun.mutateAsync({ id: selectedRun.id, action: 'close', rowVersion: selectedRun.rowVersion }).then(
                      () => {
                        toast.success('Closing run closed');
                        setSelectedRun(null);
                      },
                      (err) => toast.error(err instanceof Error ? err.message : 'Close failed')
                    )
                  }
                >
                  Close Period
                </Button>
              ) : null}
              {selectedRun.status === 'Closed' ? (
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() =>
                    void transitionRun.mutateAsync({ id: selectedRun.id, action: 'lock', rowVersion: selectedRun.rowVersion }).then(
                      () => {
                        toast.success('Closing run locked');
                        setSelectedRun(null);
                      },
                      (err) => toast.error(err instanceof Error ? err.message : 'Lock failed')
                    )
                  }
                >
                  Lock Period
                </Button>
              ) : null}
            </div>
          ) : null
        }
      >
        <div className="space-y-3 text-xs">
          <h3 className="font-bold uppercase tracking-wider text-slate-500">Closing Checklist Tasks</h3>
          {(runDetailQ.data?.tasks ?? []).length === 0 ? (
            <p className="text-slate-500">No checklist tasks for this run.</p>
          ) : (
            (runDetailQ.data?.tasks ?? []).map((task) => (
              <div key={task.id} className="flex items-center justify-between gap-2 rounded-lg border border-slate-200 dark:border-slate-800 px-3 py-2.5">
                <div>
                  <p className="font-semibold">
                    {task.taskOrder}. {task.taskName}
                  </p>
                  <p className="text-[11px] text-slate-500">{task.isMandatory ? 'Mandatory' : 'Optional'}</p>
                </div>
                {task.status === 'Completed' ? (
                  <StatusBadge status="COMPLETED" label="Completed" />
                ) : canWrite ? (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() =>
                      void completeTask.mutateAsync({ id: task.id, rowVersion: task.rowVersion }).then(
                        () => toast.success(`Task "${task.taskName}" completed`),
                        (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                      )
                    }
                  >
                    Complete
                  </Button>
                ) : (
                  <StatusBadge status="PENDING" label={task.status} />
                )}
              </div>
            ))
          )}
        </div>
      </SideDrawer>

      <AppDialog open={createRunOpen} onOpenChange={setCreateRunOpen} title="Start Closing Run" size="md">
        <form onSubmit={handleCreateRun} className="space-y-4">
          <FormField label="Fiscal Period UUID" htmlFor="fcPeriod" required>
            <Input id="fcPeriod" value={fiscalPeriodRef} onChange={(e) => setFiscalPeriodRef(e.target.value)} required />
          </FormField>
          <FormField label="Run Type" htmlFor="fcType">
            <Select
              id="fcType"
              value={runType}
              onValueChange={setRunType}
              options={[
                { value: 'MonthEnd', label: 'Month-End Close' },
                { value: 'QuarterEnd', label: 'Quarter-End Close' },
                { value: 'YearEnd', label: 'Year-End Close' },
              ]}
            />
          </FormField>
          {templates.length > 0 ? (
            <FormField label="Template (Optional)" htmlFor="fcTpl">
              <Select
                id="fcTpl"
                value={templateId}
                onValueChange={setTemplateId}
                options={[{ value: '', label: 'None' }, ...templates.map((t) => ({ value: t.id, label: `${t.code} · ${t.name}` }))]}
              />
            </FormField>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateRunOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createRun.isPending}>Start Run</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={reopenOpen} onOpenChange={setReopenOpen} title="Request Period Reopening" size="sm">
        <form onSubmit={handleRequestReopen} className="space-y-4">
          <FormField label="Fiscal Period UUID" htmlFor="roPeriod" required>
            <Input id="roPeriod" value={fiscalPeriodRef} onChange={(e) => setFiscalPeriodRef(e.target.value)} required />
          </FormField>
          <FormField label="Audit Justification" htmlFor="roReason" required>
            <Textarea id="roReason" value={reopenReason} onChange={(e) => setReopenReason(e.target.value)} rows={3} required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setReopenOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={requestReopen.isPending}>Submit Request</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
