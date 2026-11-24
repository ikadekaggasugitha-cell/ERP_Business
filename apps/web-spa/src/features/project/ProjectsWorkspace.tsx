import * as React from 'react';
import { FolderKanban, Plus, Receipt } from 'lucide-react';
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
import { addDecimals, formatCurrency, formatDate } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useApproveProjectBillingMutation,
  useCreateProjectMutation,
  useProjectBillingsQuery,
  useProjectsQuery,
  useProjectWbsQuery,
  useTransitionProjectMutation,
  type ProjectBillingDto,
  type ProjectDto,
} from './project-api';

export function ProjectsWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const orgId = activeOrg?.id;
  const currency = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, ['project:write', 'project.create']);

  const [tab, setTab] = React.useState('projects');
  const [statusFilter, setStatusFilter] = React.useState('ALL');
  const [selectedProject, setSelectedProject] = React.useState<ProjectDto | null>(null);
  const [createOpen, setCreateOpen] = React.useState(false);

  const [code, setCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [projectType, setProjectType] = React.useState('Construction');
  const [startDate, setStartDate] = React.useState(new Date().toISOString().slice(0, 10));
  const [endDate, setEndDate] = React.useState('');

  const projectsQ = useProjectsQuery(orgId, statusFilter);
  const wbsQ = useProjectWbsQuery(orgId, selectedProject?.id);
  const billingsQ = useProjectBillingsQuery(orgId);

  const createProject = useCreateProjectMutation();
  const transitionProject = useTransitionProjectMutation();
  const approveBilling = useApproveProjectBillingMutation();

  const projects = projectsQ.data?.items ?? [];
  const billings = billingsQ.data?.items ?? [];

  const totalBilling = React.useMemo(
    () => billings.reduce((acc, b) => addDecimals(acc, b.amount, 4), '0'),
    [billings]
  );

  const projectCols: ColumnDef<ProjectDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '130px', className: 'font-mono font-semibold' },
    { header: 'Project Name', accessorKey: 'name' },
    { header: 'Type', accessorKey: 'projectType', width: '140px' },
    { header: 'Start Date', width: '130px', cell: (r) => (r.startDate ? formatDate(r.startDate, locale) : '—') },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
  ];

  const billingCols: ColumnDef<ProjectBillingDto>[] = [
    { header: 'Billing Type', accessorKey: 'billingType', width: '140px', className: 'font-semibold' },
    { header: 'AR Invoice Ref', width: '160px', cell: (r) => <span className="font-mono">{r.arInvoiceRef ?? '—'}</span> },
    {
      header: 'Amount',
      align: 'right',
      width: '160px',
      cell: (r) => <span className="font-mono tabular-nums">{formatCurrency(r.amount, currency, locale)}</span>,
    },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '120px',
      cell: (r) =>
        canWrite && r.status === 'Draft' ? (
          <Button
            size="sm"
            variant="primary"
            onClick={(e) => {
              e.stopPropagation();
              void approveBilling.mutateAsync({ id: r.id, rowVersion: r.rowVersion }).then(
                (res) => toast.success(`Billing approved (${res.arInvoiceRef})`),
                (err) => toast.error(err instanceof Error ? err.message : 'Approve failed')
              );
            }}
          >
            Approve
          </Button>
        ) : null,
    },
  ];

  const handleCreateProject = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createProject.mutateAsync({
        code: code.trim(),
        name: name.trim(),
        projectType,
        currencyCode: currency,
        startDate: startDate || undefined,
        endDate: endDate || undefined,
      });
      toast.success('Project created');
      setCreateOpen(false);
      setCode('');
      setName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Create failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Operations' }, { label: 'Projects & WBS' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Project Management, WBS & Progress Billing
          </h1>
        </div>
        {canWrite ? (
          <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateOpen(true)}>
            New Project
          </Button>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Total Projects" value={projects.length} icon={<FolderKanban className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Active Projects" value={projects.filter((p) => p.status === 'Active').length} icon={<FolderKanban className="h-4 w-4" />} accent="emerald" />
        <KpiCard
          label="Total Project Billings"
          value={<span className="font-mono tabular-nums">{formatCurrency(totalBilling, currency, locale)}</span>}
          icon={<Receipt className="h-4 w-4" />}
          accent="sky"
        />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="projects">Projects ({projects.length})</TabsTrigger>
          <TabsTrigger value="billings">Billings ({billings.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="projects" className="pt-4">
          <VirtualizedTable
            data={projects}
            columns={projectCols}
            isLoading={projectsQ.isLoading}
            error={projectsQ.error}
            onRetry={() => void projectsQ.refetch()}
            onRowClick={setSelectedProject}
            emptyTitle="No projects yet"
            toolbar={{
              filters: (
                <Select
                  value={statusFilter}
                  onValueChange={setStatusFilter}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Draft', label: 'Draft' },
                    { value: 'Active', label: 'Active' },
                    { value: 'OnHold', label: 'On Hold' },
                    { value: 'Closed', label: 'Closed' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
        <TabsContent value="billings" className="pt-4">
          <VirtualizedTable data={billings} columns={billingCols} isLoading={billingsQ.isLoading} error={billingsQ.error} onRetry={() => void billingsQ.refetch()} emptyTitle="No project billings" />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedProject)}
        onOpenChange={(open) => !open && setSelectedProject(null)}
        title={selectedProject ? `${selectedProject.code} · ${selectedProject.name}` : 'Project Detail'}
        titleBadge={selectedProject ? <StatusBadge status={selectedProject.status.toUpperCase()} label={selectedProject.status} /> : null}
        footer={
          selectedProject && canWrite ? (
            <div className="flex gap-2 justify-end">
              {selectedProject.status === 'Draft' ? (
                <Button
                  size="sm"
                  variant="primary"
                  onClick={() =>
                    void transitionProject.mutateAsync({ id: selectedProject.id, action: 'activate', rowVersion: selectedProject.rowVersion }).then(
                      () => {
                        toast.success('Project activated');
                        setSelectedProject(null);
                      },
                      (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                    )
                  }
                >
                  Activate
                </Button>
              ) : null}
              {selectedProject.status === 'Active' ? (
                <>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() =>
                      void transitionProject.mutateAsync({ id: selectedProject.id, action: 'hold', rowVersion: selectedProject.rowVersion }).then(
                        () => {
                          toast.success('Project placed on hold');
                          setSelectedProject(null);
                        },
                        (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                      )
                    }
                  >
                    Hold
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() =>
                      void transitionProject.mutateAsync({ id: selectedProject.id, action: 'close', rowVersion: selectedProject.rowVersion }).then(
                        () => {
                          toast.success('Project closed');
                          setSelectedProject(null);
                        },
                        (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                      )
                    }
                  >
                    Close
                  </Button>
                </>
              ) : null}
            </div>
          ) : null
        }
      >
        <div className="space-y-3 text-xs">
          <h3 className="font-bold uppercase tracking-wider text-slate-500">Work Breakdown Structure (WBS)</h3>
          {(wbsQ.data ?? []).length === 0 ? (
            <p className="text-slate-500">No WBS elements defined yet.</p>
          ) : (
            (wbsQ.data ?? []).map((w) => (
              <div
                key={w.id}
                style={{ paddingLeft: `${Math.max(0, w.levelNo - 1) * 16 + 12}px` }}
                className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-800 py-2 pr-3"
              >
                <span className="font-mono font-semibold">{w.code}</span>
                <span className="flex-1 px-2 truncate">{w.name}</span>
                <span className="text-slate-400 font-mono">L{w.levelNo}</span>
              </div>
            ))
          )}
        </div>
      </SideDrawer>

      <AppDialog open={createOpen} onOpenChange={setCreateOpen} title="Create Project" size="md">
        <form onSubmit={handleCreateProject} className="space-y-4">
          <FormField label="Project Code" htmlFor="prjCode" required>
            <Input id="prjCode" value={code} onChange={(e) => setCode(e.target.value)} placeholder="PRJ-2026-01" required />
          </FormField>
          <FormField label="Project Name" htmlFor="prjName" required>
            <Input id="prjName" value={name} onChange={(e) => setName(e.target.value)} placeholder="Sinergi Tower B Fit-Out" required />
          </FormField>
          <FormField label="Project Type" htmlFor="prjType">
            <Select
              id="prjType"
              value={projectType}
              onValueChange={setProjectType}
              options={[
                { value: 'Construction', label: 'Construction' },
                { value: 'InternalCapex', label: 'Internal Capex' },
                { value: 'ClientService', label: 'Client Service' },
              ]}
            />
          </FormField>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Start Date" htmlFor="prjStart">
              <Input id="prjStart" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </FormField>
            <FormField label="End Date" htmlFor="prjEnd">
              <Input id="prjEnd" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
            </FormField>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createProject.isPending}>Create</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
