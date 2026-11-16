import { useMemo, useState } from 'react';
import { Check, CheckSquare, FileCheck2, GitBranch, Layers, Plus, ShieldAlert, X } from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { Textarea } from '../../components/ui/Textarea';
import { ApiClientError, extractApiFieldErrors } from '../../lib/api';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { CreateWorkflowDefinitionDialog } from './CreateWorkflowDefinitionDialog';
import { CreateWorkflowInstanceDialog } from './CreateWorkflowInstanceDialog';
import { WorkflowInstanceDrawer } from './WorkflowInstanceDrawer';
import {
  useApproveWorkflowTaskMutation,
  useRejectWorkflowTaskMutation,
  useWorkflowDefinitionsQuery,
  useWorkflowInstancesQuery,
  useWorkflowTasksQuery,
  type WorkflowDefinition,
  type WorkflowInstance,
  type WorkflowTask,
} from './workflow-api';
import { useWorkflowText } from './workflow-i18n';

export function WorkflowWorkspace() {
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;
  const canApprove = hasUserPermission(user, 'workflow:approve');
  const canWrite = hasUserPermission(user, 'workflow:write');

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <WorkflowWorkspaceInner
        orgId={orgId}
        currentUserId={user?.id}
        canApprove={canApprove}
        canWrite={canWrite}
      />
    </DomainErrorBoundary>
  );
}

interface WorkflowWorkspaceInnerProps {
  orgId: string | undefined;
  currentUserId: string | undefined;
  canApprove: boolean;
  canWrite: boolean;
}

function WorkflowWorkspaceInner({
  orgId,
  currentUserId,
  canApprove,
  canWrite,
}: WorkflowWorkspaceInnerProps) {
  const txt = useWorkflowText();
  const [activeTab, setActiveTab] = useState('tasks');
  const [taskStatusFilter, setTaskStatusFilter] = useState('pending');
  const [taskPage, setTaskPage] = useState(1);
  const [taskSearch, setTaskSearch] = useState('');

  const [instStatusFilter, setInstStatusFilter] = useState('ALL');
  const [instPage, setInstPage] = useState(1);
  const [instSearch, setInstSearch] = useState('');

  const [docTypeFilter, setDocTypeFilter] = useState('');
  const [defPage, setDefPage] = useState(1);

  const [drawerInstanceId, setDrawerInstanceId] = useState<string | null>(null);
  const [createDefOpen, setCreateDefOpen] = useState(false);
  const [createInstOpen, setCreateInstOpen] = useState(false);

  const [decisionModal, setDecisionModal] = useState<{
    mode: 'Approve' | 'Reject';
    task: WorkflowTask;
  } | null>(null);
  const [decisionComment, setDecisionComment] = useState('');
  const [sodAlertMessage, setSodAlertMessage] = useState<string | null>(null);

  const tasksQuery = useWorkflowTasksQuery(orgId, taskStatusFilter, taskPage, 25);
  const instancesQuery = useWorkflowInstancesQuery(orgId, instStatusFilter, instPage, 25);
  const definitionsQuery = useWorkflowDefinitionsQuery(orgId, docTypeFilter, defPage, 50);

  const approveMutation = useApproveWorkflowTaskMutation();
  const rejectMutation = useRejectWorkflowTaskMutation();

  const tasks = useMemo(() => tasksQuery.data?.items ?? [], [tasksQuery.data]);
  const taskMeta = tasksQuery.data?.page;
  const instances = useMemo(() => instancesQuery.data?.items ?? [], [instancesQuery.data]);
  const instanceMeta = instancesQuery.data?.page;
  const definitions = useMemo(() => definitionsQuery.data?.items ?? [], [definitionsQuery.data]);
  const definitionMeta = definitionsQuery.data?.page;

  const filteredTasks = useMemo(() => {
    const q = taskSearch.trim().toLowerCase();
    if (!q) return tasks;
    return tasks.filter(
      (t) =>
        t.documentRef.toLowerCase().includes(q) ||
        t.assigneeId.toLowerCase().includes(q) ||
        t.instanceId.toLowerCase().includes(q),
    );
  }, [tasks, taskSearch]);

  const filteredInstances = useMemo(() => {
    const q = instSearch.trim().toLowerCase();
    if (!q) return instances;
    return instances.filter(
      (i) => i.documentRef.toLowerCase().includes(q) || i.status.toLowerCase().includes(q),
    );
  }, [instances, instSearch]);

  const pendingOnPage = useMemo(() => tasks.filter((t) => !t.decision).length, [tasks]);
  const sodCountOnPage = useMemo(
    () => tasks.filter((t) => Boolean(currentUserId && t.createdBy === currentUserId)).length,
    [tasks, currentUserId],
  );

  const openDecisionDialog = (mode: 'Approve' | 'Reject', task: WorkflowTask) => {
    setDecisionComment('');
    setSodAlertMessage(null);
    setDecisionModal({ mode, task });
  };

  const handleConfirmDecision = async () => {
    if (!decisionModal) return;
    const { mode, task } = decisionModal;
    try {
      if (mode === 'Approve') {
        await approveMutation.mutateAsync({
          id: task.id,
          rowVersion: task.rowVersion,
          comment: decisionComment,
        });
        toast.success(txt.approvedToast);
      } else {
        await rejectMutation.mutateAsync({
          id: task.id,
          rowVersion: task.rowVersion,
          comment: decisionComment,
        });
        toast.success(txt.rejectedToast);
      }
      setDecisionModal(null);
    } catch (err) {
      if (err instanceof ApiClientError && err.code === 'SOD_VIOLATION') {
        const msg = `${err.message} (${err.code})`;
        setSodAlertMessage(msg);
        toast.error(msg);
        return;
      }
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const taskColumns = useMemo<ColumnDef<WorkflowTask>[]>(
    () => [
      {
        accessorKey: 'documentRef',
        header: 'Document Ref',
        cell: (row) => (
          <div>
            <div className="font-mono text-xs font-semibold text-[var(--text-primary)]">
              {row.documentRef}
            </div>
            <div className="font-mono text-[11px] text-[var(--text-muted)]">
              inst: {row.instanceId.slice(0, 8)}
            </div>
          </div>
        ),
      },
      {
        accessorKey: 'assigneeId',
        header: 'Assignee',
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-secondary)]">
            {row.assigneeId}
          </span>
        ),
      },
      {
        accessorKey: 'decision',
        header: 'Decision',
        cell: (row) => <StatusBadge status={row.decision ?? 'Pending'} />,
      },
      {
        accessorKey: 'instanceStatus',
        header: 'Instance Status',
        cell: (row) => <StatusBadge status={row.instanceStatus} />,
      },
      {
        header: 'SoD Guard',
        cell: (row) => {
          const isSelf = Boolean(currentUserId && row.createdBy === currentUserId);
          return isSelf ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300">
              <ShieldAlert className="h-3 w-3" />
              {txt.sodBadge}
            </span>
          ) : (
            <span className="text-xs text-[var(--text-muted)]">Eligible</span>
          );
        },
      },
      {
        header: 'Actions',
        cell: (task) => {
          const isPending = !task.decision;
          return (
            <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
              {canApprove && isPending && (
                <>
                  <Button
                    size="sm"
                    onClick={() => openDecisionDialog('Approve', task)}
                  >
                    <Check className="mr-1 h-3.5 w-3.5" />
                    {txt.approve}
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() => openDecisionDialog('Reject', task)}
                  >
                    <X className="mr-1 h-3.5 w-3.5" />
                    {txt.reject}
                  </Button>
                </>
              )}
              <Button
                size="sm"
                variant="outline"
                onClick={() => setDrawerInstanceId(task.instanceId)}
              >
                {txt.inspectTimeline}
              </Button>
            </div>
          );
        },
      },
    ],
    [canApprove, currentUserId, txt],
  );

  const instanceColumns = useMemo<ColumnDef<WorkflowInstance>[]>(
    () => [
      {
        accessorKey: 'documentRef',
        header: 'Document Ref',
        cell: (row) => (
          <span className="font-mono text-xs font-semibold text-[var(--text-primary)]">
            {row.documentRef}
          </span>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: (row) => <StatusBadge status={row.status} />,
      },
      {
        accessorKey: 'currentStep',
        header: 'Step',
        cell: (row) => <span className="font-mono text-xs">{row.currentStep ?? 1}</span>,
      },
      {
        accessorKey: 'definitionId',
        header: 'Definition ID',
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-muted)]">{row.definitionId}</span>
        ),
      },
      {
        header: 'Actions',
        cell: (row) => (
          <div onClick={(e) => e.stopPropagation()}>
            <Button size="sm" variant="outline" onClick={() => setDrawerInstanceId(row.id)}>
              {txt.inspectTimeline}
            </Button>
          </div>
        ),
      },
    ],
    [txt.inspectTimeline],
  );

  const definitionColumns = useMemo<ColumnDef<WorkflowDefinition>[]>(
    () => [
      {
        accessorKey: 'code',
        header: 'Code',
        cell: (row) => (
          <span className="font-mono text-xs font-semibold text-[var(--text-primary)]">
            {row.code}
          </span>
        ),
      },
      {
        accessorKey: 'documentType',
        header: 'Document Type',
        cell: (row) => <span className="font-mono text-xs">{row.documentType}</span>,
      },
      {
        accessorKey: 'version',
        header: 'Version',
        cell: (row) => <span className="font-mono text-xs">v{row.version}</span>,
      },
      {
        accessorKey: 'isActive',
        header: 'State',
        cell: (row) => <StatusBadge status={row.isActive ? 'ACTIVE' : 'INACTIVE'} />,
      },
    ],
    [],
  );

  const isMutatingDecision = approveMutation.isPending || rejectMutation.isPending;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--text-primary)]">
            {txt.title}
          </h1>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">{txt.subtitle}</p>
        </div>

        {canWrite && (
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="secondary" onClick={() => setCreateInstOpen(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              {txt.createInstance}
            </Button>
            <Button onClick={() => setCreateDefOpen(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              {txt.createDefinition}
            </Button>
          </div>
        )}
      </div>

      {(!canApprove || !canWrite) && (
        <div
          role="status"
          className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-2.5 text-xs font-medium text-amber-800 dark:text-amber-300"
        >
          {txt.readOnlyHint}
        </div>
      )}

      {sodAlertMessage && (
        <div
          role="alert"
          className="flex items-center justify-between gap-3 rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-3 text-sm text-rose-800 dark:text-rose-200"
        >
          <div className="flex items-center gap-2">
            <ShieldAlert className="h-4 w-4 shrink-0" />
            <span>{sodAlertMessage}</span>
          </div>
          <Button size="sm" variant="ghost" onClick={() => setSodAlertMessage(null)}>
            Dismiss
          </Button>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label={txt.kpiPendingTasks}
          value={String(taskStatusFilter === 'pending' ? (taskMeta?.totalElements ?? pendingOnPage) : pendingOnPage)}
          accent="amber"
          icon={<CheckSquare className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiInstances}
          value={String(instanceMeta?.totalElements ?? instances.length)}
          accent="sky"
          icon={<GitBranch className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiDefinitions}
          value={String(definitionMeta?.totalElements ?? definitions.length)}
          accent="emerald"
          icon={<Layers className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiSodGuarded}
          value={String(sodCountOnPage)}
          accent="rose"
          icon={<ShieldAlert className="h-4 w-4" />}
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="tasks">
            <FileCheck2 className="mr-1.5 h-4 w-4" />
            {txt.tabTasks}
          </TabsTrigger>
          <TabsTrigger value="instances">
            <GitBranch className="mr-1.5 h-4 w-4" />
            {txt.tabInstances}
          </TabsTrigger>
          <TabsTrigger value="definitions">
            <Layers className="mr-1.5 h-4 w-4" />
            {txt.tabDefinitions}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="tasks" className="mt-4">
          <VirtualizedTable
            data={filteredTasks}
            columns={taskColumns}
            getRowId={(row) => row.id}
            isLoading={tasksQuery.isLoading}
            error={tasksQuery.error}
            onRetry={() => void tasksQuery.refetch()}
            onRowClick={(row) => setDrawerInstanceId(row.instanceId)}
            emptyTitle={txt.emptyTasksTitle}
            emptyDescription={txt.emptyTasksDesc}
            onResetFilters={() => {
              setTaskStatusFilter('ALL');
              setTaskSearch('');
              setTaskPage(1);
            }}
            pagination={{
              page: taskPage,
              size: 25,
              totalElements: taskMeta?.totalElements ?? filteredTasks.length,
              totalPages: Math.max(1, taskMeta?.totalPages ?? 1),
              pageBase: 1,
              onPageChange: setTaskPage,
            }}
            toolbar={{
              searchValue: taskSearch,
              onSearchChange: setTaskSearch,
              searchPlaceholder: 'Search document ref or assignee...',
              filters: (
                <div className="w-44">
                  <Select
                    value={taskStatusFilter}
                    onValueChange={(next) => {
                      setTaskStatusFilter(next);
                      setTaskPage(1);
                    }}
                    options={[
                      { value: 'pending', label: txt.filterPending },
                      { value: 'ALL', label: txt.filterAll },
                      { value: 'decided', label: txt.filterDecided },
                    ]}
                  />
                </div>
              ),
            }}
          />
        </TabsContent>

        <TabsContent value="instances" className="mt-4">
          <VirtualizedTable
            data={filteredInstances}
            columns={instanceColumns}
            getRowId={(row) => row.id}
            isLoading={instancesQuery.isLoading}
            error={instancesQuery.error}
            onRetry={() => void instancesQuery.refetch()}
            onRowClick={(row) => setDrawerInstanceId(row.id)}
            emptyTitle={txt.emptyInstancesTitle}
            emptyDescription={txt.emptyInstancesDesc}
            onResetFilters={() => {
              setInstStatusFilter('ALL');
              setInstSearch('');
              setInstPage(1);
            }}
            pagination={{
              page: instPage,
              size: 25,
              totalElements: instanceMeta?.totalElements ?? filteredInstances.length,
              totalPages: Math.max(1, instanceMeta?.totalPages ?? 1),
              pageBase: 1,
              onPageChange: setInstPage,
            }}
            toolbar={{
              searchValue: instSearch,
              onSearchChange: setInstSearch,
              searchPlaceholder: 'Search instances...',
              filters: (
                <div className="w-40">
                  <Select
                    value={instStatusFilter}
                    onValueChange={(next) => {
                      setInstStatusFilter(next);
                      setInstPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Statuses' },
                      { value: 'Pending', label: 'Pending' },
                      { value: 'Approved', label: 'Approved' },
                      { value: 'Rejected', label: 'Rejected' },
                    ]}
                  />
                </div>
              ),
            }}
          />
        </TabsContent>

        <TabsContent value="definitions" className="mt-4">
          <VirtualizedTable
            data={definitions}
            columns={definitionColumns}
            getRowId={(row) => row.id}
            isLoading={definitionsQuery.isLoading}
            error={definitionsQuery.error}
            onRetry={() => void definitionsQuery.refetch()}
            emptyTitle={txt.emptyDefinitionsTitle}
            emptyDescription={txt.emptyDefinitionsDesc}
            onResetFilters={() => {
              setDocTypeFilter('');
              setDefPage(1);
            }}
            pagination={{
              page: defPage,
              size: 50,
              totalElements: definitionMeta?.totalElements ?? definitions.length,
              totalPages: Math.max(1, definitionMeta?.totalPages ?? 1),
              pageBase: 1,
              onPageChange: setDefPage,
            }}
            toolbar={{
              filters: (
                <div className="w-60">
                  <Input
                    value={docTypeFilter}
                    onChange={(e) => {
                      setDocTypeFilter(e.target.value);
                      setDefPage(1);
                    }}
                    placeholder="Filter exact documentType..."
                  />
                </div>
              ),
            }}
          />
        </TabsContent>
      </Tabs>

      <WorkflowInstanceDrawer
        open={Boolean(drawerInstanceId)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDrawerInstanceId(null);
        }}
        instanceId={drawerInstanceId}
        orgId={orgId}
        currentUserId={currentUserId}
      />

      <CreateWorkflowDefinitionDialog open={createDefOpen} onOpenChange={setCreateDefOpen} />

      <CreateWorkflowInstanceDialog
        open={createInstOpen}
        onOpenChange={setCreateInstOpen}
        definitions={definitions}
      />

      <AppDialog
        open={Boolean(decisionModal)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDecisionModal(null);
        }}
        title={decisionModal?.mode === 'Reject' ? txt.rejectTitle : txt.approveTitle}
        description={
          decisionModal
            ? `Document ${decisionModal.task.documentRef} · Task v${decisionModal.task.rowVersion}`
            : undefined
        }
        size="md"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setDecisionModal(null)}
              disabled={isMutatingDecision}
            >
              {txt.cancel}
            </Button>
            <Button
              variant={decisionModal?.mode === 'Reject' ? 'danger' : 'primary'}
              onClick={handleConfirmDecision}
              disabled={isMutatingDecision}
            >
              {decisionModal?.mode === 'Reject' ? txt.reject : txt.approve}
            </Button>
          </div>
        }
      >
        {decisionModal && (
          <div className="space-y-4">
            {sodAlertMessage && (
              <div
                role="alert"
                className="flex items-center gap-2 rounded-lg border border-rose-500/40 bg-rose-500/10 p-3 text-xs font-medium text-rose-800 dark:text-rose-200"
              >
                <ShieldAlert className="h-4 w-4 shrink-0" />
                <span>{sodAlertMessage}</span>
              </div>
            )}

            {currentUserId && decisionModal.task.createdBy === currentUserId && (
              <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-800 dark:text-amber-300">
                <div className="flex items-center gap-1.5 font-semibold">
                  <ShieldAlert className="h-4 w-4" />
                  SoD Guard Warning
                </div>
                <p className="mt-1">{txt.sodBlockedHint}</p>
              </div>
            )}

            <FormField label={txt.decisionCommentLabel} htmlFor="wf-decision-comment">
              <Textarea
                id="wf-decision-comment"
                value={decisionComment}
                onChange={(e) => setDecisionComment(e.target.value)}
                placeholder={
                  decisionModal.mode === 'Reject'
                    ? 'Provide clear reason for rejection...'
                    : 'Optional approval note...'
                }
                rows={3}
              />
            </FormField>
          </div>
        )}
      </AppDialog>
    </div>
  );
}
