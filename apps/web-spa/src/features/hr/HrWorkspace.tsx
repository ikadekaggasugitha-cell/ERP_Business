import { useMemo, useState } from 'react';
import { Briefcase, Building2, Plus, UserCheck, UserPlus, Users, UserX } from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { extractApiFieldErrors } from '../../lib/api';
import { formatDate } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { EmployeeDetailDrawer } from './EmployeeDetailDrawer';
import { EmployeeFormDialog } from './EmployeeFormDialog';
import { PositionFormDialog } from './PositionFormDialog';
import {
  useEmployeesQuery,
  usePositionsQuery,
  useTerminateEmployeeMutation,
  type HrEmployee,
  type HrPosition,
} from './hr-api';
import { useHrText } from './hr-i18n';

export function HrWorkspace() {
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;
  const currencyCode = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, 'hr:write');

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <HrWorkspaceInner orgId={orgId} currencyCode={currencyCode} canWrite={canWrite} />
    </DomainErrorBoundary>
  );
}

interface HrWorkspaceInnerProps {
  orgId: string | undefined;
  currencyCode: string;
  canWrite: boolean;
}

function HrWorkspaceInner({ orgId, currencyCode, canWrite }: HrWorkspaceInnerProps) {
  const txt = useHrText();
  const [activeTab, setActiveTab] = useState('employees');
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [nationalIdLookup, setNationalIdLookup] = useState('');
  const [searchTerm, setSearchTerm] = useState('');

  const [posPage, setPosPage] = useState(1);
  const [posSearch, setPosSearch] = useState('');

  const [employeeDialogOpen, setEmployeeDialogOpen] = useState(false);
  const [editingEmployee, setEditingEmployee] = useState<HrEmployee | null>(null);
  const [drawerEmployeeId, setDrawerEmployeeId] = useState<string | null>(null);
  const [terminateTarget, setTerminateTarget] = useState<HrEmployee | null>(null);
  const [positionDialogOpen, setPositionDialogOpen] = useState(false);

  const employeesQuery = useEmployeesQuery(orgId, {
    page,
    size: 25,
    status: statusFilter,
    nationalId: nationalIdLookup,
  });

  const positionsQuery = usePositionsQuery(orgId, posPage, 100);
  const terminateMutation = useTerminateEmployeeMutation();

  const employees = useMemo(() => employeesQuery.data?.items ?? [], [employeesQuery.data]);
  const employeeMeta = employeesQuery.data?.page;
  const positions = useMemo(() => positionsQuery.data?.items ?? [], [positionsQuery.data]);
  const positionMeta = positionsQuery.data?.page;

  const filteredEmployees = useMemo(() => {
    const q = searchTerm.trim().toLowerCase();
    if (!q) return employees;
    return employees.filter(
      (e) =>
        e.fullName.toLowerCase().includes(q) ||
        e.employeeNo.toLowerCase().includes(q) ||
        (e.email ?? '').toLowerCase().includes(q),
    );
  }, [employees, searchTerm]);

  const filteredPositions = useMemo(() => {
    const q = posSearch.trim().toLowerCase();
    if (!q) return positions;
    return positions.filter(
      (p) => p.code.toLowerCase().includes(q) || p.title.toLowerCase().includes(q),
    );
  }, [positions, posSearch]);

  const positionById = useMemo(() => {
    const map = new Map<string, HrPosition>();
    for (const p of positions) map.set(p.id, p);
    return map;
  }, [positions]);

  const activeCountOnPage = useMemo(
    () => employees.filter((e) => e.status.toUpperCase() === 'ACTIVE').length,
    [employees],
  );
  const terminatedCountOnPage = useMemo(
    () => employees.filter((e) => e.status.toUpperCase() === 'TERMINATED').length,
    [employees],
  );

  const handleConfirmTerminate = async () => {
    if (!terminateTarget) return;
    try {
      await terminateMutation.mutateAsync({
        id: terminateTarget.id,
        rowVersion: terminateTarget.rowVersion,
      });
      setTerminateTarget(null);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const employeeColumns = useMemo<ColumnDef<HrEmployee>[]>(
    () => [
      {
        accessorKey: 'employeeNo',
        header: txt.colEmployeeNo,
        cell: (row) => (
          <span className="font-mono text-xs font-medium text-[var(--text-primary)]">
            {row.employeeNo}
          </span>
        ),
      },
      {
        accessorKey: 'fullName',
        header: txt.colFullName,
        cell: (row) => (
          <div>
            <div className="font-medium text-[var(--text-primary)]">{row.fullName}</div>
            {row.email && <div className="text-xs text-[var(--text-muted)]">{row.email}</div>}
          </div>
        ),
      },
      {
        accessorKey: 'nationalIdMasked',
        header: txt.colNationalId,
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-secondary)]">
            {row.nationalIdMasked ?? '—'}
          </span>
        ),
      },
      {
        accessorKey: 'hireDate',
        header: txt.colHireDate,
        cell: (row) => (
          <span className="text-xs text-[var(--text-secondary)]">{formatDate(row.hireDate)}</span>
        ),
      },
      {
        accessorKey: 'status',
        header: txt.colStatus,
        cell: (row) => <StatusBadge status={row.status} />,
      },
      {
        header: txt.colActions,
        cell: (emp) => {
          const isTerminated = emp.status.toUpperCase() === 'TERMINATED';
          return (
            <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
              <Button size="sm" variant="outline" onClick={() => setDrawerEmployeeId(emp.id)}>
                {txt.viewDetails}
              </Button>
              {canWrite && !isTerminated && (
                <Button size="sm" variant="danger" onClick={() => setTerminateTarget(emp)}>
                  {txt.terminate}
                </Button>
              )}
            </div>
          );
        },
      },
    ],
    [canWrite, txt],
  );

  const positionColumns = useMemo<ColumnDef<HrPosition>[]>(
    () => [
      {
        accessorKey: 'code',
        header: txt.colPositionCode,
        cell: (row) => (
          <span className="font-mono text-xs font-semibold text-[var(--text-primary)]">
            {row.code}
          </span>
        ),
      },
      {
        accessorKey: 'title',
        header: txt.colPositionTitle,
      },
      {
        accessorKey: 'parentPositionId',
        header: txt.colParentPosition,
        cell: (row) => {
          const parent = row.parentPositionId ? positionById.get(row.parentPositionId) : undefined;
          return (
            <span className="text-xs text-[var(--text-secondary)]">
              {parent ? `${parent.code} — ${parent.title}` : row.parentPositionId ?? '—'}
            </span>
          );
        },
      },
      {
        accessorKey: 'organizationRefId',
        header: txt.colOrgRef,
        cell: (row) => (
          <span className="font-mono text-xs text-[var(--text-muted)]">
            {row.organizationRefId}
          </span>
        ),
      },
    ],
    [positionById, txt],
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
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="secondary" onClick={() => setPositionDialogOpen(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              {txt.addPosition}
            </Button>
            <Button
              onClick={() => {
                setEditingEmployee(null);
                setEmployeeDialogOpen(true);
              }}
            >
              <UserPlus className="mr-1.5 h-4 w-4" />
              {txt.addEmployee}
            </Button>
          </div>
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
          label={txt.kpiTotalEmployees}
          value={String(employeeMeta?.totalElements ?? employees.length)}
          icon={<Users className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiActiveEmployees}
          value={String(activeCountOnPage)}
          accent="emerald"
          icon={<UserCheck className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiTerminated}
          value={String(terminatedCountOnPage)}
          accent="rose"
          icon={<UserX className="h-4 w-4" />}
        />
        <KpiCard
          label={txt.kpiPositions}
          value={String(positionMeta?.totalElements ?? positions.length)}
          accent="sky"
          icon={<Building2 className="h-4 w-4" />}
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="employees">
            <Users className="mr-1.5 h-4 w-4" />
            {txt.tabEmployees}
          </TabsTrigger>
          <TabsTrigger value="positions">
            <Briefcase className="mr-1.5 h-4 w-4" />
            {txt.tabPositions}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="employees" className="mt-4">
          <VirtualizedTable
            data={filteredEmployees}
            columns={employeeColumns}
            getRowId={(row) => row.id}
            isLoading={employeesQuery.isLoading}
            error={employeesQuery.error}
            onRetry={() => void employeesQuery.refetch()}
            onRowClick={(emp) => setDrawerEmployeeId(emp.id)}
            emptyTitle={txt.emptyEmployeesTitle}
            emptyDescription={txt.emptyEmployeesDesc}
            onResetFilters={() => {
              setStatusFilter('ALL');
              setNationalIdLookup('');
              setSearchTerm('');
              setPage(1);
            }}
            pagination={{
              page,
              size: 25,
              totalElements: employeeMeta?.totalElements ?? filteredEmployees.length,
              totalPages: Math.max(1, employeeMeta?.totalPages ?? 1),
              pageBase: 1,
              onPageChange: setPage,
            }}
            toolbar={{
              searchValue: searchTerm,
              onSearchChange: setSearchTerm,
              searchPlaceholder: txt.searchPlaceholder,
              onExportCsv: () =>
                exportToCsv(
                  filteredEmployees,
                  [
                    { header: txt.colEmployeeNo, accessor: (r) => r.employeeNo },
                    { header: txt.colFullName, accessor: (r) => r.fullName },
                    { header: txt.colNationalId, accessor: (r) => r.nationalIdMasked ?? '' },
                    { header: txt.colHireDate, accessor: (r) => r.hireDate },
                    { header: txt.colStatus, accessor: (r) => r.status },
                  ],
                  'hr-employees.csv',
                ),
              filters: (
                <div className="flex flex-wrap items-center gap-2">
                  <div className="w-40">
                    <Select
                      value={statusFilter}
                      onValueChange={(next) => {
                        setStatusFilter(next);
                        setPage(1);
                      }}
                      options={[
                        { value: 'ALL', label: txt.statusAll },
                        { value: 'Active', label: txt.statusActive },
                        { value: 'Terminated', label: txt.statusTerminated },
                      ]}
                    />
                  </div>
                  <div className="w-56">
                    <Input
                      value={nationalIdLookup}
                      onChange={(e) => {
                        setNationalIdLookup(e.target.value);
                        setPage(1);
                      }}
                      placeholder={txt.nationalIdFilterPlaceholder}
                    />
                  </div>
                </div>
              ),
            }}
          />
        </TabsContent>

        <TabsContent value="positions" className="mt-4">
          <VirtualizedTable
            data={filteredPositions}
            columns={positionColumns}
            getRowId={(row) => row.id}
            isLoading={positionsQuery.isLoading}
            error={positionsQuery.error}
            onRetry={() => void positionsQuery.refetch()}
            emptyTitle={txt.emptyPositionsTitle}
            emptyDescription={txt.emptyPositionsDesc}
            onResetFilters={() => {
              setPosSearch('');
              setPosPage(1);
            }}
            pagination={{
              page: posPage,
              size: 100,
              totalElements: positionMeta?.totalElements ?? filteredPositions.length,
              totalPages: Math.max(1, positionMeta?.totalPages ?? 1),
              pageBase: 1,
              onPageChange: setPosPage,
            }}
            toolbar={{
              searchValue: posSearch,
              onSearchChange: setPosSearch,
              searchPlaceholder: 'Search positions...',
              onExportCsv: () =>
                exportToCsv(
                  filteredPositions,
                  [
                    { header: txt.colPositionCode, accessor: (r) => r.code },
                    { header: txt.colPositionTitle, accessor: (r) => r.title },
                    { header: txt.colOrgRef, accessor: (r) => r.organizationRefId },
                  ],
                  'hr-positions.csv',
                ),
            }}
          />
        </TabsContent>
      </Tabs>

      <EmployeeFormDialog
        open={employeeDialogOpen}
        onOpenChange={(nextOpen) => {
          setEmployeeDialogOpen(nextOpen);
          if (!nextOpen) setEditingEmployee(null);
        }}
        employee={editingEmployee}
      />

      <PositionFormDialog
        open={positionDialogOpen}
        onOpenChange={setPositionDialogOpen}
        positions={positions}
        defaultOrgId={orgId}
      />

      <EmployeeDetailDrawer
        open={Boolean(drawerEmployeeId)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDrawerEmployeeId(null);
        }}
        employeeId={drawerEmployeeId}
        orgId={orgId}
        currencyCode={currencyCode}
        positions={positions}
        canWrite={canWrite}
        onEdit={(emp) => {
          setEditingEmployee(emp);
          setEmployeeDialogOpen(true);
        }}
        onRequestTerminate={(emp) => setTerminateTarget(emp)}
      />

      <AppDialog
        open={Boolean(terminateTarget)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setTerminateTarget(null);
        }}
        title={txt.terminateTitle}
        description={txt.terminateDescription}
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setTerminateTarget(null)}
              disabled={terminateMutation.isPending}
            >
              {txt.cancel}
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmTerminate}
              disabled={terminateMutation.isPending}
            >
              {txt.terminateConfirm}
            </Button>
          </div>
        }
      >
        {terminateTarget && (
          <div className="rounded-lg border border-rose-500/20 bg-rose-500/5 p-3 text-sm">
            <div className="font-semibold text-[var(--text-primary)]">
              {terminateTarget.fullName}
            </div>
            <div className="font-mono text-xs text-[var(--text-secondary)]">
              {terminateTarget.employeeNo} · v{terminateTarget.rowVersion}
            </div>
          </div>
        )}
      </AppDialog>
    </div>
  );
}
