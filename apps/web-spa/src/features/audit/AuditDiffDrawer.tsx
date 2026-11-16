import { useMemo, useState } from 'react';
import { Check, Copy, FileDiff, ShieldCheck } from 'lucide-react';
import { Button } from '../../components/ui/Button';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { formatDateTime } from '../../lib/decimal';
import {
  computeForensicDiff,
  useAuditEventDetailQuery,
  type DiffStatus,
} from './audit-api';

interface AuditDiffDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  eventId: string | null;
  orgId: string | undefined;
}

const DIFF_ROW_STYLE: Record<DiffStatus, string> = {
  added: 'bg-emerald-500/10 text-emerald-900 dark:text-emerald-200',
  removed: 'bg-rose-500/10 text-rose-900 dark:text-rose-200',
  changed: 'bg-amber-500/10 text-amber-900 dark:text-amber-200',
  unchanged: 'text-[var(--text-secondary)]',
};

const DIFF_BADGE: Record<DiffStatus, string> = {
  added: 'ADDED',
  removed: 'REMOVED',
  changed: 'MODIFIED',
  unchanged: 'SAME',
};

function prettyJson(value: unknown): string {
  if (value === undefined || value === null) return 'null';
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function AuditDiffDrawer({
  open,
  onOpenChange,
  eventId,
  orgId,
}: AuditDiffDrawerProps) {
  const [copiedCorr, setCopiedCorr] = useState(false);
  const detailQuery = useAuditEventDetailQuery(orgId, open ? eventId : null);
  const event = detailQuery.data;

  const diffEntries = useMemo(
    () => (event ? computeForensicDiff(event.beforeValue, event.afterValue) : []),
    [event],
  );

  const handleCopyCorrelation = async () => {
    if (!event?.correlationId || typeof navigator === 'undefined' || !navigator.clipboard) return;
    await navigator.clipboard.writeText(event.correlationId);
    setCopiedCorr(true);
    setTimeout(() => setCopiedCorr(false), 1500);
  };

  return (
    <SideDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={event ? `Forensic Event: ${event.entityType} · ${event.action}` : 'Forensic Diff Viewer'}
      description={event ? formatDateTime(event.occurredAt) : undefined}
      size="xl"
    >
      {detailQuery.isLoading ? (
        <div className="space-y-3 py-6">
          <div className="h-20 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
          <div className="h-44 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
        </div>
      ) : detailQuery.isError ? (
        <ErrorState error={detailQuery.error} onRetry={() => void detailQuery.refetch()} />
      ) : event ? (
        <div className="space-y-5">
          {/* Forensic Metadata */}
          <div className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--border-subtle)] pb-3">
              <div className="flex items-center gap-2">
                <ShieldCheck className="h-4 w-4 text-emerald-600" />
                <span className="font-mono font-semibold text-[var(--text-primary)]">
                  {event.entityType}
                </span>
                <StatusBadge status={event.action} />
              </div>

              {event.correlationId && (
                <Button size="sm" variant="outline" onClick={handleCopyCorrelation}>
                  {copiedCorr ? (
                    <Check className="mr-1.5 h-3.5 w-3.5 text-emerald-600" />
                  ) : (
                    <Copy className="mr-1.5 h-3.5 w-3.5" />
                  )}
                  <span className="font-mono text-xs">{event.correlationId}</span>
                </Button>
              )}
            </div>

            <dl className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <dt className="text-xs text-[var(--text-muted)]">Actor ID</dt>
                <dd className="mt-0.5 font-mono text-xs text-[var(--text-primary)]">
                  {event.actorId ?? 'system'}
                </dd>
              </div>
              <div>
                <dt className="text-xs text-[var(--text-muted)]">Entity ID</dt>
                <dd className="mt-0.5 font-mono text-xs text-[var(--text-primary)]">
                  {event.entityId ?? '—'}
                </dd>
              </div>
              <div>
                <dt className="text-xs text-[var(--text-muted)]">Reason</dt>
                <dd
                  className="mt-0.5 text-xs text-[var(--text-primary)]"
                  data-testid="audit-detail-reason"
                >
                  {event.reason ?? '—'}
                </dd>
              </div>
            </dl>
          </div>

          {/* Field-by-field Diff Table */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
              <FileDiff className="h-4 w-4" />
              Field Delta (`beforeValue` → `afterValue`)
            </div>

            {diffEntries.length === 0 ? (
              <EmptyState title="No before/after state payload captured for this event" />
            ) : (
              <div className="overflow-hidden rounded-xl border border-[var(--border-subtle)]">
                <table className="w-full text-left text-xs">
                  <thead className="bg-[var(--surface-subtle)] text-[var(--text-muted)]">
                    <tr>
                      <th className="px-3 py-2 font-medium">Field</th>
                      <th className="px-3 py-2 font-medium">Delta</th>
                      <th className="px-3 py-2 font-medium">Before</th>
                      <th className="px-3 py-2 font-medium">After</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[var(--border-subtle)] font-mono">
                    {diffEntries.map((entry) => (
                      <tr key={entry.key} className={DIFF_ROW_STYLE[entry.status]}>
                        <td className="px-3 py-2 font-semibold">{entry.key}</td>
                        <td className="px-3 py-2 text-[10px] font-bold">
                          {DIFF_BADGE[entry.status]}
                        </td>
                        <td className="max-w-[220px] break-all px-3 py-2">{entry.beforeText}</td>
                        <td className="max-w-[220px] break-all px-3 py-2">{entry.afterText}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Raw Side-by-Side JSON */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div className="space-y-1.5">
              <div className="text-xs font-semibold text-[var(--text-muted)]">
                Raw `beforeValue`
              </div>
              <pre className="max-h-64 overflow-auto rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-3 font-mono text-xs text-[var(--text-primary)]">
                {prettyJson(event.beforeValue)}
              </pre>
            </div>
            <div className="space-y-1.5">
              <div className="text-xs font-semibold text-[var(--text-muted)]">
                Raw `afterValue`
              </div>
              <pre className="max-h-64 overflow-auto rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-3 font-mono text-xs text-[var(--text-primary)]">
                {prettyJson(event.afterValue)}
              </pre>
            </div>
          </div>
        </div>
      ) : null}
    </SideDrawer>
  );
}
