import { useMemo, useState } from 'react';
import { Calculator, CheckCircle2, FileSpreadsheet, Plus, Send } from 'lucide-react';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { exportToCsv } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { CreatePayrollRunDialog } from './CreatePayrollRunDialog';
import { PayrollRunDrawer } from './PayrollRunDrawer';
import { usePayrollRunsQuery, type PayrollRun } from './payroll-api';
import { formatPayrollPeriod, usePayrollText } from './payroll-i18n';

export function PayrollWorkspace() {
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;
  const currencyCode = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, 'payroll:write');

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <PayrollWorkspaceInner orgId={orgId} currencyCode={currencyCode} canWrite={canWrite} />
    </DomainErrorBoundary>
  );
}

interface PayrollWorkspaceInnerProps {
  orgId: string | undefined;
  currencyCode: string;
  canWrite: boolean;
}

function PayrollWorkspaceInner({ orgId, currencyCode, canWrite }: PayrollWorkspaceInnerProps) {
  const txt = usePayrollText();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchTerm, setSearchTerm] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [drawerRunId, setDrawerRunId] = useState<string | null>(null);

  const runsQuery = usePayrollRunsQuery(orgId, {
    page,
    size: 25,
    status: statusFilter,
  });

  const runs = useMemo(() => runsQuery.data?.items ?? [], [runsQuery.data]);
  const meta = runsQuery.data?.page;

  const filteredRuns = useMemo(() => {
    const q = searchTerm.trim().toLowerCase();
    if (!q) return runs;
    return runs.filter(
      (r) =>
        r.runNo.toLowerCase().includes(q) ||
        (r.glJournalRef ?? '').toLowerCase().includes(q) ||
        formatPayrollPeriod(r.periodMonth, r.periodYear).includes(q),
    );
  }, [runs, searchTerm]);

  const draftCount = useMemo(
    () => runs.filter((r) => r.status.toUpperCase() === 'DRAFT').length,
    [runs],
  );
  const readyCount = useMemo(
    () =>
      runs.filter(
        (r) => r.status.toUpperCase() === 'CALCULATED' || r.status.toUpperCase() === 'APPROVED',
      ).length,
    [runs],
  );
  const postedCount = useMemo(
    () => runs.filter((r) => r.status.toUpperCase() === 'POSTED').length,
    [runs],
  );

  const columns = useMemo<ColumnDef<PayrollRun>[]>(
    () => [
      {
        accessorKey: 'runNo',
        header: txt.colRunNo,
        cell: (row) => (
          <span className="font-mono text-xs font-semibold text-[var(--text-primary)]">
            {row.runNo}
          </span>
        ),
      },
      {
        header: txt.colPeriod,
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-secondary)]">
            {formatPayrollPeriod(row.periodMonth, row.periodYear)}
          </span>
        ),
      },
      {
        accessorKey: 'status',
        header: txt.colStatus,
        cell: (row) => <StatusBadge status={row.status} />,
      },
      {
        accessorKey: 'glJournalRef',
        header: txt.colGlRef,
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-secondary)]">
            {row.glJournalRef ?? '—'}
          </span>
        ),
      },
      {
        accessorKey: 'rowVersion',
        header: txt.colVersion,
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-muted)]">v{row.rowVersion}</span>
        ),
      },
      {
        header: txt.colActions,
        cell: (row) => (
          <div onClick={(e) => e.stopPropagation()}>
            <Button size="sm" variant="outline" onClick={() => setDrawerRunId(row.id)}>
              {txt.inspect}
            </Button>
          </div>
        ),
      },
    ],
    [txt],
  );

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
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-1.5 h-4 w-4" />
            {txt.newRun}
          </Button>
        )}
      </div>

      {!canWrite && (
        <div
          role="status"
          className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-2.5 text-xs font-medium text-amber-800 dark:text-amber-300"
        >
          {txt.readOnlyHint}
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label={txt.kpiTotalRuns}
          value={String(meta?.totalElements ?? runs.length)}
          icon={<FileSpreadsheet className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiDraft}
          value={String(draftCount)}
          accent="amber"
          icon={<Calculator className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiCalculatedApproved}
          value={String(readyCount)}
          accent="sky"
          icon={<CheckCircle2 className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiPosted}
          value={String(postedCount)}
          accent="emerald"
          icon={<Send className="h-4 w-4" />}
        />
      </div>

      <VirtualizedTable
        data={filteredRuns}
        columns={columns}
        getRowId={(row) => row.id}
        isLoading={runsQuery.isLoading}
        error={runsQuery.error}
        onRetry={() => void runsQuery.refetch()}
        onRowClick={(row) => setDrawerRunId(row.id)}
        emptyTitle={txt.emptyTitle}
        emptyDescription={txt.emptyDesc}
        onResetFilters={() => {
          setStatusFilter('ALL');
          setSearchTerm('');
          setPage(1);
        }}
        pagination={{
          page,
          size: 25,
          totalElements: meta?.totalElements ?? filteredRuns.length,
          totalPages: Math.max(1, meta?.totalPages ?? 1),
          pageBase: 1,
          onPageChange: setPage,
        }}
        toolbar={{
          searchValue: searchTerm,
          onSearchChange: setSearchTerm,
          searchPlaceholder: txt.searchPlaceholder,
          onExportCsv: () =>
            exportToCsv(
              filteredRuns,
              [
                { header: txt.colRunNo, accessor: (r) => r.runNo },
                {
                  header: txt.colPeriod,
                  accessor: (r) => formatPayrollPeriod(r.periodMonth, r.periodYear),
                },
                { header: txt.colStatus, accessor: (r) => r.status },
                { header: txt.colGlRef, accessor: (r) => r.glJournalRef ?? '' },
              ],
              'payroll-runs.csv',
            ),
          filters: (
            <div className="w-44">
              <Select
                value={statusFilter}
                onValueChange={(next) => {
                  setStatusFilter(next);
                  setPage(1);
                }}
                options={[
                  { value: 'ALL', label: txt.statusAll },
                  { value: 'Draft', label: 'Draft' },
                  { value: 'Calculated', label: 'Calculated' },
                  { value: 'Approved', label: 'Approved' },
                  { value: 'Posted', label: 'Posted' },
                ]}
              />
            </div>
          ),
        }}
      />

      <CreatePayrollRunDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(id) => setDrawerRunId(id)}
      />

      <PayrollRunDrawer
        open={Boolean(drawerRunId)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDrawerRunId(null);
        }}
        runId={drawerRunId}
        orgId={orgId}
        currencyCode={currencyCode}
        canWrite={canWrite}
      />
    </div>
  );
}
