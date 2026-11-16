import { GitBranch, ShieldAlert } from 'lucide-react';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { useWorkflowInstanceDetailQuery } from './workflow-api';

interface WorkflowInstanceDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  instanceId: string | null;
  orgId: string | undefined;
  currentUserId: string | undefined;
}

export function WorkflowInstanceDrawer({
  open,
  onOpenChange,
  instanceId,
  orgId,
  currentUserId,
}: WorkflowInstanceDrawerProps) {
  const detailQuery = useWorkflowInstanceDetailQuery(orgId, open ? instanceId : null);
  const instance = detailQuery.data;

  return (
    <SideDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={instance ? `Instance ${instance.documentRef}` : 'Workflow Instance Timeline'}
      description={
        instance
          ? `Step ${instance.currentStep ?? 1} · v${instance.rowVersion}`
          : undefined
      }
      size="lg"
    >
      {detailQuery.isLoading ? (
        <div className="space-y-3 py-6">
          <div className="h-16 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
          <div className="h-32 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
        </div>
      ) : detailQuery.isError ? (
        <ErrorState error={detailQuery.error} onRetry={() => void detailQuery.refetch()} />
      ) : instance ? (
        <div className="space-y-5">
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4 text-sm">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <GitBranch className="h-4 w-4 text-brand-600" />
                <span className="font-mono font-semibold text-[var(--text-primary)]">
                  {instance.documentRef}
                </span>
                <StatusBadge status={instance.status} />
              </div>
              <div className="font-mono text-xs text-[var(--text-muted)]">
                Definition ID: {instance.definitionId}
              </div>
            </div>
            {instance.createdBy && (
              <div className="text-right text-xs text-[var(--text-secondary)]">
                <div>Created By</div>
                <div className="font-mono">{instance.createdBy}</div>
              </div>
            )}
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-[var(--text-primary)]">
              Approval Task Timeline ({instance.tasks.length})
            </h3>
            {instance.tasks.length === 0 ? (
              <EmptyState title="No tasks spawned for this instance" />
            ) : (
              <ol className="space-y-3 border-l-2 border-[var(--border-subtle)] pl-4">
                {instance.tasks.map((task, idx) => {
                  const isSelfCreated = Boolean(currentUserId && task.createdBy === currentUserId);
                  return (
                    <li
                      key={task.id}
                      className="relative rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-3 text-sm"
                    >
                      <span
                        aria-hidden="true"
                        className="absolute -left-[21px] top-4 h-2.5 w-2.5 rounded-full border-2 border-[var(--surface-card)] bg-brand-600"
                      />
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <span className="font-medium text-[var(--text-primary)]">
                          Task #{idx + 1} · Assignee{' '}
                          <span className="font-mono text-xs">{task.assigneeId}</span>
                        </span>
                        <StatusBadge status={task.decision ?? 'Pending'} />
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-[var(--text-muted)]">
                        <span>Creator: {task.createdBy}</span>
                        <span>· v{task.rowVersion}</span>
                        {isSelfCreated && (
                          <span className="inline-flex items-center gap-1 rounded bg-amber-500/10 px-1.5 py-0.5 text-amber-700 dark:text-amber-300">
                            <ShieldAlert className="h-3 w-3" />
                            SoD Self-Created
                          </span>
                        )}
                      </div>
                    </li>
                  );
                })}
              </ol>
            )}
          </div>
        </div>
      ) : null}
    </SideDrawer>
  );
}
