import { useMemo, useState } from 'react';
import { Download, Eye, FileJson, FilterX, ShieldCheck, Users } from 'lucide-react';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { formatDateTime } from '../../lib/decimal';
import { exportToCsv, exportToJson } from '../../lib/export';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import { AuditDiffDrawer } from './AuditDiffDrawer';
import {
  useAuditEventsQuery,
  type AuditEventSummary,
  type AuditFilterState,
} from './audit-api';

interface AuditWorkspaceProps {
  initialFilters?: Partial<AuditFilterState>;
  onFiltersChange?: (next: AuditFilterState) => void;
}

const DEFAULT_FILTERS: AuditFilterState = {
  entityType: '',
  action: '',
  actor: '',
  dateFrom: '',
  dateTo: '',
  page: 1,
  size: 25,
};

export function AuditWorkspace({ initialFilters, onFiltersChange }: AuditWorkspaceProps) {
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <AuditWorkspaceInner
        orgId={orgId}
        initialFilters={initialFilters}
        onFiltersChange={onFiltersChange}
      />
    </DomainErrorBoundary>
  );
}

interface AuditWorkspaceInnerProps extends AuditWorkspaceProps {
  orgId: string | undefined;
}

function AuditWorkspaceInner({
  orgId,
  initialFilters,
  onFiltersChange,
}: AuditWorkspaceInnerProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';

  const [filters, setFilters] = useState<AuditFilterState>({
    ...DEFAULT_FILTERS,
    ...initialFilters,
  });
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);

  const patchFilters = (patch: Partial<AuditFilterState>) => {
    setFilters((prev) => {
      const next = { ...prev, ...patch };
      onFiltersChange?.(next);
      return next;
    });
  };

  const eventsQuery = useAuditEventsQuery(orgId, {
    page: filters.page,
    size: filters.size,
    entityType: filters.entityType,
    action: filters.action,
    from: filters.dateFrom || undefined,
    to: filters.dateTo || undefined,
  });

  const events = useMemo(() => eventsQuery.data?.items ?? [], [eventsQuery.data]);
  const meta = eventsQuery.data?.page;

  const filteredEvents = useMemo(() => {
    const actorQ = filters.actor.trim().toLowerCase();
    const fromTs = filters.dateFrom ? Date.parse(`${filters.dateFrom}T00:00:00.000Z`) : null;
    const toTs = filters.dateTo ? Date.parse(`${filters.dateTo}T23:59:59.999Z`) : null;

    return events.filter((ev) => {
      if (actorQ && !(ev.actorId ?? 'system').toLowerCase().includes(actorQ)) {
        return false;
      }
      if (fromTs !== null || toTs !== null) {
        const evTs = Date.parse(ev.occurredAt);
        if (!Number.isNaN(evTs)) {
          if (fromTs !== null && evTs < fromTs) return false;
          if (toTs !== null && evTs > toTs) return false;
        }
      }
      return true;
    });
  }, [events, filters.actor, filters.dateFrom, filters.dateTo]);

  const distinctActors = useMemo(() => {
    const set = new Set<string>();
    for (const ev of filteredEvents) {
      set.add(ev.actorId ?? 'system');
    }
    return set.size;
  }, [filteredEvents]);

  const distinctEntities = useMemo(() => {
    const set = new Set<string>();
    for (const ev of filteredEvents) {
      set.add(ev.entityType);
    }
    return set.size;
  }, [filteredEvents]);

  const columns = useMemo<ColumnDef<AuditEventSummary>[]>(
    () => [
      {
        accessorKey: 'occurredAt',
        header: isId ? 'Waktu Kejadian' : 'Occurred At',
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-secondary)]">
            {formatDateTime(row.occurredAt)}
          </span>
        ),
      },
      {
        accessorKey: 'entityType',
        header: isId ? 'Entitas' : 'Entity Type',
        cell: (row) => (
          <div>
            <div className="font-mono text-xs font-semibold text-[var(--text-primary)]">
              {row.entityType}
            </div>
            {row.entityId && (
              <div className="font-mono text-[11px] text-[var(--text-muted)]">{row.entityId}</div>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'action',
        header: isId ? 'Aksi' : 'Action',
        cell: (row) => <StatusBadge status={row.action} />,
      },
      {
        accessorKey: 'actorId',
        header: isId ? 'Aktor' : 'Actor ID',
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-secondary)]">
            {row.actorId ?? 'system'}
          </span>
        ),
      },
      {
        accessorKey: 'correlationId',
        header: 'Correlation ID',
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-muted)]">
            {row.correlationId ?? '—'}
          </span>
        ),
      },
      {
        header: isId ? 'Forensik' : 'Inspect',
        cell: (row) => (
          <div onClick={(e) => e.stopPropagation()}>
            <Button size="sm" variant="outline" onClick={() => setSelectedEventId(row.id)}>
              <Eye className="mr-1 h-3.5 w-3.5" />
              Diff
            </Button>
          </div>
        ),
      },
    ],
    [isId],
  );

  const handleExportCsv = () => {
    exportToCsv(
      filteredEvents,
      [
        { header: 'Occurred At', accessor: (r) => r.occurredAt },
        { header: 'Entity Type', accessor: (r) => r.entityType },
        { header: 'Entity ID', accessor: (r) => r.entityId ?? '' },
        { header: 'Action', accessor: (r) => r.action },
        { header: 'Actor ID', accessor: (r) => r.actorId ?? 'system' },
        { header: 'Correlation ID', accessor: (r) => r.correlationId ?? '' },
      ],
      'forensic-audit-events.csv',
    );
  };

  const handleExportJson = () => {
    exportToJson(filteredEvents, 'forensic-audit-events.json');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--text-primary)]">
            {isId ? 'Log Audit Forensik (Immutable)' : 'Forensic Audit Log'}
          </h1>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">
            {isId
              ? 'Jejak audit lintas layanan dengan diff JSON sebelum/sesudah dan pelacakan correlationId.'
              : 'Immutable cross-service audit ledger with before/after JSON diffing and correlation tracing.'}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" size="sm" onClick={handleExportCsv}>
            <Download className="mr-1.5 h-4 w-4" />
            Export CSV
          </Button>
          <Button variant="secondary" size="sm" onClick={handleExportJson}>
            <FileJson className="mr-1.5 h-4 w-4" />
            Export JSON
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard
          label={isId ? 'Total Event Tercatat' : 'Recorded Audit Events'}
          value={String(meta?.totalElements ?? filteredEvents.length)}
          accent="emerald"
          icon={<ShieldCheck className="h-4 w-4" />}
        />
        <KpiCard
          label={isId ? 'Aktor Unik (Halaman)' : 'Distinct Actors (Page)'}
          value={String(distinctActors)}
          accent="sky"
          icon={<Users className="h-4 w-4" />}
        />
        <KpiCard
          label={isId ? 'Tipe Entitas (Halaman)' : 'Entity Types (Page)'}
          value={String(distinctEntities)}
          accent="brand"
        />
      </div>

      {/* Forensic Filter Bar */}
      <div className="grid grid-cols-1 gap-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 sm:grid-cols-2 lg:grid-cols-6">
        <Input
          aria-label="Entity Type Filter"
          value={filters.entityType}
          onChange={(e) => patchFilters({ entityType: e.target.value, page: 1 })}
          placeholder="Entity Type (e.g. Employee)"
        />
        <Input
          aria-label="Action Filter"
          value={filters.action}
          onChange={(e) => patchFilters({ action: e.target.value, page: 1 })}
          placeholder="Action (e.g. POSTED)"
        />
        <Input
          aria-label="Actor Filter"
          value={filters.actor}
          onChange={(e) => patchFilters({ actor: e.target.value })}
          placeholder="Actor ID..."
        />
        <Input
          aria-label="Date From"
          type="date"
          value={filters.dateFrom}
          onChange={(e) => patchFilters({ dateFrom: e.target.value })}
        />
        <Input
          aria-label="Date To"
          type="date"
          value={filters.dateTo}
          onChange={(e) => patchFilters({ dateTo: e.target.value })}
        />
        <Button
          variant="outline"
          onClick={() =>
            patchFilters({
              ...DEFAULT_FILTERS,
              size: filters.size,
            })
          }
        >
          <FilterX className="mr-1.5 h-4 w-4" />
          Reset
        </Button>
      </div>

      <VirtualizedTable
        data={filteredEvents}
        columns={columns}
        getRowId={(row) => row.id}
        isLoading={eventsQuery.isLoading}
        error={eventsQuery.error}
        onRetry={() => void eventsQuery.refetch()}
        onRowClick={(row) => setSelectedEventId(row.id)}
        emptyTitle={isId ? 'Tidak ada event audit ditemukan' : 'No audit events match filters'}
        emptyDescription={
          isId
            ? 'Sesuaikan filter entityType, action, actor, atau rentang tanggal.'
            : 'Adjust entityType, action, actor, or date range filters.'
        }
        onResetFilters={() =>
          patchFilters({
            ...DEFAULT_FILTERS,
            size: filters.size,
          })
        }
        pagination={{
          page: filters.page,
          size: filters.size,
          totalElements: meta?.totalElements ?? filteredEvents.length,
          totalPages: Math.max(1, meta?.totalPages ?? 1),
          pageBase: 1,
          onPageChange: (nextPage) => patchFilters({ page: nextPage }),
        }}
      />

      <AuditDiffDrawer
        open={Boolean(selectedEventId)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setSelectedEventId(null);
        }}
        eventId={selectedEventId}
        orgId={orgId}
      />
    </div>
  );
}
